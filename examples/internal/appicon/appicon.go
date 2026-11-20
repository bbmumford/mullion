// Package appicon draws the examples' icon instead of shipping a file.
//
// An application would embed its own artwork with go:embed. The examples draw
// theirs so the repository carries no binary assets and so the shape of what
// icon.Icon wants -- PNG bytes, and a separate silhouette for a macOS menu bar
// -- is visible in Go.
package appicon

import (
	"image"
	"image/color"

	"github.com/bbmumford/mullion/icon"
)

// Colour is a filled rounded square, for a dock or taskbar icon.
func Colour() icon.Icon {
	ic, err := icon.Image(draw(color.NRGBA{R: 0x2b, G: 0x6c, B: 0xb0, A: 0xff}))
	if err != nil {
		// Encoding an image this package drew itself cannot fail; an empty
		// Icon is a supported state, so there is nothing to report upward.
		return icon.Icon{}
	}
	return ic
}

// Template is the same shape in black, marked as a macOS template image so the
// menu bar tints it to match the system theme. On Windows and Linux the
// template flag is ignored and this is simply a black icon, so an application
// that wants both passes Colour and sets the template variant only on macOS.
func Template() icon.Icon {
	ic, err := icon.Image(draw(color.NRGBA{A: 0xff}))
	if err != nil {
		return icon.Icon{}
	}
	return ic.AsTemplate()
}

// draw paints a 32x32 rounded square, the size a notification area wants.
func draw(fill color.NRGBA) image.Image {
	const size, radius = 32, 7
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			if inRoundedSquare(x, y, size, radius) {
				img.SetNRGBA(x, y, fill)
			}
		}
	}
	return img
}

func inRoundedSquare(x, y, size, radius int) bool {
	// Clamp the point into the rectangle the corner arcs are centred on; a
	// point is inside unless it is further than the radius from that.
	cx := min(max(x, radius), size-1-radius)
	cy := min(max(y, radius), size-1-radius)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= radius*radius
}
