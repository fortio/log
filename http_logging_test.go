package log // import "fortio.org/fortio/log"

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"net/http"
	"testing"
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
	expected := `{"level":"info","msg":"test1","method":"","url":"<nil>","proto":"","remote_addr":"","host":"foo-host:123","header.x-forwarded-proto":"","header.x-forwarded-for":"","user-agent":"","tls":"true","tls.peer_cn":"x\nyz","header.foo":"bar1,bar2"}
{"level":"info","msg":"test2","method":"","url":"<nil>","proto":"","remote_addr":"","host":"foo-host:123","header.x-forwarded-proto":"","header.x-forwarded-for":"","user-agent":"","extra1":"v1","extra2":"v2"}
`
	if actual != expected {
		t.Errorf("unexpected:\n%s\nvs:\n%s\n", actual, expected)
	}
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
