package log // import "fortio.org/fortio/log"

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"net/http"
	"testing"
)

// leave this test first/where it is as it relies on line number not changing.
// note that the real functional test is in fortio/fhttp. and this is mostly for coverage.
func TestLogRequest(t *testing.T) {
	SetLogLevel(Verbose) // make sure it's already debug when we capture
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	SetOutput(w)
	SetFlags(0) // remove timestamps
	h := http.Header{"foo": []string{"bar"}}
	r := &http.Request{TLS: &tls.ConnectionState{}, Header: h}
	LogRequest(r, "test1")
	r.TLS = nil
	r.Header = nil
	LogRequest(r, "test2")
	w.Flush()
	actual := b.String()
	expected := "test1:  <nil>   ()  \"\" https 0x0000\nHeader Host: \nHeader foo: bar\n" +
		"test2:  <nil>   ()  \"\"\nHeader Host: \n"
	if actual != expected {
		t.Errorf("unexpected:\n%q\nvs:\n%q\n", actual, expected)
	}
}
