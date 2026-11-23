# Altered source: systray's internal package

This directory is an altered version of **systray**'s `internal` package: the
tray builder and the platform code under it. It is not the original software.
systray's root package is [`../../shell`](../../shell/ALTERED.md).

- Project: <https://github.com/gogpu/systray>, module `github.com/gogpu/systray`
- Merged: commit `732654fe9ab62c537dbffadea7d41c6bf67d675e` (2026-11-11)
- Copyright (c) 2026 Andrey Kolkov and GoGPU Contributors, under the MIT
  licence: [`../../licenses/LICENSE.systray`](../../licenses/LICENSE.systray)

systray's authors did not write the alterations described here and are not
responsible for them.

## How it was altered

1. **Module and package.** `package internal` became `package tray`, with its
   imports rewritten onto `github.com/bbmumford/mullion/internal/tray`.
2. **The macOS Objective-C layer moved from goffi to purego.** `darwin/objc.go`
   was rewritten onto purego with the same API, and the callbacks in
   `platform_darwin.go` moved with it. The rest of the module calls the system
   through purego, and a binary holding both FFI libraries does not link without
   cgo: each carries its own `fakecgo`, and their two `_cgo_init` definitions
   collide. goffi is no longer a dependency. `darwin/objc_test.go`,
   `darwin/helpers_darwin_test.go` and `darwin/size_darwin_test.go` are new tests
   of that layer.
3. **Retains and releases.** `platform_darwin.go` retains the status item and its
   button across autorelease pools, where the original kept bare addresses, and
   releases the images, strings and menus it hands to AppKit once AppKit holds
   its own reference.
4. **Tray bounds on macOS.** `Bounds` reads the frame of the status item's
   window, where the original returned zeros.
5. **Notifications on macOS** return an error naming the cause when the process
   is not in an `.app` bundle (`darwin.Bundled`), before any notification call
   is made. macOS delivers nothing for an unbundled process, and the newer
   `UNUserNotificationCenter` answers one with an Objective-C exception that
   ends the process, so the check stands in front of whichever API is called.
6. **Thread pinning** moved out of `init.go` to `../mainthread`, which pins the
   main thread once for every surface in the process.
7. **Win32 bindings.** The window plumbing and menu entry points in
   `platform_windows.go` and its test come from `../win32`, shared with the
   main-thread wake and the menu bar.
8. **Targets with no notification area.** `platform_unsupported.go` reports
   `errors.ErrUnsupported` on targets built with none of the `darwin`, `linux`
   and `windows` build tags.

What systray had that was not carried is listed in
[`../../shell/ALTERED.md`](../../shell/ALTERED.md).

## The files

Each file compared with its counterpart at the merged commit, after item 1 is
applied to the upstream text:

| | files |
|---|---:|
| unchanged | 0 |
| item 1 only | 7 |
| changed further | 6 |
| added for mullion | 4 |
| **total** | **17** |

The counts leave out this file.

Changed further:

- `darwin/objc.go` (item 2)
- `init.go` (item 6)
- `platform_darwin.go` (items 2 to 5)
- `platform_windows.go`, `platform_windows_test.go` (item 7)
- `tray_test.go` (one comment)

Added for mullion:

- `darwin/helpers_darwin_test.go`, `darwin/objc_test.go`,
  `darwin/size_darwin_test.go` (item 2)
- `platform_unsupported.go` (item 8)
