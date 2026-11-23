package qtui

import (
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestMonthGrid(t *testing.T) {
	cases := []struct {
		name        string
		month       time.Time
		first, last time.Time
	}{
		// November 2026 starts on a Sunday: the page opens on the Monday before.
		{"month starting on a Sunday", date(2026, time.November, 23), date(2026, time.October, 26), date(2026, time.December, 6)},
		// June 2026 starts on a Monday: no leading days.
		{"month starting on a Monday", date(2026, time.June, 1), date(2026, time.June, 1), date(2026, time.July, 12)},
		// Across the year boundary.
		{"January", date(2027, time.January, 15), date(2026, time.December, 28), date(2027, time.February, 7)},
	}
	for _, testCase := range cases {
		days := monthGrid(testCase.month)
		if len(days) != 42 {
			t.Errorf("%s: %d days, want 42", testCase.name, len(days))
			continue
		}
		if !days[0].Equal(testCase.first) || !days[len(days)-1].Equal(testCase.last) {
			t.Errorf("%s: page runs %s..%s, want %s..%s", testCase.name,
				days[0].Format(time.DateOnly), days[len(days)-1].Format(time.DateOnly),
				testCase.first.Format(time.DateOnly), testCase.last.Format(time.DateOnly))
		}
		if days[0].Weekday() != time.Monday {
			t.Errorf("%s: page starts on a %s", testCase.name, days[0].Weekday())
		}
		for idx := 1; idx < len(days); idx++ {
			if !days[idx].Equal(days[idx-1].AddDate(0, 0, 1)) {
				t.Errorf("%s: day %d does not follow day %d", testCase.name, idx, idx-1)
			}
		}
	}
}

// TestDayOf: a calendar day is the local date of an instant, whatever its
// zone.
func TestDayOf(t *testing.T) {
	late := time.Date(2026, time.November, 23, 23, 30, 0, 0, time.FixedZone("west", -5*60*60))
	if got := dayOf(late); !got.Equal(date(2026, time.November, 23)) {
		t.Errorf("dayOf(%s) = %s", late, got)
	}
}
