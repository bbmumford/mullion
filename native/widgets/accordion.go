package widgets

// Added for mullion; not part of the merged go-mullion snapshot. Kept in a file
// of its own so a later re-sync with upstream can see at a glance what is
// local.

import (
	. "github.com/bbmumford/mullion/native"
)

// AccordionSection is one collapsible section: a header that toggles, and
// content shown while it is open.
//
// open is the caller's state, as everywhere else here, so which sections are
// expanded survives a rebuild and can be saved with the rest of the window's
// state. Passing nil draws the header without making it toggle, for a section
// that is always open.
func AccordionSection(open *bool, title string, content func()) {
	AccordionSectionStyled(open, title, content, CurrentColorScheme.Surfaces.Panel)
}

// AccordionSectionStyled is AccordionSection in colours of the caller's
// choosing.
func AccordionSectionStyled(open *bool, title string, content func(), surface SurfaceColors) {
	expanded := open == nil || *open

	Container(Attrs(Expand), func() {
		// The header is the control: a row that toggles on click, on Space and
		// on Enter, and reports itself to assistive technology as what it is.
		Container(Attrs(Row, Expand, CrossMid, Gap(comfort(6)), Pad2(comfort(6), comfort(8)),
			Corners(comfort(4))), func() {
			st := ProcessButtonEvents(open == nil)
			NextAccessRole("button")
			NextAccessName(title)
			NextAccessChecked(expanded)
			AssignAccess()

			if st.Clicked && open != nil {
				*open = !*open
				expanded = *open
				RequestNextFrame()
			}
			if st.Hovered || st.FocusVisible {
				ModAttrs(BackgroundVec(hoverOn(surface)))
			}

			// The triangle points down when open and right when closed, which
			// is the one convention every desktop shares.
			// A solid triangle both ways, rotated: a long arrow beside a
			// filled triangle reads as two different controls.
			glyph := TypMediaPlay
			if expanded {
				glyph = TypArrowSortedDown
			}
			Icon(glyph, FontSize(comfort(11)), TextColorVec(surface.Text))
			Label(title, FontWeight(WeightBold), FontSize(comfort(12)), TextColorVec(surface.Text))
		})

		if expanded && content != nil {
			Container(Attrs(Expand, Pad4(comfort(2), comfort(8), comfort(8), comfort(22))), content)
		}
	})
}

// hoverOn is the surface's text colour at the strength a hover wants: enough
// to show the row is live, not enough to read as selected.
func hoverOn(surface SurfaceColors) Vec4 {
	c := surface.Text
	c[3] *= 0.08
	return c
}
