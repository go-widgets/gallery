// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import "github.com/go-widgets/webcanvas"

// galleryApp adapts the gallery's *state to the shared [webcanvas.App]
// interface, so the DOM harness in internal/webcanvas can drive it without any
// gallery-specific knowledge. Each method forwards one-to-one to the matching
// *state handler, preserving the exact behaviour the old inline main() had:
// the mapping is a rename, not a rewrite, so the page renders and routes
// identically. It also implements [webcanvas.Ticker] to keep the 60-Hz toast
// countdown the gallery relies on.
//
// This adapter is tag-less (compiled and tested natively) even though only the
// `js && wasm` main() constructs it, so the native scene test can assert the
// forwarding is correct and the App/Ticker contracts are satisfied.
type galleryApp struct{ s *state }

// newGalleryApp builds the gallery scene and wraps it as an App.
func newGalleryApp() galleryApp { return galleryApp{s: newState(surfaceW, surfaceH)} }

// Size reports the gallery's computed surface dimensions.
func (g galleryApp) Size() (int, int) { return surfaceW, surfaceH }

// Draw paints the whole gallery scene into buf.
func (g galleryApp) Draw(buf []byte) { g.s.draw(buf) }

// Click forwards a primary press.
func (g galleryApp) Click(x, y int) bool { return g.s.handleClick(x, y) }

// Move forwards a pointer move (drag tick or chart hover).
func (g galleryApp) Move(x, y int) bool { return g.s.handleMove(x, y) }

// Release forwards the primary release.
func (g galleryApp) Release(x, y int) bool { return g.s.handleRelease(x, y) }

// Context forwards a secondary press (opens the edit menu).
func (g galleryApp) Context(x, y int) bool { return g.s.handleContext(x, y) }

// Char forwards a printable rune to the focused widget.
func (g galleryApp) Char(s string) bool { return g.s.handleChar(s) }

// KeyDown forwards a named/modified key to the focused widget.
func (g galleryApp) KeyDown(s string) bool { return g.s.handleKeyDown(s) }

// Scroll forwards a wheel / trackpad scroll (dy vertical, dx horizontal ROWS)
// at (x, y) to the scrollable widget under the pointer. Implementing
// [webcanvas.Scroller] is what makes the harness install the wheel listener for
// the gallery, so the mouse wheel scrolls the dashboard's ListBoxes, tables and
// trees instead of the page.
func (g galleryApp) Scroll(x, y, dx, dy int) bool { return g.s.handleScroll(x, y, dx, dy) }

// Tick advances the notification countdown one frame.
func (g galleryApp) Tick() { g.s.tick() }

// Compile-time proof the adapter satisfies every contract on the native build:
// the base App, the Ticker (toast countdown) and the Scroller (wheel routing).
var (
	_ webcanvas.App      = galleryApp{}
	_ webcanvas.Ticker   = galleryApp{}
	_ webcanvas.Scroller = galleryApp{}
)
