package widgets

// Added for mullion; see datefield.go.

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func TestParseAndFormatRoundTrip(t *testing.T) {
	in := day(2026, time.November, 21)
	s := FormatDate(in)
	if s != "2026-11-21" {
		t.Fatalf("FormatDate = %q, want 2026-11-21", s)
	}
	out, ok := ParseDate(s)
	if !ok || !out.Equal(in) {
		t.Errorf("round trip gave %v (ok=%v), want %v", out, ok, in)
	}
}

// "No date" has to survive the round trip, or a cleared field becomes the
// zero of the epoch rather than nothing.
func TestNoDateRoundTripsAsEmpty(t *testing.T) {
	if got := FormatDate(time.Time{}); got != "" {
		t.Errorf("a zero time formatted as %q, want empty", got)
	}
	if _, ok := ParseDate(""); ok {
		t.Error("an empty string parsed as a date")
	}
}

func TestParseRejectsWhatIsNotADate(t *testing.T) {
	for _, s := range []string{"2026", "2026-11", "21-11-2026", "11/21/2026", "2026-13-01", "2026-11-32", "nonsense"} {
		if _, ok := ParseDate(s); ok {
			t.Errorf("ParseDate(%q) accepted it", s)
		}
	}
}

func TestDateInRangeTreatsAZeroBoundAsNoBound(t *testing.T) {
	d := day(2026, time.June, 15)
	if !DateInRange(d, time.Time{}, time.Time{}) {
		t.Error("unbounded range rejected a date")
	}
	if !DateInRange(d, day(2026, time.January, 1), time.Time{}) {
		t.Error("a date after the minimum was rejected")
	}
	if DateInRange(d, day(2026, time.July, 1), time.Time{}) {
		t.Error("a date before the minimum was accepted")
	}
	if DateInRange(d, time.Time{}, day(2026, time.May, 1)) {
		t.Error("a date after the maximum was accepted")
	}
}

// The edit buffer is the part with the subtlety: a half-written date must not
// be thrown away, and must not be written back to the value either.
func TestAHalfWrittenDateIsKeptAndNotWrittenBack(t *testing.T) {
	value := day(2026, time.November, 21)
	st := &DateFieldState{text: "2026-1", valid: true, seeded: true}

	st.sync(&value, DateFieldAttrs{})
	if st.text != "2026-1" {
		t.Errorf("the partial text was changed to %q", st.text)
	}
	if st.valid {
		t.Error("a partial date was marked valid")
	}
	if !value.Equal(day(2026, time.November, 21)) {
		t.Errorf("the value moved to %v while the text was incomplete", value)
	}
}

func TestClearingTheTextClearsTheDate(t *testing.T) {
	value := day(2026, time.November, 21)
	st := &DateFieldState{seeded: true}
	st.sync(&value, DateFieldAttrs{})
	if !value.IsZero() {
		t.Errorf("clearing the text left %v, want the zero time", value)
	}
	if !st.valid {
		t.Error("an empty field was marked invalid; empty means no date, not a mistake")
	}
}

// A date outside the bounds is marked rather than written, so the caller never
// sees a value it said it would not accept.
func TestAnOutOfRangeDateIsMarkedAndNotWritten(t *testing.T) {
	value := day(2026, time.June, 1)
	st := &DateFieldState{text: "2030-01-01", seeded: true}
	st.sync(&value, DateFieldAttrs{Max: day(2027, time.January, 1)})
	if st.valid {
		t.Error("an out-of-range date was marked valid")
	}
	if !value.Equal(day(2026, time.June, 1)) {
		t.Errorf("an out-of-range date was written back: %v", value)
	}
}

func TestACompleteDateIsWrittenBack(t *testing.T) {
	var value time.Time
	st := &DateFieldState{text: "2026-11-21", seeded: true}
	st.sync(&value, DateFieldAttrs{})
	if !st.valid {
		t.Error("a complete date was marked invalid")
	}
	if !value.Equal(day(2026, time.November, 21)) {
		t.Errorf("the value is %v, want 2026-11-21", value)
	}
}

// A value changed from outside -- a reset button, a Set -- has to win over
// whatever is sitting in the buffer, or the field shows a date nobody holds.
func TestAValueChangedElsewhereWinsOverTheBuffer(t *testing.T) {
	value := day(2026, time.November, 21)
	var st DateFieldState
	st.seed(&value)
	if st.text != "2026-11-21" {
		t.Fatalf("the buffer seeded as %q", st.text)
	}

	value = day(2030, time.March, 3)
	st.seed(&value)
	if st.text != "2030-03-03" {
		t.Errorf("the buffer still reads %q after the value changed elsewhere", st.text)
	}
}
