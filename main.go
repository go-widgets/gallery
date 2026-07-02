// Command gallery is a browser wasm live-demo of the go-widgets/toolkit
// widget set. Runs on a plain <canvas> (no wasmbox, no SharedArrayBuffer,
// no server-side dep) so it drops into any static site — see
// index.html for the host shell.
//
// Layout: a MenuBar + Toolbar strip at the top, a Notebook of five tabs
// exercising the widget families (Buttons, Input, List+Tree, Feedback,
// Composites), a Statusbar at the bottom, and a Notification toast
// that fires on every menu-item click. Purely visual: real click
// handlers just show the notification with the widget name so the user
// can eyeball what fired.
//
//go:build js && wasm

package main

import (
	"syscall/js"
)

func main() {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "screen")
	if canvas.IsUndefined() || canvas.IsNull() {
		println("gallery: no #screen canvas in the host page")
		return
	}
	canvas.Set("width", surfaceW)
	canvas.Set("height", surfaceH)
	ctx := canvas.Call("getContext", "2d")

	local := make([]byte, 4*surfaceW*surfaceH)
	imageData := ctx.Call("createImageData", surfaceW, surfaceH)
	dst := imageData.Get("data")

	state := newState(surfaceW, surfaceH)

	render := func() {
		state.draw(local)
		js.CopyBytesToJS(dst, local)
		ctx.Call("putImageData", imageData, 0, 0)
	}
	render()

	// Mouse dispatch: use the canvas bounding-rect to convert
	// clientX/Y to canvas-local pixel coords. This is where a real
	// event loop would sit; keeping it inline for clarity.
	cb := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		ev := args[0]
		rect := canvas.Call("getBoundingClientRect")
		sx := rect.Get("width").Float() / float64(surfaceW)
		sy := rect.Get("height").Float() / float64(surfaceH)
		x := int((ev.Get("clientX").Float() - rect.Get("left").Float()) / sx)
		y := int((ev.Get("clientY").Float() - rect.Get("top").Float()) / sy)
		if state.handleClick(x, y) {
			render()
		}
		return nil
	})
	canvas.Call("addEventListener", "click", cb)

	// A 60-Hz tick drives the Notification.Life countdown so toasts
	// auto-hide. setInterval doesn't need a huge slice of allocs
	// because Tick is a pure counter-decrement.
	tick := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		state.tick()
		render()
		return nil
	})
	js.Global().Call("setInterval", tick, 16)

	// Park forever so the callbacks live.
	select {}
}
