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
		s.themeSwitcher.Current = i
		s.themeSwitcher.OnChange(i)
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

// TestAllWaveCallbacks exercises the OnToggle / OnAction / OnClose /
// OnClick / OnArrow callbacks attached to the wave-1/2/3 highlight
// widgets. None of them route through the clickables table (the
// widgets are wired at Draw time and the callbacks are the only
// observable outcome), so they need direct invocation to reach 100 %.
func TestAllWaveCallbacks(t *testing.T) {
	s := newState(surfaceW, surfaceH)

	// Switch ON branch.
	s.notify.Text = ""
	s.swtch.OnToggle(true)
	if s.notify.Text == "" || s.notify.Text[len(s.notify.Text)-2:] != "ON" {
		t.Fatalf("Switch ON branch: text=%q", s.notify.Text)
	}
	// Switch OFF branch.
	s.notify.Text = ""
	s.swtch.OnToggle(false)
	if s.notify.Text == "" || s.notify.Text[len(s.notify.Text)-3:] != "OFF" {
		t.Fatalf("Switch OFF branch: text=%q", s.notify.Text)
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

	// Theme switcher — every segment installs its palette and fires notify.
	for i, name := range s.themeNames {
		s.notify.Text = ""
		s.themeSwitcher.OnChange(i)
		if s.theme != s.themes[i] {
			t.Fatalf("themeSwitcher OnChange(%d) did not swap s.theme to %q", i, name)
		}
		if s.notify.Text == "" {
			t.Fatalf("themeSwitcher OnChange(%d) did not show notify for %q", i, name)
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
		s.notify.Visible = false
		s.toolbarV.Items[i].OnClick()
		if !s.notify.Visible {
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
	if !s.notify.Visible || s.notify.Text == "" {
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
	if s.notebookSide.Active != 1 {
		t.Fatalf("clicking the second left-side tab should select it; Active=%d", s.notebookSide.Active)
	}
}

// TestClickDropdownUpToggles drives a real handleClick on the upward-opening
// DropDown demo, proving it's wired through s.clickables.
func TestClickDropdownUpToggles(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	r := s.dropdownUp.Bounds()
	s.handleClick(r.X+r.W/2, r.Y+r.H/2)
	if !s.dropdownUp.Open {
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
	if s.accordion == nil || len(s.accordion.Sections) != 3 || s.accordion.Expanded != 1 {
		t.Fatal("accordion should have 3 sections with the second pre-expanded")
	}
	if s.colorPicker == nil || s.colorPicker.OnChange == nil {
		t.Fatal("colorPicker should be populated with an OnChange handler")
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
	if s.dateRange == nil || s.dateRange.Start.D != 10 || s.dateRange.End.D != 17 {
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
	if s.cmdPalette.Visible {
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
	if s.pagingBar == nil || s.pagingBar.Page != 6 || s.pagingBar.PageCount != 12 || !s.pagingBar.ShowRefresh {
		t.Fatalf("pagingBar should be Page 6 of 12 with refresh, got %+v", s.pagingBar)
	}
	if s.gridEdit == nil || s.gridEdit.GroupBy != 0 || !s.gridEdit.Columns[1].Editable {
		t.Fatal("gridEdit should be grouped by col 0 with an editable Owner column")
	}
	if s.dataView == nil || s.dataView.ItemRenderer == nil || s.dataView.Selected != 1 {
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
	if !s.cmdPalette.Visible {
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
	if !s.cmdPalette.Visible {
		t.Fatal("Open() should set Visible")
	}
	// Top-left corner sits well outside the centered panel.
	if !s.handleClick(1, 1) {
		t.Fatal("handleClick should return true while the palette is open")
	}
	if s.cmdPalette.Visible {
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
