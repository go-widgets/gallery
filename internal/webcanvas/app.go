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

import (
	"fmt"
	"os"
	"runtime/debug"
)

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

// Animator is an optional companion to [App]: a scene with time-varying content
// driven by a REAL wall clock (procedurally animated icons, say) implements it,
// and [Run] installs a requestAnimationFrame loop that hands it the elapsed dt
// between frames — in seconds — through AnimationStep. Unlike [Ticker] (a fixed
// cadence that always repaints), an Animator advances by the true frame delta and
// reports whether the frame changed anything, so Run repaints only when a pixel
// actually moved. A scene that implements neither installs no clock and repaints
// on input alone. The phase-advance logic lives in the scene (natively testable);
// only the rAF wiring is browser-side.
type Animator interface {
	// AnimationStep advances the scene's animation by dt seconds of real elapsed
	// time and reports whether the scene now needs a repaint.
	AnimationStep(dt float64) (repaint bool)
}

// Resizer is an optional companion to [App]: a scene that can adapt its layout to
// a NEW surface size implements it, and [Run] installs a window "resize" listener
// (and fits the canvas to the viewport once at startup) that re-sizes the canvas
// and framebuffer, calls Resize, and repaints. A scene that omits it keeps the
// fixed [App.Size] forever — the pre-resize behaviour every existing demo relies
// on — so Run never installs the listener and the surface never changes.
//
// Resize is handed the target pixel size (the canvas' laid-out client box) and
// returns the size it will actually render at: a scene may clamp to a sane
// minimum, and the host allocates the framebuffer from the RETURNED size, so the
// scene and the buffer can never disagree. The relayout logic lives in the scene
// (natively testable); only the DOM resize wiring is browser-side.
type Resizer interface {
	// Resize relays out the scene to fit w×h device pixels and returns the pixel
	// size (rw, rh) it will render at — the size the host sizes the canvas and
	// framebuffer to.
	Resize(w, h int) (rw, rh int)
}

// PanicReporter logs a handler panic the [guard] net recovered. The wasm host
// swaps in a console.error reporter carrying the JS stack; the default writes to
// standard error so a native run — and the recover test — still surfaces it. It
// is a package var, not a const, precisely so run_js.go can replace it.
var PanicReporter = func(r any, stack []byte) {
	fmt.Fprintf(os.Stderr, "webcanvas: recovered panic in a handler: %v\n%s\n", r, stack)
}

// guard runs fn under a recover net: a panic escaping a single event handler (or
// the paint it triggers) is caught, reported through [PanicReporter] with its
// stack, and swallowed — so ONE bad frame logs an error instead of tearing the
// whole wasm instance off the page, and the next dispatch still runs. It reports
// whether fn panicked so a test can prove the net fired. It is deliberately a
// last-resort net IN ADDITION to the toolkit's own per-widget hardening, never a
// substitute for fixing the upstream bug.
func guard(fn func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			PanicReporter(r, debug.Stack())
		}
	}()
	fn()
	return false
}
