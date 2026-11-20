//go:build darwin

package darwin

import (
	"testing"
	"unsafe"
)

// These exercise the Objective-C layer that was rewritten from goffi onto
// purego. They need no window server and no status item, so they catch a
// mis-bound symbol or a mis-placed argument in a plain test run rather than at
// the point where a tray silently fails to appear.

func TestClassesTheTrayNeedsAreFound(t *testing.T) {
	for _, name := range []string{
		"NSString",
		"NSData",
		"NSImage",
		"NSMenu",
		"NSMenuItem",
		"NSStatusBar",
		"NSApplication",
		"NSEvent",
		"NSUserNotificationCenter",
	} {
		if GetClass(name) == 0 {
			t.Errorf("GetClass(%q) = 0; the framework holding it is not loaded", name)
		}
	}
}

func TestAClassThatDoesNotExistIsZeroAndNotAPanic(t *testing.T) {
	if cls := GetClass("NSDefinitelyNotAClass"); cls != 0 {
		t.Errorf("GetClass of a made-up name returned %#x", cls)
	}
}

func TestRegisterSelectorInterns(t *testing.T) {
	a := RegisterSelector("initWithUTF8String:")
	if a == 0 {
		t.Fatal("RegisterSelector returned 0")
	}
	if b := RegisterSelector("initWithUTF8String:"); b != a {
		t.Errorf("the same name interned to two selectors: %#x and %#x", a, b)
	}
	if c := RegisterSelector("alloc"); c == a {
		t.Errorf("two different names interned to the same selector %#x", c)
	}
}

// NewNSString carries Go bytes into Objective-C through a uintptr argument,
// which is the call most likely to be mis-placed by the ABI. Reading the string
// back out proves the bytes arrived.
func TestNewNSStringRoundTrips(t *testing.T) {
	for _, want := range []string{"Verify", "", "a longer label with spaces", "ünïcodé ✓"} {
		s := NewNSString(want)
		if s.IsNil() {
			t.Errorf("NewNSString(%q) returned nil", want)
			continue
		}
		cstr := s.Send(RegisterSelector("UTF8String"))
		if cstr.IsNil() {
			t.Errorf("UTF8String on the NSString for %q returned nil", want)
			continue
		}
		if got := goString(uintptr(cstr)); got != want {
			t.Errorf("round trip = %q, want %q", got, want)
		}
	}
}

func TestNewNSDataCarriesLength(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7}
	obj := NewNSData(data)
	if obj.IsNil() {
		t.Fatal("NewNSData returned nil")
	}
	// -[NSData length] returns an NSUInteger, so the value comes back in the
	// integer return register; a wrong length means the two uintptr arguments
	// were not placed as dataWithBytes:length: expects.
	got := obj.Send(RegisterSelector("length"))
	if uintptr(got) != uintptr(len(data)) {
		t.Errorf("length = %d, want %d", uintptr(got), len(data))
	}
}

func TestNewNSDataOnNoBytesIsNilNotAPanic(t *testing.T) {
	if obj := NewNSData(nil); !obj.IsNil() {
		t.Error("NewNSData(nil) returned an object")
	}
}

// SendDouble and SendSize place a float and a two-double struct, which go in
// different registers from everything else. NSImage's initWithSize: takes the
// struct and reports it back, so the placement is checkable.
func TestSendSizePlacesTheStructArgument(t *testing.T) {
	img := GetClass("NSImage").Send(RegisterSelector("alloc"))
	if img.IsNil() {
		t.Fatal("[NSImage alloc] returned nil")
	}
	img = img.SendSize(RegisterSelector("initWithSize:"), NSSize{Width: 18, Height: 19})
	if img.IsNil() {
		t.Fatal("initWithSize: returned nil")
	}
	got := sendForSize(img, RegisterSelector("size"))
	if got.Width != 18 || got.Height != 19 {
		t.Errorf("size = %vx%v, want 18x19", got.Width, got.Height)
	}
}

func goString(cstr uintptr) string {
	if cstr == 0 {
		return ""
	}
	var n int
	for ; *(*byte)(unsafe.Pointer(cstr + uintptr(n))) != 0; n++ {
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(cstr)), n))
}
