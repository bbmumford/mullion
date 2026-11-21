package widgets

// Added for mullion; see calendar.go.

import (
	"testing"
	"time"
)

// The grid is the part with the arithmetic, and the arithmetic is where a
// calendar goes wrong: a month that starts on the week's first day, one that
// ends on it, a leap February, and a week that starts on Monday rather than
// Sunday.

func TestMonthGridCoversTheWholeMonthInWholeWeeks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		month time.Time
		start time.Weekday
	}{
		{"November 2026", day(2026, time.November, 1), time.Sunday},
		{"November 2026, Monday start", day(2026, time.November, 1), time.Monday},
		{"February 2024, a leap year", day(2024, time.February, 1), time.Monday},
		{"February 2026, 28 days", day(2026, time.February, 1), time.Sunday},
		{"March 2026", day(2026, time.March, 1), time.Monday},
	} {
		weeks := monthGrid(tc.month, tc.start)

		for i, week := range weeks {
			if len(week) != 7 {
				t.Errorf("%s: week %d has %d days, want 7", tc.name, i, len(week))
			}
			if got := week[0].Weekday(); got != tc.start {
				t.Errorf("%s: week %d starts on %v, want %v", tc.name, i, got, tc.start)
			}
		}

		// Every day of the month has to appear exactly once.
		seen := map[int]int{}
		for _, week := range weeks {
			for _, d := range week {
				if d.Month() == tc.month.Month() && d.Year() == tc.month.Year() {
					seen[d.Day()]++
				}
			}
		}
		last := firstOfMonth(tc.month).AddDate(0, 1, -1).Day()
		for d := 1; d <= last; d++ {
			if seen[d] != 1 {
				t.Errorf("%s: day %d appears %d times, want once", tc.name, d, seen[d])
			}
		}
		if len(seen) != last {
			t.Errorf("%s: the grid holds %d days of the month, want %d", tc.name, len(seen), last)
		}
	}
}

// The grid runs day by day, so a month containing a daylight-saving change
// must not lose or repeat one. Adding 24 hours would; AddDate does not.
func TestMonthGridSurvivesADaylightSavingChange(t *testing.T) {
	syd, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Skip("no timezone database here")
	}
	// Sydney moves to daylight saving on the first Sunday of October.
	oct := time.Date(2026, time.October, 1, 0, 0, 0, 0, syd)
	weeks := monthGrid(oct, time.Sunday)

	seen := map[int]int{}
	for _, week := range weeks {
		for _, d := range week {
			if d.Month() == time.October {
				seen[d.Day()]++
			}
		}
	}
	for d := 1; d <= 31; d++ {
		if seen[d] != 1 {
			t.Errorf("October %d appears %d times across the change, want once", d, seen[d])
		}
	}
}

func TestFirstOfMonthKeepsTheLocation(t *testing.T) {
	syd, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Skip("no timezone database here")
	}
	t0 := time.Date(2026, time.November, 21, 13, 45, 0, 0, syd)
	got := firstOfMonth(t0)
	if got.Day() != 1 || got.Month() != time.November || got.Hour() != 0 {
		t.Errorf("firstOfMonth = %v, want midnight on 1 November", got)
	}
	if got.Location() != syd {
		t.Errorf("location became %v, want %v", got.Location(), syd)
	}
}

func TestSameDayIgnoresTheTimeOfDay(t *testing.T) {
	morning := time.Date(2026, time.November, 21, 1, 0, 0, 0, time.Local)
	evening := time.Date(2026, time.November, 21, 23, 59, 0, 0, time.Local)
	if !sameDay(morning, evening) {
		t.Error("two times on one day were not the same day")
	}
	if sameDay(morning, morning.AddDate(0, 0, 1)) {
		t.Error("consecutive days were the same day")
	}
}

func TestCalendarStateShowsTheMonthItIsGiven(t *testing.T) {
	var s CalendarState
	s.Show(day(2026, time.November, 21))
	if got := s.Showing(); got.Day() != 1 || got.Month() != time.November {
		t.Errorf("Showing = %v, want 1 November", got)
	}
}

func TestDayNumbersRender(t *testing.T) {
	for n, want := range map[int]string{1: "1", 9: "9", 10: "10", 28: "28", 31: "31"} {
		if got := itoa(n); got != want {
			t.Errorf("itoa(%d) = %q, want %q", n, got, want)
		}
	}
}
