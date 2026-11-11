// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package browsertest loads the real gallery page in headless Chrome and reads
// its accessibility tree, the way a screen reader does. It is a module of its
// own so that the gallery depends on no browser driver.
//
//	cd browsertest && go test ./...
//
// Skipped when no Chrome is found (set CHROME to point at one).
package browsertest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// galleryPage builds the gallery for js/wasm, serves it with its own index.html
// and Go's wasm_exec.js, and waits for its first frame.
func galleryPage(t *testing.T) context.Context {
	t.Helper()
	dir := t.TempDir()
	build := exec.Command("go", "build", "-o", filepath.Join(dir, "gallery.wasm"), ".")
	build.Dir = ".."
	build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm", "GOWORK=off", "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the gallery: %v\n%s", err, out)
	}
	for src, dst := range map[string]string{
		filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js"): "wasm_exec.js",
		filepath.Join("..", "index.html"):                              "index.html",
	} {
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, dst), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	t.Cleanup(srv.Close)

	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", true))
	if p := os.Getenv("CHROME"); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}
	actx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	t.Cleanup(cancel)
	ctx, cancel2 := chromedp.NewContext(actx)
	t.Cleanup(cancel2)
	ctx, cancel3 := context.WithTimeout(ctx, 90*time.Second)
	t.Cleanup(cancel3)
	if _, err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
		t.Skipf("no Chrome to run the page in: %v", err)
	}
	if _, err := chromedp.Run(ctx, chromedp.Navigate(srv.URL)); err != nil {
		t.Fatal(err)
	}
	if _, err := chromedp.Call(ctx, emulation.SetFocusEmulationEnabled, emulation.SetFocusEmulationEnabledParams{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(60 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		if ok, err := chromedp.Run(ctx, chromedp.Evaluate[bool](`window.webcanvasReady === true`)); err == nil && ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the gallery never drew its first frame")
		}
	}
	return ctx
}

type axNode struct{ Role, Name, Value string }

func unquote(v *accessibility.Value) string {
	if v == nil {
		return ""
	}
	if s, err := strconv.Unquote(string(v.Value)); err == nil {
		return s
	}
	return string(v.Value)
}

// axTree is the unignored nodes of Chromium's accessibility tree, once the
// mirror has published (it waits for at least one named button).
func axTree(t *testing.T, ctx context.Context) []axNode {
	t.Helper()
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		res, err := chromedp.Call(ctx, accessibility.GetFullAXTree, accessibility.GetFullAXTreeParams{})
		if err != nil {
			t.Fatal(err)
		}
		var out []axNode
		buttons := 0
		for _, n := range res.Nodes {
			if n.Ignored {
				continue
			}
			a := axNode{Role: unquote(n.Role), Name: unquote(n.Name), Value: unquote(n.Value)}
			if a.Role == "button" && a.Name != "" {
				buttons++
			}
			out = append(out, a)
		}
		if buttons > 0 {
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("the gallery's mirror never published: %v", out)
		}
	}
}

// A screen reader sees the gallery's controls by name, and its multi-line
// editor as exactly one textbox holding its text.
func TestGalleryAXTreeShowsItsControls(t *testing.T) {
	ctx := galleryPage(t)
	tree := axTree(t, ctx)

	named := map[string]int{}
	for _, n := range tree {
		if n.Name != "" {
			named[n.Role]++
		}
	}
	t.Logf("%d unignored nodes; named by role: %v", len(tree), named)
	if named["button"] < 5 {
		t.Errorf("only %d named buttons in the AX tree: the controls are not published", named["button"])
	}
	if named["checkbox"]+named["radio"]+named["switch"] == 0 {
		t.Errorf("no named checkbox, radio or switch in the AX tree: %v", named)
	}

	var boxes []axNode
	for _, n := range tree {
		if n.Role == "textbox" && n.Name == "Multi-line editor" {
			boxes = append(boxes, n)
		}
	}
	if len(boxes) != 1 {
		t.Fatalf("%d \"Multi-line editor\" textboxes in the AX tree, want exactly 1", len(boxes))
	}
	const want = "Multi-line editor.\nType to insert.\nEnter splits a line.\nArrow keys navigate."
	if boxes[0].Value != want {
		t.Errorf("textbox value %q, want %q", boxes[0].Value, want)
	}
}
