// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"bytes"
	"image"
	stdcolor "image/color"
	_ "image/png" // registers the PNG decoder for image.DecodeConfig
	"math"
	"os"
	"path/filepath"
	"testing"

	gfxcolor "github.com/go-gfx/gfx/color"
	"github.com/go-gfx/gfx/iso"
	"github.com/go-gfx/gfx/raster"

	"github.com/go-widgets/isoicons"
	"github.com/go-widgets/toolkit"
)

// --- pixel helpers (same maths the widget's own overlay drawing uses) ----

func std(c toolkit.RGBA) stdcolor.RGBA { return stdcolor.RGBA{R: c.R, G: c.G, B: c.B, A: c.A} }

// screenOf projects a world point through d's live projection to the exact
// buffer pixel it lands on, so a test can name a pixel by the world coordinate
// that produced it.
func screenOf(d *toolkit.IsoDiagram, x, y, z float64) (int, int) {
	p := d.Projection().Project(iso.V(x, y, z))
	return int(math.Round(p.X)), int(math.Round(p.Y))
}

// faceColors are the three visible-face colours an isometric solid of base
// colour c is shaded into: top at factor 1 (exactly c), the +Y (left) and +X
// (right) sides progressively darker.
func faceColors(c toolkit.RGBA) []stdcolor.RGBA {
	s := std(c)
	return []stdcolor.RGBA{
		s,
		gfxcolor.Shade(s, iso.DefaultShading.Left),
		gfxcolor.Shade(s, iso.DefaultShading.Right),
	}
}

func isFace(c stdcolor.RGBA, set []stdcolor.RGBA) bool {
	for _, w := range set {
		if c == w {
			return true
		}
	}
	return false
}

// renderDiagram sizes d to a panel and captures it to an in-memory image.
func renderDiagram(t *testing.T, d *toolkit.IsoDiagram) *image.RGBA {
	t.Helper()
	img, err := toolkit.RenderImage(d, panelW, panelDrawH, toolkit.DefaultLight())
	if err != nil {
		t.Fatalf("RenderImage: %v", err)
	}
	return img
}

// nodeAtCell returns a copy of the node whose grid cell is (x,y), or nil.
func nodeAtCell(doc toolkit.IsoDocument, x, y int) *toolkit.IsoNode {
	for _, n := range doc.Nodes() {
		if n.X == x && n.Y == y {
			nn := n
			return &nn
		}
	}
	return nil
}

// idSet is the set of node ids currently in doc.
func idSet(doc *toolkit.IsoCRDTDocument) map[string]bool {
	m := map[string]bool{}
	for _, n := range doc.Nodes() {
		m[n.ID] = true
	}
	return m
}

// addedNode returns the single node present in doc but absent from before, or
// nil — the node a drop just created.
func addedNode(doc *toolkit.IsoCRDTDocument, before map[string]bool) *toolkit.IsoNode {
	for _, n := range doc.Nodes() {
		if !before[n.ID] {
			nn := n
			return &nn
		}
	}
	return nil
}

// clickBtn fires the toolbar button registered under key by routing a real
// Click+Release pair at its centre through the scene (surface click → group →
// button), so the button's handler runs exactly as a user's tap would drive it.
func clickBtn(s *isoScene, key string) {
	b := s.btnByKey[key]
	r := b.Bounds()
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	s.Click(cx, cy)
	s.Release(cx, cy)
}

// dropIcon arms icon in the palette, begins a palette gesture on the header (a
// header press keeps the armed icon), releases over the canvas at absolute
// (sx,sy), and returns the node the drop created on Site A.
func dropIcon(s *isoScene, icon string, sx, sy int) *toolkit.IsoNode {
	before := idSet(s.docA)
	s.palette.SelectIcon(icon)
	pb := s.palette.Bounds()
	s.Click(pb.X+10, pb.Y+2)
	s.Release(sx, sy)
	return addedNode(s.docA, before)
}

// --- exact-position pixel proof -----------------------------------------

// TestNodeTopAndSideFacesExact pins a bare cube node to its projected screen
// position: the top-face centre pixel is EXACTLY the base colour, and interior
// pixels of the two visible side faces are exactly the base colour shaded by the
// fixed left / right factors. Only a node painted precisely where iso.Project
// places it satisfies all three at once.
func TestNodeTopAndSideFacesExact(t *testing.T) {
	col := toolkit.RGB(220, 40, 40)
	doc := toolkit.NewIsoDoc()
	doc.PutNode(toolkit.IsoNode{ID: "n", X: 4, Y: 4, Color: col})
	d := toolkit.NewIsoDiagram(doc)
	img := renderDiagram(t, d)

	tx, ty := screenOf(d, 4.5, 4.5, 1) // top-face centre → exactly base colour
	if got := img.RGBAAt(tx, ty); got != std(col) {
		t.Fatalf("top-face centre pixel = %v, want exact base %v", got, std(col))
	}
	rx, ry := screenOf(d, 5, 4.5, 0.35) // +X (right) side interior
	if got, want := img.RGBAAt(rx, ry), gfxcolor.Shade(std(col), iso.DefaultShading.Right); got != want {
		t.Fatalf("+X side pixel = %v, want right-shade %v", got, want)
	}
	lx, ly := screenOf(d, 4.5, 5, 0.35) // +Y (left) side interior
	if got, want := img.RGBAAt(lx, ly), gfxcolor.Shade(std(col), iso.DefaultShading.Left); got != want {
		t.Fatalf("+Y side pixel = %v, want left-shade %v", got, want)
	}
}

// TestDepthOrderNearerDrawsOver proves the painter's-algorithm render order:
// rendering the far node alone and then both nodes, a pixel the far node painted
// its own colour becomes the near node's colour once the near node is added —
// the nearer solid draws over the farther one.
func TestDepthOrderNearerDrawsOver(t *testing.T) {
	far := toolkit.RGB(220, 30, 30)
	near := toolkit.RGB(30, 200, 30)
	farFaces, nearFaces := faceColors(far), faceColors(near)

	docFar := toolkit.NewIsoDoc()
	docFar.PutNode(toolkit.IsoNode{ID: "far", X: 0, Y: 0, Color: far})
	imgFar := renderDiagram(t, toolkit.NewIsoDiagram(docFar))

	docBoth := toolkit.NewIsoDoc()
	docBoth.PutNode(toolkit.IsoNode{ID: "far", X: 0, Y: 0, Color: far})
	docBoth.PutNode(toolkit.IsoNode{ID: "near", X: 1, Y: 0, Color: near})
	imgBoth := renderDiagram(t, toolkit.NewIsoDiagram(docBoth))

	for y := 0; y < panelDrawH; y++ {
		for x := 0; x < panelW; x++ {
			if isFace(imgFar.RGBAAt(x, y), farFaces) && isFace(imgBoth.RGBAAt(x, y), nearFaces) {
				return // a far-node pixel is now a near-node pixel: correct overlap.
			}
		}
	}
	t.Fatal("no pixel where the nearer node overdrew the farther one")
}

// --- interactive placement proof (palette drop → exact tile, undoable) --

// TestPaletteDropPlacesNodeAtExactTileUndoable arms an icon in the palette,
// starts a palette gesture and releases the pointer over Site A at the pixel a
// chosen ground tile projects to. It asserts a node carrying that icon appears
// at EXACTLY that tile, that the edit is undoable, and that a single Undo
// removes it — the drag-drop placement contract.
func TestPaletteDropPlacesNodeAtExactTileUndoable(t *testing.T) {
	s := newIsoScene()
	const gx, gy = 4, 4
	const icon = "router"

	before := len(s.docA.Nodes())
	s.palette.SelectIcon(icon)

	// Begin a palette gesture (a header press keeps the armed icon).
	pb := s.palette.Bounds()
	s.Click(pb.X+10, pb.Y+2)

	// Release over Site A at the exact pixel tile (gx,gy)'s centre projects to.
	db := s.diaA.Bounds()
	p := s.diaA.Projection().Project(iso.V(gx+0.5, gy+0.5, 0))
	s.Release(db.X+int(math.Round(p.X)), db.Y+int(math.Round(p.Y)))

	if got := len(s.docA.Nodes()); got != before+1 {
		t.Fatalf("Site A node count = %d, want %d (drop placed no node)", got, before+1)
	}
	placed := nodeAtCell(s.docA, gx, gy)
	if placed == nil {
		t.Fatalf("no node at dropped tile (%d,%d)", gx, gy)
	}
	if placed.Icon != icon {
		t.Fatalf("dropped node icon = %q, want %q", placed.Icon, icon)
	}
	if !s.diaA.CanUndo() {
		t.Fatal("placement is not undoable (CanUndo false)")
	}
	s.diaA.Undo()
	if nodeAtCell(s.docA, gx, gy) != nil {
		t.Fatal("Undo did not remove the placed node")
	}
}

// --- CRDT convergence proof (identical snapshot AND identical pixels) ---

// TestSeedReplicasStartIdentical checks both replicas begin byte-identical.
func TestSeedReplicasStartIdentical(t *testing.T) {
	s := newIsoScene()
	if !bytes.Equal(s.docA.Snapshot(), s.docB.Snapshot()) {
		t.Fatal("replicas do not start from an identical snapshot")
	}
}

// TestConvergenceIdenticalPixel drives a concurrent-edit-then-sync scenario
// through the scene's OWN sync path — replica A moves nodeWeb while replica B
// recolours that same node and adds a node — and proves the two replicas
// converge to a byte-identical snapshot AND render to byte-identical pixels,
// with the move and the recolour both surviving (a per-field LWW merge).
func TestConvergenceIdenticalPixel(t *testing.T) {
	s := convergedScene()

	if !bytes.Equal(s.docA.Snapshot(), s.docB.Snapshot()) {
		t.Fatal("replica snapshots differ after sync")
	}
	for name, doc := range map[string]*toolkit.IsoCRDTDocument{"A": s.docA, "B": s.docB} {
		n, ok := doc.Node(nodeWeb)
		if !ok {
			t.Fatalf("replica %s lost the shared node", name)
		}
		if n.X != collabMovedX || n.Y != collabMovedY {
			t.Fatalf("replica %s: move lost, node at (%d,%d) want (%d,%d)", name, n.X, n.Y, collabMovedX, collabMovedY)
		}
		if n.Color != collabRecolor {
			t.Fatalf("replica %s: recolour lost, colour %v want %v", name, n.Color, collabRecolor)
		}
		if _, ok := doc.Node("extra"); !ok {
			t.Fatalf("replica %s missing B's concurrently added node", name)
		}
	}

	pngA := renderPanel(s.diaA, s.theme)
	pngB := renderPanel(s.diaB, s.theme)
	if !bytes.Equal(pngA, pngB) {
		t.Fatal("converged replicas rendered different images")
	}
}

// --- animation proof (a phase step moves a pixel) -----------------------

// TestAnimationStepMovesPixels proves the live-animation path: the seed scene
// carries nodes with animated icons ("anim/*"), so a single AnimationStep both
// reports a repaint is needed AND changes the rendered pixels (the rest frame at
// phase 0 differs from the stepped frame). The phase-advance logic runs natively
// here; the browser rAF loop that drives it is the only wasm-tagged code.
func TestAnimationStepMovesPixels(t *testing.T) {
	s := newIsoScene()
	img0 := renderDiagram(t, s.diaA) // rest frame, phase 0

	if !s.AnimationStep(0.5) {
		t.Fatal("AnimationStep reported no repaint despite animated seed nodes")
	}
	img1 := renderDiagram(t, s.diaA) // stepped frame, phase != 0
	if bytes.Equal(img0.Pix, img1.Pix) {
		t.Fatal("animation advanced the phase but no pixel changed")
	}
}

// --- view-rotation proof (re-maps the plane AND keeps hit-testing exact) -

// TestRotationRemapsTileAndHitTestRoundTrips proves the view-rotation API on the
// active canvas: a clockwise turn changes the rendered plane, re-maps the grid
// tile under a FIXED screen pixel, and keeps hit-testing exact — a select-click
// at that same pixel resolves the tile the rotated drop landed on.
func TestRotationRemapsTileAndHitTestRoundTrips(t *testing.T) {
	// A pixel well inside Site A, over empty ground away from the seed nodes.
	px, py := isoPanelAX+330, isoDiaTop+430

	// A clockwise turn changes the rendered plane. Prove it on THROWAWAY scenes:
	// RenderPNG relocates the widget to the origin, so it must never touch a scene
	// we then drive by absolute surface coordinates.
	base := renderPanel(newIsoScene().diaA, toolkit.DefaultLight())
	turned := renderPanel(rotatedScene().diaA, toolkit.DefaultLight())
	if bytes.Equal(base, turned) {
		t.Fatal("RotateCW did not change the rendered plane")
	}

	// Unrotated: which tile does the drop land on under (px,py)?
	s0 := newIsoScene()
	n0 := dropIcon(s0, "box", px, py)
	if n0 == nil {
		t.Fatal("unrotated drop placed no node")
	}

	// Rotated one quarter clockwise on the active canvas (not rendered, so its
	// interactive bounds stay at the surface position).
	s1 := newIsoScene()
	s1.setActive(s1.diaA)
	s1.diaA.RotateCW()
	if s1.diaA.ViewRotation() != 1 {
		t.Fatalf("RotateCW gave view rotation %d, want 1", s1.diaA.ViewRotation())
	}
	n1 := dropIcon(s1, "box", px, py)
	if n1 == nil {
		t.Fatal("rotated drop placed no node")
	}
	if n0.X == n1.X && n0.Y == n1.Y {
		t.Fatalf("rotation did not remap the tile under a fixed pixel (both %d,%d)", n0.X, n0.Y)
	}

	// Hit-test round-trip on the rotated canvas: disarm placement, select-click
	// the SAME pixel, and the selection must resolve to a node on the exact tile
	// the rotated drop landed on.
	s1.mode.Set(toolkit.IsoModeSelect)
	s1.palette.SelectIcon("")
	s1.Click(px, py)
	sel := s1.diaA.Selected()
	if sel == "" {
		t.Fatal("select-click after rotation resolved no node")
	}
	selNode, _ := s1.docA.Node(sel)
	if selNode.X != n1.X || selNode.Y != n1.Y {
		t.Fatalf("rotated hit-test resolved tile (%d,%d), want (%d,%d)", selNode.X, selNode.Y, n1.X, n1.Y)
	}
}

// TestRotationIsLocalPerCanvas proves the two canvases rotate independently: a
// turn on Site A leaves Site B's view unrotated (rotation is LOCAL view state,
// never in the shared document).
func TestRotationIsLocalPerCanvas(t *testing.T) {
	s := newIsoScene()
	s.setActive(s.diaA)
	clickBtn(s, "rotcw")
	if s.diaA.ViewRotation() != 1 {
		t.Fatalf("Site A view rotation = %d, want 1", s.diaA.ViewRotation())
	}
	if s.diaB.ViewRotation() != 0 {
		t.Fatalf("Site B view rotation = %d, want 0 (rotation leaked across canvases)", s.diaB.ViewRotation())
	}
	// Turning Site A three more quarters wraps back to 0 via the observable.
	clickBtn(s, "rotccw")
	if s.diaA.ViewRotation() != 0 {
		t.Fatalf("Site A view rotation after CCW = %d, want 0", s.diaA.ViewRotation())
	}
}

// --- palette enrichment proof (anim + cloud-native + AWS all listed) ----

// TestPaletteListsEnrichedIcons proves the palette lists, beyond the built-in
// architecture icons, every animated icon and every cloud-native + AWS pack icon
// the scene registered — grouped by pack.
func TestPaletteListsEnrichedIcons(t *testing.T) {
	s := newIsoScene()
	listed := map[string]bool{}
	for _, e := range s.palette.Entries() {
		listed[e.ID] = true
	}
	check := func(kind string, ids []string) {
		if len(ids) == 0 {
			t.Fatalf("no %s ids to check — the pack is empty", kind)
		}
		for _, id := range ids {
			if !listed[id] {
				t.Errorf("palette does not list %s icon %q", kind, id)
			}
		}
	}
	check("animated", toolkit.IsoAnimatedIconIDs)
	check("cloud-native", isoicons.CloudNativeIDs())
	check("aws", isoicons.AWSIDs())

	var haveAnim, haveCN, haveAWS bool
	for _, g := range s.palette.Groups() {
		switch g.Name {
		case "anim":
			haveAnim = true
		case "cloudnative":
			haveCN = true
		case "aws":
			haveAWS = true
		}
	}
	if !haveAnim || !haveCN || !haveAWS {
		t.Fatalf("palette groups missing a pack heading (anim=%v cloudnative=%v aws=%v)", haveAnim, haveCN, haveAWS)
	}
}

// --- composite render + toolbar + routing coverage ----------------------

// TestDrawPaintsComposite renders the whole workspace and checks that the
// toolbar band, both canvases and the palette region each have non-background
// content — the composite is actually composed, not a blank fill.
func TestDrawPaintsComposite(t *testing.T) {
	s := newIsoScene()
	if w, h := s.Size(); w != isoSurfaceW || h != isoSurfaceH {
		t.Fatalf("Size() = %dx%d, want %dx%d", w, h, isoSurfaceW, isoSurfaceH)
	}
	buf := make([]byte, 4*isoSurfaceW*isoSurfaceH)
	s.Draw(buf)

	bg := std(s.theme.Background)
	nonBG := func(sx, sy int) bool {
		i := 4 * (sy*isoSurfaceW + sx)
		return stdcolor.RGBA{R: buf[i], G: buf[i+1], B: buf[i+2], A: buf[i+3]} != bg
	}
	for _, p := range []struct {
		name string
		x, y int
	}{
		{"toolbar", 40, isoToolbarH / 2},
		{"palette", isoPaletteX + isoPaletteW/2, isoPanelTop + 10},
		{"panel A", isoPanelAX + isoPanelW/2, isoDiaTop + panelDrawH/2},
		{"panel B", isoPanelBX + isoPanelW/2, isoDiaTop + panelDrawH/2},
	} {
		if !nonBG(p.x, p.y) {
			t.Errorf("%s region at (%d,%d) is bare background — not composed", p.name, p.x, p.y)
		}
	}
}

// TestToolbarButtonsAllFire clicks every toolbar button through the real event
// routing (surface click → ButtonGroup → Button → OnClick) and checks the
// observable effect of each, so every handler closure runs.
func TestToolbarButtonsAllFire(t *testing.T) {
	s := newIsoScene()

	clickBtn(s, "connect")
	if s.diaA.Mode != toolkit.IsoModeConnect {
		t.Fatal("Connect did not set connect mode")
	}
	clickBtn(s, "zone")
	if s.diaA.Mode != toolkit.IsoModeZone {
		t.Fatal("Zone did not set zone mode")
	}
	clickBtn(s, "text")
	if s.diaB.Mode != toolkit.IsoModeText {
		t.Fatal("Text did not set text mode on replica B")
	}
	clickBtn(s, "node")
	if s.palette.SelectedIcon().Get() != iconWeb {
		t.Fatalf("Node armed %q, want %q", s.palette.SelectedIcon().Get(), iconWeb)
	}
	clickBtn(s, "select")
	if s.diaA.Mode != toolkit.IsoModeSelect || s.palette.SelectedIcon().Get() != "" {
		t.Fatal("Select did not reset to select mode with no armed icon")
	}

	// Rotate CW then CCW on the active canvas (Site A) returns to orientation 0.
	s.setActive(s.diaA)
	clickBtn(s, "rotcw")
	if s.diaA.ViewRotation() != 1 {
		t.Fatalf("Rot CW gave rotation %d, want 1", s.diaA.ViewRotation())
	}
	clickBtn(s, "rotccw")
	if s.diaA.ViewRotation() != 0 {
		t.Fatalf("Rot CCW gave rotation %d, want 0", s.diaA.ViewRotation())
	}

	// Layer toggle hides then shows the monitor layer on both replicas.
	lA, _ := s.docA.Layer(layerMonitor)
	clickBtn(s, "layer")
	lA2, _ := s.docA.Layer(layerMonitor)
	lB2, _ := s.docB.Layer(layerMonitor)
	if lA2.Visible == lA.Visible {
		t.Fatal("Layer toggle did not flip monitor visibility")
	}
	if lA2.Visible != lB2.Visible {
		t.Fatal("Layer toggle left the replicas out of step")
	}

	// Zoom +/- change the projection scale (tile footprint).
	before := s.diaA.Projection().TileW
	clickBtn(s, "zoomin")
	if s.diaA.Projection().TileW <= before {
		t.Fatal("Zoom+ did not increase the scale")
	}
	clickBtn(s, "zoomout")

	// Delete on a selection removes it; place a node first (the drop selects it),
	// then delete.
	s.diaA.OnEvent(toolkit.Event{Kind: toolkit.EventDrop, X: 60, Y: 60, Code: toolkit.EncodeIsoIconPayload("box")})
	n := len(s.docA.Nodes())
	s.setActive(s.diaA)
	clickBtn(s, "delete")
	if len(s.docA.Nodes()) != n-1 {
		t.Fatalf("Delete removed %d nodes, want 1", n-len(s.docA.Nodes()))
	}

	// Undo / Redo round-trip the delete.
	clickBtn(s, "undo")
	if len(s.docA.Nodes()) != n {
		t.Fatal("Undo did not restore the deleted node")
	}
	clickBtn(s, "redo")
	if len(s.docA.Nodes()) != n-1 {
		t.Fatal("Redo did not re-apply the delete")
	}

	// Sync is a no-op-safe merge; just fire it.
	clickBtn(s, "sync")

	// Reset returns to the seeded scene (default node count, select mode).
	seeded := len(newIsoScene().docA.Nodes())
	clickBtn(s, "reset")
	if len(s.docA.Nodes()) != seeded || s.diaA.Mode != toolkit.IsoModeSelect {
		t.Fatalf("Reset did not restore the seed scene (%d nodes, mode %v)", len(s.docA.Nodes()), s.diaA.Mode)
	}
}

// TestEventRoutingCoverage exercises every capture branch of the pointer / key
// routing and the inert paths.
func TestEventRoutingCoverage(t *testing.T) {
	s := newIsoScene()

	// Click over Site B makes it active; a drag then release routes there.
	db := s.diaB.Bounds()
	if !s.Click(db.X+db.W/2, db.Y+db.H/2) {
		t.Fatal("click on Site B not handled")
	}
	if s.activeName() != "B" {
		t.Fatalf("active canvas = %s, want B", s.activeName())
	}
	s.Move(db.X+db.W/2+5, db.Y+db.H/2+5) // captured drag on B
	s.Release(db.X+db.W/2+5, db.Y+db.H/2+5)

	// Drag / release captured on Site A.
	da := s.diaA.Bounds()
	s.Click(da.X+da.W/2, da.Y+da.H/2)
	s.Move(da.X+da.W/2+5, da.Y+da.H/2+5)
	s.Release(da.X+da.W/2+5, da.Y+da.H/2+5)

	// Palette capture: click a body point (no icon armed), drag and release NOT
	// over a canvas — exercises the palette Move and the drop-skipped Release.
	pb := s.palette.Bounds()
	s.Click(pb.X+pb.W/2, pb.Y+pb.H/2)
	if !s.Move(pb.X+pb.W/2+2, pb.Y+pb.H/2+2) {
		t.Fatal("palette move not handled")
	}
	s.Release(pb.X+2, pb.Y+2)

	// Default Click branch: a point in no region (below the panels, above the
	// status line, between columns) captures nothing.
	if s.Click(400, isoStatusY-2) {
		t.Fatal("click in dead space should not be handled")
	}
	// Disarm first (the palette press above may have armed an icon): with nothing
	// in hand a hover over dead space repaints nothing.
	s.palette.SelectIcon("")
	if s.Move(401, isoStatusY-2) { // hover with nothing captured, nothing armed
		t.Fatal("hover with no capture and no armed icon should not repaint")
	}

	// Context over Site B opens its menu; over the toolbar it is ignored.
	if !s.Context(db.X+db.W/2, db.Y+db.H/2) {
		t.Fatal("right-click on a canvas should open the context menu")
	}
	if s.Context(40, isoToolbarH/2) {
		t.Fatal("right-click on the toolbar should be ignored")
	}

	// Char is inert; KeyDown routes to the active canvas.
	if s.Char("a") {
		t.Fatal("Char should be inert")
	}
	if !s.KeyDown("Delete") {
		t.Fatal("KeyDown should be handled")
	}
}

// --- PNG capture plumbing ------------------------------------------------

// TestGeneratePNGs writes the showcase images (into ISO_OUT when set, so a human
// can inspect them, else a temp dir), checks each is a decodable image of the
// expected size, and re-confirms the two converged replica PNGs are byte-
// identical on disk.
func TestGeneratePNGs(t *testing.T) {
	dir := os.Getenv("ISO_OUT")
	if dir == "" {
		dir = t.TempDir()
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	paths, err := generatePNGs(dir)
	if err != nil {
		t.Fatalf("generatePNGs: %v", err)
	}
	want := []struct {
		name string
		w, h int
	}{
		{"iso-editor.png", isoSurfaceW, isoSurfaceH},
		{"converged-a.png", panelW, panelDrawH},
		{"converged-b.png", panelW, panelDrawH},
		{"anim-frame.png", panelW, panelDrawH},
		{"rotated-a.png", panelW, panelDrawH},
		{"iso-ghost.png", isoSurfaceW, isoSurfaceH},
	}
	if len(paths) != len(want) {
		t.Fatalf("wrote %d files, want %d", len(paths), len(want))
	}
	for i, p := range paths {
		if filepath.Base(p) != want[i].name {
			t.Fatalf("path[%d] = %s, want base %s", i, p, want[i].name)
		}
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		cfg, _, err := image.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", p, err)
		}
		if cfg.Width != want[i].w || cfg.Height != want[i].h {
			t.Fatalf("%s is %dx%d, want %dx%d", p, cfg.Width, cfg.Height, want[i].w, want[i].h)
		}
	}
	repA, _ := os.ReadFile(paths[1])
	repB, _ := os.ReadFile(paths[2])
	if !bytes.Equal(repA, repB) {
		t.Fatal("converged replica PNG files differ on disk")
	}
	t.Logf("wrote %d showcase PNGs to %s", len(paths), dir)
}

// TestGeneratePNGsError checks generatePNGs surfaces the first write failure —
// here an output directory that does not exist.
func TestGeneratePNGsError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir")
	if _, err := generatePNGs(missing); err == nil {
		t.Fatal("generatePNGs into a missing dir: want error")
	}
}

// --- drag / placement ghost proof ---------------------------------------

// TestGhostFollowsPointerAndPaints is the ghost proof: arming an icon makes the
// preview follow the cursor EXACTLY (armed hover repaints and tracks the pointer,
// re-hitting the same pixel does not), and a frame with the ghost differs from one
// without it — so the translucent preview really paints under the pointer.
func TestGhostFollowsPointerAndPaints(t *testing.T) {
	s := newIsoScene()
	if s.ghost.Get().Active {
		t.Fatal("ghost active with nothing armed")
	}

	// Arm an icon (click-to-place style): a hover now drives the ghost.
	s.palette.SelectIcon("server")
	da := s.diaA.Bounds()
	tx, ty := da.X+120, da.Y+140
	if !s.Move(tx, ty) {
		t.Fatal("armed hover did not repaint the ghost")
	}
	if g := s.ghost.Get(); !g.Active || g.Icon != "server" || g.X != tx || g.Y != ty {
		t.Fatalf("ghost = %+v, want active server at (%d,%d)", g, tx, ty)
	}
	// Re-hitting the exact same pixel is a no-op repaint.
	if s.Move(tx, ty) {
		t.Fatal("re-hover to the same pixel should not repaint")
	}
	// The ghost follows to a new pixel.
	tx2, ty2 := tx+40, ty+25
	if !s.Move(tx2, ty2) {
		t.Fatal("ghost did not follow to a new pixel")
	}
	if g := s.ghost.Get(); g.X != tx2 || g.Y != ty2 {
		t.Fatalf("ghost at (%d,%d), want (%d,%d)", g.X, g.Y, tx2, ty2)
	}

	// A frame WITH the ghost differs from one without (only the ghost changed, the
	// document is untouched), so the preview is genuinely painted.
	withGhost := make([]byte, 4*s.w*s.h)
	s.Draw(withGhost)
	if !s.clearGhost() {
		t.Fatal("clearGhost reported no change for an active ghost")
	}
	noGhost := make([]byte, 4*s.w*s.h)
	s.Draw(noGhost)
	if bytes.Equal(withGhost, noGhost) {
		t.Fatal("the ghost frame is identical to the no-ghost frame — nothing painted")
	}
}

// TestGhostDisappearsOnDropAndCancel proves the ghost's exit paths: it vanishes
// when a drag-drop places a node AND when a grab is cancelled by releasing off any
// canvas (which places nothing).
func TestGhostDisappearsOnDropAndCancel(t *testing.T) {
	// Drop path: dropIcon arms, grabs from the palette header (ghost active) and
	// releases over Site A.
	s := newIsoScene()
	db := s.diaA.Bounds()
	p := s.diaA.Projection().Project(iso.V(4.5, 4.5, 0))
	n := dropIcon(s, "router", db.X+int(math.Round(p.X)), db.Y+int(math.Round(p.Y)))
	if n == nil {
		t.Fatal("drag-drop placed no node")
	}
	if s.ghost.Get().Active {
		t.Fatal("ghost still showing after a drop")
	}

	// Cancel path: grab, then release back on the palette (off every canvas).
	s2 := newIsoScene()
	s2.palette.SelectIcon("router")
	pb := s2.palette.Bounds()
	s2.Click(pb.X+10, pb.Y+2)
	if !s2.ghost.Get().Active {
		t.Fatal("grab did not arm the ghost")
	}
	before := len(s2.docA.Nodes())
	s2.Release(pb.X+10, pb.Y+40) // released on the palette, not a canvas
	if len(s2.docA.Nodes()) != before {
		t.Fatal("cancel off-canvas placed a node")
	}
	if s2.ghost.Get().Active {
		t.Fatal("ghost still showing after an off-canvas cancel")
	}
}

// TestEscapeCancelsGhost proves Escape drops an in-flight placement: the ghost
// disappears and the palette disarms.
func TestEscapeCancelsGhost(t *testing.T) {
	s := newIsoScene()
	s.palette.SelectIcon("server")
	da := s.diaA.Bounds()
	s.Move(da.X+100, da.Y+100)
	if !s.ghost.Get().Active {
		t.Fatal("ghost not armed before Escape")
	}
	if !s.KeyDown("Escape") {
		t.Fatal("Escape was not handled")
	}
	if s.ghost.Get().Active {
		t.Fatal("Escape did not cancel the ghost")
	}
	if s.palette.SelectedIcon().Get() != "" {
		t.Fatal("Escape did not disarm the palette")
	}
}

// TestGhostRendersSpriteAndComposites covers the two low-level ghost primitives
// deterministically: a SPRITE icon is blitted into the preview (only its opaque
// pixels), and the compositor blends the preview into the surface while clipping at
// every edge and skipping fully transparent pixels.
func TestGhostRendersSpriteAndComposites(t *testing.T) {
	// A 2×2 sprite: one opaque pixel, three transparent — so drawGhostSprite hits
	// both its copy and its transparent-skip paths.
	src := raster.New(2, 2)
	src.Set(0, 0, stdcolor.RGBA{R: 200, G: 30, B: 30, A: 255})
	icon := toolkit.IsoSpriteIcon{Img: src}
	img := renderGhostIcon(icon, isoGhostSize, stdcolor.RGBA{A: 255})
	if img.W != isoGhostSize || img.H != isoGhostSize {
		t.Fatalf("ghost image %dx%d, want %dx%d", img.W, img.H, isoGhostSize, isoGhostSize)
	}
	opaque := false
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] == 255 {
			opaque = true
			break
		}
	}
	if !opaque {
		t.Fatal("sprite ghost produced no opaque pixels")
	}

	// A 4×4 preview with one transparent pixel, composited three ways: clipped past
	// the top-left, clipped past the bottom-right, and wholly inside.
	ghost := raster.New(4, 4)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			ghost.Set(x, y, stdcolor.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	ghost.Set(0, 0, stdcolor.RGBA{}) // transparent → the a==0 skip
	const sw, sh = 10, 10
	buf := make([]byte, 4*sw*sh)
	for i := range buf {
		buf[i] = 255 // opaque white ground, so a blend is observable
	}
	compositeGhost(buf, sw, sh, ghost, 0, 0, 200)   // ox,oy negative → top/left clip
	compositeGhost(buf, sw, sh, ghost, sw, sh, 200) // dx,dy past the edge → bottom/right clip
	compositeGhost(buf, sw, sh, ghost, 5, 5, 200)   // wholly inside → the blend path
	changed := false
	for _, b := range buf {
		if b != 255 {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("compositeGhost changed no pixels")
	}
}

// --- full-page (resize) proof -------------------------------------------

// TestFullPageResizeRelayouts proves the [webcanvas.Resizer] hook: the surface
// grows to a larger viewport (the two canvases stretch to fill the width, side by
// side and equal), and a below-floor request is clamped to the minimum workable
// surface. This is the native stand-in for the browser resize wiring.
func TestFullPageResizeRelayouts(t *testing.T) {
	s := newIsoScene()
	if w, h := s.Size(); w != isoSurfaceW || h != isoSurfaceH {
		t.Fatalf("initial Size() = %dx%d, want %dx%d", w, h, isoSurfaceW, isoSurfaceH)
	}

	// Grow: the surface tracks the viewport and the canvases widen past the default.
	rw, rh := s.Resize(1600, 900)
	if rw != 1600 || rh != 900 {
		t.Fatalf("Resize returned %dx%d, want 1600x900", rw, rh)
	}
	if w, h := s.Size(); w != 1600 || h != 900 {
		t.Fatalf("Size() after grow = %dx%d, want 1600x900", w, h)
	}
	ba, bb := s.diaA.Bounds(), s.diaB.Bounds()
	if ba.W != bb.W {
		t.Fatalf("canvases unequal after resize: A.W=%d B.W=%d", ba.W, bb.W)
	}
	if ba.W <= isoPanelW {
		t.Fatalf("canvas width %d did not grow past the default %d", ba.W, isoPanelW)
	}
	if bb.X+bb.W > 1600 {
		t.Fatalf("Site B right edge %d overflows the 1600px surface", bb.X+bb.W)
	}
	if st := s.statusLabel.Bounds(); st.W != 1600-2*isoGap {
		t.Fatalf("status line width %d, want %d", st.W, 1600-2*isoGap)
	}
	// The canvases still bottom out above the status line.
	if ba.Y+ba.H >= s.statusLabel.Bounds().Y {
		t.Fatal("canvas overruns the status line after resize")
	}

	// Below the floor: clamped to the minimum workable surface.
	rw, rh = s.Resize(100, 100)
	if rw != isoMinSurfaceW || rh != isoMinSurfaceH {
		t.Fatalf("Resize(100,100) = %dx%d, want the floor %dx%d", rw, rh, isoMinSurfaceW, isoMinSurfaceH)
	}
	if s.diaA.Bounds().W <= 0 || s.diaA.Bounds().H <= 0 {
		t.Fatal("a canvas collapsed at the minimum surface")
	}
}
