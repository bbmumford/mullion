//go:build darwin && !ios && !x11darwin

package app

import (
	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/cocoabackend"
)

func openPopupWindow(options PopupWindowOptions, root native.FrameFn) (PopupWindow, error) {
	return cocoabackend.OpenPopupWindow(options, root)
}

func closePopupWindow(window PopupWindow) { cocoabackend.ClosePopupWindow(window) }

func movePopupWindow(window PopupWindow, position *native.Vec2) {
	cocoabackend.MovePopupWindow(window, position)
}
