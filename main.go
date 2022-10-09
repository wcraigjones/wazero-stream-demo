package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"io/fs"
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

func New(f fs.FS) *Runtime {
	ctx := context.TODO()
	r := wazero.NewRuntime(ctx)
	config := wazero.
		NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithStdin(os.Stdin).
		WithFS(f)
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
	if !r.Mod.Memory().Write(ctx, uint32(strPtr), []byte(s)) {
		log.Panicf("Memory.Write(%d, %d) out of range of memory size %d",
			strPtr, strSize, r.Mod.Memory().Size(ctx))
	}

	_, err = r.do.Call(ctx, strPtr, strSize)
	if err != nil {
		log.Panicln(err)
	}
}

func main() {
	fmt.Println("making queues")

	testWG := sync.WaitGroup{}
	ids := map[string]*io.PipeReader{}
	seed := map[string][]byte{}

	qID := 0
	workers := 25
	work := 100000

	pluginFS := &PluginFS{
		inFiles:  make(map[string]*PluginFile),
		outFiles: make(map[string]*PluginFile),
	}

	queues := make([][]string, workers)
	for i := 0; i < workers; i++ {
		queues[i] = make([]string, 0)
	}

	for i := 0; i < work; i++ {
		id := fmt.Sprintf("%x", rand.Int63())
		queues[qID%workers] = append(queues[qID%workers], id)
		qID += 1

		randBytes := make([]byte, 4096)
		rand.Read(randBytes)
		fIn := bytes.NewBuffer(randBytes)

		fHost, fPlugin := io.Pipe()
		pluginFS.Register(
			id,
			&PluginFile{
				name:   "in",
				mode:   false,
				reader: fIn,
			},
			&PluginFile{
				name:   "out",
				mode:   true,
				writer: fPlugin,
			},
		)

		ids[id] = fHost
		seed[id] = randBytes
	}

	execWG := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		execWG.Add(1)
		go func(id int) {
			r := New(pluginFS)
			fmt.Printf("worker %d starting\n", id)
			for _, streamID := range queues[id] {
				r.Do(streamID)
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
