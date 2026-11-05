// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/go-gfx/gpu2d"
	gpumetal "github.com/go-gfx/gpu2d/metal"
	"github.com/go-widgets/gpupainter"
	mtl "github.com/go-macos/metal"
)

// gpuFrame records the whole gallery for the GPU: the background, then every
// widget through gpupainter.
func gpuFrame(s *state, l *gpu2d.List) {
	l.Reset()
	bg := s.theme.Background
	l.FillRect(gpu2d.Rect{W: s.w, H: s.h}, gpu2d.RGBA{R: bg.R, G: bg.G, B: bg.B, A: bg.A})
	s.paint(gpupainter.New(l))
}

// The whole gallery, every widget family, drawn for the GPU and rasterized
// by gpu2d's CPU reference, equals the PixelPainter frame byte for byte.
func TestGalleryOnTheGPUPathIsPixelPainter(t *testing.T) {
	s := newState(surfaceW, surfaceH)
	if s.theme.Background.A != 255 {
		t.Skip("the theme's background is translucent: the frame starts from fillBG's raw bytes")
	}
	want := make([]byte, 4*s.w*s.h)
	s.draw(want)
	l, err := gpu2d.NewList(s.w, s.h)
	if err != nil {
		t.Fatal(err)
	}
	gpuFrame(s, l)
	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(want))
	if err := l.Rasterize(got); err != nil {
		t.Fatal(err)
	}
	t.Logf("gallery %d × %d: %d instances, %d arena bytes", s.w, s.h, len(l.Instances()), len(l.Data()))
	if !bytes.Equal(want, got) {
		for i := range want {
			if want[i] != got[i] {
				p := i / 4
				t.Fatalf("first difference at (%d,%d) channel %d: PixelPainter %d, GPU path %d", p%s.w, p/s.w, i%4, want[i], got[i])
			}
		}
	}

	r, err := gpumetal.New(s.w, s.h)
	if errors.Is(err, mtl.ErrUnsupported) || errors.Is(err, mtl.ErrNoDevice) {
		return // no Metal here: the CPU reference above is the check
	}
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.Draw(l); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, r.Pixels()) {
		t.Fatalf("the gallery drawn on %s differs from PixelPainter", r.Name())
	}
	t.Logf("drawn on %s: byte-identical", r.Name())
}

// BenchmarkGalleryFrameMetal is the whole gallery on a Mac's GPU: recording
// through gpupainter, binning, upload, dispatch and the wait. Compare with
// BenchmarkGalleryFrame, the same frame through PixelPainter.
func BenchmarkGalleryFrameMetal(b *testing.B) {
	s := newState(surfaceW, surfaceH)
	r, err := gpumetal.New(s.w, s.h)
	if errors.Is(err, mtl.ErrUnsupported) || errors.Is(err, mtl.ErrNoDevice) {
		b.Skip(err)
	}
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()
	l, _ := gpu2d.NewList(s.w, s.h)
	b.SetBytes(int64(4 * s.w * s.h))
	b.ReportAllocs()
	for b.Loop() {
		gpuFrame(s, l)
		if err := r.Draw(l); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(l.Instances())), "instances")
}
