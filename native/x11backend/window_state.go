//go:build linux || (darwin && x11darwin)

package x11backend

import (
	"github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xproto"
	"github.com/bbmumford/mullion/native"
)

type windowState struct {
	ui                    *native.UI
	owner                 *windowState
	popupID               native.PopupWindow
	onClose               func()
	unsubscribe           func()
	dirty, closed, mapped bool

	winTitle                   string
	winW, winH                 int
	winQuiet, undecorated      bool
	frameFn                    native.FrameFn
	win                        xproto.Window
	gc                         xproto.Gcontext
	colormap                   xproto.Colormap
	depth                      byte
	curW, curH                 int
	windowScale                float32
	wantsFrame                 bool
	softRenderer               native.SoftRenderer
	presentBuf                 []byte
	shmAttached, presentViaShm bool
	shmSeg                     shm.Seg
	shmID                      int
	shmData                    []byte

	ime                       imeState
	pendingPaste, pendingText string
	hasPendingPaste           bool
}

var mainWindow = &windowState{ui: native.ActiveUI(), windowScale: 1, dirty: true}
var currentWindow = mainWindow
var focusedWindow *windowState
var windowsByID = map[xproto.Window]*windowState{}
var callbackDepth int

func enterWindow(w *windowState) func() {
	previous := currentWindow
	var restore func()
	if callbackDepth == 0 {
		restore = native.EnterUI(w.ui)
	} else {
		restore = native.EnterUILocked(w.ui)
	}
	currentWindow = w
	callbackDepth++
	return func() { callbackDepth--; currentWindow = previous; restore() }
}
func liveWindows() []*windowState {
	result := []*windowState{mainWindow}
	for _, w := range popupWindows {
		if !w.closed {
			result = append(result, w)
		}
	}
	return result
}
