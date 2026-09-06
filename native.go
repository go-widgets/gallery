// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// This file (tag-less, so it is compiled and 100%-tested natively) builds the
// NATIVE-controls page of the gallery: one descriptor per platform control the
// go-widgets native-control seam offers. A native host (go-widgets/window's
// cocoa / win32 / gtk back-end — see native_main.go) turns each into a REAL OS
// widget over the framebuffer, so the gallery shows the same widget set the wasm
// demo draws, but as the platform's own controls. The wasm page stays the drawn
// toolkit; this page is its native mirror.
package main

import "github.com/go-widgets/toolkit"

// nativeW, nativeH are the native gallery's logical window size.
const nativeW, nativeH = 780, 680

// galleryControl describes one showcased control by value; nativeControls turns
// it into a positioned descriptor. Data, not closures, so every field is covered
// by one pass.
type galleryControl struct {
	name  string
	kind  toolkit.NativeKind
	text  string
	on    bool
	num   float64
	min   float64
	max   float64
	items []string
}

// galleryControls is the showcased set: every NativeKind, in reading order.
var galleryControls = []galleryControl{
	{name: "Button", kind: toolkit.NativeButton, text: "Click me"},
	{name: "Label", kind: toolkit.NativeLabel, text: "A static label"},
	{name: "Entry", kind: toolkit.NativeEntry, text: "editable"},
	{name: "SecureEntry", kind: toolkit.NativeSecureEntry, text: "secret"},
	{name: "Checkbox", kind: toolkit.NativeCheckbox, text: "on", on: true},
	{name: "Radio", kind: toolkit.NativeRadio, text: "choice", on: true},
	{name: "Switch", kind: toolkit.NativeSwitch, on: true},
	{name: "Slider", kind: toolkit.NativeSlider, min: 0, max: 100, num: 50},
	{name: "PopUp", kind: toolkit.NativePopUp, items: []string{"One", "Two", "Three"}, text: "Two"},
	{name: "Progress", kind: toolkit.NativeProgress, min: 0, max: 100, num: 65},
	{name: "Spinner", kind: toolkit.NativeSpinner, on: true},
	{name: "Stepper", kind: toolkit.NativeStepper, min: 0, max: 10, num: 4},
	{name: "Search", kind: toolkit.NativeSearch, text: "search…"},
	{name: "Combo", kind: toolkit.NativeCombo, items: []string{"Alpha", "Beta"}, text: "Alpha"},
	{name: "Segmented", kind: toolkit.NativeSegmented, items: []string{"Day", "Week", "Month"}, text: "Week"},
	{name: "TextView", kind: toolkit.NativeTextView, text: "multi-line\ntext area"},
	{name: "Link", kind: toolkit.NativeLink, text: "a hyperlink"},
	{name: "Date", kind: toolkit.NativeDate, text: "2026-09-01"},
	{name: "Color", kind: toolkit.NativeColor, text: "#3366cc"},
}

// nativeControls lays the showcased controls out in two columns, each row a
// caption label beside its control, and returns the flat descriptor list a
// [toolkit.Surface] publishes through its Controls field.
func nativeControls() []toolkit.NativeControl {
	const colW, rowH, labelW, ctlW, ctlH, perCol = 380, 62, 110, 210, 30, 10
	out := make([]toolkit.NativeControl, 0, len(galleryControls)*2)
	for i, s := range galleryControls {
		x := 20 + (i/perCol)*colW
		y := 20 + (i%perCol)*rowH
		out = append(out, toolkit.NativeControl{
			Kind: toolkit.NativeLabel, Key: "cap:" + s.name, Visible: true,
			Text: s.name + ":", Rect: toolkit.Rect{X: x, Y: y + 4, W: labelW, H: ctlH},
		})
		out = append(out, toolkit.NativeControl{
			Kind: s.kind, Key: "ctl:" + s.name, Visible: true,
			Text: s.text, On: s.on, Number: s.num, Min: s.min, Max: s.max, Items: s.items,
			Rect: toolkit.Rect{X: x + labelW, Y: y, W: ctlW, H: ctlH},
		})
	}
	return out
}

// nativeBackground is the gallery's flat ground, an opaque light RGBA buffer the
// native controls sit over.
func nativeBackground(w, h int) []byte {
	buf := make([]byte, w*h*4)
	for i := 0; i < len(buf); i += 4 {
		buf[i], buf[i+1], buf[i+2], buf[i+3] = 0xf3, 0xf3, 0xf5, 0xff
	}
	return buf
}
