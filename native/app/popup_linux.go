//go:build linux && !android

package app

import (
	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/waylandbackend"
	"github.com/bbmumford/mullion/native/x11backend"
)

func openPopupWindow(options PopupWindowOptions, root native.FrameFn) (PopupWindow, error) {
	if !useWayland {
		return x11backend.OpenPopupWindow(options, root)
	}
	return waylandbackend.OpenPopupWindow(options, root)
}

func closePopupWindow(window PopupWindow) {
	if useWayland {
		waylandbackend.ClosePopupWindow(window)
	} else {
		x11backend.ClosePopupWindow(window)
	}
}

func movePopupWindow(id PopupWindow, position *native.Vec2) {
	if useWayland {
		waylandbackend.MovePopupWindow(id, position)
	}
}
