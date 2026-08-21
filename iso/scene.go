// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Command iso is a browser wasm demonstrator of the go-widgets/toolkit
// isometric diagram widget ([toolkit.IsoDiagram]) and its collaborative CRDT
// backing store ([toolkit.IsoCRDTDocument]). It is an INTERACTIVE editor you
// drive in the page — not a static picture:
//
//   - a docked [toolkit.IsoIconPalette] over a rich registry: the built-in
//     architecture icons, the LIVE animated icons ("anim/*") and the vendored
//     cloud-native + AWS packs (go-widgets/isoicons). Drag an icon onto a canvas
//     and a node is created at the exact ground tile under the drop (undoable);
//     picking an icon also arms click-to-place through an mvvm.Observable binding;
//   - a real toolbar built from [toolkit.ButtonGroup]s of icon+label buttons
//     (go-iconoir glyphs, themed): a Modes group (Select / Node / Connect / Zone /
//     Text), an Edit group (Undo / Redo / Delete), a View group (RotateCCW /
//     RotateCW / Zoom+ / Zoom- / Layer) and a Collaborate group (Sync / Reset).
//     Left-drag empty ground pans; right-click opens the widget's own menu;
//   - TWO canvases side by side ("Site A" / "Site B"), each backed by its own
//     [toolkit.IsoCRDTDocument] on a distinct CRDT site. Every edit auto-syncs
//     the two replicas (operations exchanged both ways via OpsSince/Apply), so
//     an edit in one panel appears in the other, live, with no server — the
//     collaborative showpiece. The view rotation is LOCAL per canvas, so the two
//     panels may face different orientations without diverging the shared model.
//
// The animated icons advance from a real clock: the shared internal/webcanvas
// harness drives a requestAnimationFrame loop that hands the scene the elapsed
// dt through [isoScene.AnimationStep], which the scene forwards to both diagrams'
// [toolkit.IsoDiagram.AnimationStep]. The phase-advance logic is a pure function
// of dt, so it is fully exercised natively (the browser rAF wiring is the only
// `js && wasm` code, in run_js.go).
//
// All the scene logic lives here in a tag-less file, fully exercised by the
// native scene_test.go (it renders to a buffer and asserts exact projected
// pixel positions, drives a palette drop and a CRDT convergence, steps the
// animation and rotates the view). The only `js && wasm` code is main.go, a
// one-line hand-off to the shared internal/webcanvas harness, which drops out of
// the native build.
package main

import (
	"bytes"
	"fmt"
	"image"
	stdcolor "image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/go-crdt/crdt"
	"github.com/go-gfx/gfx/geometry"
	"github.com/go-gfx/gfx/iso"
	"github.com/go-gfx/gfx/raster"
	"github.com/go-iconoir/iconoir"
	"github.com/go-widgets/isoicons"
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/painter"
	"github.com/go-widgets/toolkit"
)

// Surface dimensions. These are the DEFAULT (and the standalone-capture) size:
// the scene opens at them, and the showcase PNGs render at them. In the browser
// the scene is a [webcanvas.Resizer], so the live surface tracks the viewport
// (see [isoScene.Resize]) — the two canvases stretch to fill the window — while
// the fixed defaults keep the native capture set byte-stable.
const (
	isoSurfaceW = 1120
	isoSurfaceH = 700

	// isoMinSurfaceW / isoMinSurfaceH floor the resizable surface so the two
	// canvases (and the status line) always keep a workable, positive size however
	// small the window is dragged.
	isoMinSurfaceW = 640
	isoMinSurfaceH = 400
)

// Layout geometry (all in surface pixels). Two equal canvas columns sit right of
// the docked palette, under the toolbar; a status line runs along the bottom.
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

// Toolbar geometry. Uniform icon+label cells clustered into ButtonGroups laid
// left-to-right with a small gap; each cell carries a go-iconoir glyph beside its
// text label.
const (
	tbY        = 6
	tbH        = isoToolbarH - 12 // 32
	tbX0       = 6
	tbBtnW     = 72 // one icon+label cell
	tbGroupGap = 8  // gap between ButtonGroups
	tbIconSz   = 16 // iconoir glyph square, device px
	tbIconPad  = 6  // left inset of the glyph
	tbTextPad  = 4  // gap between glyph and label
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

// Seed node icons. The web and db nodes carry ANIMATED icons ("anim/*"), so the
// starting scene visibly breathes/blinks once the rAF clock drives it; the cache
// node stays a still icon.
const (
	iconWeb   = "anim/server"
	iconDB    = "anim/database"
	iconCache = "storage"
)

// seedIso populates doc with the shared starting diagram every replica begins
// from: two named layers, a grouping zone, three nodes across the two layers, a
// connector and a floating annotation.
func seedIso(doc toolkit.IsoDocument) {
	doc.PutLayer(toolkit.IsoLayer{ID: layerInfra, Name: "Infrastructure", Visible: true, Order: 0})
	doc.PutLayer(toolkit.IsoLayer{ID: layerMonitor, Name: "Monitoring", Visible: true, Order: 1})
	doc.PutZone(toolkit.IsoZone{ID: "z-core", X: 1, Y: 1, W: 5, H: 3, Color: colZone, Label: "Core", Layer: layerInfra})
	doc.PutNode(toolkit.IsoNode{ID: nodeWeb, X: 2, Y: 2, Icon: iconWeb, Color: colWeb, Label: "web", Layer: layerInfra})
	doc.PutNode(toolkit.IsoNode{ID: nodeDB, X: 5, Y: 2, Icon: iconDB, Color: colDB, Label: "db", Layer: layerInfra})
	doc.PutNode(toolkit.IsoNode{ID: nodeCache, X: 7, Y: 5, Icon: iconCache, Color: colCache, Label: "cache", Layer: layerMonitor})
	doc.PutConnector(toolkit.IsoConnector{ID: "c-sql", From: nodeWeb, To: nodeDB, Arrow: toolkit.IsoArrowSingle, Label: "sql", Layer: layerInfra})
	doc.PutText(toolkit.IsoText{ID: "t-region", X: 1, Y: 8, Text: "region: eu-west", Size: 1, Layer: layerMonitor})
}

// newIsoRegistry builds the icon registry both diagrams and the palette share: a
// fresh registry seeded with the toolkit's built-in architecture icons, the
// built-in animated icons ("anim/*") and the vendored cloud-native + AWS sprite
// packs. It never mutates the package-global registry, so a native test observes
// exactly this set. The pack loaders only fail on a corrupt vendored asset (they
// are decoded from the isoicons module's own embed.FS and covered by its suite),
// so the impossible error is discarded deliberately — matching the sync/Apply
// reasoning below.
func newIsoRegistry() *toolkit.IsoIconRegistry {
	reg := toolkit.NewIsoIconRegistry()
	// Copy the built-ins across (the constructor for a seeded registry is
	// package-private in the toolkit; resolving each built-in id off the default
	// registry gives the same icon without touching the global).
	for _, id := range toolkit.IsoBuiltinIconIDs {
		icon, _ := toolkit.IsoDefaultIcons().Resolve(id)
		reg.Register(id, icon)
	}
	toolkit.RegisterAnimatedIcons(reg)
	_ = isoicons.RegisterCloudNative(reg, isoicons.ThemeLight)
	_ = isoicons.RegisterAWS(reg)
	return reg
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

// toolCmd is one toolbar cell: a stable key (for the test to fire it by name), a
// go-iconoir glyph name, a text label and its click handler.
type toolCmd struct {
	key   string
	icon  string
	label string
	fn    func()
}

// toolGroupSpec is one labelled cluster of toolbar cells rendered as a
// [toolkit.ButtonGroup].
type toolGroupSpec struct {
	name string
	cmds []toolCmd
}

// isoScene is the whole demonstrator: the toolbar, the docked palette, the two
// CRDT-backed diagram canvases and the status line, plus the pointer-capture,
// animation and sync bookkeeping that ties them together. It implements
// webcanvas.App (Size/Draw/Click/Move/Release/Context/Char/KeyDown) and the
// optional webcanvas.Animator (AnimationStep).
type isoScene struct {
	theme *toolkit.Theme

	// w, h are the CURRENT surface size in pixels. They start at the fixed
	// defaults (isoSurfaceW × isoSurfaceH) and, in the browser, follow the viewport
	// through [isoScene.Resize]; every widget bound is derived from them so a
	// resize relayouts the whole workspace.
	w, h int

	// reg is the icon registry shared by the palette and both diagrams, so an
	// icon listed in the palette resolves and renders identically on the canvas.
	reg *toolkit.IsoIconRegistry

	// Shared cross-boundary state (MVVM): the active edit mode drives both
	// diagrams; the palette's selected-icon observable is bound into both
	// diagrams' placement observables so picking an icon arms click-to-place.
	mode *mvvm.Observable[toolkit.IsoMode]

	// ghost is the drag/placement preview state (the icon in hand + the pointer
	// position it follows), held in ONE mvvm.Observable so the cross-boundary
	// preview is observed, not polled. It arms when a palette icon is grabbed or an
	// icon is armed for click-to-place, follows the cursor, and clears on drop or
	// cancel — the "you picked it up" feedback drawn translucent under the pointer.
	ghost *mvvm.Observable[isoGhost]

	groups   []*toolkit.ButtonGroup
	btnByKey map[string]*toolkit.Button
	tbGroup  *toolkit.ButtonGroup // the group holding the in-flight press, if any

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

	// dirty is set by either diagram's OnInvalidate; AnimationStep clears it, steps
	// both diagrams, and reports it so the rAF host repaints only when an animated
	// icon actually advanced a pixel.
	dirty bool

	capture isoTarget
	unbind  []func()
}

// newIsoScene builds the demonstrator with both replicas seeded to the same
// starting diagram over the shared enriched icon registry.
func newIsoScene() *isoScene {
	s := &isoScene{
		theme:    toolkit.DefaultLight(),
		w:        isoSurfaceW,
		h:        isoSurfaceH,
		reg:      newIsoRegistry(),
		mode:     mvvm.NewObservable(toolkit.IsoModeSelect),
		ghost:    mvvm.NewObservable(isoGhost{}),
		btnByKey: map[string]*toolkit.Button{},
	}

	// The palette and the two canvas titles are created here; every bound (theirs,
	// the status line's and both canvases') is assigned from the surface size by
	// relayout, called at the end of seedDocs once the diagrams exist.
	s.palette = toolkit.NewIsoIconPalette(s.reg)
	s.labelA = toolkit.NewLabel("Site A — CRDT replica 1")
	s.labelB = toolkit.NewLabel("Site B — CRDT replica 2")
	s.statusLabel = toolkit.NewLabel("")

	s.buildToolbar()
	s.seedDocs()
	return s
}

// isoGhost is the drag/placement preview: the id of the icon currently in hand and
// the surface pixel the pointer sits at (the ghost's centre). Active is false when
// nothing is in hand. It is a comparable value so the whole preview rides in one
// mvvm.Observable.
type isoGhost struct {
	Icon   string
	X, Y   int
	Active bool
}

// isoRects is the whole workspace geometry for one surface size: where the docked
// palette, the two canvas titles, the two canvases and the status line sit. Every
// bound the scene assigns comes from here, so a resize relayouts by recomputing
// this single value.
type isoRects struct {
	palette, labelA, labelB, status, diaA, diaB toolkit.Rect
}

// computeLayout derives the workspace geometry from the current surface size: a
// fixed-width palette at the left, the two equal canvas columns filling the rest
// of the width, the titles above them and the status line along the bottom. At the
// default 1120×700 it reproduces the original fixed layout exactly.
func (s *isoScene) computeLayout() isoRects {
	gap := isoGap
	panelTop := isoToolbarH + 4
	diaTop := panelTop + isoLabelH + 2
	statusY := s.h - isoStatusH - 4
	panelAX := isoPaletteX + isoPaletteW + gap
	panelW := (s.w - panelAX - 2*gap) / 2
	panelBX := panelAX + panelW + gap
	panelDrawH := statusY - diaTop - 6
	return isoRects{
		palette: toolkit.Rect{X: isoPaletteX, Y: panelTop, W: isoPaletteW, H: statusY - panelTop - 6},
		labelA:  toolkit.Rect{X: panelAX, Y: panelTop, W: panelW, H: isoLabelH},
		labelB:  toolkit.Rect{X: panelBX, Y: panelTop, W: panelW, H: isoLabelH},
		status:  toolkit.Rect{X: isoPaletteX, Y: statusY, W: s.w - 2*gap, H: isoStatusH},
		diaA:    toolkit.Rect{X: panelAX, Y: diaTop, W: panelW, H: panelDrawH},
		diaB:    toolkit.Rect{X: panelBX, Y: diaTop, W: panelW, H: panelDrawH},
	}
}

// relayout assigns every widget its bound from the current surface size. It runs
// at build time and on every resize; the diagrams always exist by the time it is
// called (seedDocs creates them, then calls this).
func (s *isoScene) relayout() {
	r := s.computeLayout()
	s.palette.SetBounds(r.palette)
	s.labelA.SetBounds(r.labelA)
	s.labelB.SetBounds(r.labelB)
	s.statusLabel.SetBounds(r.status)
	s.diaA.SetBounds(r.diaA)
	s.diaB.SetBounds(r.diaB)
}

// Resize is the [webcanvas.Resizer] hook: it floors the requested size to the
// minimum workable surface, relayouts the whole workspace to fit, and returns the
// size it will render at (the size the host sizes the canvas and framebuffer to).
// The browser calls it with the viewport box, so the editor opens and stays full
// page and the two canvases stretch to fill the window.
func (s *isoScene) Resize(w, h int) (int, int) {
	if w < isoMinSurfaceW {
		w = isoMinSurfaceW
	}
	if h < isoMinSurfaceH {
		h = isoMinSurfaceH
	}
	s.w, s.h = w, h
	s.relayout()
	return s.w, s.h
}

// toolbarSpec is the toolbar's whole command layout: four logical ButtonGroups.
// Handlers read the live scene, so they keep working across a reset that rebuilds
// the documents.
func (s *isoScene) toolbarSpec() []toolGroupSpec {
	return []toolGroupSpec{
		{name: "Modes", cmds: []toolCmd{
			{"select", "cursor-pointer", "Select", func() { s.mode.Set(toolkit.IsoModeSelect); s.palette.SelectIcon(""); s.refreshStatus() }},
			{"node", "plus-square", "Node", func() { s.mode.Set(toolkit.IsoModeSelect); s.palette.SelectIcon(iconWeb); s.refreshStatus() }},
			{"connect", "link", "Connect", func() { s.mode.Set(toolkit.IsoModeConnect); s.refreshStatus() }},
			{"zone", "frame", "Zone", func() { s.mode.Set(toolkit.IsoModeZone); s.refreshStatus() }},
			{"text", "text", "Text", func() { s.mode.Set(toolkit.IsoModeText); s.refreshStatus() }},
		}},
		{name: "Edit", cmds: []toolCmd{
			{"undo", "undo", "Undo", func() { s.active.Undo(); s.afterEdit() }},
			{"redo", "redo", "Redo", func() { s.active.Redo(); s.afterEdit() }},
			{"delete", "trash", "Delete", func() { s.active.DeleteSelection(); s.afterEdit() }},
		}},
		{name: "View", cmds: []toolCmd{
			{"rotccw", "rotate-camera-left", "CCW", func() { s.active.RotateCCW(); s.refreshStatus() }},
			{"rotcw", "rotate-camera-right", "CW", func() { s.active.RotateCW(); s.refreshStatus() }},
			{"zoomin", "zoom-in", "Zoom+", func() { s.zoomActive(toolkit.IsoZoomStep) }},
			{"zoomout", "zoom-out", "Zoom-", func() { s.zoomActive(1 / toolkit.IsoZoomStep) }},
			{"layer", "multiple-pages", "Layer", func() { s.toggleLayer() }},
		}},
		{name: "Collab", cmds: []toolCmd{
			{"sync", "refresh-double", "Sync", func() { s.sync(); s.refreshStatus() }},
			{"reset", "restart", "Reset", func() { s.reset() }},
		}},
	}
}

// buildToolbar assembles the four ButtonGroups once. Each button carries a
// go-iconoir glyph and its text label, drawn through the button's Icon seam so
// the glyph tracks the pressed / disabled tint; ButtonGroup owns the shared
// rounded chrome and inter-member dividers.
func (s *isoScene) buildToolbar() {
	x := tbX0
	for _, gs := range s.toolbarSpec() {
		btns := make([]*toolkit.Button, 0, len(gs.cmds))
		for _, c := range gs.cmds {
			b := toolkit.NewButton(c.label, c.fn)
			b.Icon = iconLabelPainter(c.icon, c.label)
			s.btnByKey[c.key] = b
			btns = append(btns, b)
		}
		g := toolkit.NewButtonGroup(btns...)
		w := len(btns) * tbBtnW
		g.SetBounds(toolkit.Rect{X: x, Y: tbY, W: w, H: tbH})
		s.groups = append(s.groups, g)
		x += w + tbGroupGap
	}
}

// iconLabelPainter returns a [toolkit.Button.Icon] closure that paints a
// go-iconoir glyph in a small left square and the text label to its right, both
// in the button's current ink (so they follow the pressed / disabled tint). The
// glyph is fetched once at build time; its name is a compile-time constant proven
// present in the vendored go-iconoir set.
func iconLabelPainter(name, label string) func(p painter.Painter, r toolkit.Rect, ink toolkit.RGBA) {
	ic := iconoir.MustGet(name)
	return func(p painter.Painter, r toolkit.Rect, ink toolkit.RGBA) {
		gy := r.Y + (r.H-tbIconSz)/2
		iconoir.DrawIcon(p, toolkit.Rect{X: r.X + tbIconPad, Y: gy, W: tbIconSz, H: tbIconSz}, ic, ink)
		lx := r.X + tbIconPad + tbIconSz + tbTextPad
		ly := r.Y + (r.H-toolkit.GlyphHeight())/2
		toolkit.DrawText(p, lx, ly, label, ink)
	}
}

// seedDocs (re)creates both replicas from one shared snapshot and rebinds the
// mode / placement / invalidate observers onto the fresh diagrams. It is the
// shared path for startup and reset.
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
	s.diaA.Icons = s.reg
	s.diaA.OnInvalidate = func() { s.dirty = true }
	s.diaB = toolkit.NewIsoDiagram(s.docB)
	s.diaB.Icons = s.reg
	s.diaB.OnInvalidate = func() { s.dirty = true }
	s.active = s.diaA
	s.applyMode(s.mode.Get())
	// Now that both diagrams exist, lay the whole workspace out for the current
	// surface size (positions the palette, titles, status line and both canvases).
	s.relayout()

	// MVVM wiring: the mode observable drives both diagrams' edit mode, and the
	// palette's selected-icon observable is bound into both diagrams' placement
	// observables (picking an icon arms click-to-place on either canvas). Disarming
	// (id "") also clears any in-flight ghost preview.
	s.unbind = append(s.unbind, s.mode.Subscribe(func(m toolkit.IsoMode) { s.applyMode(m) }))
	s.unbind = append(s.unbind, s.palette.SelectedIcon().Subscribe(func(id string) {
		s.diaA.PlacementIconObservable().Set(id)
		s.diaB.PlacementIconObservable().Set(id)
		if id == "" {
			s.clearGhost()
		}
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
// commands (undo, delete, zoom, rotate).
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
	s.statusLabel.Text().Set(fmt.Sprintf(
		"mode %s · active Site %s (view %d°) · A %d nodes / B %d nodes · armed icon %q — drag a palette icon onto a canvas to place a node; Rot CCW/CW turn the active view; left-drag empty ground pans; Sync merges the replicas",
		modeName(s.mode.Get()), s.activeName(), s.active.ViewRotation()*90, len(s.docA.Nodes()), len(s.docB.Nodes()), s.palette.SelectedIcon().Get(),
	))
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

// Size reports the current surface dimensions (the defaults until a resize).
func (s *isoScene) Size() (int, int) { return s.w, s.h }

// Draw paints the whole workspace: background, toolbar groups, canvas titles, the
// two diagrams (each of which draws its own open context menu), the docked
// palette, the status line and — on top of everything — the translucent drag
// ghost following the pointer while an icon is in hand.
func (s *isoScene) Draw(buf []byte) {
	p := painter.NewPixelPainter(buf, s.w, s.h)
	p.FillRect(toolkit.Rect{X: 0, Y: 0, W: s.w, H: s.h}, s.theme.Background)
	for _, g := range s.groups {
		g.Draw(p, s.theme)
	}
	s.labelA.Draw(p, s.theme)
	s.labelB.Draw(p, s.theme)
	s.diaA.Draw(p, s.theme)
	s.diaB.Draw(p, s.theme)
	s.palette.Draw(p, s.theme)
	s.statusLabel.Draw(p, s.theme)
	s.drawGhost(buf)
}

// local translates a surface event into w's local coordinate frame.
func local(w toolkit.Widget, kind toolkit.EventKind, x, y int) toolkit.Event {
	b := w.Bounds()
	return toolkit.Event{Kind: kind, X: x - b.X, Y: y - b.Y}
}

// groupAt returns the toolbar ButtonGroup under (x, y), if any.
func (s *isoScene) groupAt(x, y int) (*toolkit.ButtonGroup, bool) {
	for _, g := range s.groups {
		if g.Bounds().Contains(x, y) {
			return g, true
		}
	}
	return nil, false
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

// Click begins a gesture in whichever region holds (x, y): a toolbar group fires
// the button under the pointer, the palette arms an icon / starts a panel move, a
// canvas starts a diagram gesture (and becomes the active canvas).
func (s *isoScene) Click(x, y int) bool {
	if g, ok := s.groupAt(x, y); ok {
		s.capture = tgtToolbar
		s.tbGroup = g
		g.OnEvent(local(g, toolkit.EventClick, x, y))
		s.refreshStatus()
		return true
	}
	switch {
	case s.palette.Bounds().Contains(x, y):
		s.capture = tgtPalette
		s.palette.OnEvent(local(s.palette, toolkit.EventClick, x, y))
		// A press that armed an icon shows the ghost immediately under the pointer,
		// so the grab reads as "picked up" the instant it registers.
		s.updateGhost(x, y)
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
// pan, or a palette panel move) and, whenever an icon is in hand, slides the drag
// ghost to the pointer. With nothing captured it is a hover that still repaints
// when the ghost moved (so the preview tracks the cursor across the canvas), and
// is otherwise inert.
func (s *isoScene) Move(x, y int) bool {
	ghostMoved := s.updateGhost(x, y)
	switch s.capture {
	case tgtDiaA:
		s.diaA.OnEvent(local(s.diaA, toolkit.EventMouseDrag, x, y))
	case tgtDiaB:
		s.diaB.OnEvent(local(s.diaB, toolkit.EventMouseDrag, x, y))
	case tgtPalette:
		s.palette.OnEvent(local(s.palette, toolkit.EventMouseDrag, x, y))
	default:
		return ghostMoved
	}
	return true
}

// Release commits the in-flight gesture. A palette release over a canvas with an
// armed icon drops a node at the exact tile under the pointer (one undoable edit)
// and syncs it to the peer; a toolbar release clears the pressed member's face.
func (s *isoScene) Release(x, y int) bool {
	switch s.capture {
	case tgtDiaA:
		s.diaA.OnEvent(local(s.diaA, toolkit.EventMouseUp, x, y))
		s.afterEdit()
		s.clearGhost() // a click-to-place tap dropped the node (or a plain gesture ended)
	case tgtDiaB:
		s.diaB.OnEvent(local(s.diaB, toolkit.EventMouseUp, x, y))
		s.afterEdit()
		s.clearGhost()
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
		s.clearGhost() // the drag ended: dropped on a canvas, or cancelled off it
	case tgtToolbar:
		if s.tbGroup != nil {
			s.tbGroup.OnEvent(local(s.tbGroup, toolkit.EventMouseUp, x, y))
			s.tbGroup = nil
		}
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

// KeyDown routes a named key. Escape cancels an in-flight placement — it disarms
// the palette (which clears the ghost via the selected-icon binding) so a mistaken
// grab is dropped without placing anything. Every other key forwards to the active
// canvas (Delete removes the selection) and syncs any resulting edit to the peer.
func (s *isoScene) KeyDown(code string) bool {
	if code == "Escape" {
		s.palette.SelectIcon("")
		s.clearGhost()
		s.refreshStatus()
		return true
	}
	s.active.OnEvent(toolkit.Event{Kind: toolkit.EventKeyDown, Code: code})
	s.afterEdit()
	return true
}

// --- drag / placement ghost ---------------------------------------------

// isoGhostSize is the ghost preview's square side in device pixels; isoGhostOpacity
// is the translucency it is blitted at (0..255).
const (
	isoGhostSize    = 56
	isoGhostOpacity = 150
)

// updateGhost slides the ghost to the pointer while an icon is armed, reporting
// whether the preview changed (so a hover repaints to track the cursor). With
// nothing armed it does nothing — disarming clears the ghost through the palette
// binding, not here.
func (s *isoScene) updateGhost(x, y int) bool {
	armed := s.palette.SelectedIcon().Get()
	if armed == "" {
		return false
	}
	next := isoGhost{Icon: armed, X: x, Y: y, Active: true}
	if s.ghost.Get() == next {
		return false
	}
	s.ghost.Set(next)
	return true
}

// clearGhost hides the ghost (a drop landed, or the grab was cancelled), reporting
// whether it had been showing.
func (s *isoScene) clearGhost() bool {
	if !s.ghost.Get().Active {
		return false
	}
	s.ghost.Set(isoGhost{})
	return true
}

// ghostBase is the base colour the ghost icon is shaded from — the theme accent,
// the same default an un-coloured node uses.
func (s *isoScene) ghostBase() stdcolor.RGBA {
	c := s.theme.Accent
	return stdcolor.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// drawGhost composites the translucent icon preview over the finished frame at the
// pointer, when an icon is in hand. It draws last so the ghost floats above every
// widget.
func (s *isoScene) drawGhost(buf []byte) {
	g := s.ghost.Get()
	if !g.Active {
		return
	}
	icon, _ := s.reg.Resolve(g.Icon)
	img := renderGhostIcon(icon, isoGhostSize, s.ghostBase())
	compositeGhost(buf, s.w, s.h, img, g.X, g.Y, isoGhostOpacity)
}

// isoGhostWorldBox is the world-space box every ghost is fit to — a unit footprint
// two units tall, the extent of the tallest built-in icon — so a two-tall tower
// reads taller than a one-tall box, matching the palette thumbnails.
var isoGhostWorldBox = [8]iso.Vec3{
	iso.V(0, 0, 0), iso.V(1, 0, 0), iso.V(0, 1, 0), iso.V(1, 1, 0),
	iso.V(0, 0, 2), iso.V(1, 0, 2), iso.V(0, 1, 2), iso.V(1, 1, 2),
}

// ghostProjBBox is the screen bounding box of the world box projected through pr.
func ghostProjBBox(pr *iso.Projection) (minX, minY, maxX, maxY float64) {
	for i, v := range isoGhostWorldBox {
		p := pr.Project(v)
		if i == 0 {
			minX, maxX, minY, maxY = p.X, p.X, p.Y, p.Y
			continue
		}
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
	}
	return
}

// isoGhostProjection builds the 2:1 isometric projection that fits the world box
// centred into a size×size square with a size/8 margin — the same fit the palette
// thumbnails use, so the ghost matches the palette entry the user grabbed.
func isoGhostProjection(size int) *iso.Projection {
	pad := float64(size) / 8
	inner := float64(size) - 2*pad
	ref := iso.New(geometry.Pt(0, 0), 1, 0.5, 0.5)
	minX, minY, maxX, maxY := ghostProjBBox(ref)
	bw, bh := maxX-minX, maxY-minY
	scale := inner / bw
	if s := inner / bh; s < scale {
		scale = s
	}
	proj := iso.New(geometry.Pt(0, 0), scale, scale/2, scale/2)
	minX, minY, maxX, maxY = ghostProjBBox(proj)
	bw, bh = maxX-minX, maxY-minY
	proj.Origin = geometry.Pt(pad+(inner-bw)/2-minX, pad+(inner-bh)/2-minY)
	return proj
}

// renderGhostIcon renders icon into a fresh size×size TRANSPARENT buffer (so only
// the icon shows when it is composited): a sprite icon's art is blitted to fill the
// square, a primitive icon's shapes are depth-sorted through the fitted projection.
func renderGhostIcon(icon toolkit.IsoIcon, size int, base stdcolor.RGBA) *raster.Image {
	img := raster.New(size, size)
	dr := icon.Render(0, 0, base)
	if dr.Sprite != nil {
		drawGhostSprite(img, dr.Sprite)
	}
	if len(dr.Shapes) > 0 {
		iso.NewScene(isoGhostProjection(size)).Add(dr.Shapes...).Render(img)
	}
	return img
}

// drawGhostSprite blits src into dst scaled to fill it (nearest-neighbour), copying
// only non-transparent source pixels so the sprite's own transparency is preserved.
func drawGhostSprite(dst, src *raster.Image) {
	for y := 0; y < dst.H; y++ {
		sy := y * src.H / dst.H
		for x := 0; x < dst.W; x++ {
			c := src.At(x*src.W/dst.W, sy)
			if c.A == 0 {
				continue
			}
			dst.Set(x, y, c)
		}
	}
}

// compositeGhost source-over blends img, centred on (cx, cy) and scaled by opacity,
// into the surface buffer buf (surfW×surfH RGBA), clipped to the surface. It is the
// one place the ghost touches the finished frame.
func compositeGhost(buf []byte, surfW, surfH int, img *raster.Image, cx, cy int, opacity uint8) {
	ox, oy := cx-img.W/2, cy-img.H/2
	for y := 0; y < img.H; y++ {
		dy := oy + y
		if dy < 0 || dy >= surfH {
			continue
		}
		for x := 0; x < img.W; x++ {
			dx := ox + x
			if dx < 0 || dx >= surfW {
				continue
			}
			si := img.PixOffset(x, y)
			a := uint32(img.Pix[si+3]) * uint32(opacity) / 255
			if a == 0 {
				continue
			}
			di := 4 * (dy*surfW + dx)
			ia := 255 - a
			buf[di] = uint8((uint32(img.Pix[si])*a + uint32(buf[di])*ia) / 255)
			buf[di+1] = uint8((uint32(img.Pix[si+1])*a + uint32(buf[di+1])*ia) / 255)
			buf[di+2] = uint8((uint32(img.Pix[si+2])*a + uint32(buf[di+2])*ia) / 255)
			buf[di+3] = 255
		}
	}
}

// --- webcanvas.Animator -------------------------------------------------

// AnimationStep advances both diagrams' animation phase by dt (real elapsed
// seconds from the host's requestAnimationFrame clock) and reports whether either
// canvas actually needs a repaint — true exactly when a canvas holds a node
// carrying an animated icon, since only then does a phase change move a pixel.
// The phase-advance logic is a pure function of dt, so this method is exercised
// natively; the browser rAF loop that calls it is the only wasm-tagged code.
func (s *isoScene) AnimationStep(dt float64) bool {
	s.dirty = false
	s.diaA.AnimationStep(dt)
	s.diaB.AnimationStep(dt)
	return s.dirty
}

// --- PNG capture (inspectable artifacts) --------------------------------

// encodePNG renders the whole composite workspace to PNG bytes. png.Encode into
// an in-memory buffer has no I/O to fail on, so its (impossible here) error is
// discarded — matching toolkit.RenderPNG's documented reasoning.
func (s *isoScene) encodePNG() []byte {
	buf := make([]byte, 4*s.w*s.h)
	s.Draw(buf)
	img := image.NewRGBA(image.Rect(0, 0, s.w, s.h))
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

// animatedScene returns a fresh scene whose active canvas has been stepped to a
// mid-cycle animation phase, so its captured frame differs visibly from the rest
// frame. It is the animation proof, reused by the capture set.
func animatedScene() *isoScene {
	s := newIsoScene()
	// Pin an explicit one-second cycle on both canvases, then step half of it, so
	// the phase lands at 0.5 — the peak of the blink / breathe / bob curves —
	// independent of the toolkit's internal default period.
	s.diaA.AnimationPeriod = 1
	s.diaB.AnimationPeriod = 1
	s.AnimationStep(0.5)
	return s
}

// rotatedScene returns a fresh scene whose active canvas (Site A) has been turned
// one quarter-turn clockwise, so its captured frame shows the re-oriented plane.
// It is the view-rotation proof, reused by the capture set.
func rotatedScene() *isoScene {
	s := newIsoScene()
	s.setActive(s.diaA)
	s.diaA.RotateCW()
	return s
}

// ghostScene returns a fresh scene with an icon armed from the palette and the
// drag ghost hovering over the centre of Site A, so a capture shows the translucent
// preview riding under the pointer. It is the drag-ghost proof, reused by the
// capture set.
func ghostScene() *isoScene {
	s := newIsoScene()
	s.palette.SelectIcon(iconCache) // a still icon reads clearly as a ghost
	da := s.diaA.Bounds()
	s.Move(da.X+da.W/2, da.Y+da.H/2)
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
// full enriched editor, the two converged replica canvases (identical bytes), one
// mid-cycle animation frame, one clockwise-rotated view and one full-surface frame
// with the drag ghost under the pointer.
func isoCaptures() []isoCapture {
	editor := newIsoScene()
	conv := convergedScene()
	anim := animatedScene()
	rot := rotatedScene()
	ghost := ghostScene()
	return []isoCapture{
		{"iso-editor.png", editor.encodePNG()},
		{"converged-a.png", renderPanel(conv.diaA, conv.theme)},
		{"converged-b.png", renderPanel(conv.diaB, conv.theme)},
		{"anim-frame.png", renderPanel(anim.diaA, anim.theme)},
		{"rotated-a.png", renderPanel(rot.diaA, rot.theme)},
		{"iso-ghost.png", ghost.encodePNG()},
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
