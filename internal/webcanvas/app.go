// Package webcanvas is the gallery's shared browser harness: the generic
// glue that blits a toolkit RGBA framebuffer into a <canvas> and routes DOM
// pointer / keyboard events into a widget scene. It carries no widget logic
// of its own — every demo in this module (the widget gallery, the isometric
// diagram editor, …) implements the small [App] interface and hands it to
// [Run], so the DOM plumbing lives in exactly one place instead of being
// copy-pasted per demo.
//
// The interface is defined here, in a tag-less file, so a native (non-wasm)
// build and its tests can name the type and assert that a scene satisfies it;
// the DOM implementation of [Run] lives in run_js.go behind a `js && wasm`
// build tag, so it drops out of the native build entirely.
package webcanvas

// App is a self-contained canvas scene. A host (the wasm [Run] loop, or a
// native test) owns the pixel buffer and the event source; the App owns the
// widgets. Every coordinate is in canvas-local pixels (top-left origin), the
// same space [Run] derives from the pointer event's position within the
// canvas' bounding rectangle.
//
// Each event method reports whether the scene changed and therefore needs a
// repaint, so the host can skip a redraw when nothing moved. Draw is expected
// to fully paint the buffer (it is never given a dirty region).
type App interface {
	// Size returns the fixed pixel dimensions of the scene's surface. The host
	// sizes the canvas and allocates the framebuffer from it; it is read once,
	// at startup, so it must not change over the App's life.
	Size() (w, h int)

	// Draw paints the whole scene into buf, a width*height*4 RGBA byte slice
	// laid out exactly like an image.RGBA's Pix (row-major, 4 bytes/pixel).
	Draw(buf []byte)

	// Click delivers a primary (left) button press at (x, y). It begins a
	// gesture — a selection, a drag, a placement — that later Move/Release
	// calls advance and commit.
	Click(x, y int) bool

	// Move delivers a pointer move at (x, y). While a Click gesture is in
	// flight it is a drag tick; otherwise it is a hover.
	Move(x, y int) bool

	// Release delivers the primary button release at (x, y), committing any
	// in-flight gesture Click began.
	Release(x, y int) bool

	// Context delivers a secondary (right) button press at (x, y), typically
	// opening a context menu. The host suppresses the browser's own menu.
	Context(x, y int) bool

	// Char delivers a single printable character typed with no Ctrl/Meta/Alt
	// modifier — text input for a focused field.
	Char(s string) bool

	// KeyDown delivers a named key (Enter, Backspace, Delete, Arrow*, …) or a
	// modified key press, routed to the focused widget.
	KeyDown(s string) bool
}

// Ticker is an optional companion to [App]: a scene that needs a steady
// animation clock (a toast countdown, a blinking caret) implements it, and
// [Run] installs a 60-Hz timer that calls Tick and repaints. A scene with no
// time-varying state omits it, and Run installs no timer, so it never repaints
// except in response to input.
type Ticker interface {
	// Tick advances one animation frame. Run repaints after every Tick.
	Tick()
}
