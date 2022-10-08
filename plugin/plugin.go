package main

import (
	"os"
)

func main() {
	bytes, err := os.ReadFile("/test")
	if err != nil {
		os.Exit(1)
	}

	// Use write to avoid needing to worry about Windows newlines.
	os.Stdout.Write(bytes)
}
