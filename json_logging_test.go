//go:build !no_json
// +build !no_json

package log // import "fortio.org/fortio/log"

import (
	"fmt"
	"testing"
)

func TestToJSON_MarshalError(t *testing.T) {
	badValue := make(chan int)

	expected := fmt.Sprintf("\"ERR marshaling %v: %v\"", badValue, "JSON: unsupported type: chan int")
	actual := toJSON(badValue)

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}
