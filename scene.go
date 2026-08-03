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
	surfaceH = 1483
)

// themeRowH sizes the ViewSwitcher strip sitting between the Toolbar
// and the column grid. Keep the strip roomy enough that the segment
// labels don't clip on the 5×7 bitmap font.
const themeRowH = 26

// Column geometry. Three columns of equal width with an 8px outer
// margin + 8px gutter.
const (
	margin  = 8
	gutter  = 8
	colW    = (surfaceW - 2*margin - 2*gutter) / 3 // = 314
	colAX   = margin                               // 8
	colBX   = colAX + colW + gutter                // 330
	colCX   = colBX + colW + gutter                // 652
	sectGap = 6                                    // px between rows in a section
	sectPad = 10                                   // px between adjacent sections
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
	button     *toolkit.Button
	toggle     *toolkit.ToggleButton
	check      *toolkit.CheckButton
	radioGroup *toolkit.RadioGroup
	radios     []*toolkit.RadioButton

	entry    *toolkit.Entry
	spin     *toolkit.SpinButton
	scale    *toolkit.Scale
	dropdown *toolkit.DropDown

	progress *toolkit.ProgressBar
	level    *toolkit.LevelBar
	spinner  *toolkit.Spinner

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
	notebook *toolkit.Notebook

	panedLabel *toolkit.Label
	paned      *toolkit.Paned

	// Column-A Wave 1 (v0.7) highlights.
	swtch *toolkit.Switch
	alert *toolkit.Alert
	card  *toolkit.Card
	steps *toolkit.Steps

	// Column-B Wave 2 (v0.8) highlights.
	wave2Label *toolkit.Label
	toast      *toolkit.Toast
	banner     *toolkit.Banner
	headerBar  *toolkit.HeaderBar
	diff       *toolkit.Diff

	// Column-C Wave 3 (v0.9) highlights.
	wave3Label     *toolkit.Label
	stat           *toolkit.Stat
	timeline       *toolkit.Timeline
	chip           *toolkit.Chip
	progressCircle *toolkit.ProgressCircle
	splitButton    *toolkit.SplitButton

	// Column-A Wave 4 (v0.33) highlights: a vertical Toolbar side rail
	// + a vertical Steps checklist, demonstrating Toolbar.Orientation
	// and Steps.Orientation.
	toolbarV *toolkit.Toolbar
	stepsV   *toolkit.Steps

	// Column-B Wave 4 (v0.33) highlights: a second, smaller Notebook
	// with its tab strip on the left (Notebook.TabSide) + a second
	// DropDown that opens its popover upward (DropDown.OpenUp).
	wave4LabelB  *toolkit.Label
	notebookSide *toolkit.Notebook
	dropdownUp   *toolkit.DropDown

	// Column-C Wave 4 (v0.33) highlights: a Table with a right-aligned
	// numeric column (TableColumn.Align) + a horizontal Timeline ribbon
	// (Timeline.Horizontal).
	wave4LabelC *toolkit.Label
	table       *toolkit.Table
	timelineH   *toolkit.Timeline

	// Column-A Wave 5 (v0.42) highlights: Accordion (collapsible
	// sections), ColorPicker (HSV square + hue/alpha sliders) and
	// SegmentedBar (proportional multi-colour meter).
	accordion   *toolkit.Accordion
	colorPicker *toolkit.ColorPicker
	segBar      *toolkit.SegmentedBar

	// Column-B Wave 5 (v0.42) highlights: Carousel (paged slide viewer),
	// MarkdownEditor (live source/preview split) and DateRangePicker
	// (two-endpoint month-grid selection).
	wave5LabelB *toolkit.Label
	carousel    *toolkit.Carousel
	mdEditor    *toolkit.MarkdownEditor
	dateRange   *toolkit.DateRangePicker

	// Column-C Wave 5 (v0.42) highlights: Wizard (multi-step flow) and
	// TreeTable (Table-shaped grid with nesting). A CommandPalette is
	// wired as a canvas-wide overlay, opened from a trigger Button here
	// and drawn last (like Notification) so it floats above everything.
	wave5LabelC *toolkit.Label
	wizard      *toolkit.Wizard
	treeTable   *toolkit.TreeTable
	paletteBtn  *toolkit.Button
	cmdPalette  *toolkit.CommandPalette

	// Wave 6 (v0.81) highlights — the Table grid-editing family + the
	// widgets it enabled. Column A: a PropertyGrid (editable Name/Value)
	// above a PagingToolbar. Column B: a Table showing grouped rows + an
	// open inline cell editor. Column C: a ListBox with a DataView
	// ItemRenderer painting rich two-line rows.
	propGrid    *toolkit.PropertyGrid
	pagingBar   *toolkit.PagingToolbar
	wave6LabelB *toolkit.Label
	gridEdit    *toolkit.Table
	wave6LabelC *toolkit.Label
	dataView    *toolkit.ListBox

	// Theme switcher (ViewSwitcher v0.8) sits above the column grid.
	// Each segment installs a distinct palette so the whole scene
	// repaints on click — validates that the toolkit's Theme value
	// cascades through every widget uniformly, and demonstrates
	// LoadGTKTheme on the "Adwaita" entries.
	themeSwitcher *toolkit.ViewSwitcher
	themes        []*toolkit.Theme
	themeNames    []string

	// colA is Column A composed with the box-layout system: a VBox of
	// titled Frames (one per section), each Frame wrapping a VBox/HBox of
	// the section's widgets. It replaces the hand-computed rects + pushCard
	// borders + section Labels for Column A -- SetBounds cascades absolute
	// positions to every widget (so the clickables list still hit-tests), and
	// draw() renders the whole column with a single colA.Draw. (Columns B/C
	// are still hand-laid pending their own migration.)
	colA *toolkit.VBox

	// Live list of interactive widgets for click dispatch. Enumerated
	// in draw-order (matches the visual order the user sees) so hit-
	// testing prefers the top-most match.
	clickables []toolkit.Widget

	// Card outlines painted underneath the widgets. Each rect covers
	// one section — the GTK4 widget-factory pattern of a bordered
	// group. Recorded during layout in newState, stroked in draw().
	cards []toolkit.Rect
}

// cardPad is the extra pixels a card extends past its inner-most
// widget rect on every side. Kept small so cards feel tight.
const cardPad = 6

// frameChromeH is the vertical space a titled Frame adds around its child:
// the 1px top+bottom border, the FrameTitleH title bar and the default 4px
// padding on top+bottom. Used to size a section Frame from its content height.
const frameChromeH = 2 + toolkit.FrameTitleH + 2*4

// boxItem pairs a widget with the fixed height it occupies in a VBox column.
type boxItem struct {
	w toolkit.Widget
	h int
}

// sectionFrame wraps items in a titled Frame whose child is a vertical box
// (VBox) stacking them with the given spacing. It returns the Frame and the
// total height the Frame needs (content + title bar + border + padding), so a
// column VBox can AddFixed it without guessing.
func sectionFrame(title string, spacing int, items ...boxItem) (*toolkit.Frame, int) {
	vb := toolkit.NewVBox()
	vb.Spacing = spacing
	content := 0
	for i, it := range items {
		vb.AddFixed(it.w, it.h)
		content += it.h
		if i > 0 {
			content += spacing
		}
	}
	f := toolkit.NewFrame(vb)
	f.Title = title
	return f, content + frameChromeH
}

// hrowFlex lays widgets left-to-right with equal flex weight (an HBox), for a
// section row that puts two widgets side by side (e.g. Button + ToggleButton).
func hrowFlex(spacing int, ws ...toolkit.Widget) toolkit.Widget {
	hb := toolkit.NewHBox()
	hb.Spacing = spacing
	for _, w := range ws {
		hb.AddFlex(w, 1)
	}
	return hb
}

// hrowFixedFlex lays a fixed-width widget beside a flex-filling one (an HBox),
// for rows like SpinButton (fixed) + Scale (fill) or the vertical Toolbar rail
// (fixed) + vertical Steps (fill).
func hrowFixedFlex(spacing, fixedW int, fixed, flex toolkit.Widget) toolkit.Widget {
	hb := toolkit.NewHBox()
	hb.Spacing = spacing
	hb.AddFixed(fixed, fixedW)
	hb.AddFlex(flex, 1)
	return hb
}

// column stacks section Frames into a single VBox column, spaced by sectPad,
// and returns the VBox plus its total height (sum of the frame heights + the
// inter-section spacing) so newState can position it exactly.
func column(frames []*toolkit.Frame, heights []int) (*toolkit.VBox, int) {
	vb := toolkit.NewVBox()
	vb.Spacing = sectPad
	total := 0
	for i, f := range frames {
		vb.AddFixed(f, heights[i])
		total += heights[i]
		if i > 0 {
			total += sectPad
		}
	}
	return vb, total
}

func newState(w, h int) *state {
	s := &state{w: w, h: h, theme: toolkit.DefaultLight()}

	// --- top scaffold -----------------------------------------------------

	// Notification hosts a floating toast at bottom-right, just above
	// the Statusbar. Anchored bottom-right (not top-right) so it never
	// collides with the ListBox / TreeView headers at the top of
	// column C when it's visible.
	s.notify = toolkit.NewNotification("")
	s.notify.SetBounds(toolkit.Rect{X: w - 268, Y: h - toolkit.StatusbarH - 32, W: 260, H: 24})

	// CommandPalette is a canvas-wide overlay (like the menu popover):
	// Bounds spans the whole surface so it can catch an outside click
	// anywhere + centre its panel, and incoming event coordinates need
	// no translation since X/Y are already 0.
	s.cmdPalette = toolkit.NewCommandPalette([]toolkit.PaletteCommand{
		{Label: "New file", Action: func() { s.showNotify("Palette: New file") }},
		{Label: "Open recent", Action: func() { s.showNotify("Palette: Open recent") }},
		{Label: "Toggle theme", Action: func() { s.showNotify("Palette: Toggle theme") }},
		{Label: "About gallery", Action: func() { s.showNotify("Palette: About gallery") }},
	})
	s.cmdPalette.SetBounds(toolkit.Rect{X: 0, Y: 0, W: w, H: h})

	menu := func(label string) toolkit.MenuItem {
		return toolkit.MenuItem{Label: label, Action: func() { s.showNotify("clicked: " + label) }}
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
		{Label: "N", OnClick: func() { s.showNotify("Toolbar: New") }},
		{Label: "O", OnClick: func() { s.showNotify("Toolbar: Open") }},
		{Label: "S", OnClick: func() { s.showNotify("Toolbar: Save") }},
		{Separator: true},
		{Label: "C", OnClick: func() { s.showNotify("Toolbar: Copy") }},
		{Label: "X", OnClick: func() { s.showNotify("Toolbar: Cut") }},
		{Label: "V", OnClick: func() { s.showNotify("Toolbar: Paste") }},
		{Separator: true},
		{Label: "?", OnClick: func() { s.showNotify("go-widgets/toolkit @ v0.81.0") }},
	})
	s.toolbar.SetBounds(toolkit.Rect{X: 0, Y: toolkit.MenuBarH, W: w, H: toolkit.ToolbarButtonH})

	s.status = toolkit.NewStatusbar([]string{"~90 widgets", "100 % cov", "click something", "go-widgets/toolkit v0.81.0"})
	s.status.SetBounds(toolkit.Rect{X: 0, Y: h - toolkit.StatusbarH, W: w, H: toolkit.StatusbarH})

	// --- Theme switcher (ViewSwitcher v0.8) -----------------------------
	//
	// Sits between the Toolbar and the column grid. Four palettes:
	//   * Light    — toolkit.DefaultLight()
	//   * Dark     — toolkit.DefaultDark()
	//   * Adwaita  — parsed via LoadGTKTheme from an inline libadwaita
	//     palette (validates the CSS parser end-to-end at run time).
	//   * WhiteSur — parsed from the vinceliuice WhiteSur GTK theme's
	//     light-blue variant colour palette (Big Sur inspired). Source:
	//     github.com/vinceliuice/WhiteSur-gtk-theme, sass/_colors.scss +
	//     _colors-palette.scss ($theme_color_default = #0860F2,
	//     $base_color = #ffffff, $text_color = #363636, ...).
	adwaita, _ := toolkit.LoadGTKTheme(`
		@define-color window_bg_color   #fafafa;
		@define-color window_fg_color   #2e3436;
		@define-color view_bg_color     #ffffff;
		@define-color view_fg_color     #2e3436;
		@define-color card_bg_color     #f6f5f4;
		@define-color accent_bg_color   #3584e4;
		@define-color borders           #c0bfbc;
	`)
	whiteSur, _ := toolkit.LoadGTKTheme(`
		@define-color window_bg_color   #f5f5f5;
		@define-color window_fg_color   #242424;
		@define-color view_bg_color     #ffffff;
		@define-color view_fg_color     #363636;
		@define-color card_bg_color     #ececec;
		@define-color accent_bg_color   #0860F2;
		@define-color borders           #d1d1d6;
	`)
	s.themes = []*toolkit.Theme{
		toolkit.DefaultLight(),
		toolkit.DefaultDark(),
		adwaita,
		whiteSur,
	}
	s.themeNames = []string{"Light", "Dark", "Adwaita", "WhiteSur"}
	s.themeSwitcher = toolkit.NewViewSwitcher(s.themeNames, 0)
	s.themeSwitcher.OnChange = func(i int) {
		s.theme = s.themes[i]
		s.showNotify("Theme: " + s.themeNames[i])
	}
	s.themeSwitcher.SetBounds(toolkit.Rect{
		X: margin,
		Y: toolkit.MenuBarH + toolkit.ToolbarButtonH + sectPad,
		W: w - 2*margin,
		H: themeRowH,
	})

	// --- Column A: Actions & Inputs & Feedback ---------------------------

	// Column A widgets — CONSTRUCTION ONLY. Placement is owned by the
	// box-layout colA (a VBox of titled Frames) built near the end of
	// newState; no hand-computed rects here. (Columns B/C below still use the
	// running y/yB/yC hand-layout pending their own migration.)
	s.button = toolkit.NewButton("Click me", func() { s.showNotify("Button clicked") })

	s.toggle = toolkit.NewToggleButton("Toggle", false)
	s.toggle.OnToggle = func(on bool) {
		if on {
			s.showNotify("Toggle: ON")
		} else {
			s.showNotify("Toggle: OFF")
		}
	}

	s.check = toolkit.NewCheckButton("Enable feature", true)

	s.radioGroup = toolkit.NewRadioGroup()
	s.radios = []*toolkit.RadioButton{
		toolkit.NewRadioButton("Option A"),
		toolkit.NewRadioButton("Option B"),
		toolkit.NewRadioButton("Option C"),
	}
	for _, r := range s.radios {
		s.radioGroup.Add(r)
	}
	s.radios[0].Checked = true

	s.entry = toolkit.NewEntry("editable text")
	s.spin = toolkit.NewSpinButton(0, 100, 42, 1)
	s.scale = toolkit.NewScale(0, 100, 50)
	s.dropdown = toolkit.NewDropDown([]string{"UTF-8", "Latin-1", "Shift-JIS"}, 0)

	s.progress = toolkit.NewProgressBar()
	s.progress.Fraction = 0.66
	s.progress.Label = "66 %"

	s.level = toolkit.NewLevelBar(10)
	s.level.Value = 7

	s.spinner = toolkit.NewSpinner()
	s.spinner.Active = true

	// Notebook demo: three tabs each hosting a Label. Notebook.Draw
	// re-sizes its active page to fill the body, which is exactly what
	// we want here — a Label with tight bounds inherits the body's
	// full width.
	// The tabs showcase the v0.12-v0.15 display widgets — a live chart family
	// plus a Markdown view. The Notebook bounds + draws only the active page,
	// so each is auto-sized to the body area; all are display-only, so no
	// per-tab event routing is needed.
	s.notebook = toolkit.NewNotebook()
	s.notebook.AddTab("Line", toolkit.NewLineChart([]float64{3, 7, 2, 8, 5, 9, 4, 6}))
	s.notebook.AddTab("Bar", toolkit.NewBarChart([]float64{4, 7, 2, 8, 5, 3}))
	s.notebook.AddTab("Pie", toolkit.NewPieChart([]float64{3, 5, 2, 4, 1}))
	s.notebook.AddTab("Docs", toolkit.NewMarkdownView(
		"# Charts\n\nLive toolkit charts, one per tab:\n\n- line\n- bar\n- pie"))

	// --- Column B: Text, Calendar, ColorChooser --------------------------

	yB := toolkit.MenuBarH + toolkit.ToolbarButtonH + sectPad + themeRowH + sectPad
	cardStartB := yB

	s.textLabel = toolkit.NewLabel("TextView")
	s.textLabel.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight()})
	yB += toolkit.GlyphHeight() + sectGap

	s.textView = toolkit.NewTextView("Multi-line editor.\nType to insert.\nEnter splits a line.\nArrow keys navigate.")
	s.textView.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 110})
	yB += 110
	s.pushCard(colBX, cardStartB, colW, yB-cardStartB)
	yB += sectPad
	cardStartB = yB

	s.calLabel = toolkit.NewLabel("Calendar")
	s.calLabel.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight()})
	yB += toolkit.GlyphHeight() + sectGap

	s.calendar = toolkit.NewCalendar(2026, 7, 2)
	s.calendar.SetToday(2026, 7, 2)
	s.calendar.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 180})
	yB += 180
	s.pushCard(colBX, cardStartB, colW, yB-cardStartB)
	yB += sectPad
	cardStartB = yB

	s.colorLabel = toolkit.NewLabel("ColorChooser")
	s.colorLabel.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight()})
	yB += toolkit.GlyphHeight() + sectGap

	s.colorChoose = toolkit.NewColorChooser(toolkit.RGB(0x0d, 0x94, 0x88))
	s.colorChoose.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 130})
	yB += 130
	s.pushCard(colBX, cardStartB, colW, yB-cardStartB)

	// --- Column C: Selection & Structure ---------------------------------

	yC := toolkit.MenuBarH + toolkit.ToolbarButtonH + sectPad + themeRowH + sectPad
	cardStartC := yC

	s.listLabel = toolkit.NewLabel("ListBox")
	s.listLabel.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight()})
	yC += toolkit.GlyphHeight() + sectGap

	s.listBox = toolkit.NewListBox([]string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"})
	s.listBox.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 130})
	yC += 130
	s.pushCard(colCX, cardStartC, colW, yC-cardStartC)
	yC += sectPad
	cardStartC = yC

	s.treeLabel = toolkit.NewLabel("TreeView")
	s.treeLabel.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight()})
	yC += toolkit.GlyphHeight() + sectGap

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
	yC += 190
	s.pushCard(colCX, cardStartC, colW, yC-cardStartC)
	yC += sectPad
	cardStartC = yC

	s.expLabel = toolkit.NewLabel("Expander + Frame")
	s.expLabel.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight()})
	yC += toolkit.GlyphHeight() + sectGap

	// Expander wraps a Frame that hosts a Label — showcases layout
	// composition (Container → Container → leaf) without needing a
	// second-level interactive path.
	frameLabel := toolkit.NewLabel("nested widget inside Frame")
	s.frameHost = toolkit.NewFrame(frameLabel)
	s.expander = toolkit.NewExpander("Details", s.frameHost)
	s.expander.Expanded = true
	s.expander.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 88})
	yC += 88
	s.pushCard(colCX, cardStartC, colW, yC-cardStartC)
	yC += sectPad
	cardStartC = yC

	s.panedLabel = toolkit.NewLabel("Paned (horizontal split)")
	s.panedLabel.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight()})
	yC += toolkit.GlyphHeight() + sectGap

	// Paned demo: horizontal split hosting two Labels. Paned.SetBounds
	// centres the handle on first sizing, so no manual Position is needed.
	s.paned = toolkit.NewHPaned(toolkit.NewLabel("left pane"), toolkit.NewLabel("right pane"))
	s.paned.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 60})
	yC += 60
	s.pushCard(colCX, cardStartC, colW, yC-cardStartC)

	// --- Column A extension: Wave 1 (v0.7) — construction only ----------
	s.swtch = toolkit.NewSwitch(true)
	s.swtch.OnToggle = func(on bool) {
		if on {
			s.showNotify("Switch: ON")
		} else {
			s.showNotify("Switch: OFF")
		}
	}
	s.alert = toolkit.NewAlert("Saved 3 minutes ago.", toolkit.AlertSuccess)
	s.card = toolkit.NewCard("Card", "Title above.\nBody here.", "footer note")
	s.steps = toolkit.NewSteps([]string{"Plan", "Build", "Test", "Ship"}, 1)

	// --- Column A extension: Wave 4 (v0.33) — construction only ---------
	//
	// A vertical Toolbar side rail (Toolbar.Orientation = Vertical) beside a
	// vertical Steps checklist (Steps.Orientation = Vertical); colA lays them
	// side by side in an HBox (see the Wave 4 section frame).
	s.toolbarV = toolkit.NewToolbar([]toolkit.ToolbarItem{
		{Label: "A", OnClick: func() { s.showNotify("Side rail: A") }},
		{Label: "B", OnClick: func() { s.showNotify("Side rail: B") }},
		{Label: "C", OnClick: func() { s.showNotify("Side rail: C") }},
	})
	s.toolbarV.Orientation = toolkit.Vertical

	s.stepsV = toolkit.NewSteps([]string{"Plan", "Build", "Ship"}, 1)
	s.stepsV.Orientation = toolkit.Vertical

	// --- Column B extension: Wave 2 (v0.8) highlights -------------------

	yB += sectPad
	cardStartB = yB

	s.wave2Label = toolkit.NewLabel("Wave 2 (v0.8)")
	s.wave2Label.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight()})
	yB += toolkit.GlyphHeight() + sectGap

	s.headerBar = toolkit.NewHeaderBar("Files")
	s.headerBar.Subtitle = "~/Documents"
	s.headerBar.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 36})
	yB += 36 + sectGap

	s.toast = toolkit.NewToast("Copied to clipboard", toolkit.ToastSuccess)
	s.toast.Visible = true
	s.toast.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 24})
	yB += 24 + sectGap

	s.banner = toolkit.NewBanner("Update available.")
	s.banner.ButtonLabel = "Install"
	s.banner.OnAction = func() { s.showNotify("Banner action clicked") }
	s.banner.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 24})
	yB += 24 + sectGap

	s.diff = toolkit.NewDiff([]toolkit.DiffLine{
		{Text: "package main", Kind: toolkit.DiffContext},
		{Text: "old line", Kind: toolkit.DiffRemoved},
		{Text: "new line", Kind: toolkit.DiffAdded},
	})
	s.diff.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 54})
	yB += 54
	s.pushCard(colBX, cardStartB, colW, yB-cardStartB)

	// --- Column B extension: Wave 4 (v0.33) highlights -------------------
	//
	// A second, smaller Notebook with its tab strip on the left
	// (Notebook.TabSide = TabLeft) sits above a second DropDown that
	// opens its popover upward (DropDown.OpenUp), the natural choice
	// for a control this close to the Statusbar.

	yB += sectPad
	cardStartB = yB

	s.wave4LabelB = toolkit.NewLabel("Wave 4 (v0.33)")
	s.wave4LabelB.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight()})
	yB += toolkit.GlyphHeight() + sectGap

	s.notebookSide = toolkit.NewNotebook()
	s.notebookSide.TabSide = toolkit.TabLeft
	s.notebookSide.AddTab("A", toolkit.NewLabel("Tabs on the left"))
	s.notebookSide.AddTab("B", toolkit.NewLabel("(TabSide = TabLeft)"))
	s.notebookSide.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 70})
	yB += 70 + sectGap

	s.dropdownUp = toolkit.NewDropDown([]string{"Opens upward", "OpenUp = true"}, 0)
	s.dropdownUp.OpenUp = true
	s.dropdownUp.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 26})
	yB += 26
	s.pushCard(colBX, cardStartB, colW, yB-cardStartB)

	// --- Column C extension: Wave 3 (v0.9) highlights -------------------

	yC += sectPad
	cardStartC = yC

	s.wave3Label = toolkit.NewLabel("Wave 3 (v0.9)")
	s.wave3Label.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight()})
	yC += toolkit.GlyphHeight() + sectGap

	s.stat = toolkit.NewStat("Requests / min", "12,845")
	s.stat.Change = "+8.3%"
	s.stat.Trend = toolkit.StatUp
	s.stat.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW/2 - 2, H: 60})
	s.progressCircle = toolkit.NewProgressCircle()
	s.progressCircle.Fraction = 0.66
	s.progressCircle.SetBounds(toolkit.Rect{X: colCX + colW/2 + 2, Y: yC, W: colW/2 - 2, H: 60})
	yC += 60 + sectGap

	s.timeline = toolkit.NewTimeline([]toolkit.TimelineEvent{
		{Title: "PR opened", Kind: toolkit.TimelineDefault},
		{Title: "Reviewed", Detail: "LGTM", Kind: toolkit.TimelineSuccess},
		{Title: "Build failed", Kind: toolkit.TimelineError},
	})
	s.timeline.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 68})
	yC += 68 + sectGap

	s.chip = toolkit.NewChip("frontend")
	s.chip.Closable = true
	s.chip.OnClose = func() { s.showNotify("Chip closed") }
	s.chip.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: 96, H: 22})
	s.splitButton = toolkit.NewSplitButton("Deploy",
		func() { s.showNotify("SplitButton: Deploy") })
	s.splitButton.OnArrow = func() { s.showNotify("SplitButton: arrow menu") }
	s.splitButton.SetBounds(toolkit.Rect{X: colCX + 104, Y: yC, W: colW - 104, H: 22})
	yC += 22
	s.pushCard(colCX, cardStartC, colW, yC-cardStartC)

	// --- Column C extension: Wave 4 (v0.33) highlights -------------------
	//
	// A Table with its numeric column right-aligned (TableColumn.Align =
	// AlignRight) sits above a horizontal Timeline ribbon (Timeline.
	// Horizontal = true) — the same event log as the vertical Timeline
	// above, rotated onto the layout axis a status ribbon would use.

	yC += sectPad
	cardStartC = yC

	s.wave4LabelC = toolkit.NewLabel("Wave 4 (v0.33)")
	s.wave4LabelC.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight()})
	yC += toolkit.GlyphHeight() + sectGap

	s.table = toolkit.NewTable(
		[]toolkit.TableColumn{
			{Title: "Widget"},
			{Title: "Count", Width: 60, Align: toolkit.AlignRight},
		},
		[][]string{
			{"Buttons", "12"},
			{"Inputs", "9"},
			{"Charts", "3"},
		},
	)
	tableH := toolkit.TableHeaderHeight + 3*toolkit.TableRowHeight
	s.table.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: tableH})
	yC += tableH + sectGap

	s.timelineH = toolkit.NewTimeline([]toolkit.TimelineEvent{
		{Title: "Init", Kind: toolkit.TimelineDefault},
		{Title: "Build", Kind: toolkit.TimelineSuccess},
		{Title: "Deploy", Kind: toolkit.TimelineWarning},
	})
	s.timelineH.Horizontal = true
	s.timelineH.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 36})
	yC += 36
	s.pushCard(colCX, cardStartC, colW, yC-cardStartC)

	// --- Column A extension: Wave 5 (v0.42) — construction only ---------
	//
	// Accordion (collapsible sections) + ColorPicker + SegmentedBar; colA
	// stacks them in the Wave 5 section frame.
	s.accordion = toolkit.NewAccordion([]toolkit.AccordionSection{
		{Title: "Specs", Body: toolkit.NewLabel("2.4 GHz / 16 GB / 512 GB")},
		{Title: "Shipping", Body: toolkit.NewLabel("Ships in 2-3 business days")},
		{Title: "Returns", Body: toolkit.NewLabel("30-day free returns")},
	})
	s.accordion.Expanded = 1

	s.colorPicker = toolkit.NewColorPicker(toolkit.RGB(0x35, 0x84, 0xe4))
	s.colorPicker.OnChange = func(c toolkit.RGBA) { s.showNotify("ColorPicker changed") }

	s.segBar = toolkit.NewSegmentedBar([]toolkit.BarSegment{
		{Value: 62, Fill: toolkit.RGB(0x35, 0x84, 0xe4), Label: "used"},
		{Value: 18, Fill: toolkit.RGB(0xe5, 0xa5, 0x0a), Label: "reserved"},
		{Value: 20, Fill: toolkit.RGB(0xc0, 0xbf, 0xbc), Label: "free"},
	})

	// --- Column B extension: Wave 5 (v0.42) highlights -------------------
	//
	// Carousel (three paged slides) sits above a MarkdownEditor (live
	// source/preview split) and a DateRangePicker (two-endpoint month
	// grid selection), the v0.42 text/time-family additions.

	yB += sectPad
	cardStartB = yB

	s.wave5LabelB = toolkit.NewLabel("Wave 5 (v0.42)")
	s.wave5LabelB.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight()})
	yB += toolkit.GlyphHeight() + sectGap

	s.carousel = toolkit.NewCarousel([]toolkit.Widget{
		toolkit.NewCard("Slide 1", "First featured panel.", ""),
		toolkit.NewCard("Slide 2", "Second featured panel.", ""),
		toolkit.NewCard("Slide 3", "Third featured panel.", ""),
	})
	s.carousel.Wrap = true
	const carouselH = 74 + 16 // content + dots strip
	s.carousel.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: carouselH})
	yB += carouselH + sectGap

	s.mdEditor = toolkit.NewMarkdownEditor("# Notes\n\n- live *preview*\n- side by side")
	const mdEditorH = 90
	s.mdEditor.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: mdEditorH})
	yB += mdEditorH + sectGap

	s.dateRange = toolkit.NewDateRangePicker(2026, 7)
	s.dateRange.Start = toolkit.Date{Y: 2026, M: 7, D: 10}
	s.dateRange.End = toolkit.Date{Y: 2026, M: 7, D: 17}
	s.dateRange.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: 168})
	yB += 168
	s.pushCard(colBX, cardStartB, colW, yB-cardStartB)

	// --- Column C extension: Wave 5 (v0.42) highlights -------------------
	//
	// Wizard (multi-step Plan → Build → Ship flow) sits above a TreeTable
	// (Table-shaped grid with nesting) and a trigger Button that opens the
	// CommandPalette overlay constructed up in newState's top scaffold.

	yC += sectPad
	cardStartC = yC

	s.wave5LabelC = toolkit.NewLabel("Wave 5 (v0.42)")
	s.wave5LabelC.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight()})
	yC += toolkit.GlyphHeight() + sectGap

	s.wizard = toolkit.NewWizard([]toolkit.WizardStep{
		{Title: "Plan", Body: toolkit.NewLabel("Sketch the release scope.")},
		{Title: "Build", Body: toolkit.NewLabel("Wire the new widgets in.")},
		{Title: "Ship", Body: toolkit.NewLabel("Tag + publish the gallery.")},
	})
	const wizardH = toolkit.WizardStripH + 40 + toolkit.WizardButtonRowH
	s.wizard.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: wizardH})
	yC += wizardH + sectGap

	s.treeTable = toolkit.NewTreeTable(
		[]toolkit.TreeTableColumn{
			{Title: "Widget"},
			{Title: "Kind", Width: 90},
		},
		[]*toolkit.TreeTableNode{
			{Cells: []string{"Column A", "group"}, Expanded: true, Children: []*toolkit.TreeTableNode{
				{Cells: []string{"Accordion", "leaf"}},
				{Cells: []string{"ColorPicker", "leaf"}},
			}},
			{Cells: []string{"Column B", "group"}, Children: []*toolkit.TreeTableNode{
				{Cells: []string{"Carousel", "leaf"}},
			}},
		},
	)
	const treeTableH = toolkit.TreeTableHeaderHeight + 4*toolkit.TreeTableRowHeight
	s.treeTable.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: treeTableH})
	yC += treeTableH + sectGap

	s.paletteBtn = toolkit.NewButton("Open command palette ⌘", func() { s.cmdPalette.Open() })
	s.paletteBtn.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: 28})
	yC += 28
	s.pushCard(colCX, cardStartC, colW, yC-cardStartC)

	// --- Column A extension: Wave 6 (v0.81) — construction only ---------
	//
	// A PropertyGrid (2-column Name/Value, Value inline-editable) above a
	// PagingToolbar (First/Prev/"Page N of M"/Next/Last + Refresh); colA
	// stacks them in the Wave 6 section frame.
	s.propGrid = toolkit.NewPropertyGrid()
	s.propGrid.Add("Width", "1024")
	s.propGrid.Add("Height", "768")
	s.propGrid.Add("Title", "Untitled")
	s.propGrid.Add("Visible", "true")
	s.propGrid.Table().Selected = 2
	// Open the editor on the "Title" value cell so the demo shows editing.
	s.propGrid.OnEvent(toolkit.Event{Kind: toolkit.EventClick, X: colW - 40, Y: toolkit.TableHeaderHeight + 2*toolkit.TableRowHeight + 2})

	s.pagingBar = toolkit.NewPagingToolbar(6, 12)
	s.pagingBar.ShowRefresh = true

	// --- Column B extension: Wave 6 (v0.81) highlights -------------------
	//
	// A Table showing collapsible group rows (grouped by the first column)
	// with an inline cell editor open over an editable cell.

	yB += sectPad
	cardStartB = yB

	s.wave6LabelB = toolkit.NewLabel("Wave 6 (v0.81)")
	s.wave6LabelB.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: toolkit.GlyphHeight()})
	yB += toolkit.GlyphHeight() + sectGap

	s.gridEdit = toolkit.NewTable(
		[]toolkit.TableColumn{
			{Title: "Status", Width: 90},
			{Title: "Owner", Width: 90, Editable: true},
			{Title: "Task"},
		},
		[][]string{
			{"in-progress", "alice", "Cell editing"},
			{"in-progress", "bob", "Group rows"},
			{"todo", "carol", "Frozen cols"},
			{"done", "dave", "Panels"},
		},
	)
	s.gridEdit.GroupBy = 0
	const gridEditH = toolkit.TableHeaderHeight + 7*toolkit.TableRowHeight
	s.gridEdit.SetBounds(toolkit.Rect{X: colBX, Y: yB, W: colW, H: gridEditH})
	// Open the Owner editor on the first data row (visual line 1).
	s.gridEdit.OnEvent(toolkit.Event{Kind: toolkit.EventClick, X: 130, Y: toolkit.TableHeaderHeight + toolkit.TableRowHeight + 2})
	s.gridEdit.OnEvent(toolkit.Event{Kind: toolkit.EventChar, Code: "!"})
	yB += gridEditH
	s.pushCard(colBX, cardStartB, colW, yB-cardStartB)

	// --- Column C extension: Wave 6 (v0.81) highlights -------------------
	//
	// A ListBox with a DataView ItemRenderer: each row draws a colour
	// swatch + a two-line title/subtitle instead of the default text.

	yC += sectPad
	cardStartC = yC

	s.wave6LabelC = toolkit.NewLabel("Wave 6 (v0.81)")
	s.wave6LabelC.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: toolkit.GlyphHeight()})
	yC += toolkit.GlyphHeight() + sectGap

	dvSubs := []string{"12 unread · 2m ago", "3 unread · 1h ago", "all read · yesterday", "8 unread · 5m ago"}
	dvSwatch := []toolkit.RGBA{toolkit.RGB(0xe0, 0x50, 0x50), toolkit.RGB(0x50, 0xa0, 0xe0), toolkit.RGB(0x50, 0xb0, 0x70), toolkit.RGB(0xc0, 0x80, 0xe0)}
	s.dataView = toolkit.NewListBox([]string{"Reddit", "Hacker News", "Lobsters", "GitHub"})
	s.dataView.RowHeight = 32
	s.dataView.Selected = 1
	s.dataView.ItemRenderer = func(p painter.Painter, theme *toolkit.Theme, rc toolkit.Rect, i int, item string, sel bool, ink toolkit.RGBA) {
		p.FillRect(painter.Rect{X: rc.X + 8, Y: rc.Y + rc.H/2 - 6, W: 12, H: 12}, dvSwatch[i])
		title := toolkit.NewLabel(item)
		title.Ink = ink
		title.SetBounds(toolkit.Rect{X: rc.X + 28, Y: rc.Y + 4, W: rc.W - 32, H: 14})
		title.Draw(p, theme)
		sub := toolkit.NewLabel(dvSubs[i])
		subInk := ink
		if !sel {
			subInk = toolkit.RGB(0x90, 0x90, 0x90)
		}
		sub.Ink = subInk
		sub.SetBounds(toolkit.Rect{X: rc.X + 28, Y: rc.Y + 18, W: rc.W - 32, H: 12})
		sub.Draw(p, theme)
	}
	const dataViewH = 4 * 32
	s.dataView.SetBounds(toolkit.Rect{X: colCX, Y: yC, W: colW, H: dataViewH})
	yC += dataViewH
	s.pushCard(colCX, cardStartC, colW, yC-cardStartC)

	// --- Column A, re-composed with the box-layout system ----------------
	//
	// Every Column-A widget constructed above is re-wrapped here into titled
	// Frames stacked in a VBox. colA.SetBounds cascades absolute bounds to
	// all of them (overriding the hand-computed rects), and draw() paints the
	// whole column via colA.Draw -- so the section Labels + pushCard borders
	// for Column A are no longer drawn (Frame supplies the title + border).
	railW6 := 3 * toolkit.ToolbarButtonH
	stepsV6 := 3*toolkit.StepBoxH + 2*toolkit.StepConnectorW
	wave4RowH := railW6
	if stepsV6 > wave4RowH {
		wave4RowH = stepsV6
	}
	accFrameH := 3*toolkit.ExpanderHeaderH + 56
	propGH := toolkit.TableHeaderHeight + 4*toolkit.TableRowHeight

	fActions, hActions := sectionFrame("Actions", sectGap,
		boxItem{hrowFlex(sectGap, s.button, s.toggle), 28},
		boxItem{s.check, 22},
		boxItem{s.radios[0], 20}, boxItem{s.radios[1], 20}, boxItem{s.radios[2], 20})
	fInputs, hInputs := sectionFrame("Inputs", sectGap,
		boxItem{s.entry, 26},
		boxItem{hrowFixedFlex(sectGap, 120, s.spin, s.scale), 26},
		boxItem{s.dropdown, 26})
	fFeedback, hFeedback := sectionFrame("Feedback", sectGap,
		boxItem{s.progress, 18}, boxItem{s.level, 18}, boxItem{s.spinner, 24})
	fNotebook, hNotebook := sectionFrame("Notebook", sectGap, boxItem{s.notebook, 80})
	fWave1, hWave1 := sectionFrame("Wave 1 (v0.7)", sectGap,
		boxItem{s.swtch, 22}, boxItem{s.alert, 32}, boxItem{s.card, 74}, boxItem{s.steps, 32})
	fWave4, hWave4 := sectionFrame("Wave 4 (v0.33)", sectGap,
		boxItem{hrowFixedFlex(sectGap, 24, s.toolbarV, s.stepsV), wave4RowH})
	fWave5, hWave5 := sectionFrame("Wave 5 (v0.42)", sectGap,
		boxItem{s.accordion, accFrameH},
		boxItem{s.colorPicker, toolkit.ColorPickerHeight},
		boxItem{s.segBar, 22})
	fWave6, hWave6 := sectionFrame("Wave 6 (v0.81)", sectGap,
		boxItem{s.propGrid, propGH}, boxItem{s.pagingBar, toolkit.PagingBtnH})

	var totalA int
	s.colA, totalA = column(
		[]*toolkit.Frame{fActions, fInputs, fFeedback, fNotebook, fWave1, fWave4, fWave5, fWave6},
		[]int{hActions, hInputs, hFeedback, hNotebook, hWave1, hWave4, hWave5, hWave6})
	colATop := toolkit.MenuBarH + toolkit.ToolbarButtonH + sectPad + themeRowH + sectPad
	s.colA.SetBounds(toolkit.Rect{X: colAX, Y: colATop, W: colW, H: totalA})

	// --- click routing table --------------------------------------------

	s.clickables = []toolkit.Widget{
		// theme switcher first (above the column grid)
		s.themeSwitcher,
		// row order matches column-A top-to-bottom, then B, then C
		s.button, s.toggle, s.check,
		s.radios[0], s.radios[1], s.radios[2],
		s.entry, s.spin, s.scale, s.dropdown,
		s.notebook,
		// Column A wave extension
		s.swtch,
		// Column B & C classic
		s.textView,
		s.calendar,
		s.colorChoose,
		s.listBox,
		s.tree,
		s.expander,
		s.paned,
		// Column B wave extension
		s.banner,
		// Column C wave extension
		s.chip, s.splitButton,
		// Column A Wave 4 extension
		s.toolbarV,
		// Column B Wave 4 extension
		s.notebookSide, s.dropdownUp,
		// Column A Wave 5 extension
		s.accordion, s.colorPicker,
		// Column B Wave 5 extension
		s.carousel, s.mdEditor, s.dateRange,
		// Column C Wave 5 extension
		s.wizard, s.treeTable, s.paletteBtn,
	}

	return s
}

// showNotify shows text on the shared Notification, then re-anchors it to
// the bottom-right corner of the surface (just above the Statusbar) via
// Notification.AnchorIn. Anchoring after Show (rather than once at
// construction) matters because AnchorIn sizes the box from the CURRENT
// Text — a fixed-size box computed up front would either clip a long
// message or leave a wide empty box around a short one.
func (s *state) showNotify(text string) {
	s.notify.Show(text)
	s.notify.AnchorIn(toolkit.Rect{X: 0, Y: 0, W: s.w, H: s.h - toolkit.StatusbarH}, toolkit.BottomRight)
}

// pushCard records the outer rectangle of a section — extended by
// cardPad on every side so the border sits comfortably around the
// widgets. draw() strokes these before painting widgets so the
// widget bodies land on top of the border.
func (s *state) pushCard(x, y, w, h int) {
	s.cards = append(s.cards, toolkit.Rect{
		X: x - cardPad, Y: y - cardPad,
		W: w + 2*cardPad, H: h + 2*cardPad,
	})
}

// draw paints the whole dashboard onto buf. Buf is an RGBA row-major
// slice — buf and s.w/s.h are wrapped in a PixelPainter so the widget
// code sees only the painter.Painter interface. Draw order matters:
// background first, card outlines behind the widgets, then row
// scaffolding, then widget cards, then overlays (menu popover,
// notification, CommandPalette) on top.
func (s *state) draw(buf []byte) {
	fillBG(buf, s.w, s.h, s.theme.Background)
	p := painter.NewPixelPainter(buf, s.w, s.h)

	// Card outlines — a subtle 1-px Border stroke around each widget
	// group (GTK4 widget-factory pattern). Painted before the widgets
	// so widget bodies overlap the border on top.
	for _, r := range s.cards {
		p.FillRect(painter.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, s.theme.Surface)
		p.StrokeRect(painter.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, s.theme.Border, 1)
	}

	// Top scaffold.
	s.menuBar.Draw(p, s.theme)
	s.toolbar.Draw(p, s.theme)
	s.themeSwitcher.Draw(p, s.theme)

	// Column A — every section, composed with the box-layout system: a
	// single VBox of titled Frames draws its own borders/titles + all the
	// widgets inside (see the colA build in newState). Replaces the
	// hand-drawn Column-A widgets, section Labels and card borders.
	s.colA.Draw(p, s.theme)

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

	// Column B — Wave 2 (v0.8) highlights.
	s.wave2Label.Draw(p, s.theme)
	s.headerBar.Draw(p, s.theme)
	s.toast.Draw(p, s.theme)
	s.banner.Draw(p, s.theme)
	s.diff.Draw(p, s.theme)

	// Column C — Wave 3 (v0.9) highlights.
	s.wave3Label.Draw(p, s.theme)
	s.stat.Draw(p, s.theme)
	s.progressCircle.Draw(p, s.theme)
	s.timeline.Draw(p, s.theme)
	s.chip.Draw(p, s.theme)
	s.splitButton.Draw(p, s.theme)

	// Column B — Wave 4 (v0.33) highlights: side-tab Notebook + upward DropDown.
	s.wave4LabelB.Draw(p, s.theme)
	s.notebookSide.Draw(p, s.theme)
	s.dropdownUp.Draw(p, s.theme)

	// Column C — Wave 4 (v0.33) highlights: right-aligned Table + horizontal Timeline.
	s.wave4LabelC.Draw(p, s.theme)
	s.table.Draw(p, s.theme)
	s.timelineH.Draw(p, s.theme)

	// Column B — Wave 5 (v0.42) highlights: Carousel + MarkdownEditor + DateRangePicker.
	s.wave5LabelB.Draw(p, s.theme)
	s.carousel.Draw(p, s.theme)
	s.mdEditor.Draw(p, s.theme)
	s.dateRange.Draw(p, s.theme)

	// Column C — Wave 5 (v0.42) highlights: Wizard + TreeTable + CommandPalette trigger.
	s.wave5LabelC.Draw(p, s.theme)
	s.wizard.Draw(p, s.theme)
	s.treeTable.Draw(p, s.theme)
	s.paletteBtn.Draw(p, s.theme)

	// Wave 6 (v0.81) — grid-editing family (Column A is in colA above).
	s.wave6LabelB.Draw(p, s.theme)
	s.gridEdit.Draw(p, s.theme)
	s.wave6LabelC.Draw(p, s.theme)
	s.dataView.Draw(p, s.theme)

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
	// CommandPalette floats above everything (even the notification),
	// matching the "Ctrl+Shift+P" pattern's z-order in a real host.
	s.cmdPalette.Draw(p, s.theme)
}

// handleClick dispatches a click at (x, y) to whichever widget it
// falls in. Overlays (an open CommandPalette, then an open menu
// popover) take precedence; the top scaffold (menu bar, toolbar)
// comes next; the dashboard clickables come last, in draw order.
func (s *state) handleClick(x, y int) bool {
	ev := toolkit.Event{Kind: toolkit.EventClick, X: x, Y: y}

	// CommandPalette overlay first: it floats above even the menu popover
	// (see draw's z-order), and its own OnEvent already handles an
	// outside-click as "dismiss" — so every click is its concern while open.
	if s.cmdPalette.Visible {
		s.cmdPalette.OnEvent(ev)
		return true
	}

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
