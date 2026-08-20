// gallerymenu.go — the right-click edit menu. Kept tagless (like scene.go) so
// a native go test exercises it without the js && wasm build tag.
//
// handleContext hit-tests every editable widget — Kanban / Gantt / Agenda and
// the data widgets ListBox / Table / TreeView / TreeTable / PropertyGrid — via
// their exported hit helpers (CardAt / TaskAt / DayAt / IndexAt / RowAt /
// NodeAt), builds an edit Menu for the item under the cursor, and pops the
// shared ContextMenu at the click point. The menu actions mutate the widgets
// through their public API (Kanban.MoveCard, TreeView/TreeTable.Remove,
// PropertyGrid.RemoveAt, the exported Tasks/Columns/Rows/Items slices, the MVVM
// event list).

package main

import "github.com/go-widgets/toolkit"

// handleContext opens the edit context menu for whatever editable item is under
// (x, y) (surface coords): a Kanban card, Gantt task, month-view Agenda day, or
// a row/item/node of any ListBox / Table / TreeView in the scene. It returns
// whether the scene changed (a menu opened, or an already-open menu was
// dismissed) so the host re-renders.
func (s *state) handleContext(x, y int) bool {
	if r := s.kanban.Bounds(); inside(x, y, r) {
		if col, card := s.kanban.CardAt(x-r.X, y-r.Y); col >= 0 {
			s.ctxMenu.Menu = s.kanbanMenu(col, card)
			s.ctxMenu.Popup(x, y)
			return true
		}
	}
	if r := s.gantt.Bounds(); inside(x, y, r) {
		if row := s.gantt.TaskAt(x-r.X, y-r.Y); row >= 0 {
			s.ctxMenu.Menu = s.ganttMenu(row)
			s.ctxMenu.Popup(x, y)
			return true
		}
	}
	if r := s.agenda.Bounds(); inside(x, y, r) && s.agenda.View().Get() == toolkit.AgendaMonth {
		if yy, m, d, ok := s.agenda.DayAt(x-r.X, y-r.Y); ok {
			s.ctxMenu.Menu = s.agendaMenu(yy, m, d)
			s.ctxMenu.Popup(x, y)
			return true
		}
	}
	// The data widgets share one generic path, so every ListBox / Table /
	// TreeView instance in the gallery (not just one of each) gets the same
	// edit menu.
	for _, lb := range []*toolkit.ListBox{s.listBox, s.dataView} {
		if r := lb.Bounds(); inside(x, y, r) {
			if i := lb.IndexAt(x-r.X, y-r.Y); i >= 0 {
				s.ctxMenu.Menu = s.listMenu(lb, i)
				s.ctxMenu.Popup(x, y)
				return true
			}
		}
	}
	for _, tb := range []*toolkit.Table{s.table, s.gridEdit} {
		if r := tb.Bounds(); inside(x, y, r) {
			if row := tb.RowAt(x-r.X, y-r.Y); row >= 0 {
				s.ctxMenu.Menu = s.tableMenu(tb, row)
				s.ctxMenu.Popup(x, y)
				return true
			}
		}
	}
	for _, tv := range []*toolkit.TreeView{s.tree} {
		if r := tv.Bounds(); inside(x, y, r) {
			if node := tv.NodeAt(x-r.X, y-r.Y); node != nil {
				s.ctxMenu.Menu = s.treeMenu(tv, node)
				s.ctxMenu.Popup(x, y)
				return true
			}
		}
	}
	if r := s.treeTable.Bounds(); inside(x, y, r) {
		if node := s.treeTable.NodeAt(x-r.X, y-r.Y); node != nil {
			s.ctxMenu.Menu = s.treeTableMenu(node)
			s.ctxMenu.Popup(x, y)
			return true
		}
	}
	if r := s.propGrid.Bounds(); inside(x, y, r) {
		if row := s.propGrid.Table().RowAt(x-r.X, y-r.Y); row >= 0 {
			s.ctxMenu.Menu = s.propGridMenu(row)
			s.ctxMenu.Popup(x, y)
			return true
		}
	}
	// Nothing editable under the cursor: dismiss any open menu.
	wasOpen := s.ctxMenu.Open().Get()
	s.ctxMenu.Close()
	return wasOpen
}

// kanbanMenu builds the edit menu for card (col, card): move it between the
// adjacent columns, duplicate it, or delete it.
func (s *state) kanbanMenu(col, card int) *toolkit.Menu {
	last := len(s.kanban.Columns) - 1
	return toolkit.NewMenu([]toolkit.MenuItem{
		{Label: "Move left", Action: func() {
			if col > 0 {
				s.kanban.MoveCard(col, card, col-1, len(s.kanban.Columns[col-1].Cards))
				s.showNotify("Moved card to " + s.kanban.Columns[col-1].Title)
			}
		}},
		{Label: "Move right", Action: func() {
			if col < last {
				s.kanban.MoveCard(col, card, col+1, len(s.kanban.Columns[col+1].Cards))
				s.showNotify("Moved card to " + s.kanban.Columns[col+1].Title)
			}
		}},
		{Separator: true},
		{Label: "Duplicate", Action: func() {
			cards := s.kanban.Columns[col].Cards
			if card < len(cards) {
				dup := cards[card]
				cards = append(cards, toolkit.KanbanCard{})
				copy(cards[card+2:], cards[card+1:])
				cards[card+1] = dup
				s.kanban.Columns[col].Cards = cards
				s.showNotify("Card duplicated")
			}
		}},
		{Label: "Delete card", Action: func() {
			cards := s.kanban.Columns[col].Cards
			if card < len(cards) {
				s.kanban.Columns[col].Cards = append(cards[:card], cards[card+1:]...)
				s.kanban.SelectedCol().Set(-1)
				s.kanban.SelectedCard().Set(-1)
				s.showNotify("Card deleted")
			}
		}},
	})
}

// ganttMenu builds the edit menu for task row: extend/shrink its end, shift it,
// or delete it. Every action keeps a valid [Start, End) with a min span of 1.
func (s *state) ganttMenu(row int) *toolkit.Menu {
	tk := func() *toolkit.GanttTask { return &s.gantt.Tasks[row] }
	return toolkit.NewMenu([]toolkit.MenuItem{
		{Label: "Extend", Action: func() {
			tk().End++
			s.showNotify(tk().Label + " extended")
		}},
		{Label: "Shrink", Action: func() {
			if t := tk(); t.End-t.Start > 1 {
				t.End--
				s.showNotify(t.Label + " shrunk")
			}
		}},
		{Label: "Move right", Action: func() {
			t := tk()
			t.Start++
			t.End++
			s.showNotify(t.Label + " moved right")
		}},
		{Label: "Move left", Action: func() {
			if t := tk(); t.Start > 0 {
				t.Start--
				t.End--
				s.showNotify(t.Label + " moved left")
			}
		}},
		{Separator: true},
		{Label: "Delete task", Action: func() {
			label := tk().Label
			s.gantt.Tasks = append(s.gantt.Tasks[:row], s.gantt.Tasks[row+1:]...)
			s.gantt.Selected().Set(-1)
			s.showNotify(label + " deleted")
		}},
	})
}

// agendaMenu builds the edit menu for month-view day (y, m, d): add an event
// there, routed through the MVVM event list.
func (s *state) agendaMenu(y, m, d int) *toolkit.Menu {
	return toolkit.NewMenu([]toolkit.MenuItem{
		{Label: "Add event here", Action: func() {
			s.addedEvents++
			s.vm.addEvent("New "+itoa(s.addedEvents), y, m, d, toolkit.RGB(0x7a, 0x5a, 0xf0))
			s.showNotify("Added event on " + itoa(m) + "/" + itoa(d))
		}},
	})
}

// listMenu builds the edit menu for item i of ListBox lb: reorder it, duplicate
// it, or delete it. Selected is kept valid after the edit. lb is a parameter so
// the same menu serves every ListBox in the scene.
func (s *state) listMenu(lb *toolkit.ListBox, i int) *toolkit.Menu {
	items := func() []string { return lb.Items }
	return toolkit.NewMenu([]toolkit.MenuItem{
		{Label: "Move up", Action: func() {
			if it := items(); i > 0 && i < len(it) {
				it[i-1], it[i] = it[i], it[i-1]
				lb.Selected().Set(i - 1)
				s.showNotify("Item moved up")
			}
		}},
		{Label: "Move down", Action: func() {
			if it := items(); i >= 0 && i < len(it)-1 {
				it[i+1], it[i] = it[i], it[i+1]
				lb.Selected().Set(i + 1)
				s.showNotify("Item moved down")
			}
		}},
		{Separator: true},
		{Label: "Duplicate", Action: func() {
			if it := items(); i >= 0 && i < len(it) {
				it = append(it, "")
				copy(it[i+2:], it[i+1:])
				it[i+1] = it[i]
				lb.Items = it
				s.showNotify("Item duplicated")
			}
		}},
		{Label: "Delete", Action: func() {
			if it := items(); i >= 0 && i < len(it) {
				lb.Items = append(it[:i], it[i+1:]...)
				lb.Selected().Set(-1)
				s.showNotify("Item deleted")
			}
		}},
	})
}

// tableMenu builds the edit menu for row of Table tb: reorder, duplicate or
// delete it. tb is a parameter so the same menu serves every Table in the scene.
func (s *state) tableMenu(tb *toolkit.Table, row int) *toolkit.Menu {
	rows := func() [][]string { return tb.Rows }
	return toolkit.NewMenu([]toolkit.MenuItem{
		{Label: "Move up", Action: func() {
			if r := rows(); row > 0 && row < len(r) {
				r[row-1], r[row] = r[row], r[row-1]
				s.showNotify("Row moved up")
			}
		}},
		{Label: "Move down", Action: func() {
			if r := rows(); row >= 0 && row < len(r)-1 {
				r[row+1], r[row] = r[row], r[row+1]
				s.showNotify("Row moved down")
			}
		}},
		{Separator: true},
		{Label: "Duplicate row", Action: func() {
			if r := rows(); row >= 0 && row < len(r) {
				dup := append([]string(nil), r[row]...)
				r = append(r, nil)
				copy(r[row+2:], r[row+1:])
				r[row+1] = dup
				tb.Rows = r
				s.showNotify("Row duplicated")
			}
		}},
		{Label: "Delete row", Action: func() {
			if r := rows(); row >= 0 && row < len(r) {
				tb.Rows = append(r[:row], r[row+1:]...)
				s.showNotify("Row deleted")
			}
		}},
	})
}

// treeMenu builds the edit menu for node of TreeView tv (a parameter so it
// serves every TreeView): add a child, toggle its
// expansion (only when it has children), or delete it (never the root).
func (s *state) treeMenu(tv *toolkit.TreeView, node *toolkit.TreeNode) *toolkit.Menu {
	items := []toolkit.MenuItem{
		{Label: "Add child", Action: func() {
			node.Children = append(node.Children, &toolkit.TreeNode{Label: "child " + itoa(len(node.Children)+1)})
			node.Expanded = true
			s.showNotify("Child added to " + node.Label)
		}},
	}
	if len(node.Children) > 0 {
		items = append(items, toolkit.MenuItem{Label: "Toggle expand", Action: func() {
			node.Expanded = !node.Expanded
			s.showNotify("Toggled " + node.Label)
		}})
	}
	if node != tv.Root {
		items = append(items, toolkit.MenuItem{Separator: true})
		items = append(items, toolkit.MenuItem{Label: "Delete node", Action: func() {
			if tv.Remove(node) {
				if tv.Selected().Get() == node {
					tv.Selected().Set(nil)
				}
				s.showNotify("Node deleted")
			}
		}})
	}
	return toolkit.NewMenu(items)
}

// treeTableMenu builds the edit menu for TreeTable node: add a child, toggle its
// expansion (only when it has children), or delete it (via TreeTable.Remove).
func (s *state) treeTableMenu(node *toolkit.TreeTableNode) *toolkit.Menu {
	items := []toolkit.MenuItem{
		{Label: "Add child", Action: func() {
			cells := make([]string, len(node.Cells))
			if len(cells) > 0 {
				cells[0] = "child " + itoa(len(node.Children)+1)
			}
			for i := 1; i < len(cells); i++ {
				cells[i] = "leaf"
			}
			node.Children = append(node.Children, &toolkit.TreeTableNode{Cells: cells})
			node.Expanded = true
			s.showNotify("Row added")
		}},
	}
	if len(node.Children) > 0 {
		items = append(items, toolkit.MenuItem{Label: "Toggle expand", Action: func() {
			node.Expanded = !node.Expanded
			s.showNotify("Toggled row")
		}})
	}
	items = append(items, toolkit.MenuItem{Separator: true})
	items = append(items, toolkit.MenuItem{Label: "Delete node", Action: func() {
		if s.treeTable.Remove(node) {
			if s.treeTable.Selected().Get() == node {
				s.treeTable.Selected().Set(nil)
			}
			s.showNotify("Row deleted")
		}
	}})
	return toolkit.NewMenu(items)
}

// propGridMenu builds the edit menu for PropertyGrid row: duplicate the property
// or delete it (via PropertyGrid.RemoveAt).
func (s *state) propGridMenu(row int) *toolkit.Menu {
	rows := s.propGrid.Table().Rows
	name := ""
	if row >= 0 && row < len(rows) {
		name = rows[row][0]
	}
	return toolkit.NewMenu([]toolkit.MenuItem{
		{Label: "Duplicate", Action: func() {
			if r := s.propGrid.Table().Rows; row >= 0 && row < len(r) {
				s.propGrid.Add(r[row][0]+" copy", r[row][1])
				s.showNotify("Property duplicated")
			}
		}},
		{Separator: true},
		{Label: "Delete property", Action: func() {
			s.propGrid.RemoveAt(row)
			s.showNotify("Deleted " + name)
		}},
	})
}
