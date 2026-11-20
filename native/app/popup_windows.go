//go:build windows

package app

import (
	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/win32backend"
)

func openPopupWindow(options PopupWindowOptions, root native.FrameFn) (PopupWindow, error) {
	return win32backend.OpenPopupWindow(options, root)
}

func closePopupWindow(window PopupWindow) { win32backend.ClosePopupWindow(window) }

func movePopupWindow(window PopupWindow, position *native.Vec2) {
	win32backend.MovePopupWindow(window, position)
}
