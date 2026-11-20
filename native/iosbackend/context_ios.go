//go:build ios

package iosbackend

/*
#include "ios.h"
*/
import "C"

import (
	"unsafe"

	"github.com/bbmumford/mullion/native"
)

// Context is the BackendContext for the iOS backend.
// The backend sets Host.EscapeHatchBackendContext to this value at Run so
// extensions can present UIKit controllers from the shirei root view controller.
type Context struct{}

// Platform implements native.BackendContext.
func (Context) Platform() string { return "ios" }

// RootViewController returns the UIWindow root UIViewController as an
// opaque pointer, or nil if the host is not attached yet.
// Plugins cast with cgo / ObjC (__bridge UIViewController *).
func (Context) RootViewController() unsafe.Pointer {
	return unsafe.Pointer(C.ios_rootViewController())
}

var _ native.BackendContext = Context{}
