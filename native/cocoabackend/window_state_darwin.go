//go:build darwin && !ios

package cocoabackend

import (
	"unsafe"

	"github.com/bbmumford/mullion/native"
	"github.com/ebitengine/purego/objc"
)

type windowState struct {
	ui                     *native.UI
	unsubscribe            func()
	ready, closing, closed bool
	rendering              bool
	owner                  *windowState
	popupID                native.PopupWindow
	popupFloating          bool
	popupFocus             bool
	popupNonactivating     bool
	popupExactPosition     bool
	popupTopLevel          bool
	onClose                func()
	clock                  objc.ID
	surfacePool            []ioSurface
	lastPresented          unsafe.Pointer
	presentW               int
	presentH               int
	havePresented          bool

	lastPresentedHash uint64
	presentDeferred   bool
	gpuSubmitSeq      uint64
	gpuPresentedSeq   uint64

	frameSurfaces      []native.Surface
	frameGlyphRuns     []native.GlyphRun
	frameGlyphsAdded   []native.GlyphKey
	frameGlyphsEvicted []native.GlyphKey
	lastProducedW      float32
	lastProducedH      float32
	haveFrame          bool
	pendingText        string
	pendingPaste       string
	hasPendingPaste    bool
	frameHash          uint64
	softRenderer       native.SoftRenderer

	gWindow       objc.ID
	gView         objc.ID
	gContentLayer objc.ID
	gWantsFrame   bool

	markedStr              string
	markedSel              nsRange
	loggedReplacementRange bool
	accessByID             map[uint64]*accessElement
	accessRoots            []objc.ID
	accessOrder            []objc.ID
	accessFocused          objc.ID
	accessPending          []native.AccessAction
	frameFn                native.FrameFn
}

var mainWindow = newWindowState(native.ActiveUI())
var currentWindow = mainWindow
var windowsByObject = map[objc.ID]*windowState{}
var callbackDepth int
var framesInProgress int
var surfaceWindows = map[unsafe.Pointer]*windowState{}

func newWindowState(ui *native.UI) *windowState {
	return &windowState{ui: ui, gWantsFrame: true, accessByID: make(map[uint64]*accessElement)}
}

// AppKit callbacks are serialized on the main thread, including reentrant IME
// and accessibility callbacks. Only the outermost callback takes the frame lock.
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

func enterView(self objc.ID) func() {
	w := windowsByObject[self]
	if w == nil || w.closed {
		return nil
	}
	return enterWindow(w)
}
