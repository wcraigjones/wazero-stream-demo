package main

import (
	"fmt"
	"os"
	"reflect"
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
	name := ptrToString(ptr, size)
	do(name)
}

func do(id string) {
	fmt.Fprintln(os.Stdin, "[module]", id)
	// reader, err := os.OpenFile(
	// 	fmt.Sprintf("out/%v", id),
	// 	os.O_RDONLY,
	// 	0,
	// )
	// if err != nil {
	// 	fmt.Printf("error opening reader: %v\n", err)
	// 	return
	// }
	// defer reader.Close()

	// writer, err := os.OpenFile(
	// 	fmt.Sprintf("in/%v", id),
	// 	os.O_WRONLY,
	// 	0444,
	// )
	// if err != nil {
	// 	fmt.Printf("error opening writer: %v\n", err)
	// 	return
	// }
	// defer writer.Close()

	// data, err := io.ReadAll(reader)
	// if err != nil {
	// 	fmt.Printf("error reading: %v\n", err)
	// 	return
	// }

	// _, err = writer.Write(data)
	// if err != nil {
	// 	fmt.Printf("error writing: %v\n", err)
	// 	return
	// }
}

// ptrToString returns a string from WebAssembly compatible numeric types
// representing its pointer and length.
func ptrToString(ptr uint32, size uint32) string {
	// Get a slice view of the underlying bytes in the stream. We use SliceHeader, not StringHeader
	// as it allows us to fix the capacity to what was allocated.
	return *(*string)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(ptr),
		Len:  uintptr(size), // Tinygo requires these as uintptrs even if they are int fields.
		Cap:  uintptr(size), // ^^ See https://github.com/tinygo-org/tinygo/issues/1284
	}))
}
