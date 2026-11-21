package jsbackend

import "github.com/bbmumford/mullion/native"

// inputAccum is the browser-independent input state machine for jsbackend.
// DOM (or tests) only feed events; sample() writes Host once per frame.
//
// Pointer capture is a DOM concern: once the shell delivers primaryUp after a
// drag outside the canvas, this accumulator always emits MouseRelease —
// release is not gated on coordinates.
type inputAccum struct {
	appleHost bool

	x, y           float64
	lastX, lastY   float64
	haveLastSample bool

	pendingClick   bool
	pendingRelease bool
	edgeButton     native.MouseButton
	held           bool
	heldButton     native.MouseButton

	scrollX, scrollY float64

	pendingText string
	pendingKey  native.KeyCode
	downKeys    map[native.KeyCode]bool
	mods        native.Modifiers
	composing   bool
	composition string
}

func newInputAccum(appleHost bool) *inputAccum {
	return &inputAccum{
		appleHost: appleHost,
		downKeys:  make(map[native.KeyCode]bool),
	}
}

func (a *inputAccum) setPoint(x, y float64) {
	a.x, a.y = x, y
}

func (a *inputAccum) primaryDown() {
	a.pendingClick = true
	a.held = true
	a.heldButton = native.MousePrimary
	a.edgeButton = native.MousePrimary
}

func (a *inputAccum) primaryUp() {
	a.pendingRelease = true
	a.held = false
	a.edgeButton = native.MousePrimary
}

// wheel accumulates browser wheel deltas. mode is WheelEvent.deltaMode:
// 0 = pixel, 1 = line, 2 = page.
func (a *inputAccum) wheel(dx, dy float64, mode int) {
	switch mode {
	case 1:
		dx *= 16
		dy *= 16
	case 2:
		dx *= 400
		dy *= 400
	}
	a.scrollX += dx
	a.scrollY += dy
}

func (a *inputAccum) setMods(shift, ctrl, alt, meta bool) {
	var m native.Modifiers
	if shift {
		m |= native.ModShift
	}
	if ctrl {
		m |= native.ModCtrl
	}
	if alt {
		m |= native.ModAlt
	}
	// metaKey is ⌘ on Apple hosts and the Windows key elsewhere. Match native
	// backends: Command → ModCmd only (not also Super), so exact-combo checks
	// like mods == PrimaryMod() succeed for Cmd+A.
	if meta {
		if a.appleHost {
			m |= native.ModCmd
		} else {
			m |= native.ModSuper
		}
	}
	a.mods = m
}

// keyDown records a physical key press. Returns whether the browser should
// preventDefault (so Tab/Enter/edit chords and navigation keys do not hit the
// page). While an IME composition is active, editing/navigation keys belong
// to the IME — FrameInput.Key is not set (same gate as Wayland/Cocoa).
func (a *inputAccum) keyDown(code string) (preventDefault bool) {
	k := mapCode(code, a.appleHost)
	if k == 0 {
		return false
	}
	a.downKeys[k] = true
	if a.composing {
		return false
	}
	if !deliversAsFrameKey(k, a.mods) {
		return false
	}
	a.pendingKey = k
	return shouldPreventBrowserDefault(k, a.mods)
}

func (a *inputAccum) keyUp(code string) {
	k := mapCode(code, a.appleHost)
	if k != 0 {
		delete(a.downKeys, k)
	}
}

func (a *inputAccum) compositionStart() {
	a.composing = true
	a.composition = ""
}

func (a *inputAccum) compositionUpdate(s string) {
	a.composition = s
}

func (a *inputAccum) compositionEnd(s string) {
	a.composing = false
	if s != "" {
		a.pendingText += s
	}
	a.composition = ""
}

// textInsert is beforeinput / committed characters outside composition.
func (a *inputAccum) textInsert(s string) {
	if a.composing || s == "" {
		return
	}
	a.pendingText += s
}

func (a *inputAccum) clearKeyboard() {
	for k := range a.downKeys {
		delete(a.downKeys, k)
	}
	a.pendingKey = 0
	a.pendingText = ""
	a.mods = 0
	a.composing = false
	a.composition = ""
}

// sample writes one frame of accumulated input into Host and clears edges.
// Call once per rAF tick before RunFrameFn. FrameInput must already be zero
// (core clears it at end of each pass).
func (a *inputAccum) sample() {
	in := native.GetInputState()
	fi := native.GetFrameInput()

	np := native.Vec2{float32(a.x), float32(a.y)}
	if a.haveLastSample {
		prev := native.Vec2{float32(a.lastX), float32(a.lastY)}
		fi.Motion = native.Vec2Add(fi.Motion, native.Vec2Sub(np, prev))
	}
	a.lastX, a.lastY = a.x, a.y
	a.haveLastSample = true
	in.MousePoint = np

	in.Modifiers = a.mods
	in.DownKeys = in.DownKeys[:0]
	for k := range a.downKeys {
		in.DownKeys = append(in.DownKeys, k)
	}
	if a.composing {
		in.Composition = a.composition
		n := len([]rune(a.composition))
		in.CompositionSel = [2]int{n, n}
	} else {
		in.Composition = ""
		in.CompositionSel = [2]int{}
	}

	// Prefer click over release when both arrive in one tick (quick click);
	// release is delivered on the following sample while held is already false.
	if a.pendingClick {
		fi.Mouse = native.MouseClick
		in.MouseButton = a.edgeButton
		a.pendingClick = false
	} else if a.pendingRelease {
		fi.Mouse = native.MouseRelease
		in.MouseButton = a.edgeButton
		a.pendingRelease = false
	} else if a.held {
		in.MouseButton = a.heldButton
	}

	if a.scrollX != 0 || a.scrollY != 0 {
		// WheelEvent is already mullion-space: positive deltaY = scroll down
		// (increase ScrollOffset). Do not negate — Cocoa negates AppKit deltas
		// because their sign differs from the web event.
		fi.Scroll = native.Vec2{float32(a.scrollX), float32(a.scrollY)}
		a.scrollX, a.scrollY = 0, 0
	}
	if a.pendingKey != 0 {
		fi.Key = a.pendingKey
		a.pendingKey = 0
	}
	if a.pendingText != "" {
		fi.Text = a.pendingText
		a.pendingText = ""
	}
}

// deliversAsFrameKey reports keys that must appear on FrameInput.Key (not only
// as typed text). Includes Space so hotkeys see KeySpace; printable letters
// without modifiers stay on the text path via beforeinput.
func deliversAsFrameKey(k native.KeyCode, mods native.Modifiers) bool {
	switch k {
	case native.KeyLeft, native.KeyRight, native.KeyUp, native.KeyDown,
		native.KeyEnter, native.KeyEscape, native.KeyHome, native.KeyEnd,
		native.KeyDeleteBackward, native.KeyDeleteForward,
		native.KeyPageUp, native.KeyPageDown, native.KeyTab,
		native.KeySpace,
		native.KeyF1, native.KeyF2, native.KeyF3, native.KeyF4,
		native.KeyF5, native.KeyF6, native.KeyF7, native.KeyF8,
		native.KeyF9, native.KeyF10, native.KeyF11, native.KeyF12:
		return true
	}
	// Modifier chords (shortcuts) deliver the letter key.
	if mods&(native.ModCtrl|native.ModCmd|native.ModSuper|native.ModAlt) != 0 {
		return k != 0
	}
	return false
}

func shouldPreventBrowserDefault(k native.KeyCode, mods native.Modifiers) bool {
	// Space is intentionally omitted: preventDefault on keydown suppresses
	// beforeinput " " into the hidden field, so typing a space would fail.
	// Page-scroll on Space when no text field is focused is acceptable.
	switch k {
	case native.KeyTab, native.KeyEnter,
		native.KeyLeft, native.KeyRight, native.KeyUp, native.KeyDown,
		native.KeyPageUp, native.KeyPageDown, native.KeyHome, native.KeyEnd:
		return true
	}
	if mods&(native.ModCtrl|native.ModCmd|native.ModSuper) != 0 && isEditShortcutKey(k) {
		return true
	}
	return false
}

func isEditShortcutKey(k native.KeyCode) bool {
	switch k {
	case native.KeyA, native.KeyC, native.KeyV, native.KeyX, native.KeyZ, native.KeyY:
		return true
	}
	return false
}
