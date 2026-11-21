//go:build darwin || ios

// Package objc holds the Objective-C conveniences the macOS code in this
// module all needs.
//
// Three packages had their own: the WebView surface, the Go-drawn surface's
// AppKit shell, and the tray's runtime wrapper. All three sit on purego's objc
// package underneath, and what they each added was the same short list --
// interning a selector, looking up a class, making an NSString, running a block
// inside an autorelease pool.
//
// Reading a C string back into Go is deliberately not here. Two packages do it,
// not three, and it is the one piece that needs pointer arithmetic on a uintptr
// -- which go vet reports as a possible misuse of unsafe.Pointer wherever it is
// written. Moving it would have spread that report into new code to remove a
// duplicate that is not one. Two of them cached selectors in their own map,
// one did not cache at all, and one of the two had a benign race where a
// concurrent first use could intern the same name twice.
//
// Selector interning is worth caching rather than repeating: purego's
// RegisterName takes the Objective-C runtime's global lock, and this module
// asks for the same few dozen selectors over and over, on the frame path among
// others.
package objc

import (
	"runtime"
	"sync"

	"github.com/ebitengine/purego/objc"
)

// Sel interns a selector by name, caching the result.
//
// LoadOrStore rather than Store: two goroutines interning the same name at once
// both get the one value, where Store would leave each holding its own copy of
// what the runtime says is the same selector.
func Sel(name string) objc.SEL {
	if v, ok := sels.Load(name); ok {
		return v.(objc.SEL)
	}
	s := objc.RegisterName(name)
	actual, _ := sels.LoadOrStore(name, s)
	return actual.(objc.SEL)
}

// Class looks up a class by name, as an ID so it can be messaged directly.
// It reads 0 for a class the runtime does not know, which is what a caller
// should check rather than assume: a class is missing when its framework has
// not been loaded.
func Class(name string) objc.ID {
	return objc.ID(objc.GetClass(name))
}

// NSString makes an autoreleased NSString from a Go string.
//
// stringWithUTF8String: returns an autoreleased object, so the caller does not
// own it and must not release it -- the opposite of an alloc-and-init, and the
// distinction that caused an over-release in the tray's icon path. Inside an
// Autorelease block it lives until the block ends.
func NSString(s string) objc.ID {
	return Class("NSString").Send(Sel("stringWithUTF8String:"), s)
}

// Autorelease runs f inside an NSAutoreleasePool, so the autoreleased objects it
// makes are freed when it returns rather than at some later turn of the loop.
//
// A pool is drained even if f panics: without that, a panic would leave the
// pool as the thread's current one for whatever runs next, and its objects
// would be freed at a moment nothing expects.
func Autorelease(f func()) {
	// A pool belongs to the thread that made it, and draining one from another
	// thread is undefined. Pinning for the block's length is what keeps the
	// creation and the drain on the same thread even if f blocks and the
	// scheduler would otherwise move the goroutine.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	pool := Class("NSAutoreleasePool").Send(Sel("alloc")).Send(Sel("init"))
	defer pool.Send(Sel("drain"))
	f()
}

var sels sync.Map // string -> objc.SEL
