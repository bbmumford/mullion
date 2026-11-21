package widgets

// Added for mullion; see textgrid.go.

import (
	"strings"
	"testing"

	. "github.com/bbmumford/mullion/native"
)

func TestTextGridFromStringMakesOneRowPerLine(t *testing.T) {
	rows := TextGridFromString("ab\ncd\n", 8)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3 (the trailing newline ends an empty one)", len(rows))
	}
	if string(rowRunes(rows[0])) != "ab" || string(rowRunes(rows[1])) != "cd" {
		t.Errorf("rows are %q and %q", rowRunes(rows[0]), rowRunes(rows[1]))
	}
	if len(rows[2]) != 0 {
		t.Errorf("the last row has %d cells, want none", len(rows[2]))
	}
}

// A grid has no tab stops, so a raw tab would sit in one cell and throw every
// column after it out of line.
func TestTabsExpandToTheNextStop(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"\tx", "        x"},                 // a tab at column 0 fills to 8
		{"a\tx", "a       x"},                // from column 1, seven spaces
		{"abcdefg\tx", "abcdefg x"},          // from column 7, one space
		{"abcdefgh\tx", "abcdefgh        x"}, // exactly on a stop, a full width
	} {
		rows := TextGridFromString(tc.in, 8)
		if got := string(rowRunes(rows[0])); got != tc.want {
			t.Errorf("%q expanded to %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTabWidthIsHonoured(t *testing.T) {
	rows := TextGridFromString("a\tx", 4)
	if got := string(rowRunes(rows[0])); got != "a   x" {
		t.Errorf("with tabWidth 4, got %q, want %q", got, "a   x")
	}
}

// Runs are what make a grid affordable: a row of plain text has to be one
// label, not one per character.
func TestAdjacentCellsOfOneStyleBecomeOneRun(t *testing.T) {
	rows := TextGridFromString("hello world", 8)
	runs := textGridRuns(rows[0], TextGridAttrs{Foreground: red(), Background: blue()})
	if len(runs) != 1 {
		t.Fatalf("a plain row became %d runs, want 1", len(runs))
	}
	if runs[0].text != "hello world" {
		t.Errorf("the run holds %q", runs[0].text)
	}
}

// And a change of style has to break the run, or the colouring is lost.
func TestAChangeOfStyleBreaksTheRun(t *testing.T) {
	row := []TextGridCell{
		{Rune: 'a'}, {Rune: 'b'},
		{Rune: 'E', Foreground: red()},
		{Rune: 'c'},
	}
	runs := textGridRuns(row, TextGridAttrs{Foreground: blue(), Background: blue()})
	if len(runs) != 3 {
		t.Fatalf("got %d runs, want 3 (ab / E / c)", len(runs))
	}
	if runs[0].text != "ab" || runs[1].text != "E" || runs[2].text != "c" {
		t.Errorf("runs are %q, %q, %q", runs[0].text, runs[1].text, runs[2].text)
	}
	if runs[1].foreground != red() {
		t.Error("the marked cell lost its colour")
	}
	if runs[0].foreground != blue() {
		t.Error("an unstyled cell did not take the default")
	}
}

// An unset cell is a space, not a NUL that would draw as a missing glyph.
func TestAZeroRuneDrawsAsASpace(t *testing.T) {
	runs := textGridRuns([]TextGridCell{{}, {Rune: 'x'}}, TextGridAttrs{})
	var all strings.Builder
	for _, r := range runs {
		all.WriteString(r.text)
	}
	if got := all.String(); got != " x" {
		t.Errorf("a zero rune rendered as %q, want a leading space", got)
	}
}

func TestLineNumbersRenderAndAlign(t *testing.T) {
	if got := itoaWide(0); got != "0" {
		t.Errorf("itoaWide(0) = %q", got)
	}
	if got := itoaWide(12345); got != "12345" {
		t.Errorf("itoaWide(12345) = %q", got)
	}
	if got := padLeft("7", 3); got != "  7" {
		t.Errorf("padLeft gave %q, want two spaces then 7", got)
	}
	// A number wider than the field is not truncated: losing a digit is worse
	// than a gutter one character wider than planned.
	if got := padLeft("1234", 2); got != "1234" {
		t.Errorf("padLeft truncated to %q", got)
	}
}

func TestEmptyGridIsNotAPanic(t *testing.T) {
	if runs := textGridRuns(nil, TextGridAttrs{}); len(runs) != 0 {
		t.Errorf("an empty row produced %d runs", len(runs))
	}
	if rows := TextGridFromString("", 8); len(rows) != 1 || len(rows[0]) != 0 {
		t.Errorf("an empty string produced %v", rows)
	}
}

func rowRunes(row []TextGridCell) []rune {
	out := make([]rune, 0, len(row))
	for _, c := range row {
		out = append(out, c.Rune)
	}
	return out
}

// HSLA, as every colour here is: hue, saturation, lightness, alpha.
func red() Vec4  { return Vec4{0, 70, 50, 1} }
func blue() Vec4 { return Vec4{220, 70, 50, 1} }
