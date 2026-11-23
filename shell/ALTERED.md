# Altered source: systray

This directory is an altered version of **systray**'s root package. It is not
the original software. systray's internal package is
[`../internal/tray`](../internal/tray/ALTERED.md), and its FFI pointer audit is
[`../internal/ffiaudit`](../internal/ffiaudit/ALTERED.md).

- Project: <https://github.com/gogpu/systray>, module `github.com/gogpu/systray`
- Merged: commit `732654fe9ab62c537dbffadea7d41c6bf67d675e` (2026-11-11)
- Copyright (c) 2026 Andrey Kolkov and GoGPU Contributors, under the MIT
  licence, whose text is [`LICENSE`](LICENSE) beside this file and
  [`../licenses/LICENSE.systray`](../licenses/LICENSE.systray)

systray's authors did not write the alterations described here and are not
responsible for them.

## How it was altered

This package changed more than any other tree here: its public API was
replaced, and the builder it wrapped is unchanged underneath, in
`../internal/tray`.

1. **Package.** `package systray` became `package shell`, and the internal
   package's import became `../internal/tray`.
2. **A new tray API.** systray's chainable `SystemTray` (`systray.go`) was
   replaced by `Tray` (`tray.go`, written for mullion), which reports the errors
   the original discarded, takes the module's `icon.Icon`, and attaches to
   whichever event loop the process runs. `systray.go` and `systray_test.go` were
   not carried.
3. **One menu model.** `menu.go` now holds one `Item` that describes a command
   once and converts to the tray's menu and to the application menu bar;
   `menu_test.go` was rewritten to pin both conversions. `doc.go` was rewritten
   for the package's wider scope.
4. **Added for mullion.** `menubar.go` (the menu bar, over `../internal/menubar`,
   which came from glaze), `notify.go` (notifications), `shortcut.go` (parsing
   and matching menu-bar shortcuts, so the Go-drawn surface can answer them where
   no bar does), and their tests.
5. **Not carried** from systray: `examples/`, `tools/`, the project documents
   (`AGENTS.md`, `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, `CONTRIBUTING.md`,
   `README.md`, `ROADMAP.md`, `SECURITY.md`, `llms.txt`), `codecov.yml`,
   `.golangci.yml`, `.gitignore`, `.github/`, `go.mod` and `go.sum`. systray's
   `LICENSE` is carried unchanged, beside this file, `../internal/tray` and
   `../internal/ffiaudit`, and as
   [`../licenses/LICENSE.systray`](../licenses/LICENSE.systray).

## The files

| file | origin |
|---|---|
| `doc.go` | systray's `doc.go`, rewritten |
| `menu.go` | systray's `menu.go`, rewritten |
| `menu_test.go` | systray's `menu_test.go`, rewritten |
| `menubar.go` | added for mullion |
| `notify.go` | added for mullion |
| `shortcut.go`, `shortcut_test.go` | added for mullion |
| `tray.go`, `tray_test.go` | added for mullion, replacing systray's `systray.go` and `systray_test.go` |
| `LICENSE` | systray's `LICENSE`, unchanged |
