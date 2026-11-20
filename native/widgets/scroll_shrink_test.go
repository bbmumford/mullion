package widgets

// A VirtualListView whose item count shrinks while a scroll position from the
// taller content is still in effect must not index past the new items. This
// reproduces the haystack tab-switch crash: scroll a long list deep, then hand
// the same list a much shorter data set; without clamping, the retained anchor
// (index deep into the old list) is fed to itemHeightFn against the few new
// items and panics.

import (
	"testing"

	"github.com/bbmumford/mullion/native"

	. "github.com/bbmumford/mullion/native"
)

func TestVirtualListShrinkNoPanic(t *testing.T) {
	initFontsOnce.Do(native.InitFontSubsystem)

	scope := new(int)
	listKey := new(int)
	count := 60
	maxIdx := -1

	frame := func(scroll f32) {
		native.GetHost().WindowSize = Vec2{400, 200}
		native.GetInputState().MousePoint = Vec2{200, 100} // over the list
		native.GetFrameInput().Mouse = 0
		native.GetFrameInput().Scroll = Vec2{0, scroll}
		native.GetFrameInput().Motion = Vec2{}
		native.GetFrameInput().Key = 0
		native.GetFrameInput().Text = ""
		native.RunFrameFn(func() {
			native.ModAttrs(func(a *native.AttrSet) { a.Animations = 0 })
			native.ContainerWithKey(scope, Attrs(Viewport), func() {
				see := func(i int) {
					if i > maxIdx {
						maxIdx = i
					}
					if i < 0 || i >= count {
						t.Fatalf("virtual list touched index %d, count %d", i, count)
					}
				}
				VirtualListView(listKey, count,
					func(i int) any { return i },
					func(i int, w f32) f32 { see(i); return 20 },
					func(i int, w f32) { see(i) },
				)
			})
		})
	}

	frame(0)       // settle: learn the width/size
	for range 12 { // wheel down to the bottom of the 60-item list
		frame(500)
	}
	deepest := maxIdx

	// Shrink, but keep the list taller than the viewport, so the retained
	// offset only clamps down a little — under the re-anchor threshold
	// (size*2). The widget then keeps its deep anchor (index ~50) while the
	// offset drops just below it: the exact case that fell through to
	// itemHeightFn out of range.
	count = 45
	for range 3 {
		frame(0)
	}

	t.Logf("deepest index before shrink: %d; count after shrink: %d", deepest, count)
}
