//go:build windows

// Package win32 binds the user32 window plumbing that more than one package
// here needs.
//
// The tray's hidden window and the main-thread wake's message-only window were
// each resolving the same four entry points, with DefWindowProcW resolved a
// third time as well. One binding cannot drift from itself, and a window class
// registered through one handle behaves the same as one registered through
// another, so there was nothing to be gained from two.
//
// The procs are exported as *windows.LazyProc and not wrapped in Go functions,
// which matters more than it looks: LazyProc.Call carries //go:uintptrescapes,
// so a conversion written in its argument list keeps the referent alive and
// unmoved for the call. A wrapper would be an ordinary Go function, would lose
// the directive, and would turn every caller passing an address into exactly
// the hazard internal/ffiaudit exists to find.
//
// NewLazySystemDLL, not NewLazyDLL: it loads from the system directory only, so
// a user32.dll planted next to the executable cannot be picked up instead.
//
// # What is deliberately not here
//
// The menu entry points -- CreateMenu, CreatePopupMenu, AppendMenuW,
// DestroyMenu -- are bound twice in this module and stay that way. The tray
// calls them through LazyProc.Call with uintptr arguments; the application menu
// bar binds them as typed purego functions whose pointer parameters are real Go
// pointers, which the collector can see, so it needs no directive and no
// pinning at all. Sharing one set would mean choosing one calling style, and
// the only style that suits both is the uintptr one -- which would take the
// menu bar's pointer safety away to remove a duplicate. The duplication is the
// cheaper of the two.
package win32

import "golang.org/x/sys/windows"

var user32 = windows.NewLazySystemDLL("user32.dll")

// Window and message plumbing, shared by the tray's hidden window and the
// main-thread wake's message-only window.
var (
	RegisterClassExW = user32.NewProc("RegisterClassExW")
	CreateWindowExW  = user32.NewProc("CreateWindowExW")
	DefWindowProcW   = user32.NewProc("DefWindowProcW")
	PostMessageW     = user32.NewProc("PostMessageW")
)
