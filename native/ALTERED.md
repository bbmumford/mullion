# Altered source: go-shirei

This directory is an altered version of **go-shirei**. It is not the original
software.

- Project: <https://github.com/hasenj/go-shirei>, module `go.hasen.dev/shirei`
- Merged: commit `26d6369dce19fc82643f4f403d2dd87907f85437` (v0.9.1, 2026-11-16)
- Copyright (c) 2026 Judi Systems, under the zlib licence, whose text is
  [`COPYING`](COPYING) beside this file and
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
   `go.mod` and `go.sum` files. go-shirei's `COPYING` is carried unchanged,
   beside this file and as
   [`../licenses/LICENSE.shirei`](../licenses/LICENSE.shirei).
7. **Corrections to the carried prose.** Comments and documents that say who
   wrote, vendored or changed something name go-shirei
   (`internal/wayland/README.md`, `internal/wayland/cursorshape/cursorshape.go`,
   and go-shirei's issue numbers in `waylandbackend/`). Module paths written in
   documents and comments are this tree's (`github.com/bbmumford/mullion/native/...`,
   and `github.com/bbmumford/mullion/cmd/...` for the tools), and where prose
   names go-shirei's root package, as in `native.Vec4`, it uses this tree's
   package name. `docs/migration-v0.8.0.md` opens by saying it is go-shirei's
   note for its own v0.8.0 release.

### Reading the text in this directory

Item 2 reached prose as well as code, so most comments and documents that
go-shirei's authors wrote now say "mullion" where they said "shirei". Where
they describe the library (its frame model, its backends, its tools, its log
tag and file names), that library is now this tree, mullion's Go-drawn
surface, and the name is right. Where a statement is about who wrote, designed
or changed something, it was go-shirei's authors, unless the file is listed
below as added for mullion; item 7 corrected the statements that said
otherwise outright.

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
| `docs/` | 20 | 0 | 19 | 1 | 0 |
| `drive/` | 4 | 0 | 4 | 0 | 0 |
| `editorfile/` | 2 | 0 | 2 | 0 | 0 |
| `ext/` | 35 | 12 | 23 | 0 | 0 |
| `gpurender/` | 25 | 11 | 11 | 3 | 0 |
| `internal/` | 54 | 32 | 20 | 2 | 0 |
| `iosbackend/` | 4 | 0 | 4 | 0 | 0 |
| `ioshost/` | 1 | 0 | 1 | 0 | 0 |
| `jsbackend/` | 9 | 1 | 8 | 0 | 0 |
| `testdata/` | 18 | 18 | 0 | 0 | 0 |
| `tools/` | 1 | 0 | 1 | 0 | 0 |
| `waylandbackend/` | 34 | 0 | 34 | 0 | 0 |
| `widgets/` | 102 | 17 | 66 | 0 | 19 |
| `win32backend/` | 17 | 4 | 13 | 0 | 0 |
| `x11backend/` | 20 | 6 | 14 | 0 | 0 |
| **total** | **491** | **124** | **340** | **7** | **20** |

The counts leave out this file and `COPYING`.

Changed further:

- `cocoabackend/cocoa_appkit_darwin.go` (item 4)
- `gpurender/d3d11_windows.go`, `gpurender/gles.go`, `gpurender/metal_macos.go`
  (item 3)
- `docs/migration-v0.8.0.md`, `internal/wayland/README.md`,
  `internal/wayland/cursorshape/cursorshape.go` (item 7)

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
