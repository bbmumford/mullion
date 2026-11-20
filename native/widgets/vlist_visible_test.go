package widgets

// OutFirstVisible / OutLastVisible report the painted window.

import (
	"testing"

	"github.com/bbmumford/mullion/native"

	. "github.com/bbmumford/mullion/native"
)

func TestVirtualListOutVisibleRange(t *testing.T) {
	initFontsOnce.Do(native.InitFontSubsystem)

	scope := new(int)
	listKey := new(int)
	const itemCount = 100
	const rowH f32 = 20
	// viewport 200 → 10 rows painted when starting at top
	var firstVis, lastVis int
	var scrollY, maxScroll f32

	frame := func() {
		native.GetHost().WindowSize = Vec2{400, 200}
		native.GetInputState().MousePoint = Vec2{-1000, -1000}
		native.GetFrameInput().Mouse = 0
		native.GetFrameInput().Scroll = Vec2{}
		native.GetFrameInput().Motion = Vec2{}
		native.GetFrameInput().Key = 0
		native.GetFrameInput().Text = ""
		native.RunFrameFn(func() {
			native.ModAttrs(func(a *native.AttrSet) { a.Animations = 0 })
			native.ContainerWithKey(scope, Attrs(Viewport), func() {
				VirtualListViewExt(listKey, VirtualListAttrs{
					ItemCount: itemCount,
					ItemKey:   func(i int) any { return i },
					ItemHeight: func(i int, w f32) f32 {
						return rowH
					},
					ItemView:           func(i int, w f32) {},
					OutScrollOffset:    &scrollY,
					OutMaxScrollOffset: &maxScroll,
					OutFirstVisible:    &firstVis,
					OutLastVisible:     &lastVis,
				})
			})
		})
	}

	for range 4 {
		frame()
	}
	if firstVis != 0 {
		t.Fatalf("firstVis=%d want 0", firstVis)
	}
	// 200/20 = 10 rows; last inclusive index 9 (may be 9 or 10 if partial)
	if lastVis < 8 || lastVis > 12 {
		t.Fatalf("lastVis=%d want ~9", lastVis)
	}

	VirtualListView_ScrollToIndex(listKey, 40)
	for range 6 {
		frame()
	}
	if firstVis < 38 || firstVis > 42 {
		t.Fatalf("after ScrollToIndex(40) firstVis=%d want ~40", firstVis)
	}
	if lastVis < firstVis {
		t.Fatalf("lastVis=%d < firstVis=%d", lastVis, firstVis)
	}
}
