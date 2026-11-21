package widgets

// Added for mullion; see markdown_parse.go.

import (
	"strings"

	. "github.com/bbmumford/mullion/native"
)

// MarkdownAttrs tunes how a document is drawn. The zero value is the ordinary
// one.
type MarkdownAttrs struct {
	// BodySize is the text size of a paragraph, in points. Zero takes 13.
	// Headings and code are sized from it, so one number scales the document.
	BodySize float32

	// Gap separates the blocks. Zero takes 10.
	Gap float32

	// OnLink is called when a link is activated. Nil opens it in the system
	// browser, which is what a help or about pane wants; a caller showing
	// untrusted text should pass a function and decide for itself.
	OnLink func(url string)
}

// Markdown draws a document written in the subset ParseMarkdown reads.
//
// It parses on every frame, which is right for the help text and about panes
// this is for: a few hundred lines cost nothing against the drawing. A caller
// showing a large document should call ParseMarkdown once, keep the blocks,
// and pass them to MarkdownBlocks.
func Markdown(src string, attrs MarkdownAttrs) {
	MarkdownBlocks(ParseMarkdown(src), attrs)
}

// MarkdownBlocks draws a document already parsed.
func MarkdownBlocks(blocks []MarkdownBlock, attrs MarkdownAttrs) {
	if attrs.BodySize == 0 {
		attrs.BodySize = 13
	}
	if attrs.Gap == 0 {
		attrs.Gap = 10
	}
	scheme := CurrentColorScheme

	// The document carries its own text colour rather than inheriting
	// whatever the surrounding container happens to have set. A help pane
	// dropped into a plain container was unreadable without it: dark text on
	// a dark canvas, because nothing in the chain had said otherwise.
	Container(Attrs(Expand, Gap(comfort(attrs.Gap)),
		AmendTextStyle(TextColorVec(scheme.Surfaces.Canvas.Text))), func() {
		for _, b := range blocks {
			markdownBlock(b, attrs, scheme)
		}
	})
}

func markdownBlock(b MarkdownBlock, attrs MarkdownAttrs, scheme ColorScheme) {
	body := comfort(attrs.BodySize)

	switch b.Kind {
	case MarkdownHeading:
		// Six levels over a narrow range: a document whose h1 is twice the
		// body text reads as a poster rather than a page.
		scale := []float32{1.7, 1.45, 1.25, 1.1, 1.0, 0.95}[min(b.Level, 6)-1]
		Container(Attrs(Expand, Pad4(comfort(4), 0, 0, 0)), func() {
			markdownSpans(b.Spans, attrs, scheme, body*scale, WeightBold)
		})

	case MarkdownParagraph:
		Container(Attrs(Expand), func() {
			markdownSpans(b.Spans, attrs, scheme, body, WeightNormal)
		})

	case MarkdownCode:
		Container(Attrs(Expand, Pad(comfort(8)), Corners(comfort(4)),
			BackgroundVec(scheme.Surfaces.Panel.Background),
			BorderColorVec(scheme.Surfaces.Panel.Border), BorderWidth(1)), func() {
			// Every line its own label: a code block is pre-formatted, so its
			// lines must not be reflowed to the width of the window.
			for _, line := range strings.Split(b.Text, "\n") {
				Label(line, FontSize(body*0.92), Fonts(Monospace...),
					TextColorVec(scheme.Surfaces.Panel.Text))
			}
		})

	case MarkdownQuote:
		Container(Attrs(Row, Expand, Gap(comfort(8))), func() {
			// A bar down the left rather than an indent, so a quote is still
			// a quote when it wraps.
			Element(Attrs(FixWidth(comfort(3)), Expand, Corners(comfort(2)),
				BackgroundVec(scheme.List.Muted)))
			Container(Attrs(Grow(1)), func() {
				markdownSpans(b.Spans, attrs, scheme, body, WeightNormal)
			})
		})

	case MarkdownList:
		Container(Attrs(Expand, Gap(comfort(3)), Pad4(0, 0, 0, comfort(4))), func() {
			for i, item := range b.Items {
				Container(Attrs(Row, Expand, Gap(comfort(6))), func() {
					Container(Attrs(FixWidth(comfort(18))), func() {
						Label(markdownMarker(b.Ordered, i), FontSize(body),
							TextColorVec(scheme.List.Muted))
					})
					Container(Attrs(Grow(1)), func() {
						markdownSpans(item, attrs, scheme, body, WeightNormal)
					})
				})
			}
		})

	case MarkdownRule:
		Element(Attrs(Expand, FixHeight(1), BackgroundVec(scheme.Surfaces.Panel.Border)))
	}
}

// markdownMarker is the bullet or number shown against a list item.
func markdownMarker(ordered bool, i int) string {
	if !ordered {
		return "•"
	}
	return itoa(i+1) + "."
}

// markdownSpans draws a run of inline text, wrapping as a row of labels.
//
// Wrap on the row is what makes a marked-up paragraph reflow: each span is its
// own label, so bold and code keep their own styling, and the row flows them
// onto further lines as the window narrows.
func markdownSpans(spans []MarkdownSpan, attrs MarkdownAttrs, scheme ColorScheme,
	size float32, weight Weight) {
	Container(Attrs(Row, Wrap, Expand, CrossMid), func() {
		for _, s := range spans {
			markdownSpan(s, attrs, scheme, size, weight)
		}
	})
}

func markdownSpan(s MarkdownSpan, attrs MarkdownAttrs, scheme ColorScheme,
	size float32, weight Weight) {
	// A span is split on spaces so a long paragraph wraps between words rather
	// than only between spans: without this, a sentence with no markup in it
	// is one unbreakable label.
	words := splitKeepingSpaces(s.Text)

	for _, w := range words {
		if w == "" {
			continue
		}
		switch {
		case s.URL != "":
			// Folder is the scheme's "this is a path or a destination"
			// colour, and is what makes a link look like one against both
			// the light and the dark canvas.
			if attrs.OnLink == nil {
				Link(w, s.URL, FontSize(size), TextColorVec(scheme.List.Folder))
				continue
			}
			// A caller that wants to decide for itself gets a link-looking
			// label it handles, rather than one that opens a browser behind
			// its back. Untrusted text is the case this exists for.
			Container(Attrs(CrossMid), func() {
				st := ProcessButtonEvents(false)
				AssignAccess()
				fns := []TextStyleFn{FontSize(size), TextColorVec(scheme.List.Folder)}
				if st.Hovered {
					ModAttrs(AmendTextStyle(TextColorVec(scheme.List.Selected.Background)))
				}
				Label(w, fns...)
				if st.Clicked {
					attrs.OnLink(s.URL)
				}
			})
		case s.Code:
			Container(Attrs(Pad2(comfort(1), comfort(4)), Corners(comfort(3)),
				BackgroundVec(scheme.Surfaces.Panel.Background)), func() {
				Label(w, FontSize(size*0.92), Fonts(Monospace...))
			})
		default:
			fns := []TextStyleFn{FontSize(size)}
			if s.Bold || weight == WeightBold {
				fns = append(fns, FontWeight(WeightBold))
			}
			if s.Italic {
				fns = append(fns, FontStyle(StyleItalic))
			}
			Label(w, fns...)
		}
	}
}

// splitKeepingSpaces breaks text into words, keeping a trailing space on each
// so the gaps between them survive being drawn as separate labels.
func splitKeepingSpaces(s string) []string {
	if s == "" {
		return nil
	}
	fields := strings.Split(s, " ")
	out := make([]string, 0, len(fields))
	for i, f := range fields {
		if i < len(fields)-1 {
			f += " "
		}
		out = append(out, f)
	}
	return out
}
