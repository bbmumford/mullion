# Altered source: go-shirei

This directory is an altered version of **go-shirei**. It is not the original
software.

- Project: <https://github.com/hasenj/go-shirei>, module `go.hasen.dev/shirei`
- Merged: commit `26d6369dce19fc82643f4f403d2dd87907f85437` (v0.9.1, 2026-11-16)
- Copyright (c) 2026 Judi Systems, under the zlib licence:
  [`../licenses/LICENSE.shirei`](../licenses/LICENSE.shirei)

go-shirei's authors did not write the alterations described here and are not
responsible for them. go-shirei's three build tools are altered in the same
way and live in [`../cmd`](../cmd/ALTERED.md).

## How it was altered

1. **Module and package.** Import paths moved from `go.hasen.dev/shirei` to
   `github.com/bbmumford/mullion/native`, the root package clause became
   `package native`, and every qualified reference moved with it. The nested
   modules `ext/camera`, `ext/darkmode` and `ext/window` are part of this module,
   without their own `go.mod` and `go.sum`. gofmt then re-sorted the import
   blocks and realigned the lines the rewrite touched.
2. **The project's name.** Every other occurrence of the upstream name was
   replaced with `mullion`: identifiers, the C functions and Objective-C classes
   that cross the cgo boundary, environment variable names, the capability file
   (`ext/camera/mullion.capabilities`), the root file `mullion.go`, and the
   prose of comments and documents.
3. **Addresses handed to native code are pinned.** `gpurender/d3d11_windows.go`,
   `gpurender/gles.go` and `gpurender/metal_macos.go` hold Go memory with
   `runtime.Pinner` while Direct3D 11, OpenGL ES or Metal has its address, where
   the original passed a bare `uintptr`.
4. **Shared Objective-C helpers.** `cocoabackend/cocoa_appkit_darwin.go` takes its
   selector cache from `../internal/objc`, which the WebView and tray code use
   too.
5. **Added for mullion.** Package documentation (`doc.go`, `widgets/doc.go`) and
   new widgets, each in a file of its own: a value-bound `Select`, `Card`,
   `Form`, `Accordion`, `Tree`, `DateField`, `Calendar`, `Markdown` and
   `TextGrid`.
6. **Not carried.** `demos/`, `promos/`, `examples/`, `behavior_test/`,
   `layout_tests/`, the `shirei_tester` and `behavior_runner` tools, `README.md`,
   `CHANGELOG.md`, the images under `docs/` and at the top level, and the
   `go.mod` and `go.sum` files. go-shirei's `COPYING` is
   [`../licenses/LICENSE.shirei`](../licenses/LICENSE.shirei), unchanged.

### Reading the text in this directory

Item 2 reached prose as well as code, so comments and documents that
go-shirei's authors wrote now say "mullion" where they said "shirei". In a file
of this directory, read "mullion" as go-shirei, its authors or its design
unless the file is listed below as changed further or added for mullion.
"go-mullion" means go-shirei; no project of that name exists. The replacement
also reached module paths written in the documents and in some comments, which
read `go.hasen.dev/mullion/...`: no such module exists, and this tree is
imported as `github.com/bbmumford/mullion/native/...`.

## The files

Each file compared with its counterpart at the merged commit, after items 1
and 2 are applied to the upstream text:

| directory | files | unchanged | items 1–2 only | changed further | added for mullion |
|---|---:|---:|---:|---:|---:|
| (top level) | 85 | 0 | 84 | 0 | 1 |
| `androidbackend/` | 9 | 3 | 6 | 0 | 0 |
| `app/` | 36 | 14 | 22 | 0 | 0 |
| `audio/` | 6 | 6 | 0 | 0 | 0 |
| `cocoabackend/` | 9 | 0 | 8 | 1 | 0 |
| `docs/` | 20 | 0 | 20 | 0 | 0 |
| `drive/` | 4 | 0 | 4 | 0 | 0 |
| `editorfile/` | 2 | 0 | 2 | 0 | 0 |
| `ext/` | 35 | 12 | 23 | 0 | 0 |
| `gpurender/` | 25 | 11 | 11 | 3 | 0 |
| `internal/` | 54 | 32 | 22 | 0 | 0 |
| `iosbackend/` | 4 | 0 | 4 | 0 | 0 |
| `ioshost/` | 1 | 0 | 1 | 0 | 0 |
| `jsbackend/` | 9 | 1 | 8 | 0 | 0 |
| `testdata/` | 18 | 18 | 0 | 0 | 0 |
| `tools/` | 1 | 0 | 1 | 0 | 0 |
| `waylandbackend/` | 34 | 0 | 34 | 0 | 0 |
| `widgets/` | 102 | 17 | 66 | 0 | 19 |
| `win32backend/` | 17 | 4 | 13 | 0 | 0 |
| `x11backend/` | 20 | 6 | 14 | 0 | 0 |
| **total** | **491** | **124** | **343** | **4** | **20** |

The counts leave out this file.

Changed further (items 3 and 4):

- `cocoabackend/cocoa_appkit_darwin.go`
- `gpurender/d3d11_windows.go`
- `gpurender/gles.go`
- `gpurender/metal_macos.go`

Added for mullion (item 5):

- `doc.go`
- `widgets/doc.go`
- `widgets/accordion.go`
- `widgets/calendar.go`, `widgets/calendar_test.go`
- `widgets/card.go`
- `widgets/datefield.go`, `widgets/datefield_test.go`
- `widgets/form.go`, `widgets/form_test.go`
- `widgets/markdown.go`, `widgets/markdown_inline.go`,
  `widgets/markdown_parse.go`, `widgets/markdown_test.go`
- `widgets/select.go`, `widgets/select_test.go`
- `widgets/textgrid.go`, `widgets/textgrid_test.go`
- `widgets/tree.go`, `widgets/tree_test.go`

Third-party parts inside this tree that go-shirei itself carried (the Android
native-app glue, the Wayland bindings, three fonts) keep their own licences;
[`../NOTICE`](../NOTICE) lists them.
