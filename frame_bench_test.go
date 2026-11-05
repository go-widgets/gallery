// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import "testing"

// BenchmarkGalleryFrame repaints the whole gallery, every widget family, into
// one RGBA buffer: the CPU cost of a full frame on today's renderer, which is
// what a GPU painter would have to beat.
func BenchmarkGalleryFrame(b *testing.B) {
	s := newState(surfaceW, surfaceH)
	buf := make([]byte, 4*s.w*s.h)
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	b.ReportMetric(float64(s.w*s.h), "px")
	for b.Loop() {
		s.draw(buf)
	}
}
