//go:build linux

package waylandbackend

import (
	wos "github.com/bbmumford/mullion/native/internal/wayland/os"
	"github.com/bbmumford/mullion/native/internal/wayland/wl"
	"github.com/bbmumford/mullion/native/internal/wayland/wlclient"
	xkb "github.com/bbmumford/mullion/native/internal/wayland/xkbcommon"
	"time"

	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/internal/qwerty"
	g "go.hasen.dev/generic"
)

// Keyboard input. The compositor hands us an xkb keymap over an fd; we compile it
// with libxkbcommon (via neurlang's purego binding — no cgo) and keep an xkb state
// updated from wl_keyboard.modifiers. Per key we get the keysym (an X11 keysym
// value, same as the X11 backend uses) for shortcut mapping, and the UTF-32
// codepoint for typed text. Sampled into the mullion globals (sample, not queue).

var (
	seatModifierMask uint32
	keyboard         *wl.Keyboard
	xkbContext       *xkb.Context
	xkbKeymap        *xkb.Keymap
	xkbState         *xkb.State
)

// Bits of the serialized xkb modifier mask (standard keymaps use the X11
// convention, matching x11backend).
const (
	wlModShift = 1 << 0
	wlModCtrl  = 1 << 2
	wlModAlt   = 1 << 3
	wlModSuper = 1 << 6
)

// ensureKeyboard attaches a wl_keyboard listener once the seat advertises one.
func ensureKeyboard() {
	if keyboard != nil {
		return
	}
	if xkbContext == nil {
		xkbContext = xkb.ContextNew(xkb.ContextNoFlags)
	}
	kb, err := seat.GetKeyboard()
	if err != nil {
		return
	}
	keyboard = kb
	wlclient.KeyboardAddListener(keyboard, h)
}

// HandleKeyboardKeymap compiles the keymap the compositor sends over an fd.
func (*handler) HandleKeyboardKeymap(ev wl.KeyboardKeymapEvent) {
	repeatKey.window = nil
	if ev.FdError != nil {
		return
	}
	defer wos.Close(int(ev.Fd))
	if ev.Format != xkb.KeymapFormatTextV1 || xkbContext == nil {
		return
	}
	data, err := wos.Mmap(int(ev.Fd), 0, int(ev.Size), wos.ProtRead, wos.MapPrivate)
	if err != nil {
		return
	}
	defer wos.Munmap(data)

	km := xkbContext.KeymapNewFromString(data, xkb.KeymapFormatTextV1, 0)
	if km == nil {
		perfLog("[wl] keymap compile failed")
		return
	}
	st := km.StateNew()
	if st == nil {
		return
	}
	xkbKeymap, xkbState = km, st
}

// HandleKeyboardModifiers feeds the serialized modifier state into xkb (so text
// reflects shift/caps/layout) and into the mullion modifier flags.
func (*handler) HandleKeyboardModifiers(ev wl.KeyboardModifiersEvent) {
	seatModifierMask = ev.ModsDepressed | ev.ModsLatched
	if xkbState != nil {
		xkbState.UpdateMask(ev.ModsDepressed, ev.ModsLatched, ev.ModsLocked, 0, 0, ev.Group)
	}
	w := keyboardWindow
	if w == nil || w.closed {
		return
	}
	leave := enterWindow(w)
	defer leave()

	if xkbState == nil {
		return
	}
	updateModifiers(ev.ModsDepressed | ev.ModsLatched)
	currentWindow.dirty = true
	wlDebug("modifiers: depressed=%#x latched=%#x locked=%#x -> mullion mods=%04b",
		ev.ModsDepressed, ev.ModsLatched, ev.ModsLocked, native.GetInputState().Modifiers)
}

func (*handler) HandleKeyboardKey(ev wl.KeyboardKeyEvent) {
	if ev.State == wl.KeyboardKeyStateReleased && repeatKey.code == ev.Key+8 {
		repeatKey.window = nil
	}
	w := keyboardWindow
	if w == nil || w.closed {
		return
	}
	leave := enterWindow(w)
	defer leave()

	activationSerial, activationSource = ev.Serial, w
	lastSerial = ev.Serial // for set_selection (copy via Ctrl+C)
	if xkbState == nil {
		return
	}
	code := ev.Key + 8 // evdev keycode -> xkb keycode
	down := ev.State != wl.KeyboardKeyStateReleased
	onKey(code, xkbState.KeyGetOneSym(code), down)
	if down && repeatRate > 0 && xkbKeymap != nil && xkbKeymap.KeyRepeats(code) {
		repeatKey.window, repeatKey.code = w, code
		repeatKey.next = time.Now().Add(repeatDelay)
	}
	currentWindow.dirty = true
	wlDebug("key: evdev=%d down=%v (mods now %04b)", ev.Key, down, native.GetInputState().Modifiers)
}

func (*handler) HandleKeyboardEnter(ev wl.KeyboardEnterEvent) {
	w := windowsBySurface[ev.Surface]
	if w == nil || w.closed {
		return
	}
	keyboardWindow = w
	activationSerial, activationSource = ev.Serial, w
	leave := enterWindow(w)
	defer leave()

	// Keyboard focus is what "active window" means on Wayland; apps read it
	// via Host.WindowFocused (e.g. to throttle ambient animation while inactive).
	native.GetHost().WindowFocused = true
	updateModifiers(seatModifierMask)
	currentWindow.dirty = true
	wlDebug("keyboard enter %q", w.winTitle)
}

// HandleKeyboardLeave: focus lost — drop held keys so none stick. Composition
// is also cleared by text-input leave (which tracks the same focus); clear
// here too so a compositor that omits text-input leave cannot leave a stale
// underline.
func (*handler) HandleKeyboardLeave(ev wl.KeyboardLeaveEvent) {
	if repeatKey.window != nil && repeatKey.window.surface == ev.Surface {
		repeatKey.window = nil
	}
	w := windowsBySurface[ev.Surface]
	if w == nil || w.closed {
		return
	}
	if keyboardWindow == w {
		keyboardWindow = nil
	}
	leave := enterWindow(w)
	defer leave()

	native.GetHost().WindowFocused = false
	native.GetInputState().DownKeys = native.GetInputState().DownKeys[:0]
	native.GetInputState().Modifiers = 0
	clearComposition()
	currentWindow.dirty = true
	wlDebug("keyboard leave %q", w.winTitle)
}

// Repeat uses compositor timing and the keymap's per-key repeat policy.
var repeatRate int32 = 25
var repeatDelay = 600 * time.Millisecond
var repeatKey struct {
	window *windowState
	code   uint32
	next   time.Time
}

func (*handler) HandleKeyboardRepeatInfo(ev wl.KeyboardRepeatInfoEvent) {
	repeatRate = min(max(ev.Rate, 0), 1000)
	repeatDelay = time.Duration(max(ev.Delay, 0)) * time.Millisecond
	if repeatRate == 0 {
		repeatKey.window = nil
	}
}

func pumpKeyboardRepeat(now time.Time) {
	w := repeatKey.window
	if w == nil {
		return
	}
	if repeatRate <= 0 || w != keyboardWindow || w.closed || w.hidden || w.closing || xkbState == nil {
		repeatKey.window = nil
		return
	}
	if now.Before(repeatKey.next) {
		return
	}
	interval := time.Second / time.Duration(repeatRate)
	repeatKey.next = repeatKey.next.Add(interval)
	// Input is sampled once per frame; do not burst a backlog after a stall.
	if !repeatKey.next.After(now) {
		repeatKey.next = now.Add(interval)
	}
	leave := enterWindow(w)
	onKey(repeatKey.code, xkbState.KeyGetOneSym(repeatKey.code), true)
	w.dirty = true
	leave()
}

// onKey maps a key event to a mullion key code and, for printable presses, the
// typed text. Mirrors x11backend.onKey. The writing block resolves by evdev
// position — KeyW is the physical key at the US-QWERTY W position no matter
// the layout; the layout still drives the typed text below. Other keys
// resolve by keysym.
//
// While an IME composition is active, editing/navigation keys and typed text
// belong to the IME (Cocoa B1 hasMarkedText / Win32 VK_PROCESSKEY). text-input-v3
// has no per-key "consumed" flag, so we gate on non-empty Composition
// (textInputConsumesKeys). Committed text arrives via commit_string and is
// merged into pendingText on done.
//
// Do NOT suppress xkb→utf32 merely because text-input-v3 is enabled. GNOME
// Mutter (and similar) only sends commit_string for IME-routed text; ordinary
// latin/digit keys arrive solely on wl_keyboard.key with no commit_string.
// Gating on textInputEnabled left TextInput fields read-only without an IME
// (go-shirei#15). While composing, the early return above already skips xkb
// text — that is enough to avoid double-insert with preedit/commit_string.
func onKey(code, keysym uint32, down bool) {
	kc := qwerty.FromScan(uint16(code - 8)) // xkb keycode -> evdev
	if kc == native.KeyCodeNone {
		kc = mapKeysym(keysym)
	}
	composing := textInputConsumesKeys()
	if kc != native.KeyCodeNone {
		if down {
			if !composing {
				native.GetFrameInput().Key = kc
			}
			g.SliceAddUniq(&native.GetInputState().DownKeys, kc)
		} else {
			g.SliceRemove(&native.GetInputState().DownKeys, kc)
		}
	}
	if !down || composing {
		return
	}
	// Suppress text for shortcut combos and control characters (delivered as Key).
	if native.GetInputState().Modifiers&(native.ModCtrl|native.ModCmd|native.ModAlt) != 0 {
		return
	}
	// Plain typing: xkb utf32. Accumulate so multi-key frames keep all
	// characters (assign would drop earlier ones, same class of bug as Win32 W0).
	if r := rune(xkbState.KeyGetUtf32(code)); r >= 0x20 && r != 0x7f {
		appendPendingText(string(r))
	}
}

func updateModifiers(mask uint32) {
	var m native.Modifiers
	if mask&wlModShift != 0 {
		m |= native.ModShift
	}
	if mask&wlModCtrl != 0 {
		m |= native.ModCtrl
	}
	if mask&wlModAlt != 0 {
		m |= native.ModAlt
	}
	if mask&wlModSuper != 0 {
		m |= native.ModSuper
	}
	native.GetInputState().Modifiers = m

	syncModKey(m, native.ModShift, native.KeyShift)
	syncModKey(m, native.ModCtrl, native.KeyCtrl)
	syncModKey(m, native.ModAlt, native.KeyAlt)
	syncModKey(m, native.ModSuper, native.KeySuper)
}

func syncModKey(m, bit native.Modifiers, k native.KeyCode) {
	if m&bit != 0 {
		g.SliceAddUniq(&native.GetInputState().DownKeys, k)
	} else {
		g.SliceRemove(&native.GetInputState().DownKeys, k)
	}
}

// X11 keysym constants (xkb uses the same values). Mirrors x11backend; a shared
// keysym->KeyCode helper is a candidate cleanup.
const (
	xkBackSpace = 0xff08
	xkTab       = 0xff09
	xkReturn    = 0xff0d
	xkEscape    = 0xff1b
	xkHome      = 0xff50
	xkLeft      = 0xff51
	xkUp        = 0xff52
	xkRight     = 0xff53
	xkDown      = 0xff54
	xkPrior     = 0xff55 // Page Up
	xkNext      = 0xff56 // Page Down
	xkEnd       = 0xff57
	xkDelete    = 0xffff
	xkSpace     = 0x0020
	xkF1        = 0xffbe
	xkF12       = 0xffc9
)

func mapKeysym(ks uint32) native.KeyCode {
	switch ks {
	case xkLeft:
		return native.KeyLeft
	case xkRight:
		return native.KeyRight
	case xkUp:
		return native.KeyUp
	case xkDown:
		return native.KeyDown
	case xkReturn:
		return native.KeyEnter
	case xkEscape:
		return native.KeyEscape
	case xkBackSpace:
		return native.KeyDeleteBackward
	case xkDelete:
		return native.KeyDeleteForward
	case xkHome:
		return native.KeyHome
	case xkEnd:
		return native.KeyEnd
	case xkb.KeyInsert:
		return native.KeyInsert
	case xkPrior:
		return native.KeyPageUp
	case xkNext:
		return native.KeyPageDown
	case xkTab:
		return native.KeyTab
	case xkSpace:
		return native.KeySpace
	}
	if ks >= xkF1 && ks <= xkF12 {
		return native.KeyF1 + native.KeyCode(ks-xkF1)
	}
	if ks >= 'a' && ks <= 'z' {
		return native.KeyA + native.KeyCode(ks-'a')
	}
	if ks >= 'A' && ks <= 'Z' {
		return native.KeyA + native.KeyCode(ks-'A')
	}
	if ks >= '0' && ks <= '9' {
		return native.Key0 + native.KeyCode(ks-'0')
	}
	return native.KeyCodeNone
}
