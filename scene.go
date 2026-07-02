// Scene state — composes every widget family from go-widgets/toolkit
// into a single-view dashboard. Kept in a separate file (no js/wasm
// build tag) so a native go test can exercise draw() + handleClick()
// against a plain byte buffer.

package main

import (
	"github.com/go-widgets/painter"
	"github.com/go-widgets/toolkit"
)

// Canvas dimensions. Lives in scene.go (not main.go) so the native
// scene_test compiles without the js && wasm build tag — otherwise
// the constants drop out and the tests can't reference them.
//
// The dashboard is laid out on a 960×720 grid: MenuBar (top 24px),
// Toolbar (next 24px), a three-column body of widget cards, and a
// Statusbar on the bottom 20px. Every widget kind gets its own
// labelled slot rather than being hidden behind a Notebook tab.
const (
	surfaceW = 960
	surfaceH = 720
)

// Column geometry. Three columns of equal width with an 8px outer
// margin + 8px gutter.
const (
	margin  = 8
	gutter  = 8
	colW    = (surfaceW - 2*margin - 2*gutter) / 3 // = 314
	colAX   = margin                                // 8
	colBX   = colAX + colW + gutter                 // 330
	colCX   = colBX + colW + gutter                 // 652
	sectGap = 6                                     // px between rows in a section
	sectPad = 10                                    // px between adjacent sections
)

type state struct {
	w, h  int
	theme *toolkit.Theme

	// Persistent scaffold.
	menuBar *toolkit.MenuBar
	toolbar *toolkit.Toolbar
	status  *toolkit.Statusbar
	notify  *toolkit.Notification

	// Column A — Actions & Inputs.
	actionsLabel *toolkit.Label
	button       *toolkit.Button
	toggle       *toolkit.ToggleButton
	check        *toolkit.CheckButton
	radioGroup   *toolkit.RadioGroup
	radios       []*toolkit.RadioButton

	inputsLabel *toolkit.Label
	entry       *toolkit.Entry
	spin        *toolkit.SpinButton
	scale       *toolkit.Scale
	dropdown    *toolkit.DropDown

	feedbackLabel *toolkit.Label
	progress      *toolkit.ProgressBar
	level         *toolkit.LevelBar
	spinner       *toolkit.Spinner

	// Column B — Text & Time.
	textLabel   *toolkit.Label
	textView    *toolkit.TextView
	calLabel    *toolkit.Label
	calendar    *toolkit.Calendar
	colorLabel  *toolkit.Label
	colorChoose *toolkit.ColorChooser

	// Column C — Selection & Structure.
	listLabel *toolkit.Label
	listBox   *toolkit.ListBox
	treeLabel *toolkit.Label
	tree      *toolkit.TreeView
	expLabel  *toolkit.Label
	expander  *toolkit.Expander
	frameHost *toolkit.Frame

	// Container demos to fill the vertical whitespace + demonstrate
	// composition (a leaf-only widget dashboard would leave 30 % of
	// each column empty).
	notebookLabel *toolkit.Label
	notebook      *toolkit.Notebook

	panedLabel *toolkit.Label
	paned      *toolkit.Paned

	// Live list of interactive widgets for click dispatch. Enumerated
	// in draw-order (matches the visual order the user sees) so hit-
	// testing prefers the top-most match.
	clickables []toolkit.Widget
}

func newState(w, h int) *state {
	s := &state{w: w, h: h, theme: toolkit.DefaultLight()}

	// --- top scaffold -----------------------------------------------------

	s.notify = toolkit.NewNotification("")
	s.notify.SetBounds(toolkit.Rect{X: w - 268, Y: toolkit.MenuBarH + toolkit.ToolbarButtonH + 12, W: 260, H: 24})

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
		{Label: "?", OnClick: func() { s.notify.Show("go-widgets/toolkit @ v0.6") }},
	})
	s.toolbar.SetBounds(toolkit.Rect{X: 0, Y: toolkit.MenuBarH, W: w, H: toolkit.ToolbarButtonH})

	s.status = toolkit.NewStatusbar([]string{"25 widgets", "100 % cov", "click something", "go-widgets/toolkit v0.6"})
	s.status.SetBounds(toolkit.Rect{X: 0, Y: h - toolkit.StatusbarH, W: w, H: toolkit.StatusbarH})

	// --- Column A: Actions & Inputs & Feedback ---------------------------

	y := toolkit.MenuBarH + toolkit.ToolbarButtonH + sectPad

	s.actionsLabel = toolkit.NewLabel("Actions")
	s.actionsLabel.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: toolkit.GlyphHeight})
	y += toolkit.GlyphHeight + sectGap

	s.button = toolkit.NewButton("Click me", func() { s.notify.Show("Button clicked") })
	s.button.SetBounds(toolkit.Rect{X: colAX, Y: y, W: 140, H: 28})

	s.toggle = toolkit.NewToggleButton("Toggle", false)
	s.toggle.OnToggle = func(on bool) {
		if on {
			s.notify.Show("Toggle: ON")
		} else {
			s.notify.Show("Toggle: OFF")
		}
	}
	s.toggle.SetBounds(toolkit.Rect{X: colAX + 148, Y: y, W: 140, H: 28})
	y += 28 + sectGap

	s.check = toolkit.NewCheckButton("Enable feature", true)
	s.check.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: 22})
	y += 22 + sectGap

	s.radioGroup = toolkit.NewRadioGroup()
	s.radios = []*toolkit.RadioButton{
		toolkit.NewRadioButton("Option A"),
		toolkit.NewRadioButton("Option B"),
		toolkit.NewRadioButton("Option C"),
	}
	for _, r := range s.radios {
		s.radioGroup.Add(r)
		r.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: 20})
		y += 20 + sectGap/2
	}
	s.radios[0].Checked = true
	y += sectPad

	s.inputsLabel = toolkit.NewLabel("Inputs")
	s.inputsLabel.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: toolkit.GlyphHeight})
	y += toolkit.GlyphHeight + sectGap

	s.entry = toolkit.NewEntry("editable text")
	s.entry.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: 26})
	y += 26 + sectGap

	s.spin = toolkit.NewSpinButton(0, 100, 42, 1)
	s.spin.SetBounds(toolkit.Rect{X: colAX, Y: y, W: 120, H: 26})
	s.scale = toolkit.NewScale(0, 100, 50)
	s.scale.SetBounds(toolkit.Rect{X: colAX + 128, Y: y + 4, W: colW - 128, H: 18})
	y += 26 + sectGap

	s.dropdown = toolkit.NewDropDown([]string{"UTF-8", "Latin-1", "Shift-JIS"}, 0)
	s.dropdown.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: 26})
	y += 26 + sectPad

	s.feedbackLabel = toolkit.NewLabel("Feedback")
	s.feedbackLabel.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: toolkit.GlyphHeight})
	y += toolkit.GlyphHeight + sectGap

	s.progress = toolkit.NewProgressBar()
	s.progress.Fraction = 0.66
	s.progress.Label = "66 %"
	s.progress.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: 18})
	y += 18 + sectGap

	s.level = toolkit.NewLevelBar(10)
	s.level.Value = 7
	s.level.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: 18})
	y += 18 + sectGap

	s.spinner = toolkit.NewSpinner()
	s.spinner.Active = true
	s.spinner.SetBounds(toolkit.Rect{X: colAX, Y: y, W: 24, H: 24})
	y += 24 + sectPad

	s.notebookLabel = toolkit.NewLabel("Notebook")
	s.notebookLabel.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: toolkit.GlyphHeight})
	y += toolkit.GlyphHeight + sectGap

	// Notebook demo: three tabs each hosting a Label. Notebook.Draw
	// re-sizes its active page to fill the body, which is exactly what
	// we want here — a Label with tight bounds inherits the body's
	// full width.
	s.notebook = toolkit.NewNotebook()
	s.notebook.AddTab("One", toolkit.NewLabel("First tab body"))
	s.notebook.AddTab("Two", toolkit.NewLabel("Second tab body"))
	s.notebook.AddTab("Three", toolkit.NewLabel("Third tab body"))
	s.notebook.SetBounds(toolkit.Rect{X: colAX, Y: y, W: colW, H: 80})

	// --- Column B: Text, Calendar, ColorChooser --------------------------

	yB := toolkit.MenuBarH + toolkit.ToolbarButtonH + sectPad

	s.textLabel = toolkit.NewLabel("TextView")
	s.textLabel.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight})
	yB += toolkit.GlyphHeight + sectGap

	s.textView = toolkit.NewTextView("Multi-line editor.\nType to insert.\nEnter splits a line.\nArrow keys navigate.")
	s.textView.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 110})
	yB += 110 + sectPad

	s.calLabel = toolkit.NewLabel("Calendar")
	s.calLabel.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight})
	yB += toolkit.GlyphHeight + sectGap

	s.calendar = toolkit.NewCalendar(2026, 7, 2)
	s.calendar.SetToday(2026, 7, 2)
	s.calendar.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 180})
	yB += 180 + sectPad

	s.colorLabel = toolkit.NewLabel("ColorChooser")
	s.colorLabel.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight})
	yB += toolkit.GlyphHeight + sectGap

	s.colorChoose = toolkit.NewColorChooser(toolkit.RGB(0x0d, 0x94, 0x88))
	s.colorChoose.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 130})

	// --- Column C: Selection & Structure ---------------------------------

	yC := toolkit.MenuBarH + toolkit.ToolbarButtonH + sectPad

	s.listLabel = toolkit.NewLabel("ListBox")
	s.listLabel.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight})
	yC += toolkit.GlyphHeight + sectGap

	s.listBox = toolkit.NewListBox([]string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"})
	s.listBox.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 130})
	yC += 130 + sectPad

	s.treeLabel = toolkit.NewLabel("TreeView")
	s.treeLabel.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight})
	yC += toolkit.GlyphHeight + sectGap

	s.tree = toolkit.NewTreeView(&toolkit.TreeNode{
		Label: "/", Expanded: true, Children: []*toolkit.TreeNode{
			{Label: "src", Expanded: true, Children: []*toolkit.TreeNode{
				{Label: "main.go"}, {Label: "scene.go"},
			}},
			{Label: "docs"},
			{Label: "README.md"},
		},
	})
	s.tree.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 190})
	yC += 190 + sectPad

	s.expLabel = toolkit.NewLabel("Expander + Frame")
	s.expLabel.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight})
	yC += toolkit.GlyphHeight + sectGap

	// Expander wraps a Frame that hosts a Label — showcases layout
	// composition (Container → Container → leaf) without needing a
	// second-level interactive path.
	frameLabel := toolkit.NewLabel("nested widget inside Frame")
	s.frameHost = toolkit.NewFrame(frameLabel)
	s.expander = toolkit.NewExpander("Details", s.frameHost)
	s.expander.Expanded = true
	s.expander.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 88})
	yC += 88 + sectPad

	s.panedLabel = toolkit.NewLabel("Paned (horizontal split)")
	s.panedLabel.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight})
	yC += toolkit.GlyphHeight + sectGap

	// Paned demo: horizontal split hosting two Labels. Paned.SetBounds
	// centres the handle on first sizing, so no manual Position is needed.
	s.paned = toolkit.NewHPaned(toolkit.NewLabel("left pane"), toolkit.NewLabel("right pane"))
	s.paned.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 60})

	// --- click routing table --------------------------------------------

	s.clickables = []toolkit.Widget{
		// row order matches column-A top-to-bottom, then B, then C
		s.button, s.toggle, s.check,
		s.radios[0], s.radios[1], s.radios[2],
		s.entry, s.spin, s.scale, s.dropdown,
		s.notebook,
		s.textView,
		s.calendar,
		s.colorChoose,
		s.listBox,
		s.tree,
		s.expander,
		s.paned,
	}

	return s
}

// draw paints the whole dashboard onto buf. Buf is an RGBA row-major
// slice — buf and s.w/s.h are wrapped in a PixelPainter so the widget
// code sees only the painter.Painter interface. Draw order matters:
// background first, then row scaffolding, then widget cards, then
// overlays (menu popover + notification) on top.
func (s *state) draw(buf []byte) {
	fillBG(buf, s.w, s.h, s.theme.Background)
	p := painter.NewPixelPainter(buf, s.w, s.h)

	// Top scaffold.
	s.menuBar.Draw(p, s.theme)
	s.toolbar.Draw(p, s.theme)

	// Column A — Actions & Inputs & Feedback.
	s.actionsLabel.Draw(p, s.theme)
	s.button.Draw(p, s.theme)
	s.toggle.Draw(p, s.theme)
	s.check.Draw(p, s.theme)
	for _, r := range s.radios {
		r.Draw(p, s.theme)
	}
	s.inputsLabel.Draw(p, s.theme)
	s.entry.Draw(p, s.theme)
	s.spin.Draw(p, s.theme)
	s.scale.Draw(p, s.theme)
	s.dropdown.Draw(p, s.theme)
	s.feedbackLabel.Draw(p, s.theme)
	s.progress.Draw(p, s.theme)
	s.level.Draw(p, s.theme)
	s.spinner.Draw(p, s.theme)
	s.notebookLabel.Draw(p, s.theme)
	s.notebook.Draw(p, s.theme)

	// Column B — Text & Time.
	s.textLabel.Draw(p, s.theme)
	s.textView.Draw(p, s.theme)
	s.calLabel.Draw(p, s.theme)
	s.calendar.Draw(p, s.theme)
	s.colorLabel.Draw(p, s.theme)
	s.colorChoose.Draw(p, s.theme)

	// Column C — Selection & Structure.
	s.listLabel.Draw(p, s.theme)
	s.listBox.Draw(p, s.theme)
	s.treeLabel.Draw(p, s.theme)
	s.tree.Draw(p, s.theme)
	s.expLabel.Draw(p, s.theme)
	s.expander.Draw(p, s.theme)
	s.panedLabel.Draw(p, s.theme)
	s.paned.Draw(p, s.theme)

	// Bottom scaffold.
	s.status.Draw(p, s.theme)

	// Overlays.
	if s.menuBar.Active >= 0 && s.menuBar.Active < len(s.menuBar.Menus) {
		m := s.menuBar.Menus[s.menuBar.Active]
		nx := s.menuBar.NameOriginX(s.menuBar.Active)
		m.SetBounds(toolkit.Rect{X: nx, Y: toolkit.MenuBarH, W: 160, H: 4 + toolkit.MenuRowH*len(m.Items)})
		m.Draw(p, s.theme)
	}
	s.notify.Draw(p, s.theme)
}

// handleClick dispatches a click at (x, y) to whichever widget it
// falls in. Overlays (open menu popover) take precedence; the top
// scaffold (menu bar, toolbar) comes next; the dashboard clickables
// come last, in draw order.
func (s *state) handleClick(x, y int) bool {
	ev := toolkit.Event{Kind: toolkit.EventClick, X: x, Y: y}

	// Menu popover first: if one is open, prefer it.
	if s.menuBar.Active >= 0 && s.menuBar.Active < len(s.menuBar.Menus) {
		m := s.menuBar.Menus[s.menuBar.Active]
		r := m.Bounds()
		if inside(x, y, r) {
			m.OnEvent(toolkit.Event{Kind: ev.Kind, X: x - r.X, Y: y - r.Y})
			s.menuBar.Active = -1
			return true
		}
		// Any click outside dismisses the popover.
		s.menuBar.Active = -1
	}

	// Top scaffold.
	if inside(x, y, s.menuBar.Bounds()) {
		s.menuBar.OnEvent(local(ev, s.menuBar.Bounds()))
		return true
	}
	if inside(x, y, s.toolbar.Bounds()) {
		s.toolbar.OnEvent(local(ev, s.toolbar.Bounds()))
		return true
	}

	// Dashboard clickables — first hit wins (draw-order = z-order).
	for _, w := range s.clickables {
		r := w.Bounds()
		if inside(x, y, r) {
			w.OnEvent(local(ev, r))
			return true
		}
	}
	return true
}

// tick drives per-frame widget animations. Notification counts down
// its Life; Spinner advances its Phase by roughly one 60 Hz frame.
func (s *state) tick() {
	s.notify.Tick()
	s.spinner.Tick(1.0 / 60)
}

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
