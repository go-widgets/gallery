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

`scene.go` composes every widget family from the toolkit onto a
720×480 surface:

- **MenuBar** (File / Edit / View / Help) — clicking a name opens a
  popover the host paints under it
- **Toolbar** (New / Open / Save / … / About) — each click fires a
  Notification toast
- **Notebook** (Buttons / Input / List+Tree / Feedback / Composites)
  — one primary widget per tab plus a couple of auxiliaries for
  the visual weight
- **Statusbar** — 4-segment readout at the bottom
- **Notification** — top-right toast, auto-dismisses via a 60 Hz
  `setInterval` that calls `state.tick()`

## License

BSD-3-Clause.
