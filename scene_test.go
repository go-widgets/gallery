// SPDX-License-Identifier: BSD-3-Clause
//
// scene_test — off-browser tests for the scene composition. main.go
// carries a js && wasm build tag so it drops out on the native test
// host; scene.go stays tagless so this file can exercise it against a
// plain byte buffer.

package main

import (
	"testing"

	"github.com/go-widgets/toolkit"
)

func newSurface() []byte { return make([]byte, 4*surfaceW*surfaceH) }

func TestNewStateAndDraw(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	if s == nil {
		t.Fatal("newState returned nil")
	}
	if s.menuBar == nil || s.toolbar == nil || s.notebook == nil || s.status == nil || s.notify == nil {
		t.Fatal("newState left a core widget nil")
	}
	if len(s.menuBar.Menus) != 4 {
		t.Fatalf("MenuBar expected 4 menus, got %d", len(s.menuBar.Menus))
	}
	if len(s.notebook.Tabs) != 5 {
		t.Fatalf("Notebook expected 5 tabs, got %d", len(s.notebook.Tabs))
	}
	// Draw must paint into every byte of the surface (Background fill).
	surf := newSurface()
	s.draw(surf)
	nonZero := 0
	for _, b := range surf {
		if b != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatal("draw painted 0 non-zero bytes")
	}
}

func TestDrawEveryActiveTab(t *testing.T) {
	// Exercise the per-tab branch of draw() for all 5 tabs.
	s := newState(surfaceW, surfaceH)
	for tab := 0; tab < 5; tab++ {
		s.notebook.Active = tab
		surf := newSurface()
		s.draw(surf) // must not panic
	}
}

func TestDrawWithOpenMenuPopover(t *testing.T) {
	// A click on a menu name toggles Active; the next draw paints the
	// popover on top. Cover that path.
	s := newState(surfaceW, surfaceH)
	// Click on File (top-left).
	if !s.handleClick(10, 6) {
		t.Fatal("handleClick returned false")
	}
	if s.menuBar.Active != 0 {
		t.Fatalf("MenuBar Active after File click: %d, want 0", s.menuBar.Active)
	}
	// Draw with the popover open — exercises the "menu on top" branch.
	s.draw(newSurface())
}

func TestHandleClickToolbarFiresNotification(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Toolbar starts at Y=MenuBarH.
	tbY := toolkit.MenuBarH + toolkit.ToolbarButtonH/2
	// Click on the first toolbar button (x≈12).
	s.handleClick(12, tbY)
	if !s.notify.Visible {
		t.Fatal("toolbar click did not fire a Notification")
	}
	if s.notify.Text == "" {
		t.Fatal("Notification text is empty after toolbar click")
	}
}

func TestHandleClickMenuItemDismissesAndFires(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Open the File menu.
	s.handleClick(10, 6)
	if s.menuBar.Active != 0 {
		t.Fatal("File menu did not open")
	}
	// draw() is what sets the popover's Bounds. Run it once so the
	// popover has real coords before we hit-test into it.
	s.draw(newSurface())
	menu := s.menuBar.Menus[0]
	r := menu.Bounds()
	// Click on the first row of the popover.
	s.handleClick(r.X+r.W/2, r.Y+4+toolkit.MenuRowH/2)
	// After click on the row: menu dismisses + notify shows.
	if s.menuBar.Active != -1 {
		t.Fatalf("menu should dismiss after click; Active=%d", s.menuBar.Active)
	}
	if !s.notify.Visible {
		t.Fatal("menu click should fire the item's Action → Notification")
	}
}

func TestHandleClickOutsideOpenMenuDismisses(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Open the Edit menu (index 1) — click roughly over its position.
	editX := s.menuBar.NameOriginX(1) + s.menuBar.NameWidth(1)/2
	s.handleClick(editX, 6)
	if s.menuBar.Active != 1 {
		t.Fatalf("Edit menu did not open; Active=%d", s.menuBar.Active)
	}
	// Click outside the menu bar + outside the popover — anywhere in
	// the notebook body.
	s.handleClick(surfaceW/2, surfaceH-100)
	if s.menuBar.Active != -1 {
		t.Fatalf("outside click should dismiss menu; Active=%d", s.menuBar.Active)
	}
}

func TestHandleClickNotebookTab(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	bodyY := toolkit.MenuBarH + toolkit.ToolbarButtonH
	// Click on the 3rd tab strip (roughly x = 2*NotebookTabWidth + 20).
	tabX := 2*toolkit.NotebookTabWidth + 20
	tabY := bodyY + toolkit.NotebookTabStripH/2
	s.handleClick(tabX, tabY)
	if s.notebook.Active != 2 {
		t.Fatalf("Notebook.Active after tab click: %d, want 2", s.notebook.Active)
	}
}

func TestTickDrivesNotification(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Toolbar click primes the notification.
	s.handleClick(12, toolkit.MenuBarH+toolkit.ToolbarButtonH/2)
	if !s.notify.Visible {
		t.Fatal("Notification should be visible after click")
	}
	life := s.notify.Life
	s.tick()
	if s.notify.Life != life-1 {
		t.Fatalf("tick decremented Life by %d, want 1", life-s.notify.Life)
	}
}

func TestAllToolbarStubsFire(t *testing.T) {
	// Exercise the 6 non-N Toolbar OnClick closures (O / S / C / X / V / ?)
	// so the coverage gate holds. They each just show a notification.
	s := newState(surfaceW, surfaceH)
	// N is index 0 (already covered by TestHandleClickToolbarFiresNotification).
	// Separators sit at indices 3 and 7 — no OnClick.
	for _, i := range []int{1, 2, 4, 5, 6, 8} {
		s.toolbar.Items[i].OnClick()
		if !s.notify.Visible {
			t.Errorf("Items[%d].OnClick did not show a notification", i)
		}
	}
}

// Also exercise every menu-bar Action so the "menu"-lambda closures
// in scene.go are hit even without a real click walk.
func TestAllMenuBarActionsFire(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	for mi, m := range s.menuBar.Menus {
		for ii, it := range m.Items {
			if it.Separator || it.Action == nil {
				continue
			}
			it.Action()
			if s.notify.Text == "" {
				t.Errorf("menu[%d].item[%d] left notify.Text empty", mi, ii)
			}
		}
	}
}

func TestFillBGCoversWholeSurface(t *testing.T) {
	surf := newSurface()
	fillBG(surf, surfaceW, surfaceH, toolkit.RGB(0xFF, 0x00, 0xAB))
	// Every 4-byte tuple should be (0xFF, 0x00, 0xAB, 0xFF).
	for i := 0; i+3 < len(surf); i += 4 {
		if surf[i] != 0xFF || surf[i+1] != 0x00 || surf[i+2] != 0xAB || surf[i+3] != 0xFF {
			t.Fatalf("byte %d not filled: %v", i, surf[i:i+4])
		}
	}
}

func TestInsideAndLocalHelpers(t *testing.T) {
	r := toolkit.Rect{X: 10, Y: 20, W: 30, H: 40}
	if !inside(15, 25, r) {
		t.Fatal("centre inside")
	}
	if inside(0, 0, r) {
		t.Fatal("(0,0) outside")
	}
	if inside(40, 60, r) {
		t.Fatal("just past far corner outside (half-open)")
	}
	ev := local(toolkit.Event{X: 25, Y: 30}, r)
	if ev.X != 15 || ev.Y != 10 {
		t.Fatalf("local wrong: %+v", ev)
	}
}
