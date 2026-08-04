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

	// Mouse dispatch: convert clientX/Y to canvas-local pixel coords via the
	// bounding-rect, then route the full press / drag / release lifecycle so
	// draggable widgets (Kanban cards, Gantt bars) work, not just clicks.
	// mousedown is the press (EventClick grabs), mousemove is a drag tick
	// (routed only while a target is captured), mouseup releases.
	coords := func(ev js.Value) (int, int) {
		rect := canvas.Call("getBoundingClientRect")
		sx := rect.Get("width").Float() / float64(surfaceW)
		sy := rect.Get("height").Float() / float64(surfaceH)
		x := int((ev.Get("clientX").Float() - rect.Get("left").Float()) / sx)
		y := int((ev.Get("clientY").Float() - rect.Get("top").Float()) / sy)
		return x, y
	}
	listen := func(name string, handle func(x, y int) bool) {
		cb := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) == 0 {
				return nil
			}
			x, y := coords(args[0])
			if handle(x, y) {
				render()
			}
			return nil
		})
		canvas.Call("addEventListener", name, cb)
	}
	// Left-button press only: a right press is the edit menu's business (below),
	// not a grab/select. mousemove/mouseup carry no button distinction.
	canvas.Call("addEventListener", "mousedown", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 || args[0].Get("button").Int() != 0 {
			return nil
		}
		x, y := coords(args[0])
		if state.handleClick(x, y) {
			render()
		}
		return nil
	}))
	listen("mousemove", state.handleMove) // captured drag, else chart-hover tooltip
	listen("mouseup", state.handleRelease)
	// Right-click opens the edit context menu; suppress the browser's own menu.
	canvas.Call("addEventListener", "contextmenu", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		args[0].Call("preventDefault")
		x, y := coords(args[0])
		if state.handleContext(x, y) {
			render()
		}
		return nil
	}))
	// Keyboard: routed to the focused widget (last clicked) so an Entry or an
	// open grid-cell editor accepts text. Listen on the window (a <canvas> is
	// not focusable by default). A single printable rune with no Ctrl/Meta/Alt
	// is text (EventChar); every named key (Enter, Backspace, Arrow*, …) is an
	// EventKeyDown.
	js.Global().Call("addEventListener", "keydown", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		ev := args[0]
		key := ev.Get("key").String()
		var changed bool
		if len([]rune(key)) == 1 && !ev.Get("ctrlKey").Bool() && !ev.Get("metaKey").Bool() && !ev.Get("altKey").Bool() {
			changed = state.handleChar(key)
		} else {
			changed = state.handleKeyDown(key)
		}
		if changed {
			ev.Call("preventDefault")
			render()
		}
		return nil
	}))

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
