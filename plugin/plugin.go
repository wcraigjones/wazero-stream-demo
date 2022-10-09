package main

import (
	"hash/adler32"
	"unsafe"
)

func main() {}

//export do
func _do(ptr, size uint32) {
	name := ptrToString(uintptr(ptr), size)
	do(name)
}

func do(id string) {
	h := adler32.New()
	for i := 0; i < 100; i++ {
		h.Sum([]byte(id))
	}
}

// ptrToString returns a string from WebAssembly compatible numeric types
// representing its pointer and length.
func ptrToString(ptr uintptr, size uint32) string {
	return string(alivePointers[ptr])
}

var alivePointers = map[uintptr][]byte{}

//export my_malloc
func my_malloc(size uint32) uintptr {
	buf := make([]byte, size)
	ptr := &buf[0]
	unsafePtr := uintptr(unsafe.Pointer(ptr))
	alivePointers[unsafePtr] = buf
	return unsafePtr
}
