//go:build (!darwin || ios || x11darwin) && !windows && (!linux || android)

package app

import (
	"fmt"
	"github.com/bbmumford/mullion/native"
)

func openPopupWindow(PopupWindowOptions, native.FrameFn) (PopupWindow, error) {
	return 0, fmt.Errorf("native popup windows are not supported by this backend")
}

func closePopupWindow(PopupWindow) {}

func movePopupWindow(PopupWindow, *native.Vec2) {}
