.PHONY: plugin
plugin:
	tinygo build -o plugin/plugin.wasm -scheduler=none --no-debug -target=wasi plugin/plugin.go