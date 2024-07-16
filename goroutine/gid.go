package goroutine

import (
	"github.com/kortschak/goroutine" // Rely on and forward to the original rather than maintain our own copy.
)

// ID returns the runtime ID of the calling goroutine.
func ID() int64 {
	return goroutine.ID()
}
