//go:build windows

package mainthread

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Wake asks the main thread to call Drain, by posting a message to a
// message-only window this package owns.
//
// A plain PostThreadMessage would reach the pump but no window procedure, and a
// pump that does not recognise the message drops it. A window of our own means
// any pump in the process delivers the wake, because DispatchMessage routes a
// message to the procedure of the window it was posted to -- the tray's pump,
// the Go-drawn surface's pump and a webview's pump all call it.
func Wake() {
	if w := windowsWake(); w != nil {
		w()
	}
}

const wmMainthreadDrain = 0x0400 + 0x0A11 // WM_APP + a value unlikely to collide

var windowsWake = sync.OnceValue(func() func() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	createWindowEx := user32.NewProc("CreateWindowExW")
	defWindowProc := user32.NewProc("DefWindowProcW")
	registerClass := user32.NewProc("RegisterClassExW")
	postMessage := user32.NewProc("PostMessageW")

	proc := windows.NewCallback(func(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
		if msg == wmMainthreadDrain {
			Drain()
			return 0
		}
		r, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wparam, lparam)
		return r
	})

	name, err := windows.UTF16PtrFromString("mullion.mainthread")
	if err != nil {
		return nil
	}
	// A message-only window (HWND_MESSAGE as parent) never appears on screen
	// and takes part in no window management; it exists to carry the wake.
	type wndClassEx struct {
		size, style                        uint32
		wndProc                            uintptr
		clsExtra, wndExtra                 int32
		instance, icon, cursor, background uintptr
		menuName, className                *uint16
		iconSm                             uintptr
	}
	class := wndClassEx{
		wndProc:   proc,
		className: name,
	}
	class.size = uint32(unsafe.Sizeof(class))
	if atom, _, _ := registerClass.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return nil
	}
	const hwndMessage = ^uintptr(2) // (HWND)-3
	hwnd, _, _ := createWindowEx.Call(
		0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)),
		0, 0, 0, 0, 0, hwndMessage, 0, 0, 0)
	if hwnd == 0 {
		return nil
	}
	return func() { postMessage.Call(hwnd, wmMainthreadDrain, 0, 0) }
})
