package widgets

// Added for mullion; not part of the merged go-shirei snapshot. Kept in a file
// of its own so a later re-sync with upstream can see at a glance what is
// local.

import (
	"strings"

	. "github.com/bbmumford/mullion/native"
)

// A TextGridCell is one character and how it is painted.
//
// A zero Foreground or Background takes the grid's default, so a run of plain
// text needs no styling at all and only the cells that differ carry colour.
//
// The colours are HSLA, not RGBA: hue in 0 to 360, saturation and lightness in
// 0 to 100, alpha in 0 to 1. Vec4{120, 60, 45, 1} is a green. Reading it as
// RGBA gives a near-black grey, since a red of 1 is a hue of one degree at one
// per cent saturation, which is the first thing to check when a colour comes
// out wrong.
type TextGridCell struct {
	Rune       rune
	Foreground Vec4
	Background Vec4
	Bold       bool
}

// TextGridAttrs tunes a grid. The zero value is the ordinary one.
type TextGridAttrs struct {
	// FontSize is the character size in points. Zero takes 12.
	FontSize float32

	// Foreground and Background are the defaults for cells that name none.
	// Zero values take the scheme's panel colours.
	Foreground, Background Vec4

	// ShowLineNumbers puts a gutter down the left.
	ShowLineNumbers bool

	// FirstLineNumber is the number given to the first row when the gutter is
	// shown. Zero means one, so a grid showing a whole file counts from 1 and
	// a window onto the middle of one can say where it starts.
	FirstLineNumber int
}

// TextGrid draws rows of characters in a monospace face, each able to carry
// its own colour.
//
// It is for the things a text widget cannot do: a terminal, a hex view, a diff
// with per-character marking, a log with coloured levels. For prose use Label,
// for an editable document use TextEditor; both of those shape and wrap text,
// and this deliberately does neither. A row is as long as it is, and nothing
// reflows.
//
// Cells are drawn in runs of matching style rather than one at a time, so a
// row of plain text costs one label and not eighty.
func TextGrid(rows [][]TextGridCell, attrs TextGridAttrs) {
	scheme := CurrentColorScheme
	if attrs.FontSize == 0 {
		attrs.FontSize = 12
	}
	if attrs.Foreground == (Vec4{}) {
		attrs.Foreground = scheme.Surfaces.Panel.Text
	}
	if attrs.Background == (Vec4{}) {
		attrs.Background = scheme.Surfaces.Panel.Background
	}
	first := attrs.FirstLineNumber
	if first == 0 {
		first = 1
	}
	size := comfort(attrs.FontSize)

	// The gutter is as wide as its widest number, measured in digits, so the
	// text beside it does not shift as the view scrolls past line 99.
	gutter := 0
	if attrs.ShowLineNumbers {
		gutter = len(itoaWide(first + len(rows) - 1))
	}

	Container(Attrs(Expand, BackgroundVec(attrs.Background), Pad(comfort(6))), func() {
		for y, row := range rows {
			Container(Attrs(Row, Expand), func() {
				if attrs.ShowLineNumbers {
					Container(Attrs(Pad4(0, comfort(8), 0, 0)), func() {
						Label(padLeft(itoaWide(first+y), gutter), FontSize(size),
							Fonts(Monospace...), TextColorVec(scheme.List.Muted))
					})
				}
				for _, run := range textGridRuns(row, attrs) {
					textGridRun(run, size)
				}
			})
		}
	})
}

// TextGridFromString builds rows from plain text, one row per line, in the
// default colours. Tabs are expanded to the next multiple of tabWidth, because
// a grid has no tab stops of its own and a raw tab would occupy one cell.
func TextGridFromString(s string, tabWidth int) [][]TextGridCell {
	if tabWidth <= 0 {
		tabWidth = 8
	}
	var rows [][]TextGridCell
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		var row []TextGridCell
		col := 0
		for _, r := range line {
			if r == '\t' {
				for n := tabWidth - col%tabWidth; n > 0; n-- {
					row = append(row, TextGridCell{Rune: ' '})
					col++
				}
				continue
			}
			row = append(row, TextGridCell{Rune: r})
			col++
		}
		rows = append(rows, row)
	}
	return rows
}

// a run of adjacent cells sharing a style, which is what actually gets drawn.
type textGridRunSpan struct {
	text       string
	foreground Vec4
	background Vec4
	bold       bool
}

// textGridRuns groups a row into runs of matching style.
func textGridRuns(row []TextGridCell, attrs TextGridAttrs) []textGridRunSpan {
	var runs []textGridRunSpan
	var cur textGridRunSpan
	var b strings.Builder
	started := false

	flush := func() {
		if started && b.Len() > 0 {
			cur.text = b.String()
			runs = append(runs, cur)
			b.Reset()
		}
	}

	for _, cell := range row {
		fg, bg := cell.Foreground, cell.Background
		if fg == (Vec4{}) {
			fg = attrs.Foreground
		}
		if bg == (Vec4{}) {
			bg = attrs.Background
		}
		next := textGridRunSpan{foreground: fg, background: bg, bold: cell.Bold}
		if !started || next != cur {
			flush()
			cur, started = next, true
		}
		r := cell.Rune
		if r == 0 {
			r = ' ' // a zero rune is an unset cell, not a NUL to draw
		}
		b.WriteRune(r)
	}
	flush()
	return runs
}

func textGridRun(run textGridRunSpan, size float32) {
	fns := []TextStyleFn{FontSize(size), Fonts(Monospace...), TextColorVec(run.foreground)}
	if run.bold {
		fns = append(fns, FontWeight(WeightBold))
	}
	// A run whose background matches the grid's needs no box of its own.
	Container(Attrs(BackgroundVec(run.background)), func() {
		Label(run.text, fns...)
	})
}

// itoaWide renders a non-negative integer of any width, where itoa handles
// only one or two digits.
func itoaWide(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoaWide(-n)
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}

// padLeft right-aligns s in a field of width w, which is what a line-number
// gutter wants.
func padLeft(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return strings.Repeat(" ", w-len(s)) + s
}
