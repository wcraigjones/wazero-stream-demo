package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

//go:embed plugin/plugin.wasm
var plugin []byte

func main() {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	pluginFS := &PluginFS{
		inFiles:  make(map[string]*PluginFile),
		outFiles: make(map[string]*PluginFile),
	}

	pluginStdinReader, pluginStdinWriter := io.Pipe()

	config := wazero.
		NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(pluginStdinReader).
		WithFS(pluginFS)

	// Instantiate WASI, which implements system I/O such as console output.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, plugin)
	if err != nil {
		log.Panicln(err)
	}

	wasmWG := sync.WaitGroup{}
	wasmWG.Add(1)

	go func() {
		defer wasmWG.Done()
		_, err = r.InstantiateModule(ctx, code, config)
		if err != nil {
			if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
				fmt.Fprintf(os.Stderr, "exit_code: %d\n", exitErr.ExitCode())
			} else if !ok {
				log.Panicln(err)
			}
		}
	}()

	testWG := sync.WaitGroup{}
	ids := map[string]*io.PipeReader{}
	seed := map[string][]byte{}

	for i := 0; i < 100000; i++ {
		randBytes := make([]byte, 4096)
		rand.Read(randBytes)
		fIn := bytes.NewBuffer(randBytes)
		id := fmt.Sprint(rand.Int63())

		fHost, fPlugin := io.Pipe()
		pluginFS.Register(
			id,
			&PluginFile{
				name:   "in",
				mode:   true,
				writer: fPlugin,
			},
			&PluginFile{
				name:   "out",
				mode:   false,
				reader: fIn,
			},
		)

		ids[id] = fHost
		seed[id] = randBytes
	}

	fmt.Println("Prepared.")
	start := time.Now()
	for id, fHost := range ids {
		randBytes := seed[id]
		testWG.Add(1)
		go func() {
			defer testWG.Done()
			data, err := io.ReadAll(fHost)
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(randBytes, data) {
				fmt.Println("Error data mismatch!")
			}
		}()

		pluginStdinWriter.Write([]byte(id + "\n"))
	}

	testWG.Wait()
	fmt.Println("Processed:", time.Since(start))

	pluginStdinWriter.Close()
	wasmWG.Wait()
}
