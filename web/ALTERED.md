# Altered source: glaze

This directory is an altered version of **glaze**. It is not the original
software.

- Project: <https://github.com/crgimenes/glaze>, module `github.com/crgimenes/glaze`
- Merged: commit `f07fcba9e1805dbd739ace8fa94337f8fd75b50b` (2026-11-16)
- Copyright (c) 2025 Adam Bouqdib (original go-webview base), Copyright (c) 2026
  crgimenes, under the MIT licence:
  [`../licenses/LICENSE.glaze`](../licenses/LICENSE.glaze)

glaze's authors did not write the alterations described here and are not
responsible for them. glaze's `menu` package is
[`../internal/menubar`](../internal/menubar/ALTERED.md).

## How it was altered

1. **Module and package.** Import paths moved from `github.com/crgimenes/glaze`
   to `github.com/bbmumford/mullion/web`, `package glaze` became `package web`,
   and the `menu` import became `../internal/menubar`.
2. **One main thread for the module.** Work posted to the main thread goes
   through `../internal/mainthread` (libdispatch's main queue on macOS, GLib's
   idle source on Linux), and the "am I on the UI thread" checks answer from it:
   on Windows, where no WebView has recorded its own UI thread, the answer is
   the thread `internal/mainthread` pinned. `webview_darwin.go` takes its
   selector, class, string and autorelease-pool helpers from `../internal/objc`.
3. **Dialogs without a WebView.** The macOS open and save panel presenters in
   `dialog_darwin.go` were split so they run with no WebView; `filedialog*.go`
   adds open, save and folder dialogs parented to any window handle, and
   `question*.go` a yes/no prompt. `../dialog` uses both for the Go-drawn and the
   WebView surface alike.
4. **Living under another surface's loop.** `pump_linux.go` gives GTK's main
   context a non-blocking turn; mullion's runtime calls it each frame while a
   WebView window is open under the Go-drawn window's loop on Linux.
   `pump_other.go` is the empty counterpart elsewhere.
5. **Targets with no WebView.** `webview_unsupported.go` reports
   `errors.ErrUnsupported` on targets built with none of the `darwin`, `linux`
   and `windows` build tags, js/wasm among them.
6. **Package documentation** in `doc.go`.
7. **Not carried.** `examples/`, `imgs/`, `editor/` and its scenario test (the
   line of `webview_darwin_test.go` that ran it went with it), `Makefile`,
   `CONTRIBUTING.md`, `.github/`, `go.mod` and `go.sum`. glaze's `LICENSE` is
   [`../licenses/LICENSE.glaze`](../licenses/LICENSE.glaze), unchanged.

`README.md` in this directory is glaze's own README, unchanged. It describes
glaze, its import path `github.com/crgimenes/glaze` and its examples, which are
not in this tree; in mullion the package is `github.com/bbmumford/mullion/web`.

## The files

Each file compared with its counterpart at the merged commit, after item 1 is
applied to the upstream text:

| | files |
|---|---:|
| unchanged | 2 |
| item 1 only | 54 |
| changed further | 5 |
| added for mullion | 15 |
| **total** | **76** |

The counts leave out this file.

Changed further (items 2 to 4 and 7):

- `dialog_darwin.go`
- `webview_darwin.go`
- `webview_darwin_test.go`
- `webview_linux.go`
- `webview_windows.go`

Added for mullion (items 3 to 6):

- `doc.go`
- `filedialog.go`, `filedialog_darwin.go`, `filedialog_linux.go`,
  `filedialog_windows.go`, `filedialog_unsupported.go`
- `question.go`, `question_darwin.go`, `question_linux.go`,
  `question_windows.go`, `question_unsupported.go`, `question_test.go`
- `pump_linux.go`, `pump_other.go`
- `webview_unsupported.go`
