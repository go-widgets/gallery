// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-gfx/gpu2d"
	gpumetal "github.com/go-gfx/gpu2d/metal"
	mtl "github.com/go-macos/metal"
	"github.com/go-widgets/gpupainter"
	"github.com/go-widgets/painter"
	"github.com/go-widgets/toolkit"
	"golang.org/x/image/font/gofont/gomono"
)

// The question of go-widgets/toolkit#548: a terminal redraws its whole grid on
// output, and gpupainter rasterises every string on the CPU into a mask. Is a
// full 80 × 24 redraw on the GPU path slower than on the CPU path?

const termCols, termRows = 80, 24

// termFonts are the faces measured: the toolkit's built-in bitmap font (the
// default, 4 px cells) and Go Mono at 16 px, the size of a real terminal.
func termFonts(tb testing.TB) []struct {
	name string
	font toolkit.Font
} {
	tb.Helper()
	mono, err := toolkit.NewTrueTypeFont(gomono.TTF, 16)
	if err != nil {
		tb.Fatal(err)
	}
	return []struct {
		name string
		font toolkit.Font
	}{{"bitmap", nil}, {"mono16", mono}}
}

// fullTerminal is an 80 × 24 Terminal in font f (nil: the default) with every
// cell printed, in the eight standard colours by line, sized to exactly its
// grid.
func fullTerminal(tb testing.TB, f toolkit.Font) (*toolkit.Terminal, int, int) {
	tb.Helper()
	t := toolkit.NewTerminal(termCols, termRows, 0)
	t.Font = f
	w := fitAxis(tb, func(n int) int { t.SetBounds(toolkit.Rect{W: n, H: 4096}); c, _ := t.Screen().Size(); return c }, termCols)
	h := fitAxis(tb, func(n int) int { t.SetBounds(toolkit.Rect{W: w, H: n}); _, r := t.Screen().Size(); return r }, termRows)
	var b strings.Builder
	for row := range termRows {
		if row > 0 {
			b.WriteString("\r\n")
		}
		fmt.Fprintf(&b, "\x1b[3%dm", row%8)
		for col := range termCols {
			b.WriteByte(byte('!' + (row*termCols+col)%94)) // every printable ASCII glyph
		}
	}
	if _, err := t.Write([]byte(b.String())); err != nil {
		tb.Fatal(err)
	}
	return t, w, h
}

// fitAxis returns the smallest extent n for which cells(n) reaches want.
func fitAxis(tb testing.TB, cells func(int) int, want int) int {
	tb.Helper()
	for n := 1; n <= 4096; n++ {
		if cells(n) >= want {
			return n
		}
	}
	tb.Fatalf("no extent up to 4096 holds %d cells", want)
	return 0
}

// TestTerminalOnTheGPUPathIsPixelPainter: the full 80 × 24 terminal recorded
// through gpupainter equals the PixelPainter frame byte for byte, on the CPU
// reference and on Metal, so the benchmarks below compare the same picture.
func TestTerminalOnTheGPUPathIsPixelPainter(t *testing.T) {
	for _, f := range termFonts(t) {
		t.Run(f.name, func(t *testing.T) { terminalOnTheGPUPath(t, f.font) })
	}
}

func terminalOnTheGPUPath(t *testing.T, f toolkit.Font) {
	term, w, h := fullTerminal(t, f)
	theme := toolkit.DefaultLight()
	want := make([]byte, 4*w*h)
	term.Draw(painter.NewPixelPainter(want, w, h), theme)
	l, err := gpu2d.NewList(w, h)
	if err != nil {
		t.Fatal(err)
	}
	gp := gpupainter.New(l)
	term.Draw(gp, theme)
	gp.Flush()
	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(want))
	if err := l.Rasterize(got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Fatal("the terminal recorded for the GPU differs from PixelPainter")
	}
	t.Logf("terminal %d × %d px: %d instances, %d arena bytes", w, h, len(l.Instances()), len(l.Data()))
	r, err := gpumetal.New(w, h)
	if errors.Is(err, mtl.ErrUnsupported) || errors.Is(err, mtl.ErrNoDevice) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.Draw(l); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, r.Pixels()) {
		t.Fatalf("the terminal drawn on %s differs from PixelPainter", r.Name())
	}
}

// BenchmarkTerminal80x24 is one full redraw:
//   - cpu: Terminal.Draw into a PixelPainter, the CPU path;
//   - record: Terminal.Draw through gpupainter into a gpu2d list, the GPU
//     path's CPU half, where each cell's glyph is rasterised into a mask;
//   - metal: record, then draw the list on the GPU and wait for it.
func BenchmarkTerminal80x24(b *testing.B) {
	for _, f := range termFonts(b) {
		b.Run(f.name, func(b *testing.B) { benchTerminal(b, f.font) })
	}
}

func benchTerminal(b *testing.B, f toolkit.Font) {
	term, w, h := fullTerminal(b, f)
	theme := toolkit.DefaultLight()
	b.Run("cpu", func(b *testing.B) {
		p := painter.NewPixelPainter(make([]byte, 4*w*h), w, h)
		b.ReportAllocs()
		for b.Loop() {
			term.Draw(p, theme)
		}
	})
	l, err := gpu2d.NewList(w, h)
	if err != nil {
		b.Fatal(err)
	}
	gp := gpupainter.New(l)
	record := func() {
		l.Reset()
		gp.Reset(l)
		term.Draw(gp, theme)
		gp.Flush()
	}
	b.Run("record", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			record()
		}
	})
	b.Run("metal", func(b *testing.B) {
		r, err := gpumetal.New(w, h)
		if errors.Is(err, mtl.ErrUnsupported) || errors.Is(err, mtl.ErrNoDevice) {
			b.Skip(err)
		}
		if err != nil {
			b.Fatal(err)
		}
		defer r.Close()
		b.ReportAllocs()
		for b.Loop() {
			record()
			if err := r.Draw(l); err != nil {
				b.Fatal(err)
			}
		}
	})
}
