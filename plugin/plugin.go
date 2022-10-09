package main

import (
	"fmt"
	"io"
	"os"
	"unsafe"
)

func main() {}

//export do
func _do(ptr, size uint32) {
	name := ptrToString(uintptr(ptr), size)
	do(name)
}

func do(id string) {
	reader, err := os.OpenFile(
		fmt.Sprintf("in/%v", id),
		os.O_RDONLY,
		0,
	)
	if err != nil {
		fmt.Printf("error opening reader: %v\n", err)
		return
	}
	defer reader.Close()

	writer, err := os.OpenFile(
		fmt.Sprintf("out/%v", id),
		os.O_WRONLY,
		0444,
	)
	if err != nil {
		fmt.Printf("error opening writer: %v\n", err)
		return
	}
	defer writer.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		fmt.Printf("error reading: %v\n", err)
		return
	}

	_, err = writer.Write(data)
	if err != nil {
		fmt.Printf("error writing: %v\n", err)
		return
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
