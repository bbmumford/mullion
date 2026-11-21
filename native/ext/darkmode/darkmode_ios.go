//go:build ios && cgo

package darkmode

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework UIKit -framework Foundation
#include "darkmode_ios.h"
*/
import "C"

import (
	"unsafe"

	"github.com/bbmumford/mullion/native"
)

//export mullionExtDarkmodeIOSUpdate
func mullionExtDarkmodeIOSUpdate(isDark C.int) {
	setDarkMode(isDark == 1)
}

func initPlatform() {
	var root unsafe.Pointer
	if ctx, ok := native.GetHost().EscapeHatchBackendContext.(interface{ RootViewController() unsafe.Pointer }); ok {
		root = ctx.RootViewController()
	}
	isDark := C.mullion_ext_darkmode_ios_start_observer(root)
	setDarkMode(isDark == 1)
}
