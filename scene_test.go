// SPDX-License-Identifier: BSD-3-Clause
//
// scene_test — off-browser tests for the scene composition. main.go
// carries a js && wasm build tag so it drops out on the native test
// host; scene.go stays tagless so this file can exercise it against
// a plain byte buffer.

package main

import (
	"testing"

	"github.com/go-widgets/toolkit"
)

func newSurface() []byte { return make([]byte, 4*surfaceW*surfaceH) }

// --- scaffold + draw ------------------------------------------------------

func TestNewStateFillsScaffold(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	if s == nil {
		t.Fatal("newState returned nil")
	}
	if s.menuBar == nil || s.toolbar == nil || s.status == nil || s.notify == nil {
		t.Fatal("newState left a core scaffold widget nil")
	}
	if len(s.menuBar.Menus) != 4 {
		t.Fatalf("MenuBar expected 4 menus, got %d", len(s.menuBar.Menus))
	}
	if len(s.clickables) < 10 {
		t.Fatalf("clickables list unexpectedly small: %d", len(s.clickables))
	}
}

func TestNewStatePopulatesEveryColumn(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Column A representative widgets.
	if s.button == nil || s.toggle == nil || s.check == nil {
		t.Fatal("column A action row missing widgets")
	}
	if len(s.radios) != 3 {
		t.Fatalf("expected 3 radio buttons, got %d", len(s.radios))
	}
	if !s.radios[0].Checked {
		t.Fatal("first radio should start checked")
	}
	if s.entry == nil || s.spin == nil || s.scale == nil || s.dropdown == nil {
		t.Fatal("column A inputs row missing widgets")
	}
	if s.progress == nil || s.level == nil || s.spinner == nil {
		t.Fatal("column A feedback row missing widgets")
	}
	// Column B.
	if s.textView == nil || s.calendar == nil || s.colorChoose == nil {
		t.Fatal("column B missing widgets")
	}
	// Column C.
	if s.listBox == nil || s.tree == nil || s.expander == nil || s.frameHost == nil {
		t.Fatal("column C missing widgets")
	}
}

func TestDrawPaintsInto(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	surf := newSurface()
	s.draw(surf)
	// Background must have filled every 4-byte tuple to non-zero
	// alpha; use that as a global sanity check.
	for i := 3; i+3 < len(surf); i += 4 {
		if surf[i] == 0 {
			t.Fatalf("draw left alpha 0 at byte %d — background fill missing", i)
		}
	}
}

func TestDrawWithOpenMenuPopover(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	if !s.handleClick(10, 6) {
		t.Fatal("handleClick returned false")
	}
	if s.menuBar.Active != 0 {
		t.Fatalf("MenuBar Active after File click: %d, want 0", s.menuBar.Active)
	}
	s.draw(newSurface()) // must not panic
}

// --- top scaffold click routing -------------------------------------------

func TestHandleClickToolbarFiresNotification(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// First toolbar button (item 0, x≈12).
	s.handleClick(12, toolkit.MenuBarH+toolkit.ToolbarButtonH/2)
	if !s.notify.Visible {
		t.Fatal("toolbar click did not fire a Notification")
	}
	if s.notify.Text == "" {
		t.Fatal("Notification text is empty after toolbar click")
	}
}

func TestHandleClickMenuItemDismissesAndFires(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.handleClick(10, 6) // open File menu
	if s.menuBar.Active != 0 {
		t.Fatal("File menu did not open")
	}
	// draw() sets the popover's Bounds — run it once before hit-testing.
	s.draw(newSurface())
	menu := s.menuBar.Menus[0]
	r := menu.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+4+toolkit.MenuRowH/2)
	if s.menuBar.Active != -1 {
		t.Fatalf("menu should dismiss after item click; Active=%d", s.menuBar.Active)
	}
	if !s.notify.Visible {
		t.Fatal("menu-item click should fire the item's Action → Notification")
	}
}

func TestHandleClickOutsideOpenMenuDismisses(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	editX := s.menuBar.NameOriginX(1) + s.menuBar.NameWidth(1)/2
	s.handleClick(editX, 6)
	if s.menuBar.Active != 1 {
		t.Fatalf("Edit menu did not open; Active=%d", s.menuBar.Active)
	}
	// Click near the bottom-right of the canvas — well outside any menu.
	s.handleClick(surfaceW-20, surfaceH-40)
	if s.menuBar.Active != -1 {
		t.Fatalf("outside click should dismiss menu; Active=%d", s.menuBar.Active)
	}
}

// --- dashboard clickable dispatch -----------------------------------------

func TestClickButtonFiresHandler(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.button.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if !s.notify.Visible || s.notify.Text == "" {
		t.Fatal("Button click did not fire the Notification")
	}
}

func TestClickToggleFiresOnToggle(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.toggle.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if !s.toggle.Pressed {
		t.Fatal("Toggle click did not flip Pressed to true")
	}
	if !s.notify.Visible {
		t.Fatal("Toggle click did not fire the Notification")
	}
	// Click again — flips back.
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if s.toggle.Pressed {
		t.Fatal("second Toggle click did not flip Pressed back")
	}
}

func TestClickRadioActivatesGroup(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Click radio #2. First is checked by default; group.Add wires them.
	r := s.radios[1].Bounds()
	s.handleClick(r.X+5, r.Y+r.H/2)
	if !s.radios[1].Checked {
		t.Fatal("Radio 2 should be checked after click")
	}
	if s.radios[0].Checked {
		t.Fatal("Radio 1 should be cleared once Radio 2 is checked (group mutual-excl)")
	}
}

func TestClickListBoxSelects(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.listBox.Bounds()
	// Click 2 rows down.
	rowH := s.listBox.RowHeight
	s.handleClick(r.X+10, r.Y+rowH*2+rowH/2)
	if s.listBox.Selected < 0 {
		t.Fatalf("ListBox click did not select a row; Selected=%d", s.listBox.Selected)
	}
}

func TestClickEntryFocuses(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.entry.Bounds()
	s.handleClick(r.X+10, r.Y+r.H/2)
	if !s.entry.Focused {
		t.Fatal("Entry click should focus the entry")
	}
}

// The dashboard has empty ("dead") space between widget cards; a
// click there must return true (event consumed / no widget hit) and
// leave the notify off.
func TestClickDeadSpaceIsNoOp(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Between the Statusbar and the last card — should hit nothing.
	s.handleClick(surfaceW/2, surfaceH-toolkit.StatusbarH-2)
	if s.notify.Visible {
		t.Fatal("dead-space click should not trigger any Notification")
	}
}

// --- tick + helpers -------------------------------------------------------

func TestTickDrivesNotificationAndSpinner(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Prime the notification via toolbar click.
	s.handleClick(12, toolkit.MenuBarH+toolkit.ToolbarButtonH/2)
	life := s.notify.Life
	phaseBefore := s.spinner.Phase
	s.tick()
	if s.notify.Life != life-1 {
		t.Fatalf("tick decremented Life by %d, want 1", life-s.notify.Life)
	}
	if s.spinner.Phase == phaseBefore {
		t.Fatal("tick should advance Spinner Phase")
	}
}

func TestAllToolbarStubsFire(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Separators sit at indices 3 and 7 — no OnClick.
	for _, i := range []int{0, 1, 2, 4, 5, 6, 8} {
		s.notify.Visible = false
		s.toolbar.Items[i].OnClick()
		if !s.notify.Visible {
			t.Errorf("Items[%d].OnClick did not show a notification", i)
		}
	}
}

func TestAllMenuBarActionsFire(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	for mi, m := range s.menuBar.Menus {
		for ii, it := range m.Items {
			if it.Separator || it.Action == nil {
				continue
			}
			s.notify.Text = ""
			it.Action()
			if s.notify.Text == "" {
				t.Errorf("menu[%d].item[%d] left notify.Text empty", mi, ii)
			}
		}
	}
}

func TestAllToggleBranches(t *testing.T) {
	// Directly exercise the OFF branch of s.toggle.OnToggle (the ON
	// branch is covered by TestClickToggleFiresOnToggle → true).
	s := newState(surfaceW, surfaceH)
	s.toggle.OnToggle(false)
	if s.notify.Text == "" || s.notify.Text[len(s.notify.Text)-3:] != "OFF" {
		t.Fatalf("Toggle OFF branch not covered; text=%q", s.notify.Text)
	}
}

func TestFillBGCoversWholeSurface(t *testing.T) {
	surf := newSurface()
	fillBG(surf, surfaceW, surfaceH, toolkit.RGB(0xFF, 0x00, 0xAB))
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
