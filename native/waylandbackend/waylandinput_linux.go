//go:build linux

package waylandbackend

import (
	"github.com/bbmumford/mullion/native/internal/wayland/wl"
	"github.com/bbmumford/mullion/native/internal/wayland/wlclient"

	"github.com/bbmumford/mullion/native"
)

// Pointer input. Wayland delivers pointer coordinates in surface-local *logical*
// units (already scaled), which is exactly what mullion wants — so, unlike X11
// (device pixels ÷ scale), we pass them straight through. Buttons arrive as Linux
// evdev codes; scroll as wl_pointer.axis. Input is sampled into the mullion globals
// and consumed by the next frame (sample, not queue).

// Linux evdev button codes (linux/input-event-codes.h).
const (
	btnLeft   = 0x110
	btnRight  = 0x111
	btnMiddle = 0x112
)

// wheelScale converts a wl_pointer.axis value (logical pixels) into mullion scroll
// points. Tunable to taste; ~2 lands close to the X11 backend's per-notch feel.
const wheelScale = 2

// --- seat -------------------------------------------------------------------

func (*handler) HandleSeatCapabilities(ev wl.SeatCapabilitiesEvent) {
	if ev.Capabilities&wl.SeatCapabilityPointer != 0 && pointer == nil {
		if p, err := seat.GetPointer(); err == nil {
			pointer = p
			wlclient.PointerAddListener(pointer, h)
		}
	}
	if ev.Capabilities&wl.SeatCapabilityKeyboard != 0 {
		ensureKeyboard()
	}
	ensureRelativePointer()
	ensureDataDevice() // clipboard needs the seat
	ensureTextInput()  // IME needs the seat
}

func (*handler) HandleSeatName(wl.SeatNameEvent) {}

// --- pointer ----------------------------------------------------------------

func (*handler) HandlePointerEnter(ev wl.PointerEnterEvent) {
	w := windowsBySurface[ev.Surface]
	if w == nil || w.closed {
		return
	}
	leave := enterWindow(w)
	defer leave()

	pointerWindow = w
	pointerSerial = ev.Serial
	applyCursor(ev.Serial) // Wayland needs us to set the cursor on every enter
	native.GetInputState().MousePoint = native.Vec2{ev.SurfaceX, ev.SurfaceY}
	recordAbsolutePointer(w, ev.SurfaceX, ev.SurfaceY)
	tracePointer(w, "wl.pointer.enter", "serial=%d local=[%g %g]", ev.Serial, ev.SurfaceX, ev.SurfaceY)
	currentWindow.dirty = true
}

func (*handler) HandlePointerLeave(ev wl.PointerLeaveEvent) {
	w := windowsBySurface[ev.Surface]
	if w == nil || w.closed {
		return
	}
	leave := enterWindow(w)
	defer leave()

	if pointerWindow == w {
		pointerWindow = nil
	}
	tracePointer(w, "wl.pointer.leave", "serial=%d", ev.Serial)
	pointerSerial = ev.Serial
	native.GetInputState().MousePoint = native.Vec2{-1, -1} // off-window: nothing hovers
	currentWindow.dirty = true
}

func (*handler) HandlePointerMotion(ev wl.PointerMotionEvent) {
	w := pointerWindow
	if w == nil || w.closed {
		return
	}
	leave := enterWindow(w)
	defer leave()

	setMouse(ev.SurfaceX, ev.SurfaceY)
	recordAbsolutePointer(w, ev.SurfaceX, ev.SurfaceY)
	tracePointer(w, "wl.pointer.motion", "protocol_ms=%d local=[%g %g]", ev.Time, ev.SurfaceX, ev.SurfaceY)
	currentWindow.dirty = true
}

func (*handler) HandlePointerButton(ev wl.PointerButtonEvent) {
	w := pointerWindow
	if w == nil || w.closed {
		return
	}
	leave := enterWindow(w)
	defer leave()

	wlDebug("pointer button=%d state=%d (mods=%04b)", ev.Button, ev.State, native.GetInputState().Modifiers)
	pointerSerial = ev.Serial
	activationSerial, activationSource = ev.Serial, w
	lastSerial = ev.Serial // for set_selection (copy via a button)
	// A left press in the edge border starts an interactive resize instead of a
	// click (CSD). The compositor then grabs the pointer until the resize ends.
	tracePointer(w, "wl.pointer.button.before", "serial=%d protocol_ms=%d button=%d state=%d", ev.Serial, ev.Time, ev.Button, ev.State)
	if ev.Button == btnLeft {
		w.pointerPrimaryDown = ev.State == wl.PointerButtonStatePressed
	}
	if ev.Button == btnLeft && ev.State == wl.PointerButtonStatePressed {
		w.pointerTravel = native.Vec2{}
		w.pointerHasRelative = false
		w.pointerPress = w.pointerAbsolute
		tracePointer(w, "wl.pointer.press.reset", "")
	}
	if ev.Button == btnLeft && ev.State == wl.PointerButtonStatePressed && tryStartResize(ev.Serial) {
		return
	}
	var btn native.MouseButton
	switch ev.Button {
	case btnLeft:
		btn = native.MousePrimary
	case btnRight:
		btn = native.MouseSecondary
	case btnMiddle:
		btn = native.MouseTertiary
	default:
		return
	}
	// Click-commit: accept preedit into the document before the click moves
	// focus (Cocoa B4 / Win32 W5). drawFrame inside may present one extra frame.
	if ev.State == wl.PointerButtonStatePressed {
		commitImeBeforeInterruption()
	}
	native.GetInputState().MouseButton = btn
	if ev.State == wl.PointerButtonStatePressed {
		native.GetFrameInput().Mouse = native.MouseClick
	} else {
		native.GetFrameInput().Mouse = native.MouseRelease
	}
	currentWindow.dirty = true
}

func (*handler) HandlePointerAxis(ev wl.PointerAxisEvent) {
	w := pointerWindow
	if w == nil || w.closed {
		return
	}
	leave := enterWindow(w)
	defer leave()

	switch ev.Axis {
	case wl.PointerAxisVerticalScroll:
		scroll(0, ev.Value*wheelScale) // Wayland: +value = down, matches mullion
	case wl.PointerAxisHorizontalScroll:
		scroll(ev.Value*wheelScale, 0)
	}
	currentWindow.dirty = true
}

// The remaining wl_pointer events aren't needed yet; satisfy the listener.
func (*handler) HandlePointerFrame(wl.PointerFrameEvent)               {}
func (*handler) HandlePointerAxisSource(wl.PointerAxisSourceEvent)     {}
func (*handler) HandlePointerAxisStop(wl.PointerAxisStopEvent)         {}
func (*handler) HandlePointerAxisDiscrete(wl.PointerAxisDiscreteEvent) {}

// --- helpers ----------------------------------------------------------------

// setMouse records a new (logical) mouse position and accumulates this frame's
// motion delta, mirroring the other backends.
func setMouse(x, y float32) {
	np := native.Vec2{x, y}
	prev := native.GetInputState().MousePoint
	native.GetFrameInput().Motion = native.Vec2Add(native.GetFrameInput().Motion, native.Vec2Sub(np, prev))
	native.GetInputState().MousePoint = np
}

func scroll(dx, dy float32) {
	native.GetFrameInput().Scroll = native.Vec2Add(native.GetFrameInput().Scroll, native.Vec2{dx, dy})
}
