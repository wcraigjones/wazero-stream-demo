package main

import (
	"context"
	_ "embed"
	"fmt"
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

	config := wazero.
		NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin)

	// Instantiate WASI, which implements system I/O such as console output.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	fmt.Println("compiling")
	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, plugin)
	if err != nil {
		log.Panicln(err)
	}

	fmt.Println("instantiating")
	mod, err := r.InstantiateModule(ctx, code, config)
	if err != nil {
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintf(os.Stderr, "exit_code: %d\n", exitErr.ExitCode())
		} else if !ok {
			log.Panicln(err)
		}
	}

	fmt.Println("making queues")

	qID := 0
	workers := 2
	queues := make([][]string, workers)
	for i := 0; i < workers; i++ {
		queues[i] = make([]string, 0)
	}

	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("%x", rand.Int63())
		queues[qID%workers] = append(queues[qID%workers], id)
		qID += 1
	}

	fmt.Println("getting functions")

	do := mod.ExportedFunction("do")
	// These are undocumented, but exported. See tinygo-org/tinygo#2788
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	mallocMu := sync.Mutex{}

	execWG := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		execWG.Add(1)
		go func(id int) {
			fmt.Printf("worker %d starting\n", id)
			for _, streamID := range queues[id] {
				var namePtr, streamSize uint64
				{
					mallocMu.Lock()
					fmt.Println("streamID: ", streamID)
					streamSize = uint64(len(streamID))
					results, err := malloc.Call(ctx, streamSize)
					if err != nil {
						fmt.Println("malloc error")
						log.Panicln(err)
					}
					namePtr = results[0]
					fmt.Println("write")
					if !mod.Memory().Write(ctx, uint32(namePtr), []byte(streamID)) {
						log.Panicf("Memory.Write(%d, %d) out of range of memory size %d",
							namePtr, streamSize, mod.Memory().Size(ctx))
					}
					mallocMu.Unlock()
				}
				fmt.Println("malloc done")
				defer free.Call(ctx, namePtr)
				fmt.Println("call")
				_, err = do.Call(ctx, namePtr, streamSize)
				if err != nil {
					log.Panicln(err)
				}
				fmt.Println("call done")
			}

			execWG.Done()
		}(i)
	}

	start := time.Now()
	fmt.Println("Prepared.")
	execWG.Wait()
	fmt.Println("Processed:", time.Since(start))
}
