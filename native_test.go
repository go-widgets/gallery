// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"testing"

	"github.com/go-widgets/toolkit"
)

// TestNativeControls checks the native gallery publishes a caption + a control
// for every showcased kind, positioned and carrying its value.
func TestNativeControls(t *testing.T) {
	got := nativeControls()
	if want := len(galleryControls) * 2; len(got) != want {
		t.Fatalf("descriptor count = %d, want %d (a label + a control each)", len(got), want)
	}
	byKey := map[string]toolkit.NativeControl{}
	for _, c := range got {
		byKey[c.Key] = c
		if !c.Visible || c.Rect == (toolkit.Rect{}) {
			t.Errorf("%s: not visible or unpositioned (%+v)", c.Key, c.Rect)
		}
	}
	// Every kind is present as a control, its caption as a label.
	for _, s := range galleryControls {
		ctl, ok := byKey["ctl:"+s.name]
		if !ok || ctl.Kind != s.kind {
			t.Errorf("control %q missing or wrong kind (%v, want %v)", s.name, ctl.Kind, s.kind)
		}
		cap, ok := byKey["cap:"+s.name]
		if !ok || cap.Kind != toolkit.NativeLabel {
			t.Errorf("caption for %q missing or not a label", s.name)
		}
	}
	// Spot-check a value carried through.
	if byKey["ctl:Slider"].Number != 50 {
		t.Errorf("slider Number = %v, want 50", byKey["ctl:Slider"].Number)
	}
	if byKey["ctl:PopUp"].Text != "Two" {
		t.Errorf("pop-up Text = %q, want Two", byKey["ctl:PopUp"].Text)
	}
}

// TestNativeBackground checks the ground buffer is the right size and opaque.
func TestNativeBackground(t *testing.T) {
	buf := nativeBackground(4, 3)
	if len(buf) != 4*3*4 {
		t.Fatalf("background len = %d, want %d", len(buf), 4*3*4)
	}
	if buf[3] != 0xff {
		t.Errorf("background not opaque: alpha = %d", buf[3])
	}
}
