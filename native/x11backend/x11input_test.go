//go:build linux || (darwin && x11darwin)

package x11backend

import (
	"testing"

	"github.com/bbmumford/mullion/native"
)

func resetMouseSample() {
	native.ResetInputSession()
	native.GetHost().WindowScale = 1
	native.GetInputState().MousePoint = native.Vec2{8, 9}
}

func TestPressAndReleaseInOneDrainKeepBothBits(t *testing.T) {
	resetMouseSample()

	mouseButton(native.MousePrimary, native.MouseClick, 8, 9)
	mouseButton(native.MousePrimary, native.MouseRelease, 18, 9)

	got := native.GetFrameInput().Mouse
	if got != native.MouseClick|native.MouseRelease {
		t.Fatalf("primary sample = %v, want click|release", got)
	}
	if p := native.GetInputState().MousePoint; p != (native.Vec2{18, 9}) {
		t.Fatalf("MousePoint = %v, want release position", p)
	}
	if m := native.GetFrameInput().Motion; m != (native.Vec2{10, 0}) {
		t.Fatalf("Motion = %v, want {10, 0}", m)
	}
	if native.GetInputState().MouseButton != native.MousePrimary {
		t.Fatalf("MouseButton = %v", native.GetInputState().MouseButton)
	}

	// The press was already sampled. The later release is only a release.
	*native.GetFrameInput() = native.FrameInputData{}
	mouseButton(native.MousePrimary, native.MouseRelease, 18, 14)
	if native.GetFrameInput().Mouse != native.MouseRelease {
		t.Fatalf("held release = %v, want MouseRelease", native.GetFrameInput().Mouse)
	}

	for _, btn := range []native.MouseButton{native.MouseSecondary, native.MouseTertiary} {
		*native.GetFrameInput() = native.FrameInputData{}
		mouseButton(btn, native.MouseClick, 4, 5)
		mouseButton(btn, native.MouseRelease, 4, 5)
		if native.GetFrameInput().Mouse != native.MouseClick|native.MouseRelease {
			t.Fatalf("button %v sample = %v", btn, native.GetFrameInput().Mouse)
		}
		if native.GetInputState().MouseButton != btn {
			t.Fatalf("MouseButton = %v, want %v", native.GetInputState().MouseButton, btn)
		}
	}

	// A second click in the same sample collapses onto the same two bits.
	*native.GetFrameInput() = native.FrameInputData{}
	mouseButton(native.MousePrimary, native.MouseClick, 1, 1)
	mouseButton(native.MousePrimary, native.MouseRelease, 1, 1)
	mouseButton(native.MousePrimary, native.MouseClick, 1, 1)
	mouseButton(native.MousePrimary, native.MouseRelease, 1, 1)
	if native.GetFrameInput().Mouse != native.MouseClick|native.MouseRelease {
		t.Fatalf("collapsed sample = %v", native.GetFrameInput().Mouse)
	}

	// A different button replaces the sample.
	*native.GetFrameInput() = native.FrameInputData{}
	mouseButton(native.MousePrimary, native.MouseClick, 1, 1)
	mouseButton(native.MouseTertiary, native.MouseClick, 2, 2)
	if native.GetFrameInput().Mouse != native.MouseClick || native.GetInputState().MouseButton != native.MouseTertiary {
		t.Fatalf("replaced mouse=%v button=%v", native.GetFrameInput().Mouse, native.GetInputState().MouseButton)
	}

	// Wheel deltas still accumulate and do not touch the button sample.
	*native.GetFrameInput() = native.FrameInputData{}
	scroll(0, wheelNotchPts)
	scroll(-wheelNotchPts, 0)
	if got := native.GetFrameInput().Scroll; got != (native.Vec2{-wheelNotchPts, wheelNotchPts}) {
		t.Fatalf("Scroll = %v", got)
	}
	if native.GetFrameInput().Mouse != 0 {
		t.Fatalf("wheel set Mouse = %v", native.GetFrameInput().Mouse)
	}
}
