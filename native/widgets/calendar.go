package widgets

// Added for mullion; not part of the merged go-mullion snapshot. Kept in a file
// of its own so a later re-sync with upstream can see at a glance what is
// local.

import (
	"time"

	. "github.com/bbmumford/mullion/native"
)

// CalendarAttrs tunes a calendar. The zero value is the ordinary one.
type CalendarAttrs struct {
	// Min and Max bound what can be chosen. Zero times mean unbounded, and a
	// day outside them is drawn but cannot be picked.
	Min, Max time.Time

	// WeekStart is the day a row begins on. The zero value is Sunday, which is
	// Go's own zero for a Weekday; pass time.Monday for most of the world.
	WeekStart time.Weekday

	// CellSize is the width and height of a day, in points. Zero takes 28.
	CellSize float32
}

// CalendarState is which month the grid is showing. The caller owns it, so
// paging back three months and picking a day does not reset to today on the
// next rebuild.
//
// The zero value shows the month of the selected date, or the current month
// when nothing is selected.
type CalendarState struct {
	month  time.Time // the first of the month being shown
	seeded bool
}

// Showing reports the month on display.
func (s *CalendarState) Showing() time.Time { return s.month }

// Show moves to the month containing t.
func (s *CalendarState) Show(t time.Time) {
	s.month = firstOfMonth(t)
	s.seeded = true
}

// Calendar draws a month of days and reports the one chosen this frame.
//
// value is the selected date, shown highlighted and written when a day is
// picked. A zero time means nothing is selected.
//
//	if picked, ok := widgets.Calendar(&due, &month, widgets.CalendarAttrs{WeekStart: time.Monday}); ok {
//	    log.Println("chose", picked)
//	}
func Calendar(value *time.Time, state *CalendarState, attrs CalendarAttrs) (time.Time, bool) {
	if value == nil || state == nil {
		return time.Time{}, false
	}
	if attrs.CellSize == 0 {
		attrs.CellSize = 28
	}
	if !state.seeded {
		if value.IsZero() {
			state.Show(time.Now())
		} else {
			state.Show(*value)
		}
	}

	scheme := CurrentColorScheme
	cell := comfort(attrs.CellSize)
	var picked time.Time
	var ok bool

	Container(Attrs(Gap(comfort(4))), func() {
		// Header: the month, and a way out of it in both directions.
		Container(Attrs(Row, CrossMid, Gap(comfort(4))), func() {
			if CtrlButton(TypChevronLeft, "", true) {
				state.Show(state.month.AddDate(0, -1, 0))
			}
			Container(Attrs(Grow(1), Center), func() {
				Label(state.month.Format("January 2006"), FontWeight(WeightBold), FontSize(comfort(12)))
			})
			if CtrlButton(TypChevronRight, "", true) {
				state.Show(state.month.AddDate(0, 1, 0))
			}
		})

		// The weekday initials, in the same order the grid uses.
		Container(Attrs(Row), func() {
			for i := range 7 {
				d := time.Weekday((int(attrs.WeekStart) + i) % 7)
				Container(Attrs(FixSize(cell, cell*0.7), Center), func() {
					Label(d.String()[:1], FontSize(comfort(10)), TextColorVec(scheme.List.Muted))
				})
			}
		})

		for _, week := range monthGrid(state.month, attrs.WeekStart) {
			Container(Attrs(Row), func() {
				for _, day := range week {
					if hit, got := calendarDay(value, day, state.month, attrs, scheme, cell); got {
						picked, ok = hit, true
					}
				}
			})
		}
	})
	return picked, ok
}

// calendarDay draws one day and reports whether it was chosen.
func calendarDay(value *time.Time, day, showing time.Time, attrs CalendarAttrs,
	scheme ColorScheme, cell float32) (time.Time, bool) {
	inMonth := day.Month() == showing.Month()
	selectable := DateInRange(day, attrs.Min, attrs.Max)
	selected := !value.IsZero() && sameDay(day, *value)
	today := sameDay(day, time.Now())

	var picked time.Time
	var ok bool
	Container(Attrs(FixSize(cell, cell), Center, Corners(cell/2)), func() {
		st := ProcessButtonEvents(!selectable)
		NextAccessRole("button")
		NextAccessName(day.Format("2 January 2006"))
		NextAccessChecked(selected)
		NextAccessDisabled(!selectable)
		AssignAccess()

		switch {
		case selected:
			ModAttrs(BackgroundVec(scheme.List.Selected.Background))
		case st.Hovered && selectable:
			ModAttrs(BackgroundVec(scheme.List.Hovered.Background))
		case today:
			// Today is ringed rather than filled, so it cannot be mistaken for
			// the selection when both are on screen.
			ModAttrs(BorderColorVec(scheme.List.Muted), BorderWidth(1))
		}

		if st.Clicked && selectable {
			*value = day
			picked, ok = day, true
			RequestNextFrame()
		}

		colour := scheme.List.Surface.Text
		switch {
		case selected:
			colour = scheme.List.Selected.Text
		case !selectable:
			colour = scheme.List.Disabled
		case !inMonth:
			// The days either side of the month are shown so the weeks are
			// whole, but quietly: they are context, not choices being offered.
			colour = scheme.List.Muted
		}
		Label(itoa(day.Day()), FontSize(comfort(11)), TextColorVec(colour))
	})
	return picked, ok
}

// monthGrid is the weeks to draw: whole weeks covering the month, including
// the days either side that fall in the first and last ones.
func monthGrid(month time.Time, weekStart time.Weekday) [][]time.Time {
	first := firstOfMonth(month)
	// Step back to the week's first day. The modulo keeps it positive whatever
	// the week starts on.
	lead := (int(first.Weekday()) - int(weekStart) + 7) % 7
	start := first.AddDate(0, 0, -lead)

	var weeks [][]time.Time
	day := start
	for {
		week := make([]time.Time, 7)
		for i := range week {
			week[i] = day
			day = day.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
		// Stop once the week just added has carried us past the month.
		if day.Month() != first.Month() || day.Year() != first.Year() {
			break
		}
	}
	return weeks
}

// firstOfMonth is midnight on the first day of t's month, in t's location.
func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// sameDay reports whether two times fall on the same calendar day.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// itoa renders a day number without pulling strconv into this file for one
// call that is always one or two digits.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
