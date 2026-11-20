package window

import (
	"testing"

	"github.com/bbmumford/mullion/native"
)

// Replay the synchronous native-position path through real UI input frames and
// the extension's deferred placement queue, including layout settling passes.
func TestDragMoveLocalPointerAndRelease(t *testing.T) {
	restore := native.EnterUI(native.NewUI())
	defer restore()
	position := native.Vec2{40, 80}
	native.GetHost().WindowSize = native.Vec2{300, 200}
	frame := func() {
		native.Container(native.Attrs(native.FixSize(200, 80)), func() {
			DragMove(&position)
			native.Label("Drag handle")
		})
	}
	run := func(mouse native.MouseAction, point, motion native.Vec2) {
		t.Helper()
		native.GetInputState().MouseButton = native.MousePrimary
		native.GetInputState().MousePoint = point
		native.GetFrameInput().Mouse = mouse
		native.GetFrameInput().Motion = motion
		native.RunFrameFnLocked(frame)
	}
	run(0, native.Vec2{60, 30}, native.Vec2{})
	run(native.MouseClick, native.Vec2{60, 30}, native.Vec2{})
	run(0, native.Vec2{80, 40}, native.Vec2{20, 10})
	if position != (native.Vec2{60, 90}) {
		t.Fatalf("drag or settling applies displacement incorrectly: %v", position)
	}
	// Native repositioning puts the grabbed point back under the cursor.
	run(0, native.Vec2{60, 30}, native.Vec2{-20, -10})
	if position != (native.Vec2{60, 90}) {
		t.Fatal("the window's own move becomes another drag step")
	}
	run(0, native.Vec2{67, 34}, native.Vec2{7, 4})
	if position != (native.Vec2{67, 94}) {
		t.Fatalf("second pointer step is lost: %v", position)
	}
	run(native.MouseRelease, native.Vec2{90, 50}, native.Vec2{23, 16})
	run(0, native.Vec2{110, 60}, native.Vec2{20, 10})
	if position != (native.Vec2{67, 94}) {
		t.Fatal("released pointer keeps moving the window")
	}
	c := config()
	if c.pendingPlacement != placeAt || c.pendingX != 67 || c.pendingY != 94 {
		t.Fatalf("drag does not queue the final native position: %v %d,%d", c.pendingPlacement, c.pendingX, c.pendingY)
	}
}
