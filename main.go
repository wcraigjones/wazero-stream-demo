package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/psanford/memfs"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

//go:embed plugin/plugin.wasm
var plugin []byte

func main() {
	// Choose the context to use for function calls.
	ctx := context.Background()

	// Create a new WebAssembly Runtime.
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx) // This closes everything this Runtime created.

	rootFS := memfs.New()
	err := rootFS.WriteFile("test", bytes.NewBufferString("hello world\n").Bytes(), fs.FileMode(os.O_RDONLY))
	if err != nil {
		panic(err)
	}

	// Combine the above into our baseline config, overriding defaults.
	config := wazero.NewModuleConfig().
		// By default, I/O streams are discarded and there's no file system.
		WithStdout(os.Stdout).WithStderr(os.Stderr).WithFS(rootFS)

	// Instantiate WASI, which implements system I/O such as console output.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, plugin)
	if err != nil {
		log.Panicln(err)
	}

	// InstantiateModule runs the "_start" function, WASI's "main".
	// * Set the program name (arg[0]) to "wasi"; arg[1] should be "/test.txt".
	if _, err = r.InstantiateModule(ctx, code, config.WithArgs("wasi")); err != nil {

		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintf(os.Stderr, "exit_code: %d\n", exitErr.ExitCode())
		} else if !ok {
			log.Panicln(err)
		}
	}
}
