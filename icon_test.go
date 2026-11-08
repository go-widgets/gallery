// Copyright (c) 2026 the go-widgets/gallery authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package main

import (
	"encoding/xml"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
)

// brandMark is go-widgets/brand v0.1.0 svg/color/go-widgets.svg, attribute
// for attribute: the mark the pages inline as their icon.
var brandMark = []map[string]string{
	{"x": "8", "y": "8", "width": "240", "height": "240", "rx": "48", "fill": "#0d9488"},
	{"x": "48", "y": "56", "width": "160", "height": "32", "rx": "8", "fill": "#ffffff", "opacity": "0.95"},
	{"x": "48", "y": "112", "width": "112", "height": "32", "rx": "8", "fill": "#ffffff", "opacity": "0.82"},
	{"x": "48", "y": "168", "width": "160", "height": "32", "rx": "8", "fill": "#ffffff", "opacity": "0.70"},
}

var iconHref = regexp.MustCompile(`<link rel="icon" type="image/svg\+xml" href="data:image/svg\+xml,([^"]+)">`)

// Each page carries the go-widgets mark as its icon, decoded exactly: a
// transcription slip in the inlined data URI would show here, not as a
// wrong tab icon nobody looks at.
func TestPagesCarryTheBrandIcon(t *testing.T) {
	for _, page := range []string{"index.html", "iso.html"} {
		b, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		m := iconHref.FindSubmatch(b)
		if m == nil {
			t.Fatalf("%s has no inlined SVG icon", page)
		}
		svg, err := url.PathUnescape(string(m[1]))
		if err != nil {
			t.Fatalf("%s: %v", page, err)
		}
		var doc struct {
			ViewBox string `xml:"viewBox,attr"`
			Rects   []struct {
				Attrs []xml.Attr `xml:",any,attr"`
			} `xml:"rect"`
		}
		if err := xml.NewDecoder(strings.NewReader(svg)).Decode(&doc); err != nil {
			t.Fatalf("%s: the icon is not well-formed SVG: %v\n%s", page, err, svg)
		}
		if doc.ViewBox != "0 0 256 256" || len(doc.Rects) != len(brandMark) {
			t.Fatalf("%s: viewBox %q with %d rects, want the brand's 0 0 256 256 with %d", page, doc.ViewBox, len(doc.Rects), len(brandMark))
		}
		for i, r := range doc.Rects {
			got := map[string]string{}
			for _, a := range r.Attrs {
				got[a.Name.Local] = a.Value
			}
			for k, v := range brandMark[i] {
				if got[k] != v {
					t.Errorf("%s: rect %d %s = %q, the brand's is %q", page, i, k, got[k], v)
				}
			}
			if len(got) != len(brandMark[i]) {
				t.Errorf("%s: rect %d has %d attributes, the brand's has %d", page, i, len(got), len(brandMark[i]))
			}
		}
	}
}
