# Altered source: glaze's menu package

This directory is an altered version of **glaze**'s `menu` package. It is not
the original software. glaze's root package is [`../../web`](../../web/ALTERED.md).

- Project: <https://github.com/crgimenes/glaze>, module `github.com/crgimenes/glaze`
- Merged: commit `f07fcba9e1805dbd739ace8fa94337f8fd75b50b` (2026-11-16)
- Copyright (c) 2025 Adam Bouqdib (original go-webview base), Copyright (c) 2026
  crgimenes, under the MIT licence:
  [`../../licenses/LICENSE.glaze`](../../licenses/LICENSE.glaze)

glaze's authors did not write the alterations described here and are not
responsible for them.

## How it was altered

1. **Module and package.** `github.com/crgimenes/glaze/menu` became
   `github.com/bbmumford/mullion/internal/menubar`, and `package menu` became
   `package menubar`. Being internal, it is reached through `../../shell`
   (`shell.SetMenuBar`), which converts the module's one menu model into this
   package's items.
2. **Win32 bindings.** On Windows the menu calls (`CreateMenu`,
   `CreatePopupMenu`, `AppendMenuW`, `DestroyMenu`) come from `../win32`, which
   the tray uses too (`menu_windows.go` and its test).

The comments in these files are glaze's and still speak of the package as
`menu` and of glaze as its first consumer.

## The files

Each file compared with its counterpart at the merged commit, after item 1 is
applied to the upstream text: 9 files, of which 7 differ by item 1 only and 2
were changed further, `menu_windows.go` and `menu_windows_test.go` (item 2).
None was added for mullion, and the counts leave out this file.
