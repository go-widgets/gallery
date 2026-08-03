// SPDX-License-Identifier: BSD-3-Clause
//
// gallerymenu_test — the right-click edit context menu: hit-testing, per-widget
// menu actions (with their guards), and dismissal / click routing.

package main

import (
	"testing"

	"github.com/go-widgets/toolkit"
)

// kanbanCardPoint returns a surface point inside card (col, i).
func kanbanCardPoint(s *state, col, i int) (int, int) {
	kr := s.kanban.Bounds()
	colW := (kr.W - 2*toolkit.KanbanColGap) / 3
	x := kr.X + col*(colW+toolkit.KanbanColGap) + toolkit.KanbanCardGap + 4
	y := kr.Y + toolkit.KanbanHeaderH + toolkit.KanbanCardGap + i*(toolkit.KanbanCardH+toolkit.KanbanCardGap) + 4
	return x, y
}

func TestContextMenuKanbanActions(t *testing.T) {
	s := newState(surfaceW, surfaceH)

	// Right-click "Design" (col 0, card 0) opens the menu.
	x, y := kanbanCardPoint(s, 0, 0)
	if !s.handleContext(x, y) || !s.ctxMenu.Open {
		t.Fatal("right-click on a card did not open the menu")
	}
	items := s.ctxMenu.Menu.Items

	// Move left on column 0 is a guarded no-op.
	items[0].Action()
	if len(s.kanban.Columns[0].Cards) != 2 {
		t.Fatalf("Move-left from col 0 mutated the board: %+v", s.kanban.Columns[0].Cards)
	}
	// Duplicate.
	items[3].Action()
	if len(s.kanban.Columns[0].Cards) != 3 || s.kanban.Columns[0].Cards[1].Title != "Design" {
		t.Fatalf("Duplicate: %+v", s.kanban.Columns[0].Cards)
	}
	// Move right → column 1.
	items[1].Action()
	if s.kanban.Columns[1].Cards[len(s.kanban.Columns[1].Cards)-1].Title != "Design" {
		t.Fatalf("Move-right did not append Design to col 1: %+v", s.kanban.Columns[1].Cards)
	}
	// Delete the (now duplicated) card at col 0, index 0.
	before := len(s.kanban.Columns[0].Cards)
	items[4].Action()
	if len(s.kanban.Columns[0].Cards) != before-1 {
		t.Fatalf("Delete did not remove a card: %d, want %d", len(s.kanban.Columns[0].Cards), before-1)
	}

	// Move right on the LAST column is a guarded no-op.
	lx, ly := kanbanCardPoint(s, 2, 0)
	s.handleContext(lx, ly)
	doneBefore := len(s.kanban.Columns[2].Cards)
	s.ctxMenu.Menu.Items[1].Action() // Move right →
	if len(s.kanban.Columns[2].Cards) != doneBefore {
		t.Fatalf("Move-right from last column mutated the board")
	}
}

func TestContextMenuGanttActions(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	gr := s.gantt.Bounds()
	rowY := gr.Y + toolkit.GanttHeaderH + 2 // task 0 "Design" Start 0 End 3
	if !s.handleContext(gr.X+gr.W/2, rowY) || !s.ctxMenu.Open {
		t.Fatal("right-click on a task did not open the menu")
	}
	it := s.ctxMenu.Menu.Items
	t0 := &s.gantt.Tasks[0]

	it[0].Action() // Extend
	if t0.End != 4 {
		t.Fatalf("Extend: End=%d, want 4", t0.End)
	}
	it[1].Action() // Shrink
	if t0.End != 3 {
		t.Fatalf("Shrink: End=%d, want 3", t0.End)
	}
	it[2].Action() // Move right
	if t0.Start != 1 || t0.End != 4 {
		t.Fatalf("Move-right: %+v", *t0)
	}
	it[3].Action() // Move left
	if t0.Start != 0 || t0.End != 3 {
		t.Fatalf("Move-left: %+v", *t0)
	}
	it[3].Action() // Move left again — guarded (Start already 0)
	if t0.Start != 0 {
		t.Fatalf("Move-left past 0: Start=%d", t0.Start)
	}
	// Shrink down to the min span, then once more (guarded).
	s.gantt.Tasks[0].Start, s.gantt.Tasks[0].End = 2, 3
	it[1].Action()
	if s.gantt.Tasks[0].End != 3 {
		t.Fatalf("Shrink at min span changed End: %d", s.gantt.Tasks[0].End)
	}
	// Delete.
	n := len(s.gantt.Tasks)
	it[5].Action()
	if len(s.gantt.Tasks) != n-1 {
		t.Fatalf("Delete: %d tasks, want %d", len(s.gantt.Tasks), n-1)
	}
}

func TestContextMenuAgendaAdd(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	ar := s.agenda.Bounds()
	first := toolkit.WeekdayOfFirst(2026, 7)
	idx := first + 15 // day 16
	row, col := idx/7, idx%7
	x := ar.X + col*ar.W/7 + (ar.W/7)/2
	yy := ar.Y + toolkit.AgendaHeaderH + row*toolkit.AgendaDayCellH + toolkit.AgendaDayCellH/2
	before := len(s.agenda.Events)
	if !s.handleContext(x, yy) || !s.ctxMenu.Open {
		t.Fatal("right-click on a day did not open the menu")
	}
	s.ctxMenu.Menu.Items[0].Action() // Add event here
	if len(s.agenda.Events) != before+1 {
		t.Fatalf("Add-event: %d events, want %d", len(s.agenda.Events), before+1)
	}
	if got := s.agenda.Events[len(s.agenda.Events)-1]; got.D != 16 {
		t.Fatalf("added event on day %d, want 16", got.D)
	}
}

// TestContextMenuDismissAndRouting covers handleContext's dismiss path and
// handleClick's ctxMenu-open branch.
func TestContextMenuDismissAndRouting(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	x, y := kanbanCardPoint(s, 0, 0)
	s.handleContext(x, y) // open

	// A left click inside the menu activates a row and closes it.
	mb := s.ctxMenu.MenuBounds()
	if !s.handleClick(mb.X+mb.W/2, mb.Y+toolkit.MenuRowH/2) {
		t.Fatal("handleClick inside menu returned false")
	}
	if s.ctxMenu.Open {
		t.Fatal("menu did not close after activating a row")
	}

	// Re-open, then a right-click on empty space dismisses it.
	s.handleContext(x, y)
	if !s.handleContext(surfaceW/2, toolkit.MenuBarH+2) { // empty scaffold area
		t.Fatal("dismiss of an open menu should report a change")
	}
	if s.ctxMenu.Open {
		t.Fatal("menu not dismissed by an empty-space right-click")
	}
	// Right-click empty space with no menu open is a no-op (false).
	if s.handleContext(surfaceW/2, toolkit.MenuBarH+2) {
		t.Fatal("dismiss with no menu open should report no change")
	}
}

// TestContextMenuKanbanMoveLeftAndGuards covers the actual Move-left (col>0) and
// the defensive out-of-range guards in Duplicate / Delete.
func TestContextMenuKanbanMoveLeftAndGuards(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	// Move left an actual card: col 1 "Build" → col 0.
	s.kanbanMenu(1, 0).Items[0].Action()
	if got := s.kanban.Columns[0].Cards[len(s.kanban.Columns[0].Cards)-1].Title; got != "Build" {
		t.Fatalf("Move-left did not append Build to col 0: last=%q", got)
	}
	// Out-of-range card: Duplicate + Delete are guarded no-ops.
	n0 := len(s.kanban.Columns[0].Cards)
	s.kanbanMenu(0, 99).Items[3].Action() // Duplicate
	s.kanbanMenu(0, 99).Items[4].Action() // Delete
	if len(s.kanban.Columns[0].Cards) != n0 {
		t.Fatalf("out-of-range guards mutated the board: %d, want %d", len(s.kanban.Columns[0].Cards), n0)
	}
}

// menuHasLabel reports whether menu m contains an item with the given label.
func menuHasLabel(m *toolkit.Menu, label string) bool {
	for _, it := range m.Items {
		if it.Label == label {
			return true
		}
	}
	return false
}

func TestContextMenuListActions(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	lr := s.listBox.Bounds()
	// Right-click item 0 opens the menu.
	if !s.handleContext(lr.X+5, lr.Y+2) || !s.ctxMenu.Open {
		t.Fatal("right-click on a list item did not open the menu")
	}

	first := s.listBox.Items[0]
	// Move down (item 0 -> 1).
	s.listMenu(s.listBox, 0).Items[1].Action()
	if s.listBox.Items[1] != first {
		t.Fatalf("Move-down did not swap: %v", s.listBox.Items[:2])
	}
	// Move up (item 1 -> 0) restores it.
	s.listMenu(s.listBox, 1).Items[0].Action()
	if s.listBox.Items[0] != first {
		t.Fatalf("Move-up did not swap back: %v", s.listBox.Items[:2])
	}
	// Guarded no-ops at the ends.
	n := len(s.listBox.Items)
	s.listMenu(s.listBox, 0).Items[0].Action()   // Move up at top
	s.listMenu(s.listBox, n-1).Items[1].Action() // Move down at bottom
	if len(s.listBox.Items) != n {
		t.Fatal("end guards mutated the list length")
	}
	// Duplicate then delete.
	s.listMenu(s.listBox, 0).Items[3].Action()
	if len(s.listBox.Items) != n+1 {
		t.Fatalf("Duplicate: %d, want %d", len(s.listBox.Items), n+1)
	}
	s.listMenu(s.listBox, 0).Items[4].Action()
	if len(s.listBox.Items) != n {
		t.Fatalf("Delete: %d, want %d", len(s.listBox.Items), n)
	}
	// Out-of-range index: every action is a guarded no-op.
	for _, k := range []int{0, 1, 3, 4} {
		s.listMenu(s.listBox, 999).Items[k].Action()
	}
	if len(s.listBox.Items) != n {
		t.Fatal("out-of-range list actions mutated the list")
	}
}

func TestContextMenuTableActions(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	tr := s.table.Bounds()
	if !s.handleContext(tr.X+5, tr.Y+toolkit.TableHeaderHeight+2) || !s.ctxMenu.Open {
		t.Fatal("right-click on a table row did not open the menu")
	}

	first := append([]string(nil), s.table.Rows[0]...)
	s.tableMenu(s.table, 0).Items[1].Action() // Move down
	if s.table.Rows[1][0] != first[0] {
		t.Fatalf("Move-down did not swap rows")
	}
	s.tableMenu(s.table, 1).Items[0].Action() // Move up
	if s.table.Rows[0][0] != first[0] {
		t.Fatalf("Move-up did not swap back")
	}
	n := len(s.table.Rows)
	s.tableMenu(s.table, 0).Items[0].Action()   // Move up at top (guard)
	s.tableMenu(s.table, n-1).Items[1].Action() // Move down at bottom (guard)
	s.tableMenu(s.table, 0).Items[3].Action()   // Duplicate
	if len(s.table.Rows) != n+1 {
		t.Fatalf("Duplicate: %d rows, want %d", len(s.table.Rows), n+1)
	}
	s.tableMenu(s.table, 0).Items[4].Action() // Delete
	if len(s.table.Rows) != n {
		t.Fatalf("Delete: %d rows, want %d", len(s.table.Rows), n)
	}
	for _, k := range []int{0, 1, 3, 4} {
		s.tableMenu(s.table, 999).Items[k].Action()
	}
	if len(s.table.Rows) != n {
		t.Fatal("out-of-range table actions mutated the rows")
	}
}

func TestContextMenuTreeActions(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	tr := s.tree.Bounds()
	if !s.handleContext(tr.X+40, tr.Y+2) || !s.ctxMenu.Open { // row 0 = Root
		t.Fatal("right-click on a tree node did not open the menu")
	}

	// Root menu: Add child + Toggle expand, but NO delete.
	rootMenu := s.treeMenu(s.tree, s.tree.Root)
	if !menuHasLabel(rootMenu, "Add child") || !menuHasLabel(rootMenu, "Toggle expand") {
		t.Fatal("root menu missing Add child / Toggle expand")
	}
	if menuHasLabel(rootMenu, "Delete node") {
		t.Fatal("root menu must not offer Delete node")
	}
	before := len(s.tree.Root.Children)
	rootMenu.Items[0].Action() // Add child
	if len(s.tree.Root.Children) != before+1 {
		t.Fatalf("Add child: %d, want %d", len(s.tree.Root.Children), before+1)
	}
	exp := s.tree.Root.Expanded
	rootMenu.Items[1].Action() // Toggle expand
	if s.tree.Root.Expanded == exp {
		t.Fatal("Toggle expand did not flip Root.Expanded")
	}

	// A freshly-added leaf (no children, not root): Add child + Delete, no Toggle.
	leaf := s.tree.Root.Children[len(s.tree.Root.Children)-1]
	leafMenu := s.treeMenu(s.tree, leaf)
	if menuHasLabel(leafMenu, "Toggle expand") {
		t.Fatal("leaf menu must not offer Toggle expand")
	}
	if !menuHasLabel(leafMenu, "Delete node") {
		t.Fatal("non-root leaf menu must offer Delete node")
	}
	s.tree.Selected = leaf
	// Delete node is the last item.
	leafMenu.Items[len(leafMenu.Items)-1].Action()
	if s.tree.Selected != nil {
		t.Fatal("deleting the selected node did not clear Selected")
	}
	for _, c := range s.tree.Root.Children {
		if c == leaf {
			t.Fatal("Delete node did not detach the leaf")
		}
	}
}

// TestContextMenuOtherInstances proves the generic dispatch reaches the SECOND
// ListBox (dataView) and Table (gridEdit) instances, not just the first.
func TestContextMenuOtherInstances(t *testing.T) {
	s := newState(surfaceW, surfaceH)

	dr := s.dataView.Bounds()
	if !s.handleContext(dr.X+5, dr.Y+2) || !s.ctxMenu.Open {
		t.Fatal("right-click on dataView (2nd ListBox) did not open a menu")
	}
	// Deleting through the menu mutates dataView, not listBox.
	n := len(s.dataView.Items)
	m := len(s.listBox.Items)
	s.listMenu(s.dataView, 0).Items[4].Action() // Delete
	if len(s.dataView.Items) != n-1 {
		t.Fatalf("dataView delete: %d, want %d", len(s.dataView.Items), n-1)
	}
	if len(s.listBox.Items) != m {
		t.Fatal("editing dataView must not touch listBox")
	}

	gr := s.gridEdit.Bounds()
	// gridEdit is grouped, so its first visual line is a group header (RowAt
	// == -1); scan down to the first real data-row line.
	gy, found := 0, false
	for line := 0; line < 10; line++ {
		yy := toolkit.TableHeaderHeight + line*toolkit.TableRowHeight + 2
		if s.gridEdit.RowAt(5, yy) >= 0 {
			gy, found = gr.Y+yy, true
			break
		}
	}
	if !found {
		t.Fatal("no data row found in gridEdit")
	}
	if !s.handleContext(gr.X+5, gy) || !s.ctxMenu.Open {
		t.Fatal("right-click on gridEdit (2nd Table) did not open a menu")
	}
	gn := len(s.gridEdit.Rows)
	s.tableMenu(s.gridEdit, 0).Items[4].Action() // Delete row
	if len(s.gridEdit.Rows) != gn-1 {
		t.Fatalf("gridEdit delete: %d, want %d", len(s.gridEdit.Rows), gn-1)
	}
}
