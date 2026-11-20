package mullion

import (
	"unsafe"

	"github.com/bbmumford/mullion/native"
)

// nativeWindowHandle reads the Go-drawn window's platform handle out of the
// backend escape hatch.
//
// The hatch is how the native package means its backends' handles to be
// reached: the concrete type lives with each backend, and a caller that needs a
// handle asserts for the method it wants. Asserting on the method rather than
// on the backend type keeps this one file instead of six, and keeps the root
// package from importing every backend to name them.
//
// Only macOS and Windows have anything to give. A Wayland toplevel and an X11
// window id are not what GTK's dialogs and menus take for a parent, and they
// read nil, which those calls accept as no parent.
func nativeWindowHandle() unsafe.Pointer {
	switch ctx := native.GetHost().EscapeHatchBackendContext.(type) {
	case interface{ NSWindow() unsafe.Pointer }:
		return ctx.NSWindow()
	case interface{ HWND() unsafe.Pointer }:
		return ctx.HWND()
	}
	return nil
}
