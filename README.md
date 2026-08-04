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
