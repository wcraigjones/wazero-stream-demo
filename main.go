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
	"runtime/debug"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

//go:embed plugin/plugin.wasm
var plugin []byte

func main() {
	debug.SetMemoryLimit(2 * 1024 * 1024 * 1024)
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	pluginFS := &PluginFS{
		inFiles:  make(map[string]*PluginFile),
		outFiles: make(map[string]*PluginFile),
	}

	config := wazero.
		NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin)
		// WithFS(pluginFS)

	// Instantiate WASI, which implements system I/O such as console output.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, plugin)
	if err != nil {
		log.Panicln(err)
	}

	mod, err := r.InstantiateModule(ctx, code, config)
	if err != nil {
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintf(os.Stderr, "exit_code: %d\n", exitErr.ExitCode())
		} else if !ok {
			log.Panicln(err)
		}
	}

	testWG := sync.WaitGroup{}
	ids := map[string]*io.PipeReader{}
	seed := map[string][]byte{}
	qID := 0

	workers := 2
	queues := make([][]string, workers)
	for i := 0; i < workers; i++ {
		queues[i] = make([]string, 0)
	}

	for i := 0; i < 10; i++ {
		randBytes := make([]byte, 4096)
		rand.Read(randBytes)
		fIn := bytes.NewBuffer(randBytes)
		id := fmt.Sprintf("%x", rand.Int63())

		queues[qID%workers] = append(queues[qID%workers], id)
		qID += 1

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

	do := mod.ExportedFunction("do")
	// These are undocumented, but exported. See tinygo-org/tinygo#2788
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")

	execWG := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		execWG.Add(1)
		go func(id int) {
			for _, streamID := range queues[id] {
				// Let's use the argument to this main function in Wasm.
				streamSize := uint64(len(streamID))

				// Instead of an arbitrary memory offset, use TinyGo's allocator. Notice
				// there is nothing string-specific in this allocation function. The same
				// function could be used to pass binary serialized data to Wasm.
				results, err := malloc.Call(ctx, streamSize)
				if err != nil {
					log.Panicln(err)
				}
				namePtr := results[0]
				// This pointer is managed by TinyGo, but TinyGo is unaware of external usage.
				// So, we have to free it when finished
				defer free.Call(ctx, namePtr)

				// The pointer is a linear memory offset, which is where we write the name.
				if !mod.Memory().Write(ctx, uint32(namePtr), []byte(streamID)) {
					log.Panicf("Memory.Write(%d, %d) out of range of memory size %d",
						namePtr, streamSize, mod.Memory().Size(ctx))
				}

				// Now, we can call "greet", which reads the string we wrote to memory!
				_, err = do.Call(ctx, namePtr, streamSize)
				if err != nil {
					log.Panicln(err)
				}
			}

			execWG.Done()
		}(i)
	}

	fmt.Println("Prepared.")
	start := time.Now()
	for id, fHost := range ids {
		randBytes := seed[id]
		testWG.Add(1)
		go func(id string, fHost *io.PipeReader) {
			defer testWG.Done()
			data, err := io.ReadAll(fHost)
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(randBytes, data) {
				fmt.Println("Error data mismatch!")
			}
		}(id, fHost)
	}

	testWG.Wait()
	execWG.Wait()
	fmt.Println("Processed:", time.Since(start))
}
