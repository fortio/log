//go:build tinygo

package goroutine

import (
	"sync"
	"unsafe"
)

const IsTinyGo = true

var (
	counter int64
	mapping = make(map[uintptr]int64)
	lock    sync.Mutex
)

func ID() int64 {
	task := uintptr(currentTask())
	lock.Lock()
	if id, ok := mapping[task]; ok {
		lock.Unlock()
		return id
	}
	counter++
	mapping[task] = counter
	lock.Unlock()
	return counter
	// or, super fast but ugly large numbers/pointers:
	//return int64(uintptr(currentTask()))
}

//go:linkname currentTask internal/task.Current
func currentTask() unsafe.Pointer
