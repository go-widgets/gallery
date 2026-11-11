// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import "testing"

// The gallery publishes its controls to the accessibility mirror: named, with
// distinct keys, and the multi-line editor as one textbox with a text layer.
func TestGalleryPublishesItsControls(t *testing.T) {
	app := newGalleryApp()
	nodes := app.A11yNodes()
	if len(nodes) < 20 {
		t.Fatalf("only %d nodes published", len(nodes))
	}
	if len(nodes) > 5000 {
		t.Fatalf("%d nodes: past webcanvas's 5000-node cap", len(nodes))
	}
	seen := map[string]bool{}
	named := 0
	for _, n := range nodes {
		if seen[n.Key] {
			t.Errorf("two nodes share the key %q", n.Key)
		}
		seen[n.Key] = true
		if n.W <= 0 || n.H <= 0 {
			t.Errorf("node %q has no area", n.Key)
		}
		if n.Name != "" {
			named++
		}
	}
	if named < 20 {
		t.Errorf("only %d named nodes", named)
	}

	var textboxes []string
	for _, n := range nodes {
		if n.Role == "textbox" && n.Multiline {
			textboxes = append(textboxes, n.Key)
		}
	}
	found := false
	for _, k := range textboxes {
		if c := app.TextInputFor(k); c != nil {
			if c != app.s.textView {
				t.Errorf("key %q is not the gallery's editor", k)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("the multi-line editor has no text layer (textboxes %v)", textboxes)
	}
	if app.TextInputFor("nope") != nil {
		t.Error("an unknown key got a text layer")
	}
}

func TestGalleryKeysAreStableAcrossFrames(t *testing.T) {
	app := newGalleryApp()
	before := app.A11yNodes()
	app.Click(5, 5)
	app.Tick()
	after := app.A11yNodes()
	keys := map[string]bool{}
	for _, n := range before {
		keys[n.Key] = true
	}
	same := 0
	for _, n := range after {
		if keys[n.Key] {
			same++
		}
	}
	if same < len(before)*9/10 {
		t.Errorf("only %d of %d keys survived a frame", same, len(before))
	}
}
