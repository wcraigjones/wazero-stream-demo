package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

//go:embed plugin/plugin.wasm
var plugin []byte

type Runtime struct {
	R      wazero.Runtime
	Mod    api.Module
	malloc api.Function
	do     api.Function
}

func New() *Runtime {
	ctx := context.TODO()
	r := wazero.NewRuntime(ctx)
	config := wazero.
		NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
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
	do := mod.ExportedFunction("do")
	if do == nil {
		panic("nil do")
	}
	malloc := mod.ExportedFunction("my_malloc")
	if malloc == nil {
		panic("nil malloc")
	}

	return &Runtime{
		R:      r,
		Mod:    mod,
		malloc: malloc,
		do:     do,
	}
}

func (r *Runtime) Do(s string) {
	ctx := context.Background()
	strSize := uint64(len(s))
	results, err := r.malloc.Call(ctx, strSize)
	if err != nil {
		fmt.Println("malloc error")
		log.Panicln(err)
	}
	strPtr := results[0]
	fmt.Println("write")
	if !r.Mod.Memory().Write(ctx, uint32(strPtr), []byte(s)) {
		log.Panicf("Memory.Write(%d, %d) out of range of memory size %d",
			strPtr, strSize, r.Mod.Memory().Size(ctx))
	}
	fmt.Printf("alloc done: %d %x\n", strSize, strPtr)

	_, err = r.do.Call(ctx, strPtr, strSize)
	if err != nil {
		log.Panicln(err)
	}
	fmt.Println("call done")
}

func main() {
	fmt.Println("making queues")

	qID := 0
	workers := 1
	work := 100000

	queues := make([][]string, workers)
	for i := 0; i < workers; i++ {
		queues[i] = make([]string, 0)
	}

	for i := 0; i < work; i++ {
		id := fmt.Sprintf("%x", rand.Int63())
		queues[qID%workers] = append(queues[qID%workers], id)
		qID += 1
	}

	execWG := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		execWG.Add(1)
		go func(id int) {
			r := New()
			fmt.Printf("worker %d starting\n", id)
			for _, streamID := range queues[id] {
				r.Do(streamID)
			}
			execWG.Done()
		}(i)
	}

	start := time.Now()
	fmt.Println("Prepared.")
	execWG.Wait()
	fmt.Println("Processed:", time.Since(start))
}
