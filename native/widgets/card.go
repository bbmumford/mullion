package widgets

// Added for mullion; not part of the merged go-shirei snapshot. Kept in a file
// of its own so a later re-sync with upstream can see at a glance what is
// local.

import (
	. "github.com/bbmumford/mullion/native"
)

// CardAttrs tunes a card. The zero value is the ordinary one.
type CardAttrs struct {
	// Title is the heading above the content. Empty means no heading.
	Title string

	// Subtitle sits under the title in the quieter text colour.
	Subtitle string

	// Pad is the inset around the content, in points. Zero takes 12.
	Pad float32

	// Gap separates the heading from the content and the content's own
	// children. Zero takes 8.
	Gap float32

	// Grow makes the card take a share of the space along its parent's axis,
	// the way Grow does elsewhere. Zero leaves it at its natural size.
	Grow float32
}

// Card groups related content on a raised panel with a border and an optional
// heading.
//
// It is layout rather than behaviour: nothing in a card is clickable by being
// in one, and the content is whatever fn draws. Use it to say that a group of
// controls belongs together, which a bare container with a background does not
// quite say on its own.
func Card(attrs CardAttrs, fn func()) {
	CardStyled(attrs, CurrentColorScheme.Surfaces.Panel, fn)
}

// CardStyled is Card in colours of the caller's choosing.
func CardStyled(attrs CardAttrs, surface SurfaceColors, fn func()) {
	pad := attrs.Pad
	if pad == 0 {
		pad = 12
	}
	gap := attrs.Gap
	if gap == 0 {
		gap = 8
	}
	pad, gap = comfort(pad), comfort(gap)

	outer := Attrs(Expand, Pad(pad), Gap(gap), Corners(comfort(6)),
		BackgroundVec(surface.Background), BorderColorVec(surface.Border), BorderWidth(1),
		AmendTextStyle(TextColorVec(surface.Text)))
	if attrs.Grow > 0 {
		outer = AttrsWith(outer, Grow(attrs.Grow))
	}

	Container(outer, func() {
		if attrs.Title != "" || attrs.Subtitle != "" {
			Container(Attrs(Gap(comfort(2))), func() {
				if attrs.Title != "" {
					Label(attrs.Title, FontWeight(WeightBold), FontSize(comfort(13)))
				}
				if attrs.Subtitle != "" {
					Label(attrs.Subtitle, FontSize(comfort(11)), TextColorVec(mutedOn(surface)))
				}
			})
		}
		if fn != nil {
			fn()
		}
	})
}

// mutedOn is the surface's text colour at reduced strength, for the secondary
// line of a heading. Taking the alpha down rather than mixing towards grey
// keeps it legible on a light and a dark surface alike, which a fixed grey
// does not.
func mutedOn(surface SurfaceColors) Vec4 {
	c := surface.Text
	c[3] *= 0.65
	return c
}
