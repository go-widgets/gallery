# go-widgets/gallery

[![CI](https://github.com/go-widgets/gallery/actions/workflows/ci.yml/badge.svg)](https://github.com/go-widgets/gallery/actions/workflows/ci.yml)
[![pages](https://github.com/go-widgets/gallery/actions/workflows/pages.yml/badge.svg)](https://github.com/go-widgets/gallery/actions/workflows/pages.yml)
[![live demo](https://img.shields.io/badge/live-demo-14b8a6)](https://go-widgets.github.io/gallery/)
![coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)
[![license](https://img.shields.io/badge/license-BSD--3--Clause-blue)](./LICENSE)

Live wasm demo of [go-widgets/toolkit](https://github.com/go-widgets/toolkit).

Deploys to <https://go-widgets.github.io/gallery/> on every push to
`main` (see `.github/workflows/pages.yml`).

## Local dev

```text
pkgx task build   # compiles gallery.wasm + copies wasm_exec.js
pkgx task serve   # http://localhost:8090/
```

The gallery uses a plain `<canvas>` + `putImageData` — no
SharedArrayBuffer, no COOP/COEP, no server-side dep. That makes it
easy to iframe from any static host (`python3 -m http.server`, GitHub
Pages, S3, ...).

## Screen readers

The page is readable by assistive technology. The dashboard's widget trees
(menu bar, toolbar, columns, status bar) are published to
[webcanvas](https://github.com/go-widgets/webcanvas)'s accessibility mirror
through `go-widgets/window/webmirror`, the one toolkit-to-webcanvas conversion,
and the multi-line editor gets webcanvas's text layer, so it is one named
`textbox`. `browsertest/` is a module of its own that loads the real page in
headless Chrome and reads `Accessibility.getFullAXTree`
(`cd browsertest && go test ./...`; skipped without Chrome, `CHROME=` points at
one).

## Native controls gallery

The wasm page above shows the toolkit's **drawn** widgets. Its native
mirror shows the same widget set as the **platform's own controls** —
one `toolkit.NativeControl` descriptor per `NativeKind`, hosted by
`go-widgets/window`'s cocoa / win32 / gtk back-end (real
`NSButton`/`GtkEntry`/… over the framebuffer), pure Go, `CGO_ENABLED=0`.

```text
go build -tags gallerynative -o gallery-native .
./gallery-native                     # macOS / Windows
GO_WIDGETS_GTK=1 ./gallery-native    # Linux (GTK4-hosted back-end)
```

`native.go` builds the descriptor grid (tag-less, 100%-tested);
`native_main.go` (behind the `gallerynative` tag, so the wasm build and
the coverage-gated tests never pull in the windowing stack) opens the
window and runs it.

## Layout

`scene.go` composes widgets from the toolkit onto a **960-wide**
surface whose height is **computed from the box-layout content** (no
hard-coded height — the canvas grows as widgets are added) as a
three-column card grid plus a full-width band, every widget kind in
its own labelled slot rather than being hidden behind a tab. Around
it:

- **MenuBar** (File / Edit / View / Help) — clicking a name opens
  a popover the host paints under it.
- **Toolbar** (New / Open / Save / … / About) — each click fires a
  Notification toast.
- **Statusbar** — 4-segment readout at the bottom.
- **Notification** — top-right toast, auto-dismisses via a 60 Hz
  `setInterval` that calls `state.tick()`.

The scene composes the full toolkit catalogue (~98 widgets across
the v0.7–v0.85 waves) into labelled slots — leaf widgets, the
container/layout system (VBox/HBox/Grid/Frame/Border/Paned), and the
Wave-7 dashboard widgets (Kanban, Gantt, Agenda, Sparkline, Area/
Scatter/Radar charts).

### Retina (2×)

The surface is 960 wide at 1×. `newStateAt(2)` renders the same scene for a
Retina display: it calls `toolkit.SetMetricScale(2)` (every widget metric and
the OpenType face follow) and lays the layout out at 1920 device pixels wide,
every logical length in `scene.go` going through `sc` (`toolkit.Scaled`). The
widgets are drawn at 2×, not a bitmap upscale. The parity test checks that
the GPU path (`gpupainter`, gpu2d's CPU reference and Metal) is byte-identical
to `PixelPainter` at 1× and at 2×, and `BenchmarkGalleryFrame2x` /
`BenchmarkGalleryFrameMetal2x` measure CPU against Metal at both scales.

One `gpupainter.Painter` is kept and `Reset` each frame, and the pipelined
benchmarks (`BenchmarkGalleryFrameMetalPipe`, `…Pipe2x`) `Submit` frame N,
record frame N+1, then `Wait` for N. Measured on an M4 Max, single process,
abbench `--max-load 8`, 8 kept rounds each, benchstat medians per frame,
toolkit v0.355.1. Each Metal mode was its own interleaved pass, so each has
its own CPU baseline:

| | CPU | Metal, synchronous | CPU | Metal, pipelined |
|---|---|---|---|---|
| 1× 960×2834 | 2.84 ms | 4.04 ms (+42 %, p=0.000) | 2.94 ms | 2.47 ms (−16 %, p=0.021) |
| 2× 1920×5668 | 10.0 ms | 14.0 ms (+40 %, p=0.000) | 9.67 ms | 8.82 ms (~, p=0.195) |

Pipelined, the GPU wins at 1× and is not significantly different at 2×;
synchronous, it loses at both, because recording the frame is CPU work either
way. The rounds ran at a 1-minute load just under 8, which is why the CPU
times are higher than the 1.98 ms measured on a quieter host.

## Interactions

The gallery is a demonstrator you **drive**, not just look at:

- **Drag** — Kanban cards move between columns; Gantt bars move, and
  their edges resize (`mousedown`→`mousemove`→`mouseup` routed through
  a scene drag-capture into the widgets' `EventClick`/`EventMouseDrag`/
  `EventMouseUp`).
- **Right-click edit menu** — a `ContextMenu` on every editable widget
  (Kanban card, Gantt task, Agenda day, ListBox item, Table row,
  TreeView / TreeTable node, PropertyGrid row): move / duplicate /
  delete / add-child / … The scene hit-tests via the toolkit's exported
  `CardAt`/`TaskAt`/`DayAt`/`IndexAt`/`RowAt`/`NodeAt` helpers.
- **Agenda** — a Week/Month/Quarter/Year switcher; click (or right-
  click) an empty day to add an event.
- **Theme switcher** — recolours the whole scene (Light/Dark/Adwaita/
  WhiteSur) by swapping one `Theme` value.
- **Full window** — the ⛶ button (or double-click) expands the canvas
  to fill the browser window in a scrollable overlay; **Esc** exits.
  It is deliberately *not* the OS Fullscreen API — it stays inside the
  browser window.

Interactive state flows through
[`go-widgets/mvvm`](https://github.com/go-widgets/mvvm): the theme and
Agenda-view switchers are two-way-bound `Observable`s, the Agenda
events an `ObservableList` mirrored into the widget, and the command
palette a `Command` (see `viewmodel.go`).

The SVG-per-widget variant of the same catalogue (43 widgets, one
`.svg` + `.png` each) is available at <https://go-widgets.github.io/svg/>
via [`go-widgets/svg`](https://github.com/go-widgets/svg)'s
`gallery-render` command, which is regenerated on every toolkit dep
bump.

## License

BSD-3-Clause.
