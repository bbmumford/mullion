package widgets

// A checkbox must toggle both ways. It used to share a base function with the
// radio button, whose press handler only ever assigns (*target = value), so a
// checkbox could turn on but never off. This drives a real press gesture
// (down then up over the widget) twice and pins that the bound bool flips each
// time.

import (
	"testing"

	"github.com/bbmumford/mullion/native"

	. "github.com/bbmumford/mullion/native"
)

func TestCheckBoxToggles(t *testing.T) {
	initFontsOnce.Do(native.InitFontSubsystem)

	scope := new(int)
	on := false
	view := func() { CheckBox(&on, "toggle me") }

	frame := func(mouse Vec2, action MouseAction) {
		native.GetHost().WindowSize = Vec2{600, 400}
		native.GetInputState().MousePoint = mouse
		native.GetFrameInput().Mouse = action
		native.GetFrameInput().Scroll = Vec2{}
		native.GetFrameInput().Motion = Vec2{}
		native.GetFrameInput().Key = 0
		native.GetFrameInput().Text = ""
		native.RunFrameFn(func() {
			native.ModAttrs(func(a *native.AttrSet) { a.Animations = 0 })
			native.ContainerWithKey(scope, Attrs(Viewport), view)
		})
	}

	at := Vec2{6, 9}          // inside the checkbox row at top-left
	off := Vec2{-1000, -1000} // parked, no hover
	frame(off, 0)             // settle: establish the widget's rect

	// a full press gesture: down then up, both over the widget
	press := func() {
		frame(at, MouseClick)
		frame(at, MouseRelease)
	}

	press()
	if !on {
		t.Fatalf("after first press: want on=true, got false — checkbox never turned on")
	}
	press()
	if on {
		t.Fatalf("after second press: want on=false, got true — checkbox can turn on but not off")
	}
}
