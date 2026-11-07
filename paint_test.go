// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"bytes"
	"testing"

	"github.com/go-widgets/painter"
)

// Paint, what webcanvas/gpu records, paints Draw's frame: through a
// PixelPainter onto a fresh buffer the two are byte for byte the same.
func TestPaintIsDraw(t *testing.T) {
	g := newGalleryApp()
	if g.s.theme.Background.A != 255 {
		t.Skip("a translucent background: Draw starts from fillBG's raw bytes")
	}
	w, h := g.Size()
	want := make([]byte, 4*w*h)
	g.Draw(want)
	got := make([]byte, 4*w*h)
	g.Paint(painter.NewPixelPainter(got, w, h))
	if !bytes.Equal(got, want) {
		for i := range want {
			if got[i] != want[i] {
				p := i / 4
				t.Fatalf("first difference at (%d,%d) channel %d: Draw %d, Paint %d", p%w, p/w, i%4, want[i], got[i])
			}
		}
	}
}
