package window

import (
	"math"

	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/internal/dragtrace"
)

type dragMoveKey struct{}

type dragMoveState struct {
	active, relative bool
	epoch            int64
	point, origin    native.Vec2
}

// DragMove makes the current container a primary-button window drag handle.
// Position is the caller's requested window origin in logical screen points;
// keep it in sync with other position controls. Call every frame inside the
// handle's Container, on the UI thread. It returns true when it requests a move.
//
// Wayland tracks pointer motion independently of asynchronous surface movement,
// including absolute tablet input. It requires a positioned layer window or
// relative-pointer support; ordinary toplevels ignore position requests. Other backends
// keep the grabbed local point under the pointer using their position APIs.
func DragMove(position *native.Vec2) bool {
	return dragMove(position, true)
}

// DragMoveNonactivating is DragMove for a window opened with
// PopupWindowOptions.Nonactivating. It starts the drag even while the app is
// inactive, so a floating widget can be moved without taking focus.
func DragMoveNonactivating(position *native.Vec2) bool {
	return dragMove(position, false)
}

func dragMove(position *native.Vec2, requireActive bool) (moved bool) {
	state := native.Use[dragMoveState](dragMoveKey{})
	native.PressAction()
	input, frame := native.GetInputState(), native.GetFrameInput()
	action, sample, source := "unchanged", input.MousePoint, "local"
	if dragtrace.Enabled() && (state.active || native.IsActive() || frame.Mouse.Has(native.MouseClick) || frame.Mouse.Has(native.MouseRelease)) {
		var before native.Vec2
		if position != nil {
			before = *position
		}
		activeBefore := state.active
		defer func() {
			dragtrace.Logf("drag.step", "epoch=%d action=%s moved=%t active_before=%t active_after=%t native_active=%t mouse=%v button=%v local=%v motion=%v source=%s sample=%v grabbed=%v start_origin=%v before=%v requested=%v", native.InputEpoch(), action, moved, activeBefore, state.active, native.IsActive(), frame.Mouse, input.MouseButton, input.MousePoint, frame.Motion, source, sample, state.point, state.origin, before, position)
		}()
	}
	if position == nil || (requireActive && !native.IsActive()) || input.MouseButton != native.MousePrimary || frame.Mouse.Has(native.MouseRelease) {
		state.active = false
		action = "inactive_or_release"
		return false
	}
	if !native.IsActive() {
		// Start only after PressAction captured a press on this handle. A newly
		// opened popup can inherit the button that opened it; that must not
		// begin a drag.
		state.active = false
		return false
	}
	if state.active && state.epoch == native.InputEpoch() {
		action = "same_epoch"
		return false
	}
	state.epoch = native.InputEpoch()
	point := input.MousePoint
	provider, relative := native.GetHost().EscapeHatchBackendContext.(interface {
		PointerTravel() (native.Vec2, bool)
	})
	if relative {
		source = "travel"
		var available bool
		point, available = provider.PointerTravel()
		sample = point
		if !available {
			action = "travel_unavailable"
			state.active = false
			return false
		}
	}
	if !state.active || frame.Mouse.Has(native.MouseClick) || state.relative != relative {
		state.active, state.relative = true, relative
		state.point, state.origin = point, *position
		action = "start"
		return false
	}
	origin := *position
	if relative {
		origin = state.origin
	} else if frame.Motion == (native.Vec2{}) {
		action = "no_motion"
		return false
	}
	next := native.Vec2Add(origin, native.Vec2Sub(point, state.point))
	for axis := range next {
		next[axis] = float32(math.Round(float64(next[axis])))
	}
	if next == *position {
		return false
	}
	action = "request"
	*position = next
	Position(int(next[0]), int(next[1]))
	return true
}
