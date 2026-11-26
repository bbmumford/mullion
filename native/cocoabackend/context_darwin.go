//go:build darwin && !ios

package cocoabackend

import (
	"unsafe"

	"github.com/bbmumford/mullion/native"
)

// Context is the BackendContext for the macOS (AppKit) backend.
// The backend sets Host.EscapeHatchBackendContext to this value at Run.
type Context struct{ window *windowState }

// Platform implements native.BackendContext.
func (Context) Platform() string { return "darwin" }

// NSWindow returns the live NSWindow as an opaque pointer, or nil if the host
// has not created a window yet or it is closed. Call on the UI thread; cast
// with objc / AppKit (NSWindow *).
func (c Context) NSWindow() unsafe.Pointer {
	if c.window == nil || c.window.closed {
		return nil
	}
	return cPointer(uintptr(c.window.gWindow))
}

// cPointer turns an address a foreign call returned into a pointer. The
// memory is C's or Objective-C's (an NSString's UTF-8 bytes, an IOSurface's
// pixels, an MTLBuffer's contents, an NSWindow), which Go's garbage collector
// neither moves nor frees, so a Go pointer to it stays valid for as long as
// its owner keeps it. The uintptr's bits are read as a pointer through the
// variable that holds them, instead of converting the integer, which go vet's
// unsafeptr check cannot tell apart from arithmetic on a Go pointer.
func cPointer(addr uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&addr)) }

var _ native.BackendContext = Context{}
