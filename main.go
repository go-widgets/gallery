// Command gallery is a browser wasm live-demo of the go-widgets/toolkit
// widget set. Runs on a plain <canvas> (no wasmbox, no SharedArrayBuffer,
// no server-side dep) so it drops into any static site — see index.html for
// the host shell.
//
// Layout: a MenuBar + Toolbar strip at the top, a Notebook of tabs exercising
// the widget families, a Statusbar at the bottom, and a Notification toast
// that fires on every menu-item click.
//
// The DOM plumbing — canvas blit + pointer/keyboard routing — lives in the
// shared internal/webcanvas harness, which every demo in this module reuses;
// this command only builds the scene (via newGalleryApp, an [webcanvas.App]
// adapter over *state) and hands it to webcanvas.Run.
//
//go:build js && wasm

package main

import "github.com/go-widgets/webcanvas"

func main() {
	webcanvas.Run("screen", newGalleryApp())
}
