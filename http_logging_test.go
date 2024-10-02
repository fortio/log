//go:build !no_http && !no_net
// +build !no_http,!no_net

package log // import "fortio.org/fortio/log"

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// There is additional functional testing in fortio.org/fortio/fhttp.
func TestLogRequest(t *testing.T) {
	SetLogLevel(Verbose) // make sure it's already verbose when we capture
	Config.LogFileAndLine = false
	Config.JSON = true
	Config.NoTimestamp = true
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	h := http.Header{"FoO": []string{"bar1", "bar2"}, "X-Forwarded-Host": []string{"fOO.fortio.org"}}
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "x\nyz"}} // make sure special chars are escaped
	r := &http.Request{TLS: &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}, Header: h, Host: "foo-host:123"}
	LogRequest(r, "test1")
	r.TLS = nil
	r.Header = nil
	LogRequest(r, "test2", Str("extra1", "v1"), Str("extra2", "v2"))
	w.Flush()
	actual := b.String()
	//nolint: lll
	expected := `{"level":"info","msg":"test1","method":"","url":null,"host":"foo-host:123","proto":"","remote_addr":"","tls":true,"tls.peer_cn":"x\nyz","header.foo":"bar1,bar2","header.x-forwarded-host":"fOO.fortio.org"}
{"level":"info","msg":"test2","method":"","url":null,"host":"foo-host:123","proto":"","remote_addr":"","extra1":"v1","extra2":"v2"}
`
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL != nil && r.URL.Path == "/panicbefore" {
		panic("some test handler panic before response")
	}
	if r.URL != nil && r.URL.Path == "/tea" {
		w.WriteHeader(http.StatusTeapot)
	}
	w.Write([]byte("hello"))
	time.Sleep(100 * time.Millisecond)
	if r.URL != nil && r.URL.Path == "/panicafter" {
		panic("some test handler panic after response")
	}
}

type NullHTTPWriter struct {
	doErr   bool
	headers http.Header
}

func (n *NullHTTPWriter) Header() http.Header {
	if n.headers == nil {
		n.headers = make(http.Header)
	}
	return n.headers
}

func (n *NullHTTPWriter) Write(b []byte) (int, error) {
	if n.doErr {
		return 0, errors.New("some fake http write error")
	}
	return len(b), nil
}

// Also implement http.Flusher interface.
func (n *NullHTTPWriter) Flush() {
}

func (n *NullHTTPWriter) WriteHeader(_ int) {}

func TestLogAndCall(t *testing.T) {
	Config.LogFileAndLine = true // yet won't show up in output
	Config.JSON = true
	Config.NoTimestamp = true
	Config.CombineRequestAndResponse = false // Separate request and response logging
	SetLogLevelQuiet(Info)
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	hr := &http.Request{}
	hr.Header = http.Header{"foo": []string{"bar1", "bar2"}, "X-Forwarded-Host": []string{"foo2.fortio.org"}}
	n := &NullHTTPWriter{}
	hw := &ResponseRecorder{w: n}
	LogAndCall("test-log-and-call", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual := b.String()
	//nolint: lll
	expectedPrefix := `{"level":"info","msg":"test-log-and-call","method":"","url":null,"host":"","proto":"","remote_addr":"","header.x-forwarded-host":"foo2.fortio.org"}
{"level":"info","msg":"test-log-and-call","status":200,"size":5,"microsec":1` // the 1 is for the 100ms sleep
	if !strings.HasPrefix(actual, expectedPrefix) {
		t.Errorf("unexpected:\n%s\nvs should start with:\n%s\n", actual, expectedPrefix)
	}
	if len(hw.Header()) != 0 {
		t.Errorf("unexpected non nil header: %v", hw.Header())
	}
	hr.URL = &url.URL{Path: "/tea"}
	b.Reset()
	Config.CombineRequestAndResponse = true // Combined logging test
	LogAndCall("test-log-and-call2", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual = b.String()
	//nolint: lll
	expectedPrefix = `{"level":"info","msg":"test-log-and-call2","method":"","url":"/tea","host":"","proto":"","remote_addr":"","header.x-forwarded-host":"foo2.fortio.org","status":418,"size":5,"microsec":10`
	if !strings.HasPrefix(actual, expectedPrefix) {
		t.Errorf("unexpected:\n%s\nvs should start with:\n%s\n", actual, expectedPrefix)
	}
	b.Reset()
	n.doErr = true
	LogAndCall("test-log-and-call3", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual = b.String()
	expectedFragment := `"header.x-forwarded-host":"foo2.fortio.org","status":500,"size":0,"microsec":1`
	if !strings.Contains(actual, expectedFragment) {
		t.Errorf("unexpected:\n%s\nvs should contain error:\n%s\n", actual, expectedFragment)
	}
	hr.URL = &url.URL{Path: "/panicafter"}
	n.doErr = false
	SetLogLevelQuiet(Verbose)
	b.Reset()
	LogAndCall("test-log-and-call4", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual = b.String()
	expectedFragment = `"status":-500,`
	Config.GoroutineID = false
	if !strings.Contains(actual, expectedFragment) {
		t.Errorf("unexpected:\n%s\nvs should contain error:\n%s\n", actual, expectedFragment)
	}
	if !strings.Contains(actual, `{"level":"crit","msg":"panic in handler","error":"some test handler panic after response"}`) {
		t.Errorf("unexpected:\n%s\nvs should contain error:\n%s\n", actual, "some test handler panic after response")
	}
	hr.URL = &url.URL{Path: "/panicbefore"}
	b.Reset()
	LogAndCall("test-log-and-call5", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual = b.String()
	expectedFragment = `"status":-500,`
	Config.GoroutineID = false
	if !strings.Contains(actual, expectedFragment) {
		t.Errorf("unexpected:\n%s\nvs should contain error:\n%s\n", actual, expectedFragment)
	}
	if !strings.Contains(actual, `{"level":"crit","msg":"panic in handler","error":"some test handler panic before response"}`) {
		t.Errorf("unexpected:\n%s\nvs should contain error:\n%s\n", actual, "some test handler panic before response")
	}
	// restore for other tests
	Config.GoroutineID = true
	// check for flusher interface
	var hwi http.ResponseWriter = hw
	flusher, ok := hwi.(http.Flusher)
	if !ok {
		t.Fatalf("expected http.ResponseWriter to be an http.Flusher")
	}
	flusher.Flush()
}

func TestLogResponseOnHTTPResponse(t *testing.T) {
	SetLogLevel(Info)
	Config.LogFileAndLine = false
	Config.JSON = true
	Config.NoTimestamp = true
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	r := &http.Response{StatusCode: http.StatusTeapot, ContentLength: 123}
	LogResponse(r, "test1")
	w.Flush()
	actual := b.String()
	expected := `{"level":"info","msg":"test1","status":418,"size":123}
`
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
	SetLogLevelQuiet(Warning)
	b.Reset()
	LogResponse(r, "test2")
	w.Flush()
	actual = b.String()
	if actual != "" {
		t.Errorf("unexpected: %q", actual)
	}
	SetLogLevelQuiet(Verbose)
}

func TestLogRequestNoLog(t *testing.T) {
	SetLogLevel(Warning) // make sure it's already debug when we capture
	Config.LogFileAndLine = false
	Config.JSON = true
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	r := &http.Request{}
	LogRequest(r, "test1")
	w.Flush()
	actual := b.String()
	if actual != "" {
		t.Errorf("unexpected: %q", actual)
	}
}

// Test for the "old" TLSInfo.
func TestTLSInfo(t *testing.T) {
	h := http.Header{"foo": []string{"bar"}}
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "x\nyz"}} // make sure special chars are escaped
	r := &http.Request{TLS: &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}, Header: h}
	got := TLSInfo(r)
	expected := " https 0x0000 \"CN=x\\nyz\""
	if got != expected {
		t.Errorf("unexpected for tls:\n%s\nvs:\n%s\n", got, expected)
	}
	r.TLS = nil
	got = TLSInfo(r)
	expected = ""
	if got != expected {
		t.Errorf("unexpected for no tls:\n%s\nvs:\n%s\n", got, expected)
	}
}

func TestInterceptStandardLogger(t *testing.T) {
	SetLogLevel(Warning)
	Config.LogFileAndLine = true
	Config.JSON = false // check that despite this, it'll be json anyway (so it doesn't go infinite loop)
	Config.NoTimestamp = true
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	InterceptStandardLogger(Warning)
	log.Printf("\n\na test\n\n")
	w.Flush()
	actual := b.String()
	expected := `{"level":"warn","msg":"a test","src":"std"}` + "\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func TestNewStdLogger(t *testing.T) {
	SetLogLevel(Info)
	Config.LogFileAndLine = true
	Config.JSON = false
	Config.NoTimestamp = true
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	logger := NewStdLogger("test src", Info)
	logger.Printf("\n\nanother test\n\n")
	w.Flush()
	actual := b.String()
	expected := `{"level":"info","msg":"another test","src":"test src"}` + "\n"
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}
