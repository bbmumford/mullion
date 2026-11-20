package jsbackend

import (
	"slices"
	"testing"

	"github.com/bbmumford/mullion/native"
)

func clearFrameInput() {
	*native.GetFrameInput() = native.FrameInputData{}
}

// TestWheelPixelModePositiveIsScrollDown: browser WheelEvent positive deltaY
// means scroll down; that must map to positive FrameInput.Scroll.Y (same as
// X11 wheel-down). Negating to "match Cocoa" inverts trackpad/page feel.
func TestWheelPixelModePositiveIsScrollDown(t *testing.T) {
	native.ResetInputSession()
	a := newInputAccum(false)

	a.wheel(0, 40, 0) // deltaMode pixel
	a.sample()
	if got := native.GetFrameInput().Scroll; got != (native.Vec2{0, 40}) {
		t.Fatalf("Scroll = %v, want {0, 40}", got)
	}

	clearFrameInput()
	a.wheel(-12, -8, 0)
	a.sample()
	if got := native.GetFrameInput().Scroll; got != (native.Vec2{-12, -8}) {
		t.Fatalf("Scroll = %v, want {-12, -8}", got)
	}
}

// TestWheelLineModeScales: deltaMode 1 (line) multiplies by 16 before sample.
func TestWheelLineModeScales(t *testing.T) {
	native.ResetInputSession()
	a := newInputAccum(false)

	a.wheel(0, 3, 1) // 3 lines → 48 px
	a.sample()
	if got := native.GetFrameInput().Scroll; got != (native.Vec2{0, 48}) {
		t.Fatalf("Scroll = %v, want {0, 48}", got)
	}
}

// TestWheelAccumulatesAcrossEvents: multiple wheel events before sample sum.
func TestWheelAccumulatesAcrossEvents(t *testing.T) {
	native.ResetInputSession()
	a := newInputAccum(false)

	a.wheel(0, 10, 0)
	a.wheel(0, 15, 0)
	a.sample()
	if got := native.GetFrameInput().Scroll; got != (native.Vec2{0, 25}) {
		t.Fatalf("Scroll = %v, want {0, 25}", got)
	}

	clearFrameInput()
	a.sample()
	if got := native.GetFrameInput().Scroll; got != (native.Vec2{}) {
		t.Fatalf("second sample Scroll = %v, want zero", got)
	}
}

// TestMotionAccumulatedFromSamplePoints pins FrameInput.Motion: each sample
// after the first reports the delta from the previous sample point. Drag-drop
// and other Motion consumers rely on this; the old shell never set Motion.
func TestMotionAccumulatedFromSamplePoints(t *testing.T) {
	native.ResetInputSession()
	a := newInputAccum(false)

	a.setPoint(10, 20)
	a.sample()
	if m := native.GetFrameInput().Motion; m != (native.Vec2{}) {
		t.Fatalf("first sample Motion = %v, want zero", m)
	}
	if p := native.GetInputState().MousePoint; p != (native.Vec2{10, 20}) {
		t.Fatalf("MousePoint = %v, want {10,20}", p)
	}

	clearFrameInput()
	a.setPoint(40, 50)
	a.sample()
	want := native.Vec2{30, 30}
	if m := native.GetFrameInput().Motion; m != want {
		t.Fatalf("Motion = %v, want %v", m, want)
	}
	if p := native.GetInputState().MousePoint; p != (native.Vec2{40, 50}) {
		t.Fatalf("MousePoint = %v, want {40,50}", p)
	}
}

// TestPrimaryReleaseOutsideCanvasStillDeliversRelease: pointer capture on the
// DOM is what ensures primaryUp arrives after a drag outside the canvas; the
// accumulator must not drop that release based on coordinates (regression for
// stuck IsActive / drag-drop when the shell forgets to feed up).
func TestPrimaryReleaseOutsideCanvasStillDeliversRelease(t *testing.T) {
	native.ResetInputSession()
	a := newInputAccum(false)

	a.setPoint(100, 100)
	a.primaryDown()
	a.sample()
	if native.GetFrameInput().Mouse != native.MouseClick {
		t.Fatalf("down: Mouse = %v, want MouseClick", native.GetFrameInput().Mouse)
	}

	clearFrameInput()
	// Pointer moved outside the logical canvas; up still arrives (capture).
	a.setPoint(-20, 500)
	a.primaryUp()
	a.sample()
	if native.GetFrameInput().Mouse != native.MouseRelease {
		t.Fatalf("up outside: Mouse = %v, want MouseRelease", native.GetFrameInput().Mouse)
	}
	if p := native.GetInputState().MousePoint; p != (native.Vec2{-20, 500}) {
		t.Fatalf("MousePoint after outside up = %v", p)
	}
}

// TestCompositionSuppressesFrameKey: during IME composition, Backspace/arrows
// must not set FrameInput.Key (textinput trusts the backend and would delete
// from the document while the IME still owns the preedit).
func TestCompositionSuppressesFrameKey(t *testing.T) {
	native.ResetInputSession()
	a := newInputAccum(false)

	a.compositionStart()
	a.compositionUpdate("に")
	a.setMods(false, false, false, false)
	if a.keyDown("Backspace") {
		t.Fatal("Backspace during composition should not preventDefault via key path")
	}
	a.sample()
	if native.GetFrameInput().Key != 0 {
		t.Fatalf("Key = %v during composition, want 0", native.GetFrameInput().Key)
	}
	if native.GetInputState().Composition != "に" {
		t.Fatalf("Composition = %q, want に", native.GetInputState().Composition)
	}

	clearFrameInput()
	a.compositionEnd("日")
	a.sample()
	if native.GetFrameInput().Key != 0 {
		t.Fatalf("Key after composition end = %v, want 0", native.GetFrameInput().Key)
	}
	if native.GetFrameInput().Text != "日" {
		t.Fatalf("Text = %q, want 日", native.GetFrameInput().Text)
	}
	if native.GetInputState().Composition != "" {
		t.Fatalf("Composition still set: %q", native.GetInputState().Composition)
	}

	// After composition, Backspace is a normal edit key again.
	clearFrameInput()
	a.keyDown("Backspace")
	a.sample()
	if native.GetFrameInput().Key != native.KeyDeleteBackward {
		t.Fatalf("Key = %v, want KeyDeleteBackward", native.GetFrameInput().Key)
	}
}

// TestSpaceDeliversAsFrameKey: Space must appear on FrameInput.Key for hotkeys
// (games jump, etc.). Typed space still also arrives as Text via beforeinput
// when the hidden field is focused; editdecode ignores KeySpace for insert.
// preventDefault is false so beforeinput can still deliver " " into text fields.
func TestSpaceDeliversAsFrameKey(t *testing.T) {
	native.ResetInputSession()
	a := newInputAccum(false)
	a.setMods(false, false, false, false)
	if a.keyDown("Space") {
		t.Fatal("Space must not preventDefault (would block beforeinput space)")
	}
	a.sample()
	if native.GetFrameInput().Key != native.KeySpace {
		t.Fatalf("Key = %v, want KeySpace", native.GetFrameInput().Key)
	}
}

// TestMetaMapsToKeyCommandOnAppleHost: Cocoa uses KeyCommand for ⌘; web-Mac
// must put KeyCommand in DownKeys, not KeySuper.
func TestMetaMapsToKeyCommandOnAppleHost(t *testing.T) {
	if got := mapCode("MetaLeft", true); got != native.KeyCommand {
		t.Fatalf("apple MetaLeft = %v, want KeyCommand", got)
	}
	if got := mapCode("MetaRight", true); got != native.KeyCommand {
		t.Fatalf("apple MetaRight = %v, want KeyCommand", got)
	}
	if got := mapCode("MetaLeft", false); got != native.KeySuper {
		t.Fatalf("non-apple MetaLeft = %v, want KeySuper", got)
	}

	native.ResetInputSession()
	a := newInputAccum(true)
	a.setMods(false, false, false, true)
	a.keyDown("MetaLeft")
	a.sample()
	if !slices.Contains(native.GetInputState().DownKeys, native.KeyCommand) {
		t.Fatalf("DownKeys = %v, want KeyCommand", native.GetInputState().DownKeys)
	}
	if native.GetInputState().Modifiers&native.ModCmd == 0 {
		t.Fatal("mods missing ModCmd on apple meta")
	}
	if native.GetInputState().Modifiers&native.ModSuper != 0 {
		t.Fatal("mods must not set ModSuper on apple meta")
	}
}

// TestCmdADeliversEditShortcut: primary-mod letter chords set FrameInput.Key
// and request preventDefault so the browser does not select-all the page.
func TestCmdADeliversEditShortcut(t *testing.T) {
	native.ResetInputSession()
	a := newInputAccum(true)
	a.setMods(false, false, false, true) // meta → ModCmd
	if !a.keyDown("KeyA") {
		t.Fatal("Cmd+A should preventDefault")
	}
	a.sample()
	if native.GetFrameInput().Key != native.KeyA {
		t.Fatalf("Key = %v, want KeyA", native.GetFrameInput().Key)
	}
	if native.GetInputState().Modifiers&native.ModCmd == 0 {
		t.Fatal("expected ModCmd")
	}
}
