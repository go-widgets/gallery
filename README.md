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

`scene.go` composes widgets from the toolkit onto a **960×720**
surface as a three-column card grid — every widget kind lives in
its own labelled slot rather than being hidden behind a tab. Around
it:

- **MenuBar** (File / Edit / View / Help) — clicking a name opens
  a popover the host paints under it.
- **Toolbar** (New / Open / Save / … / About) — each click fires a
  Notification toast.
- **Statusbar** — 4-segment readout at the bottom.
- **Notification** — top-right toast, auto-dismisses via a 60 Hz
  `setInterval` that calls `state.tick()`.

The scene currently exposes the pre-v0.7 widget catalogue (27
constructors — MenuBar/Toolbar/Notebook/DropDown/TreeView/… — that
already covered the core toolkit surface). The 29 widgets shipped
across v0.7 / v0.8 / v0.9 (Switch, Alert, Card, Steps, Table,
Toast, Banner, ActionRow, ViewSwitcher, ChatBubble, Diff, Stat,
Timeline, DropZone, Chip, FormField, ProgressCircle, …) are
compiled into the wasm bundle but not yet composed into a slot —
adding them is a slot-geometry follow-up.

The SVG-per-widget variant of the same catalogue (43 widgets, one
`.svg` + `.png` each) is available at <https://go-widgets.github.io/svg/>
via [`go-widgets/svg`](https://github.com/go-widgets/svg)'s
`gallery-render` command, which is regenerated on every toolkit dep
bump.

## License

BSD-3-Clause.
