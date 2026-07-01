# go-widgets/gallery

Live wasm demo of [go-widgets/toolkit](https://github.com/go-widgets/toolkit).

Deploys to <https://go-widgets.github.io/gallery/> on every push to
`main` (see `.github/workflows/pages.yml`).

## Local dev

```
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
