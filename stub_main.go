// The native build without a tag. The gallery's two entrypoints are main.go
// (js && wasm, the live demo) and native_main.go (gallerynative, the native
// controls gallery); with neither tag a `package main` has no `main` and
// `go build ./...` fails to link on a native host -- which CI never saw, as
// it builds this package for js/wasm and with the gallerynative tag only.
//
// The body is empty on purpose: it holds no statement, so the coverage gate
// has nothing here to count.
//
//go:build !(js && wasm) && !gallerynative

package main

func main() {}
