//go:build darwin

package darwin

import (
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// The helpers below reach shapes of objc_msgSend the package's own Send methods
// do not offer, because only the tests need them.

// sendForDouble reads a double return, which comes back in a floating-point
// register rather than the integer one.
func sendForDouble(id ID, sel SEL) float64 {
	return objc.Send[float64](objc.ID(id), objc.SEL(sel))
}

// sendForEvent sends the four-argument event-dequeue message.
func sendForEvent(id ID, sel SEL, mask uint64, until ID, mode ID, dequeue bool) ID {
	return ID(objc.Send[objc.ID](objc.ID(id), objc.SEL(sel), mask, objc.ID(until), objc.ID(mode), dequeue))
}

// nsStringGoValue reads an NSString back into Go.
func nsStringGoValue(s ID) string {
	if s.IsNil() {
		return ""
	}
	cstr := s.Send(RegisterSelector("UTF8String"))
	if cstr.IsNil() {
		return ""
	}
	p := uintptr(cstr)
	var n int
	for ; *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0; n++ {
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
}

// newTestCallback builds a method implementation that calls fn, with the
// (self, _cmd, sender) signature an Objective-C method is given.
func newTestCallback(fn func()) uintptr {
	return purego.NewCallback(func(self, cmd, sender uintptr) uintptr {
		fn()
		return 0
	})
}
