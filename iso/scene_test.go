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
// routing (surface click → HBox → Button → OnClick) and checks the observable
// effect of each, so every handler closure runs.
func TestToolbarButtonsAllFire(t *testing.T) {
	s := newIsoScene()

	click := func(b *toolkit.Button) {
		r := b.Bounds()
		s.Click(r.X+r.W/2, r.Y+r.H/2)
		s.Release(r.X+r.W/2, r.Y+r.H/2)
	}
	byLabel := map[string]*toolkit.Button{}
	for _, b := range s.buttons {
		byLabel[b.Label] = b
	}

	click(byLabel["Connect"])
	if s.diaA.Mode != toolkit.IsoModeConnect {
		t.Fatal("Connect did not set connect mode")
	}
	click(byLabel["Zone"])
	if s.diaA.Mode != toolkit.IsoModeZone {
		t.Fatal("Zone did not set zone mode")
	}
	click(byLabel["Text"])
	if s.diaB.Mode != toolkit.IsoModeText {
		t.Fatal("Text did not set text mode on replica B")
	}
	click(byLabel["Node"])
	if s.palette.SelectedIcon().Get() != "server" {
		t.Fatal("Node did not arm the server icon")
	}
	click(byLabel["Select"])
	if s.diaA.Mode != toolkit.IsoModeSelect || s.palette.SelectedIcon().Get() != "" {
		t.Fatal("Select did not reset to select mode with no armed icon")
	}

	// Layer toggle hides then shows the monitor layer on both replicas.
	lA, _ := s.docA.Layer(layerMonitor)
	click(byLabel["Layer"])
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
	click(byLabel["Zoom+"])
	if s.diaA.Projection().TileW <= before {
		t.Fatal("Zoom+ did not increase the scale")
	}
	click(byLabel["Zoom-"])

	// Delete on a selection removes it; place a node first, select it, delete.
	s.diaA.OnEvent(toolkit.Event{Kind: toolkit.EventDrop, X: 60, Y: 60, Code: toolkit.EncodeIsoIconPayload("box")})
	n := len(s.docA.Nodes())
	s.setActive(s.diaA)
	click(byLabel["Delete"])
	if len(s.docA.Nodes()) != n-1 {
		t.Fatalf("Delete removed %d nodes, want 1", n-len(s.docA.Nodes()))
	}

	// Undo / Redo round-trip the delete.
	click(byLabel["Undo"])
	if len(s.docA.Nodes()) != n {
		t.Fatal("Undo did not restore the deleted node")
	}
	click(byLabel["Redo"])
	if len(s.docA.Nodes()) != n-1 {
		t.Fatal("Redo did not re-apply the delete")
	}

	// Sync is a no-op-safe merge; just fire it.
	click(byLabel["Sync"])

	// Reset returns to the seeded scene (default node count, select mode).
	seeded := len(newIsoScene().docA.Nodes())
	click(byLabel["Reset"])
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
	if s.Move(401, isoStatusY-2) { // hover with nothing captured
		t.Fatal("hover with no capture should not repaint")
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
