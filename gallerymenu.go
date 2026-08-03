// gallerymenu.go — the right-click edit menu. Kept tagless (like scene.go) so
// a native go test exercises it without the js && wasm build tag.
//
// handleContext hit-tests the interactive Wave-7 widgets (Kanban / Gantt /
// Agenda) via their exported CardAt / TaskAt / DayAt helpers, builds an edit
// Menu for the item under the cursor, and pops the shared ContextMenu at the
// click point. The menu actions mutate the widgets through their public API
// (Kanban.MoveCard, the exported Tasks/Columns slices, the MVVM event list).

package main

import "github.com/go-widgets/toolkit"

// handleContext opens the edit context menu for the Kanban card, Gantt task or
// month-view Agenda day under (x, y) (surface coords). It returns whether the
// scene changed (a menu opened, or an already-open menu was dismissed) so the
// host re-renders.
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
	if r := s.agenda.Bounds(); inside(x, y, r) && s.agenda.View == toolkit.AgendaMonth {
		if yy, m, d, ok := s.agenda.DayAt(x-r.X, y-r.Y); ok {
			s.ctxMenu.Menu = s.agendaMenu(yy, m, d)
			s.ctxMenu.Popup(x, y)
			return true
		}
	}
	// Nothing editable under the cursor: dismiss any open menu.
	wasOpen := s.ctxMenu.Open
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
				s.kanban.SelectedCol, s.kanban.SelectedCard = -1, -1
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
			s.gantt.Selected = -1
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
