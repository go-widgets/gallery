// viewmodel.go — the gallery's MVVM layer. Kept tagless (like scene.go) so a
// native go test exercises it without the js && wasm build tag.
//
// Rather than mutate scene state directly in each widget callback, the
// interactive widgets bind to the bindables below (go-widgets/mvvm
// Observables, a Command and an ObservableList). Subscriptions apply the
// effects, so the gallery genuinely demonstrates the MVVM pattern: the theme
// and Agenda-view switchers edit Observables, the "add event" gesture appends
// to an ObservableList that mirrors into the widget, and the CommandPalette
// button runs a Command.

package main

import (
	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// neverEq is an Observable equality that always reports "different", so every
// Set notifies. It preserves the pre-MVVM behaviour where re-selecting the
// current switcher segment still re-applies + re-announces it.
func neverEq(a, b int) bool { return false }

// viewModel holds the gallery's interactive state as MVVM bindables.
type viewModel struct {
	// themeIndex two-way binds the theme ViewSwitcher; a subscription installs
	// the matching palette. agendaView likewise drives the Agenda's view.
	themeIndex *mvvm.Observable[int]
	agendaView *mvvm.Observable[int]
	// events is the Agenda's event list as an ObservableList; a subscription
	// mirrors it into the widget, so appending an event repaints the calendar.
	events *mvvm.ObservableList[toolkit.AgendaEvent]
	// paletteCmd opens the CommandPalette, bound to the palette Button.
	paletteCmd *mvvm.Command

	// unbind detaches every binding/subscription (kept for symmetry with a
	// real app teardown; the gallery lives for the page's lifetime).
	unbind []func()
}

// newViewModel builds the bindables and wires them to s's already-constructed
// widgets. Called from newState once every widget exists.
func newViewModel(s *state) *viewModel {
	vm := &viewModel{
		themeIndex: mvvm.NewObservableEq(0, neverEq),
		agendaView: mvvm.NewObservableEq(int(toolkit.AgendaMonth), neverEq),
		events:     mvvm.NewObservableList(s.agenda.Events...),
	}

	// Theme: the switcher edits themeIndex; the subscription applies the palette
	// and announces it. BindField makes OnChange call Set and keeps Current in
	// sync when themeIndex is set programmatically.
	vm.themeIndex.Subscribe(func(i int) {
		s.theme = s.themes[i]
		s.showNotify("Theme: " + s.themeNames[i])
	})
	vm.unbind = append(vm.unbind, mvvm.BindField(vm.themeIndex, &s.themeSwitcher.Current, &s.themeSwitcher.OnChange, nil))

	// Agenda view: the switcher edits agendaView; the subscription sets the view.
	vm.agendaView.Subscribe(func(i int) {
		s.agenda.View = toolkit.AgendaView(i)
		s.showNotify("Agenda view: " + s.agendaSwitcher.Views[i])
	})
	vm.unbind = append(vm.unbind, mvvm.BindField(vm.agendaView, &s.agendaSwitcher.Current, &s.agendaSwitcher.OnChange, nil))

	// Events: mirror the ObservableList into the widget on every change, so a
	// day-activate that appends an event repaints the Agenda.
	vm.unbind = append(vm.unbind, vm.events.SubscribeChanged(func() {
		s.agenda.Events = vm.events.Slice()
	}))

	// CommandPalette: route the trigger Button through a Command.
	vm.paletteCmd = mvvm.NewCommand(func() { s.cmdPalette.Open() }, nil)
	vm.unbind = append(vm.unbind, mvvm.BindCommand(vm.paletteCmd, &s.paletteBtn.OnClick, nil))

	return vm
}

// addEvent appends a new event on (y, m, d) through the ObservableList, whose
// subscription mirrors it into the Agenda widget. It is the MVVM path the
// Agenda's OnDayActivate uses instead of mutating the widget slice directly.
func (vm *viewModel) addEvent(title string, y, m, d int, fill toolkit.RGBA) {
	vm.events.Append(toolkit.AgendaEvent{Title: title, Y: y, M: m, D: d, Fill: fill})
}
