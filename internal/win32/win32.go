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
// The menu entry points are wrapped rather than exported raw, so both callers
// can share one binding without either giving up a safety property it had. See
// the Menus block below.
package win32

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var user32 = windows.NewLazySystemDLL("user32.dll")

// Window and message plumbing, shared by the tray's hidden window and the
// main-thread wake's message-only window.
var (
	RegisterClassExW = user32.NewProc("RegisterClassExW")
	CreateWindowExW  = user32.NewProc("CreateWindowExW")
	DefWindowProcW   = user32.NewProc("DefWindowProcW")
	PostMessageW     = user32.NewProc("PostMessageW")
)

// Menus. Typed at the Go boundary, so a caller hands over a real *uint16 the
// collector can see for the length of the call, and the uintptr conversion
// happens below in the argument list of LazyProc.Call, where
// //go:uintptrescapes keeps the referent alive and unmoved. That is both
// properties at once, and it is why these are wrappers where the window
// plumbing above is not.
//
// The two callers had drifted into different conventions for the same four
// entry points: the tray passed addresses as uintptr and depended on the
// directive, and the menu bar bound typed purego functions and depended on the
// pointer being visible. Neither was wrong, but one of them had to be chosen to
// share a binding, and a typed wrapper over a directive-carrying Call is
// stronger than either alone.
var (
	createMenu      = user32.NewProc("CreateMenu")
	createPopupMenu = user32.NewProc("CreatePopupMenu")
	appendMenuW     = user32.NewProc("AppendMenuW")
	destroyMenu     = user32.NewProc("DestroyMenu")
)

// CreateMenu makes an empty menu bar. It reads 0 on failure.
func CreateMenu() uintptr {
	h, _, _ := createMenu.Call()
	return h
}

// CreatePopupMenu makes an empty pop-up menu, for a context menu or a submenu.
// It reads 0 on failure.
func CreatePopupMenu() uintptr {
	h, _, _ := createPopupMenu.Call()
	return h
}

// AppendMenu adds one item to the end of menu and reports whether it was added.
//
// item is a command id for an ordinary item, or the submenu's handle when flags
// carry MF_POPUP. text may be nil, which is what a separator wants.
func AppendMenu(menu uintptr, flags uint32, item uintptr, text *uint16) bool {
	r, _, _ := appendMenuW.Call(menu, uintptr(flags), item, uintptr(unsafe.Pointer(text)))
	return r != 0
}

// DestroyMenu frees a menu and everything under it, and reports whether it did.
// SetMenu does not free the bar it replaces, so a caller swapping one has to.
func DestroyMenu(menu uintptr) bool {
	r, _, _ := destroyMenu.Call(menu)
	return r != 0
}
