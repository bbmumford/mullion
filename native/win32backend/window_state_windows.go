package win32backend

import (
	"syscall"
	"unsafe"

	"github.com/bbmumford/mullion/native"
)

type windowState struct {
	ui                     *native.UI
	unsubscribe            func()
	owner                  *windowState
	popupID                native.PopupWindow
	onClose                func()
	ready, closing, closed bool
	access                 *windowAccess

	winTitle         string
	winW             int
	winH             int
	winTransparent   bool
	winUndecorated   bool
	winFloating      bool
	winFocus         bool
	winTopLevel      bool
	winNonactivating bool
	winExactPosition bool
	frameFn          native.FrameFn

	hwnd syscall.Handle

	// DIB present surface: top-down 32bpp BGRA selected into a memory DC.
	// SoftRenderer writes dibBuf; presentDC sends it to the window.
	memDC    syscall.Handle
	dibBM    syscall.Handle
	dibOldBM syscall.Handle
	dibBuf   []byte
	dibW     int
	dibH     int
	dibBits  unsafe.Pointer

	softRenderer native.SoftRenderer

	dirty      bool // a new frame must be produced before the next present
	haveFrame  bool
	wantsFrame bool // last frame asked to be re-run (animation/async work)
	timerOn    bool

	// Content-hash present skip (mirrors cocoabackend / androidbackend).
	lastPresentedHash uint64
	havePresented     bool
	presentW          int
	presentH          int

	pendingHi   uint16 // pending UTF-16 high surrogate from WM_CHAR
	pendingText string // committed text to deliver on the next frame

	layeredDC, layeredBM, layeredOldBM uintptr
	layeredPixels                      []byte
	layeredW, layeredH                 int
	presentingLayered                  bool
}

var mainWindow = newWindowState(native.ActiveUI())
var currentWindow = mainWindow
var windowsByHandle = map[uintptr]*windowState{}
var creatingWindow *windowState
var callbackDepth, framesInProgress int
var winQuiet bool
var winBackground bool
var hinstance syscall.Handle
var wndProcCB = syscall.NewCallback(wndProc)

func newWindowState(ui *native.UI) *windowState {
	return &windowState{ui: ui, dirty: true, access: newWindowAccess()}
}

// Native callbacks can reenter while Windows processes another window's event.
// Only the outermost callback takes the frame lock.
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
