// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-gfx/gpu2d"
	gpumetal "github.com/go-gfx/gpu2d/metal"
	mtl "github.com/go-macos/metal"
	"github.com/go-widgets/gpupainter"
	"github.com/go-widgets/toolkit"
)

// gpuFrame records the whole gallery for the GPU: the background, then every
// widget through gpupainter.
func gpuFrame(s *state, l *gpu2d.List) { newGPURec(l).frame(s) }

// gpuRec keeps ONE gpupainter.Painter across frames: frame resets the list,
// then the painter (Painter.Reset does not reset the list), and records again.
type gpuRec struct {
	l  *gpu2d.List
	gp *gpupainter.Painter
}

func newGPURec(l *gpu2d.List) *gpuRec { return &gpuRec{l: l, gp: gpupainter.New(l)} }

func (g *gpuRec) frame(s *state) {
	g.l.Reset()
	bg := s.theme.Background
	g.l.FillRect(gpu2d.Rect{W: s.w, H: s.h}, gpu2d.RGBA{R: bg.R, G: bg.G, B: bg.B, A: bg.A})
	g.gp.Reset(g.l)
	s.paint(g.gp)
	g.gp.Flush() // pixels still being batched into an image
}

// scaledState builds the gallery at the given pixel ratio and puts the
// toolkit's global metric scale back to 1 when the test or benchmark ends, so
// the next one starts from the 1× default.
func scaledState(tb testing.TB, scale float64) *state {
	tb.Helper()
	tb.Cleanup(func() { toolkit.SetMetricScale(1) })
	return newStateAt(scale)
}

// The whole gallery, every widget family, drawn for the GPU and rasterized
// by gpu2d's CPU reference, equals the PixelPainter frame byte for byte, at 1×
// and at Retina 2× (widgets laid out and drawn at 2×, not a bitmap upscale).
func TestGalleryOnTheGPUPathIsPixelPainter(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		t.Run(fmt.Sprintf("%gx", scale), func(t *testing.T) { galleryOnTheGPUPath(t, scale) })
	}
}

func galleryOnTheGPUPath(t *testing.T, scale float64) {
	s := scaledState(t, scale)
	if s.w != int(float64(surfaceW)*scale) {
		t.Fatalf("surface width %d, want %d", s.w, int(float64(surfaceW)*scale))
	}
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
func BenchmarkGalleryFrameMetal(b *testing.B) { benchGalleryFrameMetal(b, 1) }

// BenchmarkGalleryFrameMetal2x is the same at Retina 2×: 1920 wide, widgets at 2×.
func BenchmarkGalleryFrameMetal2x(b *testing.B) { benchGalleryFrameMetal(b, 2) }

func benchGalleryFrameMetal(b *testing.B, scale float64) {
	benchMetal(b, scale, false)
}

// BenchmarkGalleryFrameMetalPipe is the pipelined frame: Submit frame N, record
// frame N+1 while the GPU draws it, then Wait for N. Same at 2×.
func BenchmarkGalleryFrameMetalPipe(b *testing.B)   { benchMetal(b, 1, true) }
func BenchmarkGalleryFrameMetalPipe2x(b *testing.B) { benchMetal(b, 2, true) }

func benchMetal(b *testing.B, scale float64, pipelined bool) {
	s := scaledState(b, scale)
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
	rec := newGPURec(l)
	// Submit's own wall time, reported apart: a CPU profile on macOS can
	// charge another thread's time to a goroutine blocked in a C call.
	var inSubmit time.Duration
	if pipelined {
		rec.frame(s)
		for b.Loop() {
			t0 := time.Now()
			err := r.Submit(l) // copies the list
			inSubmit += time.Since(t0)
			if err != nil {
				b.Fatal(err)
			}
			rec.frame(s) // record N+1 while the GPU draws N
			if err := r.Wait(); err != nil {
				b.Fatal(err)
			}
		}
	} else {
		for b.Loop() {
			rec.frame(s)
			if err := r.Draw(l); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ReportMetric(float64(len(l.Instances())), "instances")
	if pipelined {
		b.ReportMetric(float64(inSubmit.Nanoseconds())/float64(b.N), "submit-ns/op")
	}
}
