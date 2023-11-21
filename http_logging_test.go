package log // import "fortio.org/fortio/log"

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// There is additional functional testing in fortio.org/fortio/fhttp.
func TestLogRequest(t *testing.T) {
	SetLogLevel(Verbose) // make sure it's already debug when we capture
	Config.LogFileAndLine = false
	Config.JSON = true
	Config.NoTimestamp = true
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	h := http.Header{"foo": []string{"bar1", "bar2"}}
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "x\nyz"}} // make sure special chars are escaped
	r := &http.Request{TLS: &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}, Header: h, Host: "foo-host:123"}
	LogRequest(r, "test1")
	r.TLS = nil
	r.Header = nil
	LogRequest(r, "test2", Str("extra1", "v1"), Str("extra2", "v2"))
	w.Flush()
	actual := b.String()
	//nolint: lll
	expected := `{"level":"info","msg":"test1","method":"","url":null,"proto":"","remote_addr":"","host":"foo-host:123","header.x-forwarded-proto":"","header.x-forwarded-for":"","user-agent":"","tls":true,"tls.peer_cn":"x\nyz","header.foo":"bar1,bar2"}
{"level":"info","msg":"test2","method":"","url":null,"proto":"","remote_addr":"","host":"foo-host:123","header.x-forwarded-proto":"","header.x-forwarded-for":"","user-agent":"","extra1":"v1","extra2":"v2"}
`
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL != nil && r.URL.Path == "/tea" {
		w.WriteHeader(http.StatusTeapot)
	}
	w.Write([]byte("hello"))
	time.Sleep(100 * time.Millisecond)
}

type NullHTTPWriter struct {
	doErr   bool
	doPanic bool
}

func (n *NullHTTPWriter) Header() http.Header {
	return nil
}

func (n *NullHTTPWriter) Write(b []byte) (int, error) {
	if n.doPanic {
		panic("some fake http write panic")
	}
	if n.doErr {
		return 0, fmt.Errorf("some fake http write error")
	}
	return len(b), nil
}

func (n *NullHTTPWriter) WriteHeader(_ int) {}

func TestLogAndCall(t *testing.T) {
	Config.LogFileAndLine = false
	Config.JSON = true
	Config.NoTimestamp = true
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	hr := &http.Request{}
	n := &NullHTTPWriter{}
	hw := &ResponseRecorder{w: n}
	LogAndCall("test-log-and-call", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual := b.String()
	//nolint: lll
	expectedPrefix := `{"level":"info","msg":"test-log-and-call","method":"","url":null,"proto":"","remote_addr":"","host":"","header.x-forwarded-proto":"","header.x-forwarded-for":"","user-agent":""}
{"level":"info","msg":"test-log-and-call","status":200,"size":5,"microsec":1` // the 1 is for the 100ms sleep
	if !strings.HasPrefix(actual, expectedPrefix) {
		t.Errorf("unexpected:\n%s\nvs should start with:\n%s\n", actual, expectedPrefix)
	}
	if hw.Header() != nil {
		t.Errorf("unexpected non nil header: %v", hw.Header())
	}
	hr.URL = &url.URL{Path: "/tea"}
	b.Reset()
	Config.CombineRequestAndResponse = true // single line test
	LogAndCall("test-log-and-call2", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual = b.String()
	//nolint: lll
	expectedPrefix = `{"level":"info","msg":"test-log-and-call2","method":"","url":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/tea","RawPath":"","OmitHost":false,"ForceQuery":false,"RawQuery":"","Fragment":"","RawFragment":""},"proto":"","remote_addr":"","host":"","header.x-forwarded-proto":"","header.x-forwarded-for":"","user-agent":"","status":418,"size":5,"microsec":1`
	if !strings.HasPrefix(actual, expectedPrefix) {
		t.Errorf("unexpected:\n%s\nvs should start with:\n%s\n", actual, expectedPrefix)
	}
	b.Reset()
	n.doErr = true
	LogAndCall("test-log-and-call3", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual = b.String()
	expectedFragment := `"user-agent":"","status":500,"size":0,"microsec":1`
	if !strings.Contains(actual, expectedFragment) {
		t.Errorf("unexpected:\n%s\nvs should contain error:\n%s\n", actual, expectedFragment)
	}
	n.doPanic = true
	n.doErr = false
	b.Reset()
	LogAndCall("test-log-and-call4", testHandler).ServeHTTP(hw, hr)
	w.Flush()
	actual = b.String()
	expectedFragment = `,"size":0,`
	Config.GoroutineID = false
	if !strings.Contains(actual, expectedFragment) {
		t.Errorf("unexpected:\n%s\nvs should contain error:\n%s\n", actual, expectedFragment)
	}
	if !strings.Contains(actual, `{"level":"crit","msg":"panic in handler","error":"some fake http write panic"`) {
		t.Errorf("unexpected:\n%s\nvs should contain error:\n%s\n", actual, "some fake http write panic")
	}
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
