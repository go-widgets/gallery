// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"testing"

	"github.com/go-widgets/gallery/internal/webcanvas"
)

// TestGalleryAppForwards drives every galleryApp adapter method and checks it
// forwards to the wrapped *state identically to a directly-built state — the
// control that the harness refactor did not change the gallery's behaviour.
func TestGalleryAppForwards(t *testing.T) {
	g := newGalleryApp()

	if w, h := g.Size(); w != surfaceW || h != surfaceH {
		t.Fatalf("Size() = %dx%d, want %dx%d", w, h, surfaceW, surfaceH)
	}

	// Draw must paint the same bytes the wrapped state paints directly.
	ref := newState(surfaceW, surfaceH)
	want := make([]byte, 4*surfaceW*surfaceH)
	ref.draw(want)
	got := make([]byte, 4*surfaceW*surfaceH)
	g.Draw(got)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Draw diverged from state.draw at byte %d: got %d want %d", i, got[i], want[i])
		}
	}

	// Each event method forwards to the matching *state handler and returns its
	// bool. Compared against a freshly-built reference state fed the same input,
	// the adapter's result must match, proving a pure pass-through.
	cmp := func(name string, viaApp, direct bool) {
		if viaApp != direct {
			t.Fatalf("%s: adapter returned %v, state handler %v", name, viaApp, direct)
		}
	}
	// A menu-name click opens a popover on both (same coords, fresh states).
	cmp("Click", g.Click(surfaceW/2, 10), newState(surfaceW, surfaceH).handleClick(surfaceW/2, 10))
	cmp("Move", g.Move(surfaceW/2, surfaceH/2), newState(surfaceW, surfaceH).handleMove(surfaceW/2, surfaceH/2))
	cmp("Release", g.Release(surfaceW/2, surfaceH/2), newState(surfaceW, surfaceH).handleRelease(surfaceW/2, surfaceH/2))
	cmp("Context", g.Context(surfaceW/2, surfaceH/2), newState(surfaceW, surfaceH).handleContext(surfaceW/2, surfaceH/2))
	cmp("Char", g.Char("a"), newState(surfaceW, surfaceH).handleChar("a"))
	cmp("KeyDown", g.KeyDown("Enter"), newState(surfaceW, surfaceH).handleKeyDown("Enter"))
	// Scroll forwards to handleScroll: a wheel over the ListBox scrolls it (true),
	// a wheel over empty space is a no-op (false) — both compared to a fresh state.
	lb := newState(surfaceW, surfaceH).listBox.Bounds()
	cmp("Scroll/list", g.Scroll(lb.X+lb.W/2, lb.Y+lb.H/2, 0, 3),
		newState(surfaceW, surfaceH).handleScroll(lb.X+lb.W/2, lb.Y+lb.H/2, 0, 3))
	cmp("Scroll/empty", g.Scroll(0, 0, 0, 3), newState(surfaceW, surfaceH).handleScroll(0, 0, 0, 3))

	// Tick decrements the toast life; just exercising it must not panic and
	// keeps the wrapped state usable.
	g.Tick()
	g.Draw(got)
}

// TestGalleryAppSatisfiesContracts is a runtime companion to the compile-time
// var _ assertions: the adapter value is usable as both an App and a Ticker.
func TestGalleryAppSatisfiesContracts(t *testing.T) {
	var app webcanvas.App = newGalleryApp()
	if _, ok := app.(webcanvas.Ticker); !ok {
		t.Fatal("galleryApp does not satisfy webcanvas.Ticker")
	}
	if _, ok := app.(webcanvas.Scroller); !ok {
		t.Fatal("galleryApp does not satisfy webcanvas.Scroller")
	}
}
