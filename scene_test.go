// SPDX-License-Identifier: BSD-3-Clause
//
// scene_test — off-browser tests for the scene composition. main.go
// carries a js && wasm build tag so it drops out on the native test
// host; scene.go stays tagless so this file can exercise it against
// a plain byte buffer.

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
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
	if !s.radios[0].Checked().Get() {
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

// TestDrawDumpsEveryThemeToPNG renders the scene under every theme
// exposed by the theme switcher and PNG-encodes each to a dedicated
// file. Serves two purposes:
//
//   - **Correctness signal**: each PNG must have a distinct pixel
//     signature (bg color, ink color, accent color differ between
//     themes), so an "OnChange doesn't actually swap the theme"
//     regression would trip the signature comparison.
//   - **Visual verification hook**: when the environment variable
//     GALLERY_DUMP_PNG is set to a directory, the PNGs land there
//     instead of a temporary directory the test cleans up. That's
//     the seam a developer uses to inspect the render outside CI.
//
// Runs unconditionally on CI so coverage stays honest — every
// statement inside the dump path is exercised on every run.
func TestDrawDumpsEveryThemeToPNG(t *testing.T) {
	dir := os.Getenv("GALLERY_DUMP_PNG")
	if dir == "" {
		dir = t.TempDir()
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir dump dir: %v", err)
	}
	s := newState(surfaceW, surfaceH)
	signatures := make(map[string]bool, len(s.themeNames))
	for i, name := range s.themeNames {
		// Mirror what ViewSwitcher's own OnEvent does on a click:
		// updates Current then fires OnChange. Direct OnChange calls
		// would leave the visual "selected" state stuck on 0.
		s.themeSwitcher.Current().Set(i)
		s.themeSwitcher.Current().Set(i)
		surf := newSurface()
		s.draw(surf)
		path := filepath.Join(dir, "scene-"+name+".png")
		if err := encodeSurfaceAsPNG(surf, path); err != nil {
			t.Fatalf("encode %s: %v", path, err)
		}
		sig := surfaceSignature(surf)
		if signatures[sig] {
			t.Fatalf("theme %q produced the same pixel signature as an earlier theme — OnChange likely did not swap the palette", name)
		}
		signatures[sig] = true
	}
}

// encodeSurfaceAsPNG writes the RGBA byte buffer to path as a PNG.
func encodeSurfaceAsPNG(surf []byte, path string) error {
	img := image.NewRGBA(image.Rect(0, 0, surfaceW, surfaceH))
	for y := 0; y < surfaceH; y++ {
		for x := 0; x < surfaceW; x++ {
			i := 4 * (y*surfaceW + x)
			img.Set(x, y, color.RGBA{surf[i], surf[i+1], surf[i+2], surf[i+3]})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// surfaceSignature reduces the RGBA buffer to a short, theme-
// distinguishing string by sampling a handful of representative
// pixels. Samples the top-left (background), the toolbar strip
// (surface), and the accent-heavy Wave-3 ProgressCircle area
// (accent). Two different palettes MUST hit distinct color triples
// on at least one of those samples.
func surfaceSignature(surf []byte) string {
	samples := []struct{ x, y int }{
		{4, 4},                        // background
		{surfaceW / 2, 30},            // toolbar
		{surfaceW - 40, surfaceH / 2}, // wave-3 accent zone
	}
	var buf [3 * 4]byte
	for i, p := range samples {
		off := 4 * (p.y*surfaceW + p.x)
		copy(buf[i*4:i*4+4], surf[off:off+4])
	}
	return string(buf[:])
}

func TestDrawWithOpenMenuPopover(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	if !s.handleClick(10, 6) {
		t.Fatal("handleClick returned false")
	}
	if s.menuBar.Active().Get() != 0 {
		t.Fatalf("MenuBar Active after File click: %d, want 0", s.menuBar.Active().Get())
	}
	s.draw(newSurface()) // must not panic
}

// --- top scaffold click routing -------------------------------------------

func TestHandleClickToolbarFiresNotification(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// First toolbar button (item 0, x≈12).
	s.handleClick(12, toolkit.MenuBarH+toolkit.ToolbarButtonH/2)
	if !s.notify.Visible().Get() {
		t.Fatal("toolbar click did not fire a Notification")
	}
	if s.notify.Text == "" {
		t.Fatal("Notification text is empty after toolbar click")
	}
}

func TestHandleClickMenuItemDismissesAndFires(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.handleClick(10, 6) // open File menu
	if s.menuBar.Active().Get() != 0 {
		t.Fatal("File menu did not open")
	}
	// draw() sets the popover's Bounds — run it once before hit-testing.
	s.draw(newSurface())
	menu := s.menuBar.Menus[0]
	r := menu.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+4+toolkit.MenuRowH/2)
	if s.menuBar.Active().Get() != -1 {
		t.Fatalf("menu should dismiss after item click; Active=%d", s.menuBar.Active().Get())
	}
	if !s.notify.Visible().Get() {
		t.Fatal("menu-item click should fire the item's Action → Notification")
	}
}

func TestHandleClickOutsideOpenMenuDismisses(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	editX := s.menuBar.NameOriginX(1) + s.menuBar.NameWidth(1)/2
	s.handleClick(editX, 6)
	if s.menuBar.Active().Get() != 1 {
		t.Fatalf("Edit menu did not open; Active=%d", s.menuBar.Active().Get())
	}
	// Click near the bottom-right of the canvas — well outside any menu.
	s.handleClick(surfaceW-20, surfaceH-40)
	if s.menuBar.Active().Get() != -1 {
		t.Fatalf("outside click should dismiss menu; Active=%d", s.menuBar.Active().Get())
	}
}

// --- dashboard clickable dispatch -----------------------------------------

func TestClickButtonFiresHandler(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.button.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if !s.notify.Visible().Get() || s.notify.Text == "" {
		t.Fatal("Button click did not fire the Notification")
	}
}

func TestClickToggleFiresOnToggle(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.toggle.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if !s.toggle.Pressed().Get() {
		t.Fatal("Toggle click did not flip Pressed to true")
	}
	if !s.notify.Visible().Get() {
		t.Fatal("Toggle click did not fire the Notification")
	}
	// Click again — flips back.
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if s.toggle.Pressed().Get() {
		t.Fatal("second Toggle click did not flip Pressed back")
	}
}

func TestClickRadioActivatesGroup(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Click radio #2. First is checked by default; group.Add wires them.
	r := s.radios[1].Bounds()
	s.handleClick(r.X+5, r.Y+r.H/2)
	if !s.radios[1].Checked().Get() {
		t.Fatal("Radio 2 should be checked after click")
	}
	if s.radios[0].Checked().Get() {
		t.Fatal("Radio 1 should be cleared once Radio 2 is checked (group mutual-excl)")
	}
}

func TestClickListBoxSelects(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.listBox.Bounds()
	// Click 2 rows down.
	rowH := s.listBox.RowHeight
	s.handleClick(r.X+10, r.Y+rowH*2+rowH/2)
	if s.listBox.Selected().Get() < 0 {
		t.Fatalf("ListBox click did not select a row; Selected=%d", s.listBox.Selected().Get())
	}
}

func TestClickEntryFocuses(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.entry.Bounds()
	s.handleClick(r.X+10, r.Y+r.H/2)
	if !s.entry.Focused() {
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
	if s.notify.Visible().Get() {
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
		s.notify.Visible().Set(false)
		s.toolbar.Items[i].OnClick()
		if !s.notify.Visible().Get() {
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
	s.toggle.Pressed().Set(true) // starts false; flip on so the next flip is a real change
	s.notify.Text = ""
	s.toggle.Pressed().Set(false)
	if s.notify.Text == "" || s.notify.Text[len(s.notify.Text)-3:] != "OFF" {
		t.Fatalf("Toggle OFF branch not covered; text=%q", s.notify.Text)
	}
}

// TestAllWaveCallbacks exercises the OnToggle / OnAction / OnClose /
// OnClick / OnArrow callbacks attached to the wave-1/2/3 highlight
// widgets. None of them route through the clickables table (the
// widgets are wired at Draw time and the callbacks are the only
// observable outcome), so they need direct invocation to reach 100 %.
func TestAllWaveCallbacks(t *testing.T) {
	s := newState(surfaceW, surfaceH)

	// Switch OFF branch (the switch starts ON from NewSwitch(true), so Set(false)
	// is a real change).
	s.notify.Text = ""
	s.swtch.On().Set(false)
	if s.notify.Text == "" || s.notify.Text[len(s.notify.Text)-3:] != "OFF" {
		t.Fatalf("Switch OFF branch: text=%q", s.notify.Text)
	}
	// Switch ON branch.
	s.notify.Text = ""
	s.swtch.On().Set(true)
	if s.notify.Text == "" || s.notify.Text[len(s.notify.Text)-2:] != "ON" {
		t.Fatalf("Switch ON branch: text=%q", s.notify.Text)
	}

	// Banner action.
	s.notify.Text = ""
	s.banner.OnAction()
	if s.notify.Text == "" {
		t.Fatal("Banner OnAction did not show notify")
	}

	// Chip close.
	s.notify.Text = ""
	s.chip.OnClose()
	if s.notify.Text == "" {
		t.Fatal("Chip OnClose did not show notify")
	}

	// SplitButton main click + arrow click.
	s.notify.Text = ""
	s.splitButton.OnClick()
	if s.notify.Text == "" {
		t.Fatal("SplitButton OnClick did not show notify")
	}
	s.notify.Text = ""
	s.splitButton.OnArrow()
	if s.notify.Text == "" {
		t.Fatal("SplitButton OnArrow did not show notify")
	}

	// Theme switcher — every segment installs its palette and fires notify. The
	// widget's Current() Observable dedups equal Sets, so move to another segment
	// first to make each Set(i) a genuine change (only the neverEq view-model
	// re-applies on the same value).
	for i, name := range s.themeNames {
		s.themeSwitcher.Current().Set((i + 1) % len(s.themeNames))
		s.notify.Text = ""
		s.themeSwitcher.Current().Set(i)
		if s.theme != s.themes[i] {
			t.Fatalf("themeSwitcher select(%d) did not swap s.theme to %q", i, name)
		}
		if s.notify.Text == "" {
			t.Fatalf("themeSwitcher select(%d) did not show notify for %q", i, name)
		}
	}
}

// --- Wave 4 (v0.33) new-param demos ----------------------------------------

// TestWave4ParamsWired asserts every new v0.33 placement/orientation param
// the gallery demos actually got set on the live widget instance (not just
// left at its zero value) — the whole point of the wave-4 additions is to
// exercise these fields.
func TestWave4ParamsWired(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	if s.toolbarV == nil || s.toolbarV.Orientation != toolkit.Vertical {
		t.Fatal("toolbarV should be a Vertical-orientation Toolbar")
	}
	if s.stepsV == nil || s.stepsV.Orientation != toolkit.Vertical {
		t.Fatal("stepsV should be a Vertical-orientation Steps")
	}
	if s.notebookSide == nil || s.notebookSide.TabSide != toolkit.TabLeft {
		t.Fatal("notebookSide should have TabSide = TabLeft")
	}
	if s.dropdownUp == nil || !s.dropdownUp.OpenUp {
		t.Fatal("dropdownUp should have OpenUp = true")
	}
	if s.table == nil {
		t.Fatal("table should be populated")
	}
	if len(s.table.Columns) != 2 || s.table.Columns[1].Align != toolkit.AlignRight {
		t.Fatal("table's numeric column should be Align = AlignRight")
	}
	if s.timelineH == nil || !s.timelineH.Horizontal {
		t.Fatal("timelineH should have Horizontal = true")
	}
	// The original (Column C, Wave 3) Timeline stays vertical — confirms the
	// new field defaults false and the two instances demo both axes.
	if s.timeline == nil || s.timeline.Horizontal {
		t.Fatal("the wave-3 timeline should remain the default vertical layout")
	}
}

// TestAllToolbarVStubsFire exercises the vertical side-rail Toolbar's
// OnClick closures — wired at construction time, not routed through
// s.clickables in this test (that path is covered separately below).
func TestAllToolbarVStubsFire(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	for i := range s.toolbarV.Items {
		s.notify.Visible().Set(false)
		s.toolbarV.Items[i].OnClick()
		if !s.notify.Visible().Get() {
			t.Errorf("toolbarV.Items[%d].OnClick did not show a notification", i)
		}
	}
}

// TestClickToolbarVDispatchesThroughClickables drives a real handleClick
// at the vertical toolbar's first button to prove it's wired into the
// clickables hit-test table (not just constructed).
func TestClickToolbarVDispatchesThroughClickables(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.toolbarV.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+toolkit.ToolbarButtonH/2)
	if !s.notify.Visible().Get() || s.notify.Text == "" {
		t.Fatal("clicking the vertical toolbar's first button did not fire a notification")
	}
}

// TestClickNotebookSideSwitchesTab drives a real handleClick on the second
// (left-side) tab of the side-tab Notebook demo, proving TabLeft hit-testing
// is wired through s.clickables.
func TestClickNotebookSideSwitchesTab(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.notebookSide.Bounds()
	// Tab strip runs down the left edge; tab 1 ("B") sits one
	// NotebookTabStripH below tab 0.
	s.handleClick(r.X+10, r.Y+toolkit.NotebookTabStripH+4)
	if s.notebookSide.Active().Get() != 1 {
		t.Fatalf("clicking the second left-side tab should select it; Active=%d", s.notebookSide.Active().Get())
	}
}

// TestClickDropdownUpToggles drives a real handleClick on the upward-opening
// DropDown demo, proving it's wired through s.clickables.
func TestClickDropdownUpToggles(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.dropdownUp.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if !s.dropdownUp.Open().Get() {
		t.Fatal("clicking dropdownUp should open its popover")
	}
	pop := s.dropdownUp.PopoverBounds()
	if pop.Y >= r.Y {
		t.Fatalf("OpenUp popover should sit above the control; popover.Y=%d control.Y=%d", pop.Y, r.Y)
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

// --- Wave 5 (v0.42) new-widget demos ----------------------------------------

// TestWave5WidgetsPopulated asserts every v0.42 widget the gallery demos
// exists and carries the parameters the demo is built around — mirroring
// TestWave4ParamsWired for the newest wave.
func TestWave5WidgetsPopulated(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	if s.accordion == nil || len(s.accordion.Sections) != 3 || s.accordion.Expanded().Get() != 1 {
		t.Fatal("accordion should have 3 sections with the second pre-expanded")
	}
	if s.colorPicker == nil || s.colorPicker.Color().Get() != toolkit.RGB(0x35, 0x84, 0xe4) {
		t.Fatal("colorPicker should be populated with its seeded colour")
	}
	if s.segBar == nil || s.segBar.Total() != 100 {
		t.Fatalf("segBar segments should total 100, got %v", s.segBar.Total())
	}
	if s.carousel == nil || len(s.carousel.Slides) != 3 || !s.carousel.Wrap {
		t.Fatal("carousel should have 3 slides with Wrap = true")
	}
	if s.mdEditor == nil || s.mdEditor.Source == nil || s.mdEditor.Preview == nil {
		t.Fatal("mdEditor should have both Source and Preview panes")
	}
	if s.dateRange == nil || s.dateRange.Start().Get().D != 10 || s.dateRange.End().Get().D != 17 {
		t.Fatal("dateRange should have Start=10 and End=17 preset")
	}
	if s.wizard == nil || len(s.wizard.Steps) != 3 {
		t.Fatal("wizard should have 3 steps")
	}
	if s.treeTable == nil || len(s.treeTable.Columns) != 2 || len(s.treeTable.Root) != 2 {
		t.Fatal("treeTable should have 2 columns and 2 root nodes")
	}
	if s.paletteBtn == nil || s.cmdPalette == nil {
		t.Fatal("paletteBtn + cmdPalette should be populated")
	}
	if s.cmdPalette.Visible().Get() {
		t.Fatal("cmdPalette should start hidden")
	}
	if len(s.cmdPalette.Commands) != 4 {
		t.Fatalf("cmdPalette should have 4 commands, got %d", len(s.cmdPalette.Commands))
	}
}

// TestWave6WidgetsPopulated checks the v0.81 grid-editing-family demos are
// wired: a PropertyGrid (with an editor opened on the Title value), a
// PagingToolbar, a grouped+editable Table and a DataView ListBox.
func TestWave6WidgetsPopulated(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	if s.propGrid == nil || s.propGrid.Value("Width") != "1024" {
		t.Fatal("propGrid should be populated with a Width property")
	}
	if s.pagingBar == nil || s.pagingBar.Page().Get() != 6 || s.pagingBar.PageCount != 12 || !s.pagingBar.ShowRefresh {
		t.Fatalf("pagingBar should be Page 6 of 12 with refresh, got %+v", s.pagingBar)
	}
	if s.gridEdit == nil || s.gridEdit.GroupBy != 0 || !s.gridEdit.Columns[1].Editable {
		t.Fatal("gridEdit should be grouped by col 0 with an editable Owner column")
	}
	if s.dataView == nil || s.dataView.ItemRenderer == nil || s.dataView.Selected().Get() != 1 {
		t.Fatal("dataView should have an ItemRenderer and a selected row")
	}
}

// TestAllCommandPaletteActionsFire exercises every PaletteCommand.Action
// closure wired in newState — the only observable outcome of each is the
// Notification it fires, the same pattern TestAllMenuBarActionsFire uses.
func TestAllCommandPaletteActionsFire(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	for i, cmd := range s.cmdPalette.Commands {
		s.notify.Text = ""
		if cmd.Action == nil {
			t.Fatalf("command[%d] %q has a nil Action", i, cmd.Label)
		}
		cmd.Action()
		if s.notify.Text == "" {
			t.Errorf("command[%d] %q left notify.Text empty", i, cmd.Label)
		}
	}
}

// TestClickPaletteButtonOpensCommandPalette drives a real handleClick on the
// trigger Button, proving it's wired into s.clickables and that it actually
// opens the overlay (Visible flips true, Query/Selected reset).
func TestClickPaletteButtonOpensCommandPalette(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.paletteBtn.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if !s.cmdPalette.Visible().Get() {
		t.Fatal("clicking the trigger button should open the CommandPalette")
	}
}

// TestHandleClickRoutesToOpenCommandPalette proves the CommandPalette-first
// branch in handleClick: once Visible, any click (even far from the panel)
// is handed to the palette instead of falling through to the dashboard
// clickables. Clicking outside the centered panel dismisses it.
func TestHandleClickRoutesToOpenCommandPalette(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.cmdPalette.Open()
	if !s.cmdPalette.Visible().Get() {
		t.Fatal("Open() should set Visible")
	}
	// Top-left corner sits well outside the centered panel.
	if !s.handleClick(1, 1) {
		t.Fatal("handleClick should return true while the palette is open")
	}
	if s.cmdPalette.Visible().Get() {
		t.Fatal("an outside click should have dismissed the CommandPalette")
	}
}

// TestClickColorPickerSVSquareFiresOnChange drives a real handleClick inside
// the SV square (the widget's local (0,0)-(120,120) region) to prove the
// gallery's OnChange closure — whose only job is to surface a Notification —
// actually runs.
func TestClickColorPickerSVSquareFiresOnChange(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.colorPicker.Bounds()
	s.notify.Text = ""
	s.handleClick(r.X+10, r.Y+10)
	if s.notify.Text == "" {
		t.Fatal("clicking the SV square should fire colorPicker.OnChange -> Notification")
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

// TestKanbanCardDrag drives the scene's drag-capture routing end to end: a
// press on card (0,0) grabs it, a drag into column 2 marks the gesture, and a
// release drops it there -- moving the card and firing OnCardMove.
func TestKanbanCardDrag(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	kr := s.kanban.Bounds()
	colW := (kr.W - 2*toolkit.KanbanColGap) / 3
	cardY := kr.Y + toolkit.KanbanHeaderH + toolkit.KanbanCardGap + 4

	// Press on column 0, card 0 ("Design").
	if !s.handleClick(kr.X+toolkit.KanbanCardGap+4, cardY) {
		t.Fatal("handleClick(card) returned false")
	}
	if s.dragTarget == nil {
		t.Fatal("press did not arm a drag target")
	}
	// Drag + release over column 2, slot 0.
	col2X := kr.X + 2*(colW+toolkit.KanbanColGap) + toolkit.KanbanCardGap + 4
	if !s.handleDrag(col2X, cardY) {
		t.Fatal("handleDrag returned false with a target armed")
	}
	if !s.handleRelease(col2X, cardY) {
		t.Fatal("handleRelease returned false with a target armed")
	}
	if s.dragTarget != nil {
		t.Fatal("drag target not cleared after release")
	}
	if got := s.kanban.Columns[2].Cards[0].Title; got != "Design" {
		t.Fatalf("after drag, column 2 card 0 = %q, want Design", got)
	}
	// Drag/release with nothing armed are no-ops.
	if s.handleDrag(1, 1) || s.handleRelease(1, 1) {
		t.Fatal("drag/release without a target should be no-ops")
	}
}

// TestGanttBarDrag presses just inside task 0's bar, drags it right and
// releases: the bar moves (Start increases) with its span preserved, firing
// OnTaskChange.
func TestGanttBarDrag(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	gr := s.gantt.Bounds()
	axisW := gr.W - toolkit.GanttLabelW
	units := 12 // auto-derived from the largest task End (Ship ends at 12)
	barX := func(c int) int { return gr.X + toolkit.GanttLabelW + c*axisW/units }
	rowY := gr.Y + toolkit.GanttHeaderH + 2 // task 0 ("Design", Start 0 End 3)
	span0 := s.gantt.Tasks[0].End - s.gantt.Tasks[0].Start

	// Press 8px inside the bar's left edge (past the ~5px resize slop) -> move.
	s.handleClick(barX(0)+8, rowY)
	if s.dragTarget == nil {
		t.Fatal("press on a bar did not arm a drag target")
	}
	s.handleDrag(barX(9), rowY)
	s.handleRelease(barX(9), rowY)

	if s.gantt.Tasks[0].Start <= 0 {
		t.Fatalf("bar did not move: Start=%d", s.gantt.Tasks[0].Start)
	}
	if got := s.gantt.Tasks[0].End - s.gantt.Tasks[0].Start; got != span0 {
		t.Fatalf("span changed on move: got %d, want %d", got, span0)
	}
}

// TestAgendaSwitcherChangesView clicks the "Week" segment of the Agenda view
// switcher and checks the Agenda's View follows.
func TestAgendaSwitcherChangesView(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	ar := s.agendaSwitcher.Bounds()
	s.handleClick(ar.X+ar.W/8, ar.Y+ar.H/2) // segment 0 centre ("Week")
	if s.agenda.View().Get() != toolkit.AgendaWeek {
		t.Fatalf("Agenda view = %d, want AgendaWeek (%d)", s.agenda.View().Get(), toolkit.AgendaWeek)
	}
}

// TestAgendaAddEvent clicks an empty in-month day cell and checks a new event
// is appended (exercising OnDayActivate + itoa).
func TestAgendaAddEvent(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	ar := s.agenda.Bounds()
	before := len(s.agenda.Events)

	first := toolkit.WeekdayOfFirst(2026, 7)
	idx := first + 14 // day 15 (no seeded event)
	row, col := idx/7, idx%7
	cellX := ar.X + col*ar.W/7 + (ar.W/7)/2
	cellY := ar.Y + toolkit.AgendaHeaderH + row*toolkit.AgendaDayCellH + toolkit.AgendaDayCellH/2
	s.handleClick(cellX, cellY)

	if len(s.agenda.Events) != before+1 {
		t.Fatalf("events after day click = %d, want %d", len(s.agenda.Events), before+1)
	}
	if got := s.agenda.Events[len(s.agenda.Events)-1]; got.D != 15 || got.M != 7 {
		t.Fatalf("added event on %d/%d, want 7/15", got.M, got.D)
	}
}

// TestAgendaEventEditing drives the inline event editor end-to-end through the
// host: opening it (OnSelect), routing keystrokes to it (handleChar/
// handleKeyDown), and committing on an outside click (handleClick), which fires
// OnEventEdited → a notification.
func TestAgendaEventEditing(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// A chip click Sets the Agenda's Selected event; the gallery's subscription
	// opens the toolkit editor on that event. Selected starts at -1, so Set(0)
	// is a genuine change and notifies.
	s.agenda.Selected().Set(0)
	if s.agenda.Editing() != 0 {
		t.Fatalf("Editing()=%d, want 0 (editor open)", s.agenda.Editing())
	}
	// Keystrokes are captured by the open editor, not the previous keyTarget.
	if !s.handleChar("!") {
		t.Fatal("handleChar should be consumed by the open editor")
	}
	if !s.handleKeyDown("Backspace") {
		t.Fatal("handleKeyDown should be consumed by the open editor")
	}
	// A click anywhere while editing routes to the editor; (1,1) is outside the
	// centred panel, so it commits + closes and notifies via OnEventEdited.
	s.notify.Visible().Set(false)
	if !s.handleClick(1, 1) {
		t.Fatal("click while editing should be consumed")
	}
	if s.agenda.Editing() != -1 {
		t.Fatal("outside click should close the editor")
	}
	if !s.notify.Visible().Get() {
		t.Fatal("committing an edit should fire OnEventEdited → a notification")
	}
}

// TestAgendaSidebarToggle clicks a calendar row in the sidebar rail and checks
// the calendar's visibility flips (shared with the Agenda) + OnToggle notifies,
// for both the hide and the show direction.
func TestAgendaSidebarToggle(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	sb := s.calSidebar.Bounds()
	rowX := sb.X + 10
	rowY := sb.Y + toolkit.AgendaHeaderH + toolkit.AgendaSidebarRowH/2 // row 0

	s.notify.Visible().Set(false)
	s.handleClick(rowX, rowY) // hide "Team"
	if !s.agendaCals[0].Hidden {
		t.Fatal("first sidebar click should hide calendar 0")
	}
	if !s.notify.Visible().Get() {
		t.Fatal("OnToggle should notify on hide")
	}
	s.notify.Visible().Set(false)
	s.handleClick(rowX, rowY) // show it again
	if s.agendaCals[0].Hidden {
		t.Fatal("second sidebar click should show calendar 0")
	}
	if !s.notify.Visible().Get() {
		t.Fatal("OnToggle should notify on show")
	}
}

// TestItoa spot-checks the local itoa helper.
func TestItoa(t *testing.T) {
	if itoa(0) != "0" || itoa(42) != "42" || itoa(-7) != "-7" {
		t.Fatalf("itoa: %q %q %q", itoa(0), itoa(42), itoa(-7))
	}
}

// TestDropDownPopover: opening the combobox, selecting an option via its
// popover (fires OnSelect), and dismissing it with an outside click.
func TestDropDownPopover(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	dr := s.dropdown.Bounds()
	s.handleClick(dr.X+dr.W/2, dr.Y+dr.H/2) // open via the control
	if !s.dropdown.Open().Get() {
		t.Fatal("dropdown did not open")
	}
	s.draw(newSurface()) // renders the popover (covers the draw branch)

	pb := s.dropdown.PopoverBounds()
	s.notify.Visible().Set(false)
	s.handleClick(pb.X+5, pb.Y+toolkit.PopoverRowH+2) // click option row 1
	if s.dropdown.Selected().Get() != 1 {
		t.Fatalf("Selected=%d, want 1", s.dropdown.Selected().Get())
	}
	if s.dropdown.Open().Get() {
		t.Fatal("popover should close after a selection")
	}
	if !s.notify.Visible().Get() {
		t.Fatal("OnSelect did not fire a notification")
	}
	// Reopen, then an outside click dismisses it.
	s.handleClick(dr.X+dr.W/2, dr.Y+dr.H/2)
	if !s.dropdown.Open().Get() {
		t.Fatal("dropdown did not reopen")
	}
	s.handleClick(1, surfaceH-toolkit.StatusbarH-2) // dead space
	if s.dropdown.Open().Get() {
		t.Fatal("outside click should dismiss the popover")
	}
}

// TestKeyboardRouting: typing routes to the focused (last-clicked) widget; a
// dead-space click clears focus so keys become no-ops.
func TestKeyboardRouting(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	er := s.entry.Bounds()
	s.handleClick(er.X+10, er.Y+er.H/2)
	if s.keyTarget != toolkit.Widget(s.entry) {
		t.Fatal("clicking the entry did not focus it for keyboard")
	}
	before := s.entry.Text().Get()
	s.handleChar("Z")
	if s.entry.Text().Get() == before {
		t.Fatal("handleChar did not edit the entry")
	}
	s.handleKeyDown("Backspace")
	if s.entry.Text().Get() != before {
		t.Fatalf("Backspace did not restore text: %q vs %q", s.entry.Text().Get(), before)
	}
	s.handleClick(1, surfaceH-toolkit.StatusbarH-2) // dead space
	if s.keyTarget != nil {
		t.Fatal("dead-space click should clear keyTarget")
	}
	if s.handleChar("x") || s.handleKeyDown("Enter") {
		t.Fatal("no keyTarget → key handlers should be no-ops")
	}
}

// TestChartHoverTooltip: hovering the active Notebook Line/Bar chart shows a
// value tooltip; leaving hides it; a non-chart tab shows nothing.
func TestChartHoverTooltip(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.draw(newSurface()) // sets the active page's bounds

	lc := s.notebook.Tabs[0].Page.(*toolkit.LineChart)
	lr := lc.Bounds()
	if !s.handleHover(lr.X+lr.W/2, lr.Y+lr.H/2) || !s.tooltip.Visible().Get() || s.tooltip.Text == "" {
		t.Fatalf("line-chart hover: visible=%v text=%q", s.tooltip.Visible().Get(), s.tooltip.Text)
	}
	s.draw(newSurface()) // renders the tooltip (covers Draw's Visible branch)

	if !s.handleHover(1, 1) || s.tooltip.Visible().Get() { // leave the chart
		t.Fatal("leaving the chart should hide the tooltip")
	}
	if s.handleHover(1, 1) { // already hidden → no change
		t.Fatal("hover off-chart with no tooltip should report no change")
	}

	// Bar tab.
	s.notebook.Active().Set(1)
	s.draw(newSurface())
	bc := s.notebook.Tabs[1].Page.(*toolkit.BarChart)
	br := bc.Bounds()
	if !s.handleHover(br.X+br.W/2, br.Y+br.H/2) || !s.tooltip.Visible().Get() {
		t.Fatal("bar-chart hover should show a tooltip")
	}
	// Pie tab: hovering the disc shows a slice value; hovering off-chart hides.
	s.notebook.Active().Set(2)
	s.draw(newSurface())
	nb := s.notebook.Bounds()
	pc := s.notebook.Tabs[2].Page.(*toolkit.PieChart)
	if !s.handleHover(pc.Bounds().X+pc.Bounds().W/2+3, pc.Bounds().Y+pc.Bounds().H/2) || !s.tooltip.Visible().Get() {
		t.Fatal("pie-slice hover should show a tooltip")
	}
	if !s.handleHover(1, 1) || s.tooltip.Visible().Get() {
		t.Fatal("hovering off every chart should hide the tooltip")
	}

	// chartHoverText guards: outside the notebook, and an out-of-range Active.
	if txt, _, _ := s.chartHoverText(1, 1); txt != "" {
		t.Fatal("chartHoverText outside notebook should be empty")
	}
	s.notebook.Active().Set(99)
	if txt, _, _ := s.chartHoverText(nb.X+5, nb.Y+40); txt != "" {
		t.Fatal("chartHoverText with bad Active should be empty")
	}
}

// TestHandleMoveDragVsHover: handleMove drags while captured, else hovers.
func TestHandleMoveDragVsHover(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.draw(newSurface())
	lc := s.notebook.Tabs[0].Page.(*toolkit.LineChart)
	lr := lc.Bounds()
	if !s.handleMove(lr.X+lr.W/2, lr.Y+lr.H/2) { // no capture → hover
		t.Fatal("handleMove should hover when not dragging")
	}
	s.dragTarget, s.dragBounds = s.kanban, s.kanban.Bounds() // simulate capture
	if !s.handleMove(10, 10) {                               // → drag path
		t.Fatal("handleMove should route a drag when captured")
	}
}

func TestFtoa(t *testing.T) {
	if ftoa(3) != "3" || ftoa(2.5) != "2.5" {
		t.Fatalf("ftoa: %q %q", ftoa(3), ftoa(2.5))
	}
}

// TestDropdownUpSelect covers the second dropdown's OnSelect closure.
func TestDropdownUpSelect(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.notify.Visible().Set(false)
	s.dropdownUp.Select(1)
	if !s.notify.Visible().Get() {
		t.Fatal("dropdownUp OnSelect did not fire a notification")
	}
}

// TestAreaChartHoverCrosshair: hovering the Wave-7 AreaChart shows its value and
// arms the crosshair; leaving clears it. Also checks the Line crosshair arms.
func TestAreaChartHoverCrosshair(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.draw(newSurface()) // set chart bounds

	// Line chart hover arms its crosshair.
	lc := s.notebook.Tabs[0].Page.(*toolkit.LineChart)
	lr := lc.Bounds()
	s.handleHover(lr.X+lr.W/2, lr.Y+lr.H/2)
	if !lc.Hover().Get() {
		t.Fatal("line-chart hover did not arm the crosshair")
	}

	// Area chart hover: value tooltip + crosshair.
	ar := s.areaChart.Bounds()
	if !s.handleHover(ar.X+ar.W/2, ar.Y+ar.H/2) || !s.tooltip.Visible().Get() {
		t.Fatal("area-chart hover did not show a tooltip")
	}
	if !s.areaChart.Hover().Get() {
		t.Fatal("area-chart hover did not arm the crosshair")
	}
	if lc.Hover().Get() {
		t.Fatal("moving to the area chart should clear the line crosshair")
	}
	// Leaving every chart clears the area crosshair.
	s.handleHover(1, 1)
	if s.areaChart.Hover().Get() {
		t.Fatal("leaving the chart should clear the area crosshair")
	}
}

// TestAllChartsHover: every remaining chart (Scatter/Radar/Sparklines) shows a
// value tooltip and arms its hover highlight.
func TestAllChartsHover(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.draw(newSurface())

	sr := s.scatterChart.Bounds()
	if !s.handleHover(sr.X+sr.W/2, sr.Y+sr.H/2) || !s.tooltip.Visible().Get() || !s.scatterChart.Hover().Get() {
		t.Fatal("scatter hover should show a tooltip + ring")
	}
	rr := s.radarChart.Bounds()
	if !s.handleHover(rr.X+rr.W/2, rr.Y+rr.H/4) || !s.tooltip.Visible().Get() || !s.radarChart.Hover().Get() {
		t.Fatal("radar hover should show a tooltip + spoke")
	}
	sl := s.sparkLine.Bounds()
	if !s.handleHover(sl.X+sl.W/2, sl.Y+sl.H/2) || !s.tooltip.Visible().Get() || !s.sparkLine.Hover().Get() {
		t.Fatal("sparkLine hover should show a tooltip + crosshair")
	}
	sb := s.sparkBar.Bounds()
	if !s.handleHover(sb.X+sb.W/2, sb.Y+sb.H/2) || !s.tooltip.Visible().Get() || !s.sparkBar.Hover().Get() {
		t.Fatal("sparkBar hover should show a tooltip + highlight")
	}
	// A draw after arming exercises the hover-highlight render paths.
	s.draw(newSurface())
}
