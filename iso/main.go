// Command iso is the interactive isometric-diagram wasm demonstrator; see
// scene.go for the full package doc. This file is the only `js && wasm` code:
// it hands a freshly-built scene to the shared internal/webcanvas harness and
// nothing more, so every line of scene logic stays in the natively-tested
// scene.go.
//
//go:build js && wasm

package main

import "github.com/go-widgets/gallery/internal/webcanvas"

func main() {
	webcanvas.Run("screen", newIsoScene())
}
