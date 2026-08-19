// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Command iso is a browser wasm demonstrator of the go-widgets/toolkit
// isometric diagram widget ([toolkit.IsoDiagram]) and its collaborative CRDT
// backing store ([toolkit.IsoCRDTDocument]). It is an INTERACTIVE editor you
// drive in the page — not a static picture:
//
//   - a docked [toolkit.IsoIconPalette]: drag an icon onto a canvas and a node
//     is created at the exact ground tile under the drop (undoable); picking an
//     icon also arms click-to-place through an mvvm.Observable binding;
//   - a mode toolbar: Select / place Node / Connect / Zone / Text, plus
//     Undo / Redo / Delete, a layer show-hide toggle, Zoom +/- and a Sync
//     button. Left-drag empty ground pans; right-click opens the widget's own
//     context menu;
//   - TWO canvases side by side ("Site A" / "Site B"), each backed by its own
//     [toolkit.IsoCRDTDocument] on a distinct CRDT site. Every edit auto-syncs
//     the two replicas (operations exchanged both ways via OpsSince/Apply), so
//     an edit in one panel appears in the other, live, with no server — the
//     collaborative showpiece.
//
// All the scene logic lives here in a tag-less file, fully exercised by the
// native scene_test.go (it renders to a buffer and asserts exact projected
// pixel positions, drives a palette drop and a CRDT convergence). The only
// `js && wasm` code is main.go, a one-line hand-off to the shared
// internal/webcanvas harness, which drops out of the native build.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/go-crdt/crdt"
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/painter"
	"github.com/go-widgets/toolkit"
)

// Surface dimensions. Fixed (unlike the gallery, whose height is content-driven)
// because the editor is a bounded two-canvas workspace, not a scrolling
// dashboard.
const (
	isoSurfaceW = 1120
	isoSurfaceH = 700
)

// Layout geometry (all in surface pixels). Two equal canvas columns sit right of
// the docked palette, under the mode toolbar; a status line runs along the
// bottom.
const (
	isoToolbarH = 44
	isoLabelH   = 20
	isoStatusH  = 18
	isoGap      = 8

	isoPanelTop = isoToolbarH + 4              // 48
	isoDiaTop   = isoPanelTop + isoLabelH + 2  // 70
	isoStatusY  = isoSurfaceH - isoStatusH - 4 // 678

	isoPaletteW = 176
	isoPanelW   = 456

	isoPaletteX = isoGap                             // 8
	isoPanelAX  = isoPaletteX + isoPaletteW + isoGap // 192
	isoPanelBX  = isoPanelAX + isoPanelW + isoGap    // 656

	// panelDrawH is the drawable height of one canvas (and the height a
	// standalone panel capture renders at).
	panelDrawH = isoStatusY - isoDiaTop - 6 // 602
	panelW     = isoPanelW
)

// CRDT sites for the two replicas.
const (
	siteA crdt.SiteID = 1
	siteB crdt.SiteID = 2
)

// Seed-scene ids and layers, exported-in-spirit (package-private, but named) so
// the test can assert on the exact entities the editor starts with.
const (
	layerInfra   = "infra"
	layerMonitor = "monitor"

	nodeWeb   = "web"
	nodeDB    = "db"
	nodeCache = "cache"
)

// Concurrent-edit targets used by convergedScene and the convergence test:
// replica A moves nodeWeb while replica B recolours it (a per-field LWW merge)
// and adds a node.
var (
	collabMovedX, collabMovedY = 6, 6
	collabRecolor              = toolkit.RGB(255, 90, 180)
)

// Seed palette / node colours.
var (
	colWeb   = toolkit.RGB(60, 120, 220)
	colDB    = toolkit.RGB(40, 170, 90)
	colCache = toolkit.RGB(210, 60, 60)
	colZone  = toolkit.RGBA{R: 60, G: 120, B: 220, A: 70}
)

// seedIso populates doc with the shared starting diagram every replica begins
// from: two named layers, a grouping zone, three nodes across the two layers, a
// connector and a floating annotation.
func seedIso(doc toolkit.IsoDocument) {
	doc.PutLayer(toolkit.IsoLayer{ID: layerInfra, Name: "Infrastructure", Visible: true, Order: 0})
	doc.PutLayer(toolkit.IsoLayer{ID: layerMonitor, Name: "Monitoring", Visible: true, Order: 1})
	doc.PutZone(toolkit.IsoZone{ID: "z-core", X: 1, Y: 1, W: 5, H: 3, Color: colZone, Label: "Core", Layer: layerInfra})
	doc.PutNode(toolkit.IsoNode{ID: nodeWeb, X: 2, Y: 2, Icon: "server", Color: colWeb, Label: "web", Layer: layerInfra})
	doc.PutNode(toolkit.IsoNode{ID: nodeDB, X: 5, Y: 2, Icon: "database", Color: colDB, Label: "db", Layer: layerInfra})
	doc.PutNode(toolkit.IsoNode{ID: nodeCache, X: 7, Y: 5, Icon: "storage", Color: colCache, Label: "cache", Layer: layerMonitor})
	doc.PutConnector(toolkit.IsoConnector{ID: "c-sql", From: nodeWeb, To: nodeDB, Arrow: toolkit.IsoArrowSingle, Label: "sql", Layer: layerInfra})
	doc.PutText(toolkit.IsoText{ID: "t-region", X: 1, Y: 8, Text: "region: eu-west", Size: 1, Layer: layerMonitor})
}

// isoTarget identifies which region captured the current pointer press, so a
// drag/release routes back to the same widget in its own local coordinates.
type isoTarget int

const (
	tgtNone isoTarget = iota
	tgtToolbar
	tgtPalette
	tgtDiaA
	tgtDiaB
)

// isoScene is the whole demonstrator: the mode toolbar, the docked palette, the
// two CRDT-backed diagram canvases and the status line, plus the pointer-capture
// and sync bookkeeping that ties them together. It implements webcanvas.App
// (Size/Draw/Click/Move/Release/Context/Char/KeyDown).
type isoScene struct {
	theme *toolkit.Theme

	// Shared cross-boundary state (MVVM): the active edit mode drives both
	// diagrams; the palette's selected-icon observable is bound into both
	// diagrams' placement observables so picking an icon arms click-to-place.
	mode *mvvm.Observable[toolkit.IsoMode]

	toolbar     *toolkit.HBox
	buttons     []*toolkit.Button
	palette     *toolkit.IsoIconPalette
	labelA      *toolkit.Label
	labelB      *toolkit.Label
	statusLabel *toolkit.Label

	docA, docB *toolkit.IsoCRDTDocument
	diaA, diaB *toolkit.IsoDiagram
	active     *toolkit.IsoDiagram

	// syncedA/syncedB are each replica's version as of the last exchange; sync
	// sends only the operations produced since.
	syncedA, syncedB crdt.CompositeVersion

	capture isoTarget
	unbind  []func()
}

// newIsoScene builds the demonstrator with both replicas seeded to the same
// starting diagram.
func newIsoScene() *isoScene {
	s := &isoScene{
		theme: toolkit.DefaultLight(),
		mode:  mvvm.NewObservable(toolkit.IsoModeSelect),
	}

	s.palette = toolkit.NewIsoIconPalette(toolkit.IsoDefaultIcons())
	s.palette.SetBounds(toolkit.Rect{X: isoPaletteX, Y: isoPanelTop, W: isoPaletteW, H: isoStatusY - isoPanelTop - 6})

	s.labelA = toolkit.NewLabel("Site A — CRDT replica 1")
	s.labelA.SetBounds(toolkit.Rect{X: isoPanelAX, Y: isoPanelTop, W: isoPanelW, H: isoLabelH})
	s.labelB = toolkit.NewLabel("Site B — CRDT replica 2")
	s.labelB.SetBounds(toolkit.Rect{X: isoPanelBX, Y: isoPanelTop, W: isoPanelW, H: isoLabelH})

	s.statusLabel = toolkit.NewLabel("")
	s.statusLabel.SetBounds(toolkit.Rect{X: isoPaletteX, Y: isoStatusY, W: isoSurfaceW - 2*isoGap, H: isoStatusH})

	s.buildToolbar()
	s.seedDocs()
	return s
}

// buildToolbar assembles the mode / edit / view / collaborate button strip once;
// its handlers read the live scene, so they keep working across a reset that
// rebuilds the documents.
func (s *isoScene) buildToolbar() {
	type item struct {
		label string
		fn    func()
	}
	items := []item{
		{"Select", func() { s.mode.Set(toolkit.IsoModeSelect); s.palette.SelectIcon(""); s.refreshStatus() }},
		{"Node", func() { s.mode.Set(toolkit.IsoModeSelect); s.palette.SelectIcon("server"); s.refreshStatus() }},
		{"Connect", func() { s.mode.Set(toolkit.IsoModeConnect); s.refreshStatus() }},
		{"Zone", func() { s.mode.Set(toolkit.IsoModeZone); s.refreshStatus() }},
		{"Text", func() { s.mode.Set(toolkit.IsoModeText); s.refreshStatus() }},
		{"Undo", func() { s.active.Undo(); s.afterEdit() }},
		{"Redo", func() { s.active.Redo(); s.afterEdit() }},
		{"Delete", func() { s.active.DeleteSelection(); s.afterEdit() }},
		{"Layer", func() { s.toggleLayer() }},
		{"Zoom+", func() { s.zoomActive(toolkit.IsoZoomStep) }},
		{"Zoom-", func() { s.zoomActive(1 / toolkit.IsoZoomStep) }},
		{"Sync", func() { s.sync(); s.refreshStatus() }},
		{"Reset", func() { s.reset() }},
	}
	s.toolbar = toolkit.NewHBox()
	for _, it := range items {
		b := toolkit.NewButton(it.label, it.fn)
		s.buttons = append(s.buttons, b)
		s.toolbar.AddFixed(b, 80)
	}
	s.toolbar.SetBounds(toolkit.Rect{X: 6, Y: 6, W: isoSurfaceW - 12, H: isoToolbarH - 12})
}

// seedDocs (re)creates both replicas from one shared snapshot and rebinds the
// mode / placement observers onto the fresh diagrams. It is the shared path for
// startup and reset.
func (s *isoScene) seedDocs() {
	for _, u := range s.unbind {
		u()
	}
	s.unbind = nil

	s.docA = toolkit.NewIsoCRDTDocument(siteA)
	seedIso(s.docA)
	// Replica B joins from A's snapshot, so both share the same base history and
	// start byte-identical. The snapshot is self-produced and valid.
	s.docB, _ = toolkit.LoadIsoCRDTDocument(siteB, s.docA.Snapshot())

	s.diaA = toolkit.NewIsoDiagram(s.docA)
	s.diaA.SetBounds(toolkit.Rect{X: isoPanelAX, Y: isoDiaTop, W: isoPanelW, H: panelDrawH})
	s.diaB = toolkit.NewIsoDiagram(s.docB)
	s.diaB.SetBounds(toolkit.Rect{X: isoPanelBX, Y: isoDiaTop, W: isoPanelW, H: panelDrawH})
	s.active = s.diaA
	s.applyMode(s.mode.Get())

	// MVVM wiring: the mode observable drives both diagrams' edit mode, and the
	// palette's selected-icon observable is bound into both diagrams' placement
	// observables (picking an icon arms click-to-place on either canvas).
	s.unbind = append(s.unbind, s.mode.Subscribe(func(m toolkit.IsoMode) { s.applyMode(m) }))
	s.unbind = append(s.unbind, s.palette.SelectedIcon().Subscribe(func(id string) {
		s.diaA.PlacementIconObservable().Set(id)
		s.diaB.PlacementIconObservable().Set(id)
	}))
	s.diaA.PlacementIconObservable().Set(s.palette.SelectedIcon().Get())
	s.diaB.PlacementIconObservable().Set(s.palette.SelectedIcon().Get())

	s.syncedA = s.docA.Version()
	s.syncedB = s.docB.Version()
	s.refreshStatus()
}

// applyMode mirrors the shared mode onto both diagram widgets.
func (s *isoScene) applyMode(m toolkit.IsoMode) {
	s.diaA.Mode = m
	s.diaB.Mode = m
}

// reset returns to the initial mode and a fresh pair of replicas.
func (s *isoScene) reset() {
	s.mode.Set(toolkit.IsoModeSelect)
	s.palette.SelectIcon("")
	s.seedDocs()
}

// setActive marks d (Site A or B) as the target of the toolbar's per-canvas
// commands (undo, delete, zoom).
func (s *isoScene) setActive(d *toolkit.IsoDiagram) { s.active = d }

// activeName is "A" or "B" for the active canvas.
func (s *isoScene) activeName() string {
	if s.active == s.diaA {
		return "A"
	}
	return "B"
}

// zoomActive zooms the active canvas about its centre by factor (>1 in, <1 out).
func (s *isoScene) zoomActive(factor float64) {
	b := s.active.Bounds()
	s.active.ZoomAt(factor, b.W/2, b.H/2)
	s.refreshStatus()
}

// toggleLayer flips the monitor layer's visibility on both replicas to the same
// value and syncs, so the show/hide is reflected in step on both canvases.
func (s *isoScene) toggleLayer() {
	l, _ := s.docA.Layer(layerMonitor)
	vis := !l.Visible
	for _, d := range []*toolkit.IsoCRDTDocument{s.docA, s.docB} {
		ll, _ := d.Layer(layerMonitor)
		ll.Visible = vis
		d.PutLayer(ll)
	}
	s.sync()
	s.refreshStatus()
}

// afterEdit propagates an edit to the peer replica and refreshes the status.
func (s *isoScene) afterEdit() {
	s.sync()
	s.refreshStatus()
}

// sync exchanges the operations each replica is missing, both ways, then
// advances the per-replica cursors so a later sync sends only newer edits. The
// operations come from sibling replicas of one document and are always
// well-formed, so Apply cannot report an error here; the impossible error is
// discarded deliberately (convergence is proven independently by the test's
// snapshot comparison).
func (s *isoScene) sync() {
	aOps := s.docA.OpsSince(s.syncedA)
	bOps := s.docB.OpsSince(s.syncedB)
	_ = s.docB.Apply(aOps...)
	_ = s.docA.Apply(bOps...)
	s.syncedA = s.docA.Version()
	s.syncedB = s.docB.Version()
}

// refreshStatus rewrites the bottom status line from the live scene state.
func (s *isoScene) refreshStatus() {
	s.statusLabel.Text = fmt.Sprintf(
		"mode %s · active Site %s · A %d nodes / B %d nodes · armed icon %q — drag a palette icon onto a canvas to place a node; left-drag empty ground pans; right-click edits; Sync merges the replicas",
		modeName(s.mode.Get()), s.activeName(), len(s.docA.Nodes()), len(s.docB.Nodes()), s.palette.SelectedIcon().Get(),
	)
}

// modeName is the human label for an edit mode.
func modeName(m toolkit.IsoMode) string {
	switch m {
	case toolkit.IsoModeConnect:
		return "Connect"
	case toolkit.IsoModeZone:
		return "Zone"
	case toolkit.IsoModeText:
		return "Text"
	default:
		return "Select"
	}
}

// --- webcanvas.App ------------------------------------------------------

// Size reports the fixed surface dimensions.
func (s *isoScene) Size() (int, int) { return isoSurfaceW, isoSurfaceH }

// Draw paints the whole workspace: background, toolbar, canvas titles, the two
// diagrams (each of which draws its own open context menu), the docked palette
// and the status line.
func (s *isoScene) Draw(buf []byte) {
	p := painter.NewPixelPainter(buf, isoSurfaceW, isoSurfaceH)
	p.FillRect(toolkit.Rect{X: 0, Y: 0, W: isoSurfaceW, H: isoSurfaceH}, s.theme.Background)
	s.toolbar.Draw(p, s.theme)
	s.labelA.Draw(p, s.theme)
	s.labelB.Draw(p, s.theme)
	s.diaA.Draw(p, s.theme)
	s.diaB.Draw(p, s.theme)
	s.palette.Draw(p, s.theme)
	s.statusLabel.Draw(p, s.theme)
}

// local translates a surface event into w's local coordinate frame.
func local(w toolkit.Widget, kind toolkit.EventKind, x, y int) toolkit.Event {
	b := w.Bounds()
	return toolkit.Event{Kind: kind, X: x - b.X, Y: y - b.Y}
}

// diagramAt returns the canvas under (x, y), if any.
func (s *isoScene) diagramAt(x, y int) (*toolkit.IsoDiagram, bool) {
	if s.diaA.Bounds().Contains(x, y) {
		return s.diaA, true
	}
	if s.diaB.Bounds().Contains(x, y) {
		return s.diaB, true
	}
	return nil, false
}

// Click begins a gesture in whichever region holds (x, y): the toolbar fires a
// button, the palette arms an icon / starts a panel move, a canvas starts a
// diagram gesture (and becomes the active canvas).
func (s *isoScene) Click(x, y int) bool {
	switch {
	case s.toolbar.Bounds().Contains(x, y):
		s.capture = tgtToolbar
		s.toolbar.OnEvent(local(s.toolbar, toolkit.EventClick, x, y))
		s.refreshStatus()
	case s.palette.Bounds().Contains(x, y):
		s.capture = tgtPalette
		s.palette.OnEvent(local(s.palette, toolkit.EventClick, x, y))
		s.refreshStatus()
	case s.diaA.Bounds().Contains(x, y):
		s.capture = tgtDiaA
		s.setActive(s.diaA)
		s.diaA.OnEvent(local(s.diaA, toolkit.EventClick, x, y))
		s.refreshStatus()
	case s.diaB.Bounds().Contains(x, y):
		s.capture = tgtDiaB
		s.setActive(s.diaB)
		s.diaB.OnEvent(local(s.diaB, toolkit.EventClick, x, y))
		s.refreshStatus()
	default:
		s.capture = tgtNone
		return false
	}
	return true
}

// Move advances the in-flight gesture on the captured widget (a diagram drag /
// pan, or a palette panel move); with nothing captured it is an inert hover.
func (s *isoScene) Move(x, y int) bool {
	switch s.capture {
	case tgtDiaA:
		s.diaA.OnEvent(local(s.diaA, toolkit.EventMouseDrag, x, y))
	case tgtDiaB:
		s.diaB.OnEvent(local(s.diaB, toolkit.EventMouseDrag, x, y))
	case tgtPalette:
		s.palette.OnEvent(local(s.palette, toolkit.EventMouseDrag, x, y))
	default:
		return false
	}
	return true
}

// Release commits the in-flight gesture. A palette release over a canvas with an
// armed icon drops a node at the exact tile under the pointer (one undoable edit)
// and syncs it to the peer.
func (s *isoScene) Release(x, y int) bool {
	switch s.capture {
	case tgtDiaA:
		s.diaA.OnEvent(local(s.diaA, toolkit.EventMouseUp, x, y))
		s.afterEdit()
	case tgtDiaB:
		s.diaB.OnEvent(local(s.diaB, toolkit.EventMouseUp, x, y))
		s.afterEdit()
	case tgtPalette:
		s.palette.OnEvent(local(s.palette, toolkit.EventMouseUp, x, y))
		if payload := s.palette.DragData(); payload != "" {
			if d, ok := s.diagramAt(x, y); ok {
				b := d.Bounds()
				d.OnEvent(toolkit.Event{Kind: toolkit.EventDrop, X: x - b.X, Y: y - b.Y, Code: payload})
				s.setActive(d)
				s.afterEdit()
			}
		}
	case tgtToolbar:
		s.toolbar.OnEvent(local(s.toolbar, toolkit.EventMouseUp, x, y))
	}
	s.capture = tgtNone
	return true
}

// Context opens the active canvas' own context menu at the right-clicked point.
func (s *isoScene) Context(x, y int) bool {
	if d, ok := s.diagramAt(x, y); ok {
		s.setActive(d)
		d.OnEvent(local(d, toolkit.EventSecondaryClick, x, y))
		return true
	}
	return false
}

// Char consumes printable input. The editor drives text through modes and the
// toolbar rather than typing, so there is nothing to type into.
func (s *isoScene) Char(string) bool { return false }

// KeyDown forwards a named key to the active canvas (Delete removes the
// selection) and syncs any resulting edit to the peer.
func (s *isoScene) KeyDown(code string) bool {
	s.active.OnEvent(toolkit.Event{Kind: toolkit.EventKeyDown, Code: code})
	s.afterEdit()
	return true
}

// --- PNG capture (inspectable artifacts) --------------------------------

// encodePNG renders the whole composite workspace to PNG bytes. png.Encode into
// an in-memory buffer has no I/O to fail on, so its (impossible here) error is
// discarded — matching toolkit.RenderPNG's documented reasoning.
func (s *isoScene) encodePNG() []byte {
	buf := make([]byte, 4*isoSurfaceW*isoSurfaceH)
	s.Draw(buf)
	img := image.NewRGBA(image.Rect(0, 0, isoSurfaceW, isoSurfaceH))
	copy(img.Pix, buf)
	var out bytes.Buffer
	_ = png.Encode(&out, img)
	return out.Bytes()
}

// convergedScene returns a scene after a concurrent-edit-then-sync scenario:
// replica A moves nodeWeb while replica B recolours that same node and adds a
// node; after sync both replicas hold byte-identical documents and therefore
// render identically. It is the collaboration proof, reused by the capture set.
func convergedScene() *isoScene {
	s := newIsoScene()
	na, _ := s.docA.Node(nodeWeb)
	na.X, na.Y = collabMovedX, collabMovedY
	s.docA.PutNode(na)
	nb, _ := s.docB.Node(nodeWeb)
	nb.Color = collabRecolor
	s.docB.PutNode(nb)
	s.docB.PutNode(toolkit.IsoNode{ID: "extra", X: 8, Y: 6, Icon: "cloud", Color: colDB, Label: "extra", Layer: layerInfra})
	s.sync()
	return s
}

// renderPanel captures one diagram canvas at panel size. panelW/panelDrawH are
// positive, so RenderPNG cannot fail; its error is discarded.
func renderPanel(d *toolkit.IsoDiagram, theme *toolkit.Theme) []byte {
	data, _ := toolkit.RenderPNG(d, panelW, panelDrawH, theme)
	return data
}

// isoCapture is one named inspectable image.
type isoCapture struct {
	name string
	data []byte
}

// isoCaptures builds the demonstrator's showcase images in a stable order: the
// full editor, then the two converged replica canvases (identical bytes).
func isoCaptures() []isoCapture {
	editor := newIsoScene()
	conv := convergedScene()
	return []isoCapture{
		{"iso-editor.png", editor.encodePNG()},
		{"converged-a.png", renderPanel(conv.diaA, conv.theme)},
		{"converged-b.png", renderPanel(conv.diaB, conv.theme)},
	}
}

// generatePNGs writes every showcase image into dir (which must already exist),
// returning the paths written in capture order. The first write to fail aborts.
func generatePNGs(dir string) ([]string, error) {
	var paths []string
	for _, c := range isoCaptures() {
		path := filepath.Join(dir, c.name)
		if err := os.WriteFile(path, c.data, 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
