// SPDX-License-Identifier: BSD-3-Clause
//
// viewmodel_test — asserts the MVVM bindings are genuinely two-way and that
// the ObservableList mirrors into the Agenda widget.

package main

import (
	"testing"

	"github.com/go-widgets/mvvm"
	"github.com/go-widgets/toolkit"
)

// TestBindTwoWay covers the local two-way Observable bridge: seed, both
// directions, and the unbind that detaches them.
func TestBindTwoWay(t *testing.T) {
	a := mvvm.NewObservable(0)
	b := mvvm.NewObservable(9)
	unbind := bindTwoWay(a, b)
	if b.Get() != 0 {
		t.Fatalf("seed: b=%d, want a's 0", b.Get())
	}
	a.Set(3)
	if b.Get() != 3 {
		t.Fatalf("a→b: b=%d", b.Get())
	}
	b.Set(5)
	if a.Get() != 5 {
		t.Fatalf("b→a: a=%d", a.Get())
	}
	unbind()
	a.Set(7)
	if b.Get() != 5 {
		t.Fatal("unbind: a→b must stop")
	}
	b.Set(8)
	if a.Get() != 7 {
		t.Fatal("unbind: b→a must stop")
	}
}

// TestViewModelThemeBindingTwoWay: setting vm.themeIndex programmatically must
// drive both the switcher's Current and the applied theme (the VM->widget
// direction the click tests don't exercise).
func TestViewModelThemeBindingTwoWay(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	for i := range s.themeNames {
		s.vm.themeIndex.Set(i)
		if s.themeSwitcher.Current().Get() != i {
			t.Fatalf("themeIndex.Set(%d): switcher.Current=%d", i, s.themeSwitcher.Current().Get())
		}
		if s.theme != s.themes[i] {
			t.Fatalf("themeIndex.Set(%d) did not apply theme %q", i, s.themeNames[i])
		}
	}
}

// TestViewModelAgendaViewBindingTwoWay: setting vm.agendaView drives the
// switcher Current and the Agenda's View.
func TestViewModelAgendaViewBindingTwoWay(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	for _, v := range []toolkit.AgendaView{toolkit.AgendaWeek, toolkit.AgendaMonth, toolkit.AgendaQuarter, toolkit.AgendaYear} {
		s.vm.agendaView.Set(int(v))
		if s.agendaSwitcher.Current().Get() != int(v) {
			t.Fatalf("agendaView.Set(%d): switcher.Current=%d", v, s.agendaSwitcher.Current().Get())
		}
		if s.agenda.View().Get() != v {
			t.Fatalf("agendaView.Set(%d): agenda.View=%d", v, s.agenda.View().Get())
		}
	}
}

// TestViewModelEventsListMirrors: appending to the events ObservableList
// mirrors into the Agenda widget's slice.
func TestViewModelEventsListMirrors(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	before := len(s.agenda.Events)
	s.vm.addEvent("VM", 2026, 7, 18, toolkit.RGB(1, 2, 3))
	if s.vm.events.Len() != before+1 {
		t.Fatalf("list length = %d, want %d", s.vm.events.Len(), before+1)
	}
	if len(s.agenda.Events) != before+1 {
		t.Fatalf("agenda.Events not mirrored: %d, want %d", len(s.agenda.Events), before+1)
	}
	if got := s.agenda.Events[len(s.agenda.Events)-1]; got.Title != "VM" || got.D != 18 {
		t.Fatalf("mirrored event = %+v, want Title=VM D=18", got)
	}
}

// TestViewModelPaletteCommand: executing the bound Command opens the palette,
// and the palette Button routes through it.
func TestViewModelPaletteCommand(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	s.vm.paletteCmd.Execute()
	if !s.cmdPalette.Visible().Get() {
		t.Fatal("paletteCmd.Execute did not open the CommandPalette")
	}
	s.cmdPalette.Dismiss()
	// The button's OnClick was rebound to the command by BindCommand.
	s.paletteBtn.OnClick()
	if !s.cmdPalette.Visible().Get() {
		t.Fatal("paletteBtn.OnClick (bound to the Command) did not open the palette")
	}
}
