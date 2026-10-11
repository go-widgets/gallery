// Native entrypoint. The demonstrator is the wasm canvas driver in main.go,
// build-tagged js && wasm; this stub exists only so that `go build ./...`
// links on a native host. A `package main` with no `main` fails to link
// there ("function main is undeclared in the main package"), which CI never
// saw because it builds the root package and builds iso only for js/wasm.
// The scene itself (scene.go) is what the native tests exercise.
//
// The body is empty on purpose: it holds no statement, so the coverage gate
// has nothing here to count. Same convention as go-widgets/app-template.
//
//go:build !(js && wasm)

package main

func main() {}
