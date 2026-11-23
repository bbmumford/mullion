# Altered source: systray's FFI pointer audit

This package is derived from **systray**'s `ffi_pointer_audit_test.go`, and much
of its matching is that file's. It is an altered version, not the original
software.

- Project: <https://github.com/gogpu/systray>, module `github.com/gogpu/systray`
- Merged: commit `732654fe9ab62c537dbffadea7d41c6bf67d675e` (2026-11-11)
- Copyright (c) 2026 Andrey Kolkov and GoGPU Contributors, under the MIT
  licence: [`../../licenses/LICENSE.systray`](../../licenses/LICENSE.systray)

systray's authors did not write the alterations described here and are not
responsible for them.

## How it was altered

The original was one test that walked the directory it sat in and checked
calls made through the `syscall` and `windows` packages. Here it is a package,
which the module-root test `../../ffiaudit_test.go` (written for mullion) runs
over the whole module:

- it scans every package of the module, whatever its build tags, rather than
  one directory;
- it names no FFI library: only calls the compiler marks
  `//go:uintptrescapes` are taken as safe, and every other call that receives
  an address as a `uintptr` (purego's, objc's, a COM vtable wrapper's) needs a
  pin;
- it reads function literals, including the ones held in package-level
  variables, where the original read function declarations only;
- a `runtime.Pinner` pin counts only when it comes before the conversion, and
  pinning a value covers addresses inside it;
- it checks addresses stored into slice and array elements, through pointers
  and into package-level variables, not only into struct fields;
- the root test's allowlist needs a reason for each entry, and an entry whose
  code has changed is reported as stale and fails.

## The files

`ffiaudit.go`, `ffiaudit_test.go` and `helper_test.go` have no file of the same
name upstream; `ffiaudit.go` carries the original's matching forward, and the
tests are new.
