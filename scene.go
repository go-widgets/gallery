// Scene state — composes every widget family from go-widgets/toolkit
// into a single-view dashboard. Kept in a separate file (no js/wasm
// build tag) so a native go test can exercise draw() + handleClick()
// against a plain byte buffer.

package main

import (
	"strconv"

	"github.com/go-widgets/painter"
	"github.com/go-widgets/toolkit"
)

// itoa is strconv.Itoa under a short name, matching the toolkit's own helper
// so the interactivity callbacks read compactly.
func itoa(n int) string { return strconv.Itoa(n) }

// dropdownRowH is the pixel height of one row in a DropDown popover (matches
// the toolkit's PopoverBounds row step), used to map a click to an option.
const dropdownRowH = 18

// toolkitVersion is the go-widgets/toolkit release the gallery is built against,
// shown in the About menu + statusbar. One place so a dep bump updates both;
// keep in sync with go.mod.
const toolkitVersion = "v0.93.0"

// Canvas dimensions. Lives in scene.go (not main.go) so the native
// scene_test compiles without the js && wasm build tag — otherwise
// the constants drop out and the tests can't reference them.
//
// The dashboard is laid out on a 960×720 grid: MenuBar (top 24px),
// Toolbar (next 24px), a three-column body of widget cards, and a
// Statusbar on the bottom 20px. Every widget kind gets its own
// labelled slot rather than being hidden behind a Notebook tab.
const surfaceW = 960

// surfaceH is COMPUTED from the box-layout content, never hardcoded: newState
// lays out the three columns + the full-width band and reports the total
// height they need (see the s.h assignment in newState), so the canvas adapts
// automatically as widgets are added or removed — nothing is clipped.
var surfaceH = newState(surfaceW, 0).h

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
	textView    *toolkit.TextView
	calendar    *toolkit.Calendar
	colorChoose *toolkit.ColorChooser

	// Column C — Selection & Structure.
	listBox   *toolkit.ListBox
	tree      *toolkit.TreeView
	expander  *toolkit.Expander
	frameHost *toolkit.Frame

	// Container demos to fill the vertical whitespace + demonstrate
	// composition (a leaf-only widget dashboard would leave 30 % of
	// each column empty).
	notebook *toolkit.Notebook

	paned *toolkit.Paned

	// Column-A Wave 1 (v0.7) highlights.
	swtch *toolkit.Switch
	alert *toolkit.Alert
	card  *toolkit.Card
	steps *toolkit.Steps

	// Column-B Wave 2 (v0.8) highlights.
	toast     *toolkit.Toast
	banner    *toolkit.Banner
	headerBar *toolkit.HeaderBar
	diff      *toolkit.Diff

	// Column-C Wave 3 (v0.9) highlights.
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
	notebookSide *toolkit.Notebook
	dropdownUp   *toolkit.DropDown

	// Column-C Wave 4 (v0.33) highlights: a Table with a right-aligned
	// numeric column (TableColumn.Align) + a horizontal Timeline ribbon
	// (Timeline.Horizontal).
	table     *toolkit.Table
	timelineH *toolkit.Timeline

	// Column-A Wave 5 (v0.42) highlights: Accordion (collapsible
	// sections), ColorPicker (HSV square + hue/alpha sliders) and
	// SegmentedBar (proportional multi-colour meter).
	accordion   *toolkit.Accordion
	colorPicker *toolkit.ColorPicker
	segBar      *toolkit.SegmentedBar

	// Column-B Wave 5 (v0.42) highlights: Carousel (paged slide viewer),
	// MarkdownEditor (live source/preview split) and DateRangePicker
	// (two-endpoint month-grid selection).
	carousel  *toolkit.Carousel
	mdEditor  *toolkit.MarkdownEditor
	dateRange *toolkit.DateRangePicker

	// Column-C Wave 5 (v0.42) highlights: Wizard (multi-step flow) and
	// TreeTable (Table-shaped grid with nesting). A CommandPalette is
	// wired as a canvas-wide overlay, opened from a trigger Button here
	// and drawn last (like Notification) so it floats above everything.
	wizard     *toolkit.Wizard
	treeTable  *toolkit.TreeTable
	paletteBtn *toolkit.Button
	cmdPalette *toolkit.CommandPalette

	// Wave 6 (v0.81) highlights — the Table grid-editing family + the
	// widgets it enabled. Column A: a PropertyGrid (editable Name/Value)
	// above a PagingToolbar. Column B: a Table showing grouped rows + an
	// open inline cell editor. Column C: a ListBox with a DataView
	// ItemRenderer painting rich two-line rows.
	propGrid  *toolkit.PropertyGrid
	pagingBar *toolkit.PagingToolbar
	gridEdit  *toolkit.Table
	dataView  *toolkit.ListBox

	// Wave 7 (v0.82) — dashboard widgets. Column-fitting ones (Sparklines +
	// Area/Scatter/Radar charts) go into the three columns; the wide ones
	// (Agenda month view, Kanban board, Gantt chart) go into a full-width
	// band below the column grid (colWide).
	sparkLine      *toolkit.Sparkline
	sparkBar       *toolkit.Sparkline
	areaChart      *toolkit.AreaChart
	scatterChart   *toolkit.ScatterChart
	radarChart     *toolkit.RadarChart
	agenda         *toolkit.Agenda
	agendaSwitcher *toolkit.ViewSwitcher
	kanban         *toolkit.Kanban
	gantt          *toolkit.Gantt

	// addedEvents counts events the user adds by clicking empty Agenda days,
	// so each new event gets a distinct label.
	addedEvents int

	// Drag capture: the dashboard widget grabbed on press (a Kanban card or a
	// Gantt bar), with its surface bounds, so EventMouseDrag / EventMouseUp
	// route back to it in its own local coordinates even as the pointer leaves.
	dragTarget toolkit.Widget
	dragBounds toolkit.Rect

	// keyTarget is the widget that last received a click; keyboard events
	// (typing into an Entry, an open cell editor, …) route to it.
	keyTarget toolkit.Widget

	// tooltip shows the underlying value when the pointer hovers a chart.
	tooltip *toolkit.Tooltip

	// vm is the MVVM layer: the theme/Agenda-view switchers, the Agenda event
	// list and the palette Button bind to its Observables / ObservableList /
	// Command (see viewmodel.go).
	vm *viewModel

	// ctxMenu is the right-click edit menu (surface-sized overlay). Its Menu is
	// rebuilt per popup for the Kanban card / Gantt task / Agenda day under the
	// cursor (see gallerymenu.go).
	ctxMenu *toolkit.ContextMenu

	// App-shell demo: a Border with a COLLAPSIBLE sidebar (a Frame.Collapsible
	// in the West region) and a draggable splitter GRIP (Border.WestSplit),
	// its Center hosting a Paned whose handle is a second grip. Demonstrates
	// the collapsible-sidebar + splitter-grip layout system.
	appBorder *toolkit.Border
	sidebar   *toolkit.Frame
	appPaned  *toolkit.Paned

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
	// draw() renders the whole column with a single colA.Draw.
	colA *toolkit.VBox
	// colB is Column B composed with the box-layout system (same structure as
	// colA).
	colB *toolkit.VBox
	// colC is Column C composed with the box-layout system. All three columns
	// are now box-laid -- the gallery no longer hand-computes any body rects.
	colC *toolkit.VBox
	// colWide is a full-width VBox below the 3-column grid hosting the wide
	// Wave 7 dashboard widgets (Agenda month, Kanban, Gantt) that don't fit a
	// single narrow column.
	colWide *toolkit.VBox

	// Live list of interactive widgets for click dispatch. Enumerated
	// in draw-order (matches the visual order the user sees) so hit-
	// testing prefers the top-most match.
	clickables []toolkit.Widget
}

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

// newState builds the whole dashboard. The height is NOT taken from a caller
// or a magic constant: it is COMPUTED from the box-layout content once the
// three columns + the full-width band are laid out (see the s.h assignment
// near the end), so adding widgets grows the surface automatically. The second
// parameter is ignored (kept only so existing call sites compile); pass 0.
func newState(w, _ int) *state {
	// Switch the whole toolkit to anti-aliased, shaped go-opentype text (the
	// bundled Atkinson Hyperlegible face) instead of the built-in 5x7 bitmap
	// font, so the gallery renders crisp AA text and dogfoods go-opentype. Done
	// FIRST, before any layout below measures text widths, so widths + rendering
	// agree. The bundled face never fails to parse; on the impossible error the
	// toolkit keeps the bitmap default. Idempotent across newState calls.
	_ = toolkit.UseOpenTypeText()

	s := &state{w: w, theme: toolkit.DefaultLight()}

	// --- top scaffold -----------------------------------------------------

	// Notification hosts a floating toast at bottom-right, just above the
	// Statusbar. Positioned at the end of newState, once s.h is known.
	s.notify = toolkit.NewNotification("")

	// CommandPalette is a canvas-wide overlay (like the menu popover): its
	// Bounds span the whole surface (set at the end, once s.h is known) so it
	// can catch an outside click anywhere + centre its panel.
	s.cmdPalette = toolkit.NewCommandPalette([]toolkit.PaletteCommand{
		{Label: "New file", Action: func() { s.showNotify("Palette: New file") }},
		{Label: "Open recent", Action: func() { s.showNotify("Palette: Open recent") }},
		{Label: "Toggle theme", Action: func() { s.showNotify("Palette: Toggle theme") }},
		{Label: "About gallery", Action: func() { s.showNotify("Palette: About gallery") }},
	})

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
		{Label: "?", OnClick: func() { s.showNotify("go-widgets/toolkit @ " + toolkitVersion) }},
	})
	s.toolbar.SetBounds(toolkit.Rect{X: 0, Y: toolkit.MenuBarH, W: w, H: toolkit.ToolbarButtonH})

	s.status = toolkit.NewStatusbar([]string{"~98 widgets", "100 % cov", "click something", "go-widgets/toolkit " + toolkitVersion})

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
	// OnChange is installed by the MVVM layer (newViewModel binds themeSwitcher
	// to vm.themeIndex; a subscription applies the palette).
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
	s.dropdown.OnSelect = func(i int) { s.showNotify("Encoding: " + s.dropdown.Options[i]) }

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

	// --- Column B widgets — CONSTRUCTION ONLY (placement owned by colB) ---
	s.textView = toolkit.NewTextView("Multi-line editor.\nType to insert.\nEnter splits a line.\nArrow keys navigate.")

	s.calendar = toolkit.NewCalendar(2026, 7, 2)
	s.calendar.SetToday(2026, 7, 2)

	s.colorChoose = toolkit.NewColorChooser(toolkit.RGB(0x0d, 0x94, 0x88))

	// --- Column C widgets — CONSTRUCTION ONLY (placement owned by colC) ---
	s.listBox = toolkit.NewListBox([]string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"})

	s.tree = toolkit.NewTreeView(&toolkit.TreeNode{
		Label: "/", Expanded: true, Children: []*toolkit.TreeNode{
			{Label: "src", Expanded: true, Children: []*toolkit.TreeNode{
				{Label: "main.go"}, {Label: "scene.go"},
			}},
			{Label: "docs"},
			{Label: "README.md"},
		},
	})

	// Expander wraps a Frame that hosts a Label — layout composition
	// (Expander → Frame → leaf).
	frameLabel := toolkit.NewLabel("nested widget inside Frame")
	s.frameHost = toolkit.NewFrame(frameLabel)
	s.expander = toolkit.NewExpander("Details", s.frameHost)
	s.expander.Expanded = true

	s.paned = toolkit.NewHPaned(toolkit.NewLabel("left pane"), toolkit.NewLabel("right pane"))

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

	// --- Column B extension: Wave 2 (v0.8) — construction only ----------
	s.headerBar = toolkit.NewHeaderBar("Files")
	s.headerBar.Subtitle = "~/Documents"

	s.toast = toolkit.NewToast("Copied to clipboard", toolkit.ToastSuccess)
	s.toast.Visible = true

	s.banner = toolkit.NewBanner("Update available.")
	s.banner.ButtonLabel = "Install"
	s.banner.OnAction = func() { s.showNotify("Banner action clicked") }

	s.diff = toolkit.NewDiff([]toolkit.DiffLine{
		{Text: "package main", Kind: toolkit.DiffContext},
		{Text: "old line", Kind: toolkit.DiffRemoved},
		{Text: "new line", Kind: toolkit.DiffAdded},
	})

	// --- Column B extension: Wave 4 (v0.33) — construction only ---------
	//
	// A second Notebook with its tab strip on the left (TabSide = TabLeft)
	// above a DropDown that opens its popover upward (OpenUp).
	s.notebookSide = toolkit.NewNotebook()
	s.notebookSide.TabSide = toolkit.TabLeft
	s.notebookSide.AddTab("A", toolkit.NewLabel("Tabs on the left"))
	s.notebookSide.AddTab("B", toolkit.NewLabel("(TabSide = TabLeft)"))

	s.dropdownUp = toolkit.NewDropDown([]string{"Opens upward", "OpenUp = true"}, 0)
	s.dropdownUp.OpenUp = true
	s.dropdownUp.OnSelect = func(i int) { s.showNotify("Chose: " + s.dropdownUp.Options[i]) }

	// --- Column C extension: Wave 3 (v0.9) — construction only ----------
	//
	// Stat + ProgressCircle sit side by side, then a Timeline, then Chip +
	// SplitButton side by side; colC arranges the rows via HBox.
	s.stat = toolkit.NewStat("Requests / min", "12,845")
	s.stat.Change = "+8.3%"
	s.stat.Trend = toolkit.StatUp
	s.progressCircle = toolkit.NewProgressCircle()
	s.progressCircle.Fraction = 0.66

	s.timeline = toolkit.NewTimeline([]toolkit.TimelineEvent{
		{Title: "PR opened", Kind: toolkit.TimelineDefault},
		{Title: "Reviewed", Detail: "LGTM", Kind: toolkit.TimelineSuccess},
		{Title: "Build failed", Kind: toolkit.TimelineError},
	})

	s.chip = toolkit.NewChip("frontend")
	s.chip.Closable = true
	s.chip.OnClose = func() { s.showNotify("Chip closed") }
	s.splitButton = toolkit.NewSplitButton("Deploy",
		func() { s.showNotify("SplitButton: Deploy") })
	s.splitButton.OnArrow = func() { s.showNotify("SplitButton: arrow menu") }

	// --- Column C extension: Wave 4 (v0.33) — construction only ---------
	//
	// A Table with its numeric column right-aligned above a horizontal
	// Timeline ribbon (Timeline.Horizontal = true).
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

	s.timelineH = toolkit.NewTimeline([]toolkit.TimelineEvent{
		{Title: "Init", Kind: toolkit.TimelineDefault},
		{Title: "Build", Kind: toolkit.TimelineSuccess},
		{Title: "Deploy", Kind: toolkit.TimelineWarning},
	})
	s.timelineH.Horizontal = true

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

	// --- Column B extension: Wave 5 (v0.42) — construction only ---------
	s.carousel = toolkit.NewCarousel([]toolkit.Widget{
		toolkit.NewCard("Slide 1", "First featured panel.", ""),
		toolkit.NewCard("Slide 2", "Second featured panel.", ""),
		toolkit.NewCard("Slide 3", "Third featured panel.", ""),
	})
	s.carousel.Wrap = true

	s.mdEditor = toolkit.NewMarkdownEditor("# Notes\n\n- live *preview*\n- side by side")

	s.dateRange = toolkit.NewDateRangePicker(2026, 7)
	s.dateRange.Start = toolkit.Date{Y: 2026, M: 7, D: 10}
	s.dateRange.End = toolkit.Date{Y: 2026, M: 7, D: 17}

	// --- Column C extension: Wave 5 (v0.42) — construction only ---------
	//
	// Wizard (multi-step flow) above a TreeTable (nested grid) and a trigger
	// Button that opens the CommandPalette overlay.
	s.wizard = toolkit.NewWizard([]toolkit.WizardStep{
		{Title: "Plan", Body: toolkit.NewLabel("Sketch the release scope.")},
		{Title: "Build", Body: toolkit.NewLabel("Wire the new widgets in.")},
		{Title: "Ship", Body: toolkit.NewLabel("Tag + publish the gallery.")},
	})

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

	s.paletteBtn = toolkit.NewButton("Open command palette ⌘", func() { s.cmdPalette.Open() })

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

	// --- Column B extension: Wave 6 (v0.81) — construction only ---------
	//
	// A Table showing collapsible group rows (grouped by the first column)
	// with an inline cell editor open over an editable cell.
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
	// Open the Owner editor on the first data row (visual line 1).
	s.gridEdit.OnEvent(toolkit.Event{Kind: toolkit.EventClick, X: 130, Y: toolkit.TableHeaderHeight + toolkit.TableRowHeight + 2})
	s.gridEdit.OnEvent(toolkit.Event{Kind: toolkit.EventChar, Code: "!"})

	// --- Column C extension: Wave 6 (v0.81) — construction only ---------
	//
	// A ListBox with a DataView ItemRenderer: each row draws a colour
	// swatch + a two-line title/subtitle instead of the default text.
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

	// --- Wave 7 (v0.82) dashboard widgets — construction only ------------
	s.sparkLine = toolkit.NewSparkline([]float64{3, 5, 4, 8, 6, 9, 7, 11, 9, 13})
	s.sparkLine.ShowLast = true
	s.sparkBar = toolkit.NewSparkline([]float64{5, 8, 4, 9, 6, 11, 7, 10})
	s.sparkBar.Kind = toolkit.SparkBar
	s.sparkBar.ShowLast = true

	s.areaChart = toolkit.NewAreaChart([][]float64{{3, 6, 4, 8, 5, 9}, {1, 3, 2, 5, 3, 6}})
	s.scatterChart = toolkit.NewScatterChart([][]toolkit.ScatterPoint{
		{{X: 1, Y: 2}, {X: 3, Y: 5}, {X: 4, Y: 3}, {X: 6, Y: 7}, {X: 8, Y: 6}},
		{{X: 2, Y: 6}, {X: 5, Y: 2}, {X: 7, Y: 8}, {X: 9, Y: 4}},
	})
	s.radarChart = toolkit.NewRadarChart(
		[]string{"Speed", "Power", "Range", "Armor", "Agility", "Luck"},
		[][]float64{{8, 6, 7, 4, 9, 5}, {5, 9, 4, 8, 3, 7}})

	s.agenda = toolkit.NewAgenda([]toolkit.AgendaEvent{
		{Title: "Kickoff", Y: 2026, M: 7, D: 3, Fill: toolkit.RGB(0x35, 0x84, 0xe4)},
		{Title: "Review", Y: 2026, M: 7, D: 10, Fill: toolkit.RGB(0x1e, 0x9e, 0x52)},
		{Title: "Ship", Y: 2026, M: 7, D: 20, Fill: toolkit.RGB(0xe0, 0x50, 0x50)},
		{Title: "Retro", Y: 2026, M: 7, D: 24, Fill: toolkit.RGB(0xc0, 0x80, 0xe0)},
	})
	s.agenda.View = toolkit.AgendaMonth
	s.agenda.Year, s.agenda.Month = 2026, 7
	// Clicking an empty in-month day adds an event there, through the MVVM
	// event ObservableList (which mirrors back into the widget).
	s.agenda.OnDayActivate = func(y, m, d int) {
		s.addedEvents++
		s.vm.addEvent("New "+itoa(s.addedEvents), y, m, d, toolkit.RGB(0x7a, 0x5a, 0xf0))
		s.showNotify("Added event on " + itoa(m) + "/" + itoa(d))
	}
	// A four-segment switcher drives the Agenda's view (Week/Month/Quarter/Year),
	// whose enum matches these indices 0..3. Its OnChange is installed by the
	// MVVM layer (bound to vm.agendaView).
	s.agendaSwitcher = toolkit.NewViewSwitcher([]string{"Week", "Month", "Quarter", "Year"}, int(toolkit.AgendaMonth))

	s.kanban = toolkit.NewKanban([]toolkit.KanbanColumn{
		{Title: "To Do", Cards: []toolkit.KanbanCard{
			{Title: "Design", Subtitle: "spec", Accent: toolkit.RGB(0xe0, 0x50, 0x50)},
			{Title: "Research", Subtitle: "links"}}},
		{Title: "Doing", Cards: []toolkit.KanbanCard{
			{Title: "Build", Subtitle: "impl", Accent: toolkit.RGB(0x35, 0x84, 0xe4)}}},
		{Title: "Done", Cards: []toolkit.KanbanCard{
			{Title: "Ship", Subtitle: "v1", Accent: toolkit.RGB(0x1e, 0x9e, 0x52)}}},
	})
	s.kanban.SelectedCol, s.kanban.SelectedCard = 1, 0 // highlight the "Doing" card

	s.gantt = toolkit.NewGantt([]toolkit.GanttTask{
		{Label: "Design", Start: 0, End: 3, Progress: 1.0},
		{Label: "Build", Start: 2, End: 8, Progress: 0.6, Fill: toolkit.RGB(0xe5, 0xa5, 0x0a)},
		{Label: "Test", Start: 7, End: 10, Progress: 0.2, Fill: toolkit.RGB(0x50, 0xa0, 0xe0)},
		{Label: "Ship", Start: 10, End: 12, Fill: toolkit.RGB(0x1e, 0x9e, 0x52)},
	})
	s.gantt.Selected = 1 // highlight the "Build" task
	// Drag a card between columns (v0.83 interactivity).
	s.kanban.OnCardMove = func(fromCol, fromCard, toCol, toIdx int) {
		s.showNotify("Moved card to " + s.kanban.Columns[toCol].Title)
	}
	// Drag a bar to move it or its edges to resize (v0.83 interactivity).
	s.gantt.OnTaskChange = func(i, start, end int) {
		s.showNotify(s.gantt.Tasks[i].Label + ": " + itoa(start) + "→" + itoa(end))
	}

	// MVVM layer: bind the theme + Agenda-view switchers to Observables, the
	// palette Button to a Command, and the Agenda events to an ObservableList.
	// Constructed here, once every bound widget exists.
	s.vm = newViewModel(s)

	// Right-click edit menu (its Menu is rebuilt per popup; bounds set below
	// once the surface height is known).
	s.ctxMenu = toolkit.NewContextMenu(toolkit.NewMenu(nil))

	// Hover tooltip for chart values.
	s.tooltip = toolkit.NewTooltip("")
	s.tooltip.Placement = toolkit.PlaceAbove

	// App-shell: collapsible sidebar (Frame.Collapsible in a Border West
	// region) + splitter grips (Border.WestSplit + the Paned handle).
	nav := toolkit.NewVBox()
	nav.Spacing = sectGap
	nav.AddFixed(toolkit.NewLabel("  Dashboard"), 20)
	nav.AddFixed(toolkit.NewLabel("  Reports"), 20)
	nav.AddFixed(toolkit.NewLabel("  Settings"), 20)
	s.sidebar = toolkit.NewFrame(nav)
	s.sidebar.Title = "Sidebar"
	s.sidebar.Collapsible = true // click the title bar to collapse the sidebar
	s.appPaned = toolkit.NewHPaned(
		toolkit.NewLabel("  Main content — drag the grip →"),
		toolkit.NewLabel("  Detail"))
	s.appBorder = toolkit.NewBorder()
	s.appBorder.West = s.sidebar
	s.appBorder.WestSize = 150
	s.appBorder.WestSplit = true // draggable grip between sidebar and centre
	s.appBorder.Center = s.appPaned

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
	fWave7, hWave7 := sectionFrame("Wave 7 (v0.82)", sectGap,
		boxItem{s.sparkLine, 30}, boxItem{s.sparkBar, 30})

	var totalA int
	s.colA, totalA = column(
		[]*toolkit.Frame{fActions, fInputs, fFeedback, fNotebook, fWave1, fWave4, fWave5, fWave6, fWave7},
		[]int{hActions, hInputs, hFeedback, hNotebook, hWave1, hWave4, hWave5, hWave6, hWave7})
	colATop := toolkit.MenuBarH + toolkit.ToolbarButtonH + sectPad + themeRowH + sectPad
	s.colA.SetBounds(toolkit.Rect{X: colAX, Y: colATop, W: colW, H: totalA})

	// --- Column B, re-composed with the box-layout system ----------------
	carouselH6 := 74 + 16
	mdEditorH6 := 90
	gridEditH6 := toolkit.TableHeaderHeight + 7*toolkit.TableRowHeight

	fText, hText := sectionFrame("TextView", sectGap, boxItem{s.textView, 110})
	fCal, hCal := sectionFrame("Calendar", sectGap, boxItem{s.calendar, 180})
	fColor, hColor := sectionFrame("ColorChooser", sectGap, boxItem{s.colorChoose, 130})
	// Size the Diff to its content so Wave 2 adapts (its lines used to clip
	// against a fixed 54px box).
	diffH := len(s.diff.Lines)*toolkit.DiffLineH() + 2*toolkit.DiffPadY
	fWave2, hWave2 := sectionFrame("Wave 2 (v0.8)", sectGap,
		boxItem{s.headerBar, 36}, boxItem{s.toast, 24}, boxItem{s.banner, 24}, boxItem{s.diff, diffH})
	fWave4B, hWave4B := sectionFrame("Wave 4 (v0.33)", sectGap,
		boxItem{s.notebookSide, 70}, boxItem{s.dropdownUp, 26})
	fWave5B, hWave5B := sectionFrame("Wave 5 (v0.42)", sectGap,
		boxItem{s.carousel, carouselH6}, boxItem{s.mdEditor, mdEditorH6}, boxItem{s.dateRange, 168})
	fWave6B, hWave6B := sectionFrame("Wave 6 (v0.81)", sectGap, boxItem{s.gridEdit, gridEditH6})
	fWave7B, hWave7B := sectionFrame("Wave 7 (v0.82)", sectGap,
		boxItem{s.areaChart, 120}, boxItem{s.scatterChart, 120})

	var totalB int
	s.colB, totalB = column(
		[]*toolkit.Frame{fText, fCal, fColor, fWave2, fWave4B, fWave5B, fWave6B, fWave7B},
		[]int{hText, hCal, hColor, hWave2, hWave4B, hWave5B, hWave6B, hWave7B})
	s.colB.SetBounds(toolkit.Rect{X: colBX, Y: colATop, W: colW, H: totalB})

	// --- Column C, re-composed with the box-layout system ----------------
	tableH6 := toolkit.TableHeaderHeight + 3*toolkit.TableRowHeight
	wizardH6 := toolkit.WizardStripH + 40 + toolkit.WizardButtonRowH
	treeTableH6 := toolkit.TreeTableHeaderHeight + 4*toolkit.TreeTableRowHeight
	dataViewH6 := 4 * 32

	fList, hList := sectionFrame("ListBox", sectGap, boxItem{s.listBox, 130})
	fTree, hTree := sectionFrame("TreeView", sectGap, boxItem{s.tree, 190})
	fExp, hExp := sectionFrame("Expander + Frame", sectGap, boxItem{s.expander, 88})
	fPaned, hPaned := sectionFrame("Paned (horizontal split)", sectGap, boxItem{s.paned, 60})
	fWave3, hWave3 := sectionFrame("Wave 3 (v0.9)", sectGap,
		boxItem{hrowFlex(4, s.stat, s.progressCircle), 60},
		boxItem{s.timeline, 68},
		boxItem{hrowFixedFlex(8, 96, s.chip, s.splitButton), 22})
	fWave4C, hWave4C := sectionFrame("Wave 4 (v0.33)", sectGap,
		boxItem{s.table, tableH6}, boxItem{s.timelineH, 36})
	fWave5C, hWave5C := sectionFrame("Wave 5 (v0.42)", sectGap,
		boxItem{s.wizard, wizardH6}, boxItem{s.treeTable, treeTableH6}, boxItem{s.paletteBtn, 28})
	fWave6C, hWave6C := sectionFrame("Wave 6 (v0.81)", sectGap, boxItem{s.dataView, dataViewH6})
	fWave7C, hWave7C := sectionFrame("Wave 7 (v0.82)", sectGap, boxItem{s.radarChart, 200})

	var totalC int
	s.colC, totalC = column(
		[]*toolkit.Frame{fList, fTree, fExp, fPaned, fWave3, fWave4C, fWave5C, fWave6C, fWave7C},
		[]int{hList, hTree, hExp, hPaned, hWave3, hWave4C, hWave5C, hWave6C, hWave7C})
	s.colC.SetBounds(toolkit.Rect{X: colCX, Y: colATop, W: colW, H: totalC})

	// --- Full-width band below the columns: wide Wave 7 dashboard widgets -
	// Agenda (month), Kanban and Gantt need more width than a single narrow
	// column, so they stack in a full-bleed VBox under the 3-column grid.
	fShell, hShell := sectionFrame("App shell — collapsible sidebar + splitter grips (v0.63)", sectGap,
		boxItem{s.appBorder, 130})
	fAgenda, hAgenda := sectionFrame("Agenda (v0.84) — switch views above; click or right-click an empty day to add", sectGap,
		boxItem{s.agendaSwitcher, 28}, boxItem{s.agenda, 272})
	fKanban, hKanban := sectionFrame("Kanban board (v0.84) — drag a card, or right-click to edit", sectGap, boxItem{s.kanban, 190})
	fGantt, hGantt := sectionFrame("Gantt chart (v0.84) — drag a bar/edges, or right-click to edit", sectGap, boxItem{s.gantt, toolkit.GanttHeaderH + 4*toolkit.GanttRowH})
	var totalWide int
	s.colWide, totalWide = column(
		[]*toolkit.Frame{fShell, fAgenda, fKanban, fGantt},
		[]int{hShell, hAgenda, hKanban, hGantt})
	colsBottom := max(totalA, totalB, totalC)
	wideTop := colATop + colsBottom + sectPad
	s.colWide.SetBounds(toolkit.Rect{X: margin, Y: wideTop, W: surfaceW - 2*margin, H: totalWide})

	// --- Height is COMPUTED from the box-layout content -------------------
	// The surface sizes to whatever the columns + full-width band actually
	// need, plus the bottom Statusbar; nothing is clipped, and adding a widget
	// grows the canvas automatically (surfaceH is derived from this via a
	// package var). Position the deferred, height-dependent chrome now.
	s.h = wideTop + totalWide + sectPad + toolkit.StatusbarH
	s.status.SetBounds(toolkit.Rect{X: 0, Y: s.h - toolkit.StatusbarH, W: w, H: toolkit.StatusbarH})
	s.cmdPalette.SetBounds(toolkit.Rect{X: 0, Y: 0, W: w, H: s.h})
	s.ctxMenu.SetBounds(toolkit.Rect{X: 0, Y: 0, W: w, H: s.h})
	s.notify.SetBounds(toolkit.Rect{X: w - 268, Y: s.h - toolkit.StatusbarH - 32, W: 260, H: 24})

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
		// Wave 6 (v0.81) editable/selectable data widgets — a left click must
		// reach them to open the inline cell editor (gridEdit / propGrid) or
		// move the selection (table / dataView) and focus them for keyboard.
		s.propGrid, s.pagingBar, s.gridEdit, s.dataView, s.table,
		// Full-width band — the app-shell Border routes clicks to its
		// collapsible sidebar (title toggles collapse) + the Paned grip.
		s.appBorder,
		// Interactive Wave 7 widgets (v0.83): the Agenda view switcher + the
		// Agenda (day-activate), and the drag-editable Kanban and Gantt.
		s.agendaSwitcher, s.agenda, s.kanban, s.gantt,
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

// draw paints the whole dashboard onto buf. Buf is an RGBA row-major
// slice — buf and s.w/s.h are wrapped in a PixelPainter so the widget
// code sees only the painter.Painter interface. Draw order matters:
// background first, card outlines behind the widgets, then row
// scaffolding, then widget cards, then overlays (menu popover,
// notification, CommandPalette) on top.
func (s *state) draw(buf []byte) {
	fillBG(buf, s.w, s.h, s.theme.Background)
	p := painter.NewPixelPainter(buf, s.w, s.h)

	// Top scaffold.
	s.menuBar.Draw(p, s.theme)
	s.toolbar.Draw(p, s.theme)
	s.themeSwitcher.Draw(p, s.theme)

	// Column A — every section, composed with the box-layout system: a
	// single VBox of titled Frames draws its own borders/titles + all the
	// widgets inside (see the colA build in newState). Replaces the
	// hand-drawn Column-A widgets, section Labels and card borders.
	s.colA.Draw(p, s.theme)

	// Column B — every section, composed with the box-layout system (see the
	// colB build in newState). Replaces the hand-drawn Column-B widgets,
	// section Labels and card borders.
	s.colB.Draw(p, s.theme)

	// Column C — every section, composed with the box-layout system (see the
	// colC build in newState). All three columns now render via one Draw each.
	s.colC.Draw(p, s.theme)

	// Full-width band under the columns: the wide Wave 7 dashboard widgets.
	s.colWide.Draw(p, s.theme)

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
	// Open DropDown popovers render above the widgets (the host owns the
	// popover surface): a ListBox of the options at PopoverBounds.
	for _, d := range []*toolkit.DropDown{s.dropdown, s.dropdownUp} {
		if d.Open {
			lb := toolkit.NewListBox(d.Options)
			lb.Selected = d.Selected
			lb.SetBounds(d.PopoverBounds())
			lb.Draw(p, s.theme)
		}
	}
	// Chart-hover value tooltip (above the widgets, below the menus).
	s.tooltip.Draw(p, s.theme)
	// CommandPalette floats above everything (even the notification),
	// matching the "Ctrl+Shift+P" pattern's z-order in a real host.
	s.cmdPalette.Draw(p, s.theme)
	// The right-click edit menu is the very topmost overlay.
	s.ctxMenu.Draw(p, s.theme)
}

// handleClick dispatches a click at (x, y) to whichever widget it
// falls in. Overlays (an open CommandPalette, then an open menu
// popover) take precedence; the top scaffold (menu bar, toolbar)
// comes next; the dashboard clickables come last, in draw order.
func (s *state) handleClick(x, y int) bool {
	ev := toolkit.Event{Kind: toolkit.EventClick, X: x, Y: y}

	// Right-click edit menu is the topmost overlay: while open, every click is
	// its concern (activate a row inside it, or dismiss on an outside click).
	if s.ctxMenu.Open {
		s.ctxMenu.OnEvent(ev)
		return true
	}

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

	// Open DropDown popover: the host owns the popover surface, so route a click
	// inside it to Select the option row, or dismiss it on a click outside.
	for _, d := range []*toolkit.DropDown{s.dropdown, s.dropdownUp} {
		if d.Open {
			pb := d.PopoverBounds()
			if inside(x, y, pb) {
				d.Select((y - pb.Y) / dropdownRowH)
			} else {
				d.Open = false
			}
			return true
		}
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

	// Dashboard clickables — first hit wins (draw-order = z-order). The hit
	// widget also becomes the drag-capture target so a following
	// EventMouseDrag / EventMouseUp routes back to it (Kanban card / Gantt bar).
	for _, w := range s.clickables {
		r := w.Bounds()
		if inside(x, y, r) {
			s.dragTarget, s.dragBounds = w, r
			s.keyTarget = w // clicking a widget focuses it for keyboard input
			w.OnEvent(local(ev, r))
			return true
		}
	}
	s.dragTarget, s.keyTarget = nil, nil
	return true
}

// handleMove drives pointer motion: a captured drag first, else a hover check
// (chart value tooltip). Reports whether the scene changed.
func (s *state) handleMove(x, y int) bool {
	if s.handleDrag(x, y) {
		return true
	}
	return s.handleHover(x, y)
}

// handleHover shows/updates the chart-value tooltip (and the Line/Area hover
// crosshair) under the pointer, or hides them when the pointer leaves every
// chart. Reports whether anything changed.
func (s *state) handleHover(x, y int) bool {
	s.clearChartHovers() // drop any previous crosshair before re-testing
	txt, ax, ay := s.chartHoverText(x, y)
	if txt == "" {
		if s.tooltip.Visible {
			s.tooltip.Hide()
			return true
		}
		return false
	}
	s.tooltip.Text = txt
	s.tooltip.Show(toolkit.Rect{X: ax, Y: ay})
	return true
}

// notebookLine returns the active Notebook page when it is a LineChart, else nil.
// notebookPage returns the active Notebook page widget, or nil.
func (s *state) notebookPage() toolkit.Widget {
	if s.notebook.Active >= 0 && s.notebook.Active < len(s.notebook.Tabs) {
		return s.notebook.Tabs[s.notebook.Active].Page
	}
	return nil
}

// clearChartHovers turns off the hover highlight on every hoverable chart.
func (s *state) clearChartHovers() {
	switch c := s.notebookPage().(type) {
	case *toolkit.LineChart:
		c.Hover = false
	case *toolkit.BarChart:
		c.Hover = false
	case *toolkit.PieChart:
		c.Hover = false
	}
	s.areaChart.Hover = false
	s.scatterChart.Hover = false
	s.radarChart.Hover = false
	s.sparkLine.Hover = false
	s.sparkBar.Hover = false
}

// chVal is the standard "#index = value" tooltip label.
func chVal(i int, v float64) string { return "#" + itoa(i) + " = " + ftoa(v) }

// chartHoverText returns the tooltip text + anchor for the chart under (x, y),
// arming that chart's hover highlight (crosshair / bar / slice / point / spoke).
// Empty text means no chart is under the cursor.
func (s *state) chartHoverText(x, y int) (text string, ax, ay int) {
	// Active Notebook page (Line / Bar / Pie).
	switch c := s.notebookPage().(type) {
	case *toolkit.LineChart:
		if r := c.Bounds(); inside(x, y, r) {
			if i, v, ok := c.ValueAt(x - r.X); ok {
				c.Hover, c.HoverIndex = true, i
				return chVal(i, v), x, y
			}
		}
	case *toolkit.BarChart:
		if r := c.Bounds(); inside(x, y, r) {
			if i, v, ok := c.ValueAt(x - r.X); ok {
				c.Hover, c.HoverIndex = true, i
				return chVal(i, v), x, y
			}
		}
	case *toolkit.PieChart:
		if r := c.Bounds(); inside(x, y, r) {
			if i, v, ok := c.SliceAt(x-r.X, y-r.Y); ok {
				c.Hover, c.HoverIndex = true, i
				return chVal(i, v), x, y
			}
		}
	}
	// Wave-7 AreaChart (value + crosshair).
	if r := s.areaChart.Bounds(); inside(x, y, r) {
		if i, v, ok := s.areaChart.ValueAt(x - r.X); ok {
			s.areaChart.Hover, s.areaChart.HoverIndex = true, i
			return chVal(i, v), x, y
		}
	}
	// Wave-7 ScatterChart (nearest point → its X,Y + ring).
	if r := s.scatterChart.Bounds(); inside(x, y, r) {
		if si, pi, pt, ok := s.scatterChart.NearestPoint(x-r.X, y-r.Y); ok {
			s.scatterChart.Hover, s.scatterChart.HoverSeries, s.scatterChart.HoverPoint = true, si, pi
			return "(" + ftoa(pt.X) + ", " + ftoa(pt.Y) + ")", x, y
		}
	}
	// Wave-7 RadarChart (nearest axis → its label + first-series value + spoke).
	if r := s.radarChart.Bounds(); inside(x, y, r) {
		if k, ok := s.radarChart.AxisAt(x-r.X, y-r.Y); ok && len(s.radarChart.Series) > 0 && k < len(s.radarChart.Series[0]) {
			s.radarChart.Hover, s.radarChart.HoverAxis = true, k
			return s.radarChart.Axes[k] + " = " + ftoa(s.radarChart.Series[0][k]), x, y
		}
	}
	// Sparklines (value + crosshair/bar highlight).
	for _, sp := range []*toolkit.Sparkline{s.sparkLine, s.sparkBar} {
		if r := sp.Bounds(); inside(x, y, r) {
			if i, v, ok := sp.ValueAt(x - r.X); ok {
				sp.Hover, sp.HoverIndex = true, i
				return chVal(i, v), x, y
			}
		}
	}
	return "", 0, 0
}

// ftoa formats a chart value compactly (no trailing zeros).
func ftoa(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

// handleChar routes a printable character to the focused widget as an
// EventChar (text input into an Entry, an open Table/PropertyGrid cell editor,
// …). Reports whether a target consumed it.
func (s *state) handleChar(ch string) bool {
	if s.keyTarget == nil {
		return false
	}
	s.keyTarget.OnEvent(toolkit.Event{Kind: toolkit.EventChar, Code: ch})
	return true
}

// handleKeyDown routes a named key (Enter, Backspace, Arrow*, Home, End, Esc,
// …) to the focused widget as an EventKeyDown. Reports whether a target
// consumed it.
func (s *state) handleKeyDown(code string) bool {
	if s.keyTarget == nil {
		return false
	}
	s.keyTarget.OnEvent(toolkit.Event{Kind: toolkit.EventKeyDown, Code: code})
	return true
}

// handleDrag routes a pointer move (with the button held) to the widget grabbed
// on press, as an EventMouseDrag in that widget's local coordinates. It reports
// whether a captured widget consumed it (so the host re-renders).
func (s *state) handleDrag(x, y int) bool {
	if s.dragTarget == nil {
		return false
	}
	s.dragTarget.OnEvent(toolkit.Event{Kind: toolkit.EventMouseDrag, X: x - s.dragBounds.X, Y: y - s.dragBounds.Y})
	return true
}

// handleRelease routes a button release to the captured widget as an
// EventMouseUp (its local coordinates), then drops the capture. It reports
// whether a captured widget consumed it.
func (s *state) handleRelease(x, y int) bool {
	if s.dragTarget == nil {
		return false
	}
	s.dragTarget.OnEvent(toolkit.Event{Kind: toolkit.EventMouseUp, X: x - s.dragBounds.X, Y: y - s.dragBounds.Y})
	s.dragTarget = nil
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
