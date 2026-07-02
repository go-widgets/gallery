// Scene state — composes every widget family from go-widgets/toolkit
// into a single canvas layout. Kept in a separate file (no js/wasm
// build tag) so a native go test could exercise draw() + handleClick()
// against a []byte off-browser if we ever add that CI lane.

package main

import (
	"github.com/go-widgets/toolkit"
)

// Fixed canvas dimensions. Lives in scene.go (not main.go) so the
// native scene_test compiles without the js && wasm build tag —
// otherwise the constants drop out and the tests can't reference
// them.
const (
	surfaceW = 720
	surfaceH = 480
)

type state struct {
	w, h int

	theme    *toolkit.Theme
	menuBar  *toolkit.MenuBar
	toolbar  *toolkit.Toolbar
	notebook *toolkit.Notebook
	status   *toolkit.Statusbar
	notify   *toolkit.Notification

	// Per-tab primary + auxiliary widgets, kept on state so their
	// state persists across renders + a test could reach in.
	primary map[int]toolkit.Widget

	button   *toolkit.Button
	check    *toolkit.CheckButton
	entry    *toolkit.Entry
	listBox  *toolkit.ListBox
	tree     *toolkit.TreeView
	progress *toolkit.ProgressBar
	scale    *toolkit.Scale
	color    *toolkit.ColorChooser
	calendar *toolkit.Calendar
	dropdown *toolkit.DropDown
}

func newState(w, h int) *state {
	s := &state{w: w, h: h, theme: toolkit.DefaultLight(), primary: map[int]toolkit.Widget{}}
	s.status = toolkit.NewStatusbar([]string{"35 widgets", "100 % cov", "click a menu to toast", "go-widgets/toolkit"})
	s.status.SetBounds(toolkit.Rect{X: 0, Y: h - toolkit.StatusbarH, W: w, H: toolkit.StatusbarH})

	s.notify = toolkit.NewNotification("")
	s.notify.SetBounds(toolkit.Rect{X: w - 260, Y: toolkit.MenuBarH + toolkit.ToolbarButtonH + 8, W: 250, H: 24})

	menu := func(label string) toolkit.MenuItem {
		return toolkit.MenuItem{Label: label, Action: func() { s.notify.Show("clicked: " + label) }}
	}

	s.menuBar = toolkit.NewMenuBar()
	s.menuBar.Names = []string{"File", "Edit", "View", "Help"}
	s.menuBar.Menus = []*toolkit.Menu{
		toolkit.NewMenu([]toolkit.MenuItem{menu("New"), menu("Open"), {Separator: true}, menu("Quit")}),
		toolkit.NewMenu([]toolkit.MenuItem{menu("Cut"), menu("Copy"), menu("Paste")}),
		toolkit.NewMenu([]toolkit.MenuItem{menu("Zoom in"), menu("Zoom out"), menu("Reset")}),
		toolkit.NewMenu([]toolkit.MenuItem{menu("About")}),
	}
	s.menuBar.SetBounds(toolkit.Rect{X: 0, Y: 0, W: w, H: toolkit.MenuBarH})

	s.toolbar = toolkit.NewToolbar([]toolkit.ToolbarItem{
		{Label: "N", OnClick: func() { s.notify.Show("Toolbar: New") }},
		{Label: "O", OnClick: func() { s.notify.Show("Toolbar: Open") }},
		{Label: "S", OnClick: func() { s.notify.Show("Toolbar: Save") }},
		{Separator: true},
		{Label: "C", OnClick: func() { s.notify.Show("Toolbar: Copy") }},
		{Label: "X", OnClick: func() { s.notify.Show("Toolbar: Cut") }},
		{Label: "V", OnClick: func() { s.notify.Show("Toolbar: Paste") }},
		{Separator: true},
		{Label: "?", OnClick: func() { s.notify.Show("go-widgets/toolkit @ 100 % cov") }},
	})
	s.toolbar.SetBounds(toolkit.Rect{X: 0, Y: toolkit.MenuBarH, W: w, H: toolkit.ToolbarButtonH})

	// Widgets by tab.
	s.button = toolkit.NewButton("Click me", func() { s.notify.Show("Button clicked") })
	s.check = toolkit.NewCheckButton("Enable feature", true)
	s.entry = toolkit.NewEntry("editable text")
	s.dropdown = toolkit.NewDropDown([]string{"UTF-8", "Latin-1", "Shift-JIS"}, 0)

	s.listBox = toolkit.NewListBox([]string{"apple", "banana", "cherry", "date", "elderberry"})
	s.tree = toolkit.NewTreeView(&toolkit.TreeNode{
		Label: "/", Expanded: true, Children: []*toolkit.TreeNode{
			{Label: "src", Expanded: true, Children: []*toolkit.TreeNode{{Label: "main.go"}, {Label: "scene.go"}}},
			{Label: "docs"},
			{Label: "README.md"},
		},
	})

	s.progress = toolkit.NewProgressBar()
	s.progress.Fraction = 0.66
	s.scale = toolkit.NewScale(0, 100, 50)

	s.color = toolkit.NewColorChooser(toolkit.RGB(0x0d, 0x94, 0x88))
	s.calendar = toolkit.NewCalendar(2026, 7, 1)
	s.calendar.SetToday(2026, 7, 1)

	// Notebook layout.
	s.notebook = toolkit.NewNotebook()
	s.notebook.AddTab("Buttons", s.button)
	s.notebook.AddTab("Input", s.entry)
	s.notebook.AddTab("List+Tree", s.tree)
	s.notebook.AddTab("Feedback", s.progress)
	s.notebook.AddTab("Composites", s.color)

	bodyY := toolkit.MenuBarH + toolkit.ToolbarButtonH
	statusHeight := toolkit.StatusbarH
	bodyH := h - bodyY - statusHeight
	s.notebook.SetBounds(toolkit.Rect{X: 0, Y: bodyY, W: w, H: bodyH})

	tabBodyY := bodyY + toolkit.NotebookTabStripH
	tabBodyH := bodyH - toolkit.NotebookTabStripH

	s.button.SetBounds(toolkit.Rect{X: w/2 - 60, Y: tabBodyY + 30, W: 120, H: 32})
	s.check.SetBounds(toolkit.Rect{X: 40, Y: tabBodyY + 80, W: 200, H: 24})
	s.dropdown.SetBounds(toolkit.Rect{X: 40, Y: tabBodyY + 110, W: 180, H: 24})

	s.entry.SetBounds(toolkit.Rect{X: 40, Y: tabBodyY + 30, W: w - 80, H: 28})

	s.listBox.SetBounds(toolkit.Rect{X: 40, Y: tabBodyY + 20, W: 200, H: tabBodyH - 40})
	s.tree.SetBounds(toolkit.Rect{X: 260, Y: tabBodyY + 20, W: w - 300, H: tabBodyH - 40})

	s.progress.SetBounds(toolkit.Rect{X: 40, Y: tabBodyY + 30, W: w - 80, H: 20})
	s.scale.SetBounds(toolkit.Rect{X: 40, Y: tabBodyY + 70, W: w - 80, H: 20})

	s.color.SetBounds(toolkit.Rect{X: 40, Y: tabBodyY + 20, W: w - 80, H: 120})
	s.calendar.SetBounds(toolkit.Rect{X: 40, Y: tabBodyY + 160, W: 220, H: tabBodyH - 180})

	// primary widget per tab (for hit-test dispatch).
	s.primary[0] = s.button
	s.primary[1] = s.entry
	s.primary[2] = s.tree
	s.primary[3] = s.progress
	s.primary[4] = s.color
	return s
}

// draw paints everything in draw-order onto buf.
func (s *state) draw(buf []byte) {
	fillBG(buf, s.w, s.h, s.theme.Background)
	s.menuBar.Draw(buf, s.w, s.theme)
	s.toolbar.Draw(buf, s.w, s.theme)
	s.notebook.Draw(buf, s.w, s.theme)
	switch s.notebook.Active {
	case 0:
		s.check.Draw(buf, s.w, s.theme)
		s.dropdown.Draw(buf, s.w, s.theme)
	case 2:
		s.listBox.Draw(buf, s.w, s.theme)
	case 3:
		s.scale.Draw(buf, s.w, s.theme)
	case 4:
		s.calendar.Draw(buf, s.w, s.theme)
	}
	s.status.Draw(buf, s.w, s.theme)
	// Menu popovers open when the user clicks a top-level name; paint
	// them after the status so they land on top.
	if s.menuBar.Active >= 0 && s.menuBar.Active < len(s.menuBar.Menus) {
		m := s.menuBar.Menus[s.menuBar.Active]
		// Position the popover just below the clicked name.
		nx := s.menuBar.NameOriginX(s.menuBar.Active)
		m.SetBounds(toolkit.Rect{X: nx, Y: toolkit.MenuBarH, W: 160, H: 4 + toolkit.MenuRowH*len(m.Items)})
		m.Draw(buf, s.w, s.theme)
	}
	s.notify.Draw(buf, s.w, s.theme)
}

// handleClick dispatches a click at (x, y) to whichever pane it
// falls in.
func (s *state) handleClick(x, y int) bool {
	ev := toolkit.Event{Kind: toolkit.EventClick, X: x, Y: y}

	// Menu popover first — if one is open, prefer it.
	if s.menuBar.Active >= 0 && s.menuBar.Active < len(s.menuBar.Menus) {
		m := s.menuBar.Menus[s.menuBar.Active]
		r := m.Bounds()
		if inside(x, y, r) {
			m.OnEvent(toolkit.Event{Kind: ev.Kind, X: x - r.X, Y: y - r.Y})
			s.menuBar.Active = -1
			return true
		}
		// Any click outside dismisses.
		s.menuBar.Active = -1
	}

	switch {
	case inside(x, y, s.menuBar.Bounds()):
		s.menuBar.OnEvent(local(ev, s.menuBar.Bounds()))
	case inside(x, y, s.toolbar.Bounds()):
		s.toolbar.OnEvent(local(ev, s.toolbar.Bounds()))
	default:
		s.notebook.OnEvent(local(ev, s.notebook.Bounds()))
	}
	return true
}

// tick drives the Notification's Life countdown.
func (s *state) tick() { s.notify.Tick() }

// --- helpers --------------------------------------------------------------

func fillBG(buf []byte, w, h int, c toolkit.RGBA) {
	for i := 0; i+3 < len(buf); i += 4 {
		buf[i], buf[i+1], buf[i+2], buf[i+3] = c.R, c.G, c.B, c.A
	}
	_, _ = w, h
}

func inside(x, y int, r toolkit.Rect) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

func local(ev toolkit.Event, r toolkit.Rect) toolkit.Event {
	ev.X -= r.X
	ev.Y -= r.Y
	return ev
}
