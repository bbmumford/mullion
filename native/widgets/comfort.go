package widgets

import (
	. "github.com/bbmumford/mullion/native"
)

// comfort scales a design-unit size by Host.ComfortScale (touch-friendly
// control density). Use for button/input text, segment height, slider handle,
// checkbox box, etc. — not for arbitrary app layout.
func comfort(v f32) f32 {
	return v * ComfortScale()
}
