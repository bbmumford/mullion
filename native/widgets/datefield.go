package widgets

// Added for mullion; not part of the merged go-shirei snapshot. Kept in a file
// of its own so a later re-sync with upstream can see at a glance what is
// local.

import (
	"fmt"
	"time"

	. "github.com/bbmumford/mullion/native"
)

// DateFieldAttrs tunes a date field. The zero value is the ordinary one.
type DateFieldAttrs struct {
	// Placeholder is shown while no date is set. Empty takes "yyyy-mm-dd".
	Placeholder string

	// Min and Max bound what can be chosen. Zero times mean unbounded.
	Min, Max time.Time

	// Width is the field's width in points. Zero takes 130.
	Width float32
}

// DateFieldState is the text being typed, which cannot live in the time.Time:
// "2026-1" is not a date and must not be thrown away before the person has
// finished writing it.
//
// The caller owns it, as with TreeState and every other piece of widget state
// here. An earlier version kept these in a package-level map keyed by the
// value's address, which read more neatly at the call site and leaked an entry
// for every date field the program ever built.
//
// The zero value is correct for a field starting from whatever the value
// holds.
type DateFieldState struct {
	text   string
	valid  bool
	seeded bool
}

// Text is what is currently typed, for a caller that wants to show or save it.
func (s *DateFieldState) Text() string { return s.text }

// Valid reports whether what is typed is a date within the field's bounds. An
// empty field is valid: empty means no date, not a mistake.
func (s *DateFieldState) Valid() bool { return s.valid || s.text == "" }

// DateField edits a date by typing it.
//
// The value is a time.Time, truncated to a day: a date field that quietly
// carries a time of day produces appointments at 00:00:00.000001 and
// comparisons that fail for no visible reason. A zero time means no date, and
// shows the placeholder.
//
// Typing is accepted as the person goes and parsed when it is complete, so a
// half-written date is not thrown away mid-keystroke. An unparseable or
// out-of-range entry is marked rather than reverted -- taking the text away
// while someone is still typing it is the most annoying thing a date field can
// do.
//
// It reports true on the frame the date changes.
func DateField(value *time.Time, state *DateFieldState, attrs DateFieldAttrs) bool {
	if value == nil || state == nil {
		return false
	}
	if attrs.Placeholder == "" {
		attrs.Placeholder = "yyyy-mm-dd"
	}
	if attrs.Width == 0 {
		attrs.Width = 130
	}

	st := state
	st.seed(value)
	before := *value

	Container(Attrs(Row, CrossMid, Gap(comfort(4))), func() {
		Container(Attrs(FixWidth(comfort(attrs.Width))), func() {
			TextInputExt(&st.text, TextInputAttrs{
				Placeholder: attrs.Placeholder,
				FixedWidth:  true,
				MinWidth:    comfort(attrs.Width),
			})
		})
		// The validity mark sits beside the field rather than recolouring it,
		// so the text stays as readable while it is wrong as when it is right.
		if st.text != "" && !st.valid {
			Icon(SymICross, FontSize(comfort(11)), TextColorVec(CurrentColorScheme.List.Error))
		}
	})

	st.sync(value, attrs)
	return *value != before
}

// ParseDate reads a date in the format DateField writes, and reports whether
// it understood it.
//
// The format is ISO 8601's calendar date, yyyy-mm-dd, which sorts as text,
// never swaps the day and the month between readers in different countries,
// and is the one format every system agrees on.
func ParseDate(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(dateLayout, s, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// FormatDate writes a date in the format ParseDate reads. A zero time gives an
// empty string, so "no date" round-trips.
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateLayout)
}

// DateInRange reports whether t falls within min and max, treating a zero
// bound as no bound.
func DateInRange(t, min, max time.Time) bool {
	if !min.IsZero() && t.Before(min) {
		return false
	}
	if !max.IsZero() && t.After(max) {
		return false
	}
	return true
}

const dateLayout = "2006-01-02"

// seed fills the buffer from the value the first time the field is drawn, and
// whenever the value has been changed from somewhere else -- a reset button, a
// Set -- which has to win over whatever is sitting in the buffer, or the field
// shows a date nobody holds.
func (st *DateFieldState) seed(value *time.Time) {
	if !st.seeded {
		st.text, st.valid, st.seeded = FormatDate(*value), true, true
		return
	}
	if parsed, ok := ParseDate(st.text); ok && parsed.Equal(*value) {
		return // the buffer already says this date
	}
	if st.text == "" && value.IsZero() {
		return // both say "no date"
	}
	if !value.IsZero() && FormatDate(*value) != st.text {
		st.text, st.valid = FormatDate(*value), true
	}
}

// sync parses what was typed and writes it back when it is a date in range.
func (st *DateFieldState) sync(value *time.Time, attrs DateFieldAttrs) {
	if st.text == "" {
		st.valid = true
		*value = time.Time{}
		return
	}
	parsed, ok := ParseDate(st.text)
	if !ok || !DateInRange(parsed, attrs.Min, attrs.Max) {
		st.valid = false
		return
	}
	st.valid = true
	*value = parsed
}

// DateString is FormatDate under a name that reads better at a call site that
// is building a label rather than a field.
func DateString(t time.Time) string {
	if t.IsZero() {
		return "no date"
	}
	return fmt.Sprintf("%s", t.Format(dateLayout))
}
