//go:build darwin

// Package darwin is the minimal Objective-C runtime wrapper the tray needs:
// the primitives for driving NSStatusBar, NSStatusItem and NSMenu.
//
// It reaches Objective-C through purego, which is the FFI layer the rest of
// this module already uses. Upstream it went through goffi instead, and a
// binary containing both could not be linked without CGo: each ships its own
// fakecgo, and the two definitions of _cgo_init collide. Since a CGo-free
// build is the whole reason these libraries were chosen, the tray came over to
// purego rather than the other two coming over to goffi -- they are the ones
// with a GPU compositor and a WebView binding built on it.
package darwin

import (
	"errors"
	"runtime"
	"structs"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// Errors returned by Objective-C runtime operations.
var (
	ErrLibraryNotLoaded = errors.New("darwin: failed to load library")
	ErrSymbolNotFound   = errors.New("darwin: symbol not found")
	ErrClassNotFound    = errors.New("darwin: class not found")
)

// ID represents an Objective-C object pointer.
type ID uintptr

// Class represents an Objective-C class pointer.
type Class uintptr

// SEL represents an Objective-C selector (method name).
type SEL uintptr

// NSSize represents the dimensions of a rectangle.
type NSSize struct {
	_      structs.HostLayout
	Width  float64
	Height float64
}

// nsPoint is NSPoint: the same two-double layout as NSSize, named separately
// because the one call that takes one reads better for it.
type nsPoint struct {
	_ structs.HostLayout
	X float64
	Y float64
}

// NSRect is an origin and a size, as AppKit lays it out: four doubles.
type NSRect struct {
	_      structs.HostLayout
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// Frame reads a view's or window's frame.
//
// A struct of four doubles comes back in floating-point registers rather than
// the integer one the Send methods here use, so it needs the typed form of
// objc_msgSend. That is one generic call with purego and was a hand-built call
// interface with the FFI library this package used to go through, which is why
// the status item's position was left unanswered until now.
func (id ID) Frame(sel SEL) NSRect {
	if id.IsNil() {
		return NSRect{}
	}
	return objc.Send[NSRect](objc.ID(id), objc.SEL(sel))
}

// runtimeState holds what has to be loaded before any class can be looked up.
type runtimeState struct {
	once sync.Once
	err  error

	// Allocating a class pair is not in purego's objc package, so these three
	// are bound here. Everything else goes through that package.
	allocateClassPair func(super Class, name string, extraBytes uintptr) Class
	addMethod         func(cls Class, name SEL, imp uintptr, types string) bool
	registerClassPair func(cls Class)
}

var rt runtimeState

// initRuntime loads the frameworks and binds the class-creation functions.
//
// purego's objc package loads libobjc for itself, but objc_getClass can only
// find a class whose framework is in the process: without these two, every
// NSStatusBar lookup returns nil with nothing to say why.
func initRuntime() error {
	rt.once.Do(func() {
		for _, path := range []string{
			"/System/Library/Frameworks/Foundation.framework/Foundation",
			"/System/Library/Frameworks/AppKit.framework/AppKit",
		} {
			if _, err := purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
				rt.err = errors.Join(ErrLibraryNotLoaded, err)
				return
			}
		}
		libobjc, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			rt.err = errors.Join(ErrLibraryNotLoaded, err)
			return
		}
		purego.RegisterLibFunc(&rt.allocateClassPair, libobjc, "objc_allocateClassPair")
		purego.RegisterLibFunc(&rt.addMethod, libobjc, "class_addMethod")
		purego.RegisterLibFunc(&rt.registerClassPair, libobjc, "objc_registerClassPair")
	})
	return rt.err
}

// GetClass looks up an Objective-C class by name, 0 when there is none.
func GetClass(name string) Class {
	if initRuntime() != nil {
		return 0
	}
	return Class(objc.GetClass(name))
}

// RegisterSelector interns a selector by name.
func RegisterSelector(name string) SEL {
	if initRuntime() != nil {
		return 0
	}
	return SEL(objc.RegisterName(name))
}

// Send sends a message that takes no arguments.
func (id ID) Send(sel SEL) ID {
	return ID(objc.ID(id).Send(objc.SEL(sel)))
}

// Send sends a message with no arguments to a class.
func (c Class) Send(sel SEL) ID {
	return ID(objc.ID(c).Send(objc.SEL(sel)))
}

// IsNil reports whether the object is nil.
func (id ID) IsNil() bool { return id == 0 }

// Ptr returns the object as a raw pointer value.
func (id ID) Ptr() uintptr { return uintptr(id) }

// SendPtr sends a message with one pointer-sized argument.
//
// The argument crosses as a plain uintptr, so anything it points at must be
// pinned by the caller for the length of the call; the runtime does not see a
// uintptr as a reference and would be free to move or collect the referent.
func (id ID) SendPtr(sel SEL, arg uintptr) ID {
	return ID(objc.ID(id).Send(objc.SEL(sel), arg))
}

// SendBool sends a message with one BOOL argument.
func (id ID) SendBool(sel SEL, arg bool) ID {
	return ID(objc.ID(id).Send(objc.SEL(sel), arg))
}

// SendInt sends a message with one NSInteger argument.
func (id ID) SendInt(sel SEL, arg int64) ID {
	return ID(objc.ID(id).Send(objc.SEL(sel), arg))
}

// SendSize sends a message with one NSSize argument.
func (id ID) SendSize(sel SEL, size NSSize) ID {
	return ID(objc.ID(id).Send(objc.SEL(sel), size))
}

// SendDouble sends a message with one double argument.
func (id ID) SendDouble(sel SEL, arg float64) ID {
	return ID(objc.ID(id).Send(objc.SEL(sel), arg))
}

// MsgSendPtrPtr sends a message with two pointer-sized arguments. The pinning
// note on SendPtr applies to both.
func MsgSendPtrPtr(id ID, sel SEL, arg0, arg1 uintptr) ID {
	return ID(objc.ID(id).Send(objc.SEL(sel), arg0, arg1))
}

// MsgSend3Ptr sends a message with three pointer-sized arguments. The pinning
// note on SendPtr applies to all three.
func MsgSend3Ptr(id ID, sel SEL, arg0, arg1, arg2 uintptr) ID {
	return ID(objc.ID(id).Send(objc.SEL(sel), arg0, arg1, arg2))
}

// AllocateClassPair creates a class that is not yet registered. Add its methods
// with ClassAddMethod, then RegisterClassPair.
func AllocateClassPair(superclass Class, name string) Class {
	if initRuntime() != nil {
		return 0
	}
	return rt.allocateClassPair(superclass, name, 0)
}

// ClassAddMethod adds a method to an unregistered class. imp is a callback
// address from purego.NewCallback and typeEncoding its Objective-C signature,
// such as "v@:@" for a void method taking one object.
func ClassAddMethod(cls Class, sel SEL, imp uintptr, typeEncoding string) bool {
	if initRuntime() != nil {
		return false
	}
	return rt.addMethod(cls, sel, imp, typeEncoding)
}

// RegisterClassPair makes an allocated class available to the runtime.
func RegisterClassPair(cls Class) {
	if initRuntime() != nil {
		return
	}
	rt.registerClassPair(cls)
}

// NewNSString creates an NSString from a Go string.
//
// The caller owns the result and must release it: this uses alloc plus
// initWithUTF8String:, so it comes back with a retain count of one that belongs
// to the caller. Contrast NewNSData, which does not.
func NewNSString(s string) ID {
	if initRuntime() != nil {
		return 0
	}
	cls := GetClass("NSString")
	if cls == 0 {
		return 0
	}
	nsstr := cls.Send(RegisterSelector("alloc"))
	if nsstr.IsNil() {
		return 0
	}

	// SendPtr carries the address as a plain uintptr, so the bytes are pinned
	// until initWithUTF8String: has copied them.
	cstr := append([]byte(s), 0)
	var pinner runtime.Pinner
	pinner.Pin(&cstr[0])
	defer pinner.Unpin()
	return nsstr.SendPtr(RegisterSelector("initWithUTF8String:"), uintptr(unsafe.Pointer(&cstr[0])))
}

// NewNSData creates an NSData from a byte slice.
//
// The caller does NOT own the result and must not release it: this goes through
// +[NSData dataWithBytes:length:], a class convenience constructor, so the
// object comes back autoreleased. Releasing it is an over-release whose crash
// surfaces later, inside the event loop, where nothing points back to the call
// that caused it. Contrast NewNSString, which the caller does own.
func NewNSData(data []byte) ID {
	if initRuntime() != nil || len(data) == 0 {
		return 0
	}
	cls := GetClass("NSData")
	if cls == 0 {
		return 0
	}

	// dataWithBytes:length: copies the bytes; pinned until it returns, since
	// MsgSendPtrPtr passes the address as a uintptr.
	var pinner runtime.Pinner
	pinner.Pin(&data[0])
	defer pinner.Unpin()

	return MsgSendPtrPtr(ID(cls), RegisterSelector("dataWithBytes:length:"),
		uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)))
}

// PostAppDefinedEvent creates an NSEvent of type NSApplicationDefined and posts
// it to the application's event queue via [NSApp postEvent:atStart:YES].
//
// This is the wake-up mechanism for [NSApp run]: the run loop only re-checks
// the stop flag after dequeuing and processing a real event, so ending the loop
// means delivering one, and a bare CFRunLoopStop wake is not enough (see
// darwinTray.Destroy). postEvent:atStart: is thread-safe per AppKit, and the
// application retains the posted event, so nothing is released here.
func PostAppDefinedEvent(app ID) {
	if app == 0 || initRuntime() != nil {
		return
	}
	cls := GetClass("NSEvent")
	if cls == 0 {
		return
	}

	// NSEventTypeApplicationDefined. otherEventWithType: raises
	// NSInternalInconsistencyException for a type outside its accepted set.
	const nsEventTypeApplicationDefined = 15

	// The argument types follow the Apple header: type and modifierFlags are
	// NSUInteger, windowNumber/data1/data2 are NSInteger, subtype is a short,
	// timestamp an NSTimeInterval and location an NSPoint. Handing purego the
	// Go types that match lets it place them as the ABI expects.
	event := objc.Send[objc.ID](objc.ID(cls),
		objc.SEL(RegisterSelector("otherEventWithType:location:modifierFlags:timestamp:windowNumber:context:subtype:data1:data2:")),
		uint64(nsEventTypeApplicationDefined),
		nsPoint{},
		uint64(0),
		float64(0),
		int64(0),
		uintptr(0),
		int16(0),
		int64(0),
		int64(0),
	)
	if event == 0 {
		return
	}
	objc.ID(app).Send(objc.SEL(RegisterSelector("postEvent:atStart:")), event, true)
}
