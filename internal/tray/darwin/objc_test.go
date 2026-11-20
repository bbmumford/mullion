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

// The rest of this file covers the calls an ABI mistake breaks silently: a
// wrongly-placed bool, int or double does not fail, it arrives as something
// else and the tray simply misbehaves. Each case reads the value back out of
// Objective-C rather than only checking that the call returned.

// SendBool places a BOOL, which is a signed char on arm64 and not a word.
func TestSendBoolPlacesTheArgument(t *testing.T) {
	img := GetClass("NSImage").Send(RegisterSelector("alloc")).
		SendSize(RegisterSelector("initWithSize:"), NSSize{Width: 10, Height: 10})
	if img.IsNil() {
		t.Fatal("could not make an NSImage")
	}
	for _, want := range []bool{true, false} {
		img.SendBool(RegisterSelector("setTemplate:"), want)
		got := img.Send(RegisterSelector("isTemplate"))
		if (uintptr(got) != 0) != want {
			t.Errorf("setTemplate:%v read back as %v", want, uintptr(got) != 0)
		}
	}
}

// SendInt places an NSInteger. NSStatusItem's length is a double, so the
// integer case uses NSNumber, where the value comes straight back.
func TestSendIntPlacesTheArgument(t *testing.T) {
	for _, want := range []int64{0, 1, 42, -7, 1 << 40} {
		num := GetClass("NSNumber").Send(RegisterSelector("alloc")).
			SendInt(RegisterSelector("initWithInteger:"), want)
		if num.IsNil() {
			t.Fatalf("initWithInteger:%d returned nil", want)
		}
		got := num.Send(RegisterSelector("integerValue"))
		if int64(got) != want {
			t.Errorf("initWithInteger:%d read back as %d", want, int64(got))
		}
	}
}

// SendDouble places a float, which goes in a floating-point register. Getting
// this wrong is how a status item ends up with a nonsense width.
func TestSendDoublePlacesTheArgument(t *testing.T) {
	for _, want := range []float64{0, 1, 22.5, -3.25, 1e9} {
		num := GetClass("NSNumber").Send(RegisterSelector("alloc")).
			SendDouble(RegisterSelector("initWithDouble:"), want)
		if num.IsNil() {
			t.Fatalf("initWithDouble:%v returned nil", want)
		}
		got := sendForDouble(num, RegisterSelector("doubleValue"))
		if got != want {
			t.Errorf("initWithDouble:%v read back as %v", want, got)
		}
	}
}

// MsgSend3Ptr places three word arguments. NSMenuItem's designated initialiser
// takes exactly that shape and is what the tray builds every item with.
func TestMsgSend3PtrPlacesThreeArguments(t *testing.T) {
	title := NewNSString("Item title")
	key := NewNSString("k")
	if title.IsNil() || key.IsNil() {
		t.Fatal("could not make the strings")
	}
	item := GetClass("NSMenuItem").Send(RegisterSelector("alloc"))
	if item.IsNil() {
		t.Fatal("[NSMenuItem alloc] returned nil")
	}
	item = MsgSend3Ptr(item, RegisterSelector("initWithTitle:action:keyEquivalent:"),
		title.Ptr(), 0, key.Ptr())
	if item.IsNil() {
		t.Fatal("initWithTitle:action:keyEquivalent: returned nil")
	}
	// The first and third arguments arrived if the item carries them back.
	gotTitle := item.Send(RegisterSelector("title"))
	if s := nsStringGoValue(gotTitle); s != "Item title" {
		t.Errorf("title read back as %q, want %q", s, "Item title")
	}
	gotKey := item.Send(RegisterSelector("keyEquivalent"))
	if s := nsStringGoValue(gotKey); s != "k" {
		t.Errorf("keyEquivalent read back as %q, want %q", s, "k")
	}
}

// A class made at runtime with a Go callback for its method is how every tray
// click reaches Go. If the callback is not installed the menu is simply dead,
// with nothing reported.
func TestAClassCanBeCreatedWithAGoCallbackForItsMethod(t *testing.T) {
	cls := AllocateClassPair(GetClass("NSObject"), "mullionObjcTestTarget")
	if cls == 0 {
		t.Fatal("AllocateClassPair returned 0")
	}
	called := make(chan struct{}, 1)
	imp := newTestCallback(func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	sel := RegisterSelector("mullionTestFire:")
	if !ClassAddMethod(cls, sel, imp, "v@:@") {
		t.Fatal("ClassAddMethod reported failure")
	}
	RegisterClassPair(cls)

	if GetClass("mullionObjcTestTarget") == 0 {
		t.Fatal("the registered class cannot be looked up by name")
	}
	obj := ID(cls).Send(RegisterSelector("alloc"))
	if obj.IsNil() {
		t.Fatal("alloc on the new class returned nil")
	}
	obj.SendPtr(sel, 0)
	select {
	case <-called:
	default:
		t.Error("the method was dispatched but the Go callback did not run")
	}
}

// PostAppDefinedEvent is the one call with no second way to check it: it is
// what lets Remove return from the event loop. Here there is no loop, so what
// is checked is that the nine-argument call -- a struct, a double and mixed
// integer widths -- builds an event and posts it without faulting.
func TestPostAppDefinedEventBuildsAndPostsAnEvent(t *testing.T) {
	app := GetClass("NSApplication").Send(RegisterSelector("sharedApplication"))
	if app.IsNil() {
		t.Skip("no NSApplication in this process")
	}
	PostAppDefinedEvent(app)

	// The event is in the queue; take it back out, which both proves it was
	// posted and leaves the queue as it was found.
	const anyMask = ^uint64(0)
	distantPast := GetClass("NSDate").Send(RegisterSelector("distantPast"))
	ev := sendForEvent(app, RegisterSelector("nextEventMatchingMask:untilDate:inMode:dequeue:"),
		anyMask, distantPast, NewNSString("kCFRunLoopDefaultMode"), true)
	if ev.IsNil() {
		t.Fatal("nothing was posted to the application's event queue")
	}
	// NSEventTypeApplicationDefined is 15.
	if got := ev.Send(RegisterSelector("type")); uintptr(got) != 15 {
		t.Errorf("the queued event has type %d, want 15 (application defined)", uintptr(got))
	}
}

// PostAppDefinedEvent must do nothing with no application, rather than fault.
func TestPostAppDefinedEventIgnoresANilApplication(t *testing.T) {
	PostAppDefinedEvent(0)
}

// Bundled is what stands between an unbundled process and a notification call
// that would terminate it, so its answer matters more than most. A test binary
// is not in an .app bundle, which is the case it has to get right here.
func TestBundledIsFalseForAnUnbundledProcess(t *testing.T) {
	if Bundled() {
		t.Error("Bundled() is true for a test binary, which is not in an .app bundle")
	}
}
