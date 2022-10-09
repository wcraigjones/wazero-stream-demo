package main

import (
	"fmt"
	"os"
	"unsafe"
)

func main() {
	// b := make([]byte, 1)
	// for {
	// 	_, err := os.Stdin.Read(b)
	// 	if err != nil {
	// 		break
	// 	}
	// }
}

//export do
func _do(ptr, size uint32) {
	fmt.Fprintln(os.Stdout, "[module]", "loading string")
	name := ptrToString(uintptr(ptr), size)
	fmt.Fprintln(os.Stdout, "[module]", "loaded string")
	do(name)
}

func do(id string) {
	fmt.Fprintln(os.Stdout, "[module]", id)
}

// ptrToString returns a string from WebAssembly compatible numeric types
// representing its pointer and length.
func ptrToString(ptr uintptr, size uint32) string {
	return string(alivePointers[ptr])
}

var alivePointers = map[uintptr][]byte{}

//export my_malloc
func my_malloc(size uint32) uintptr {
	fmt.Fprintln(os.Stdout, "[module]", "malloc start")
	buf := make([]byte, size)
	ptr := &buf[0]
	unsafePtr := uintptr(unsafe.Pointer(ptr))
	alivePointers[unsafePtr] = buf
	fmt.Fprintln(os.Stdout, "[module]", "malloc end")
	return unsafePtr
}
