// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

//go:build gallerynative

// Command gallery, built with -tags gallerynative, is the NATIVE-controls
// gallery: a go-widgets/window whose Surface publishes one descriptor per
// NativeKind, so the platform back-end (cocoa / win32 / gtk) shows every control
// as a real OS widget. It is the native mirror of the default wasm demo, kept
// behind a build tag so the wasm build and the coverage-gated native tests never
// pull in the windowing stack. On Linux, run with GO_WIDGETS_GTK=1 for the
// GTK4-hosted back-end.
//
// Build: go build -tags gallerynative -o gallery-native .
package main

import (
	"github.com/go-widgets/toolkit"
	"github.com/go-widgets/window"
)

func main() {
	buf := nativeBackground(nativeW, nativeH)
	surf := toolkit.NewSurface(func() ([]byte, int, int) { return buf, nativeW, nativeH })
	surf.Controls = nativeControls
	w, err := window.Open(window.Config{Title: "Native Widget Gallery", Width: nativeW, Height: nativeH})
	if err != nil {
		panic(err)
	}
	if err := w.Run(surf); err != nil {
		panic(err)
	}
}
