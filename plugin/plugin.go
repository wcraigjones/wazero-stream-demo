package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		do(scanner.Text())
	}
}

func do(id string) {
	reader, err := os.OpenFile(
		fmt.Sprintf("out/%v", id),
		os.O_RDONLY,
		0,
	)
	if err != nil {
		fmt.Printf("error opening reader: %v\n", err)
		return
	}
	defer reader.Close()

	writer, err := os.OpenFile(
		fmt.Sprintf("in/%v", id),
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

	fmt.Fprintf(os.Stdout, "processed: %v\n", id)
}
