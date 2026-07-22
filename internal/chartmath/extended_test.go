package chartmath

import (
	"testing"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// extendedSeries builds a 1D series with a synthetic UTC trading day —
// pre-market 08:00–09:30, regular 09:30–16:00, after-hours 16:00–20:00 —
// and candles at the given times with closes 100, 101, 102, ...
func extendedSeries(candleTimes ...time.Time) model.Series {
	day := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	series := model.Series{
		Range:        model.Range1D,
		PreStart:     day.Add(8 * time.Hour),
		SessionStart: day.Add(9*time.Hour + 30*time.Minute),
		SessionEnd:   day.Add(16 * time.Hour),
		PostEnd:      day.Add(20 * time.Hour),
	}
	for idx, candleTime := range candleTimes {
		series.Candles = append(series.Candles, model.Candle{Time: candleTime, Close: 100 + float64(idx)})
	}
	return series
}

// clock returns an instant on the synthetic trading day.
func clock(hour, minute int) time.Time {
	return time.Date(2026, 7, 17, hour, minute, 0, 0, time.UTC)
}

func TestStateAt(t *testing.T) {
	series := extendedSeries()
	cases := []struct {
		name string
		now  time.Time
		want MarketState
	}{
		{"before pre", clock(7, 0), MarketClosed},
		{"pre opens", clock(8, 0), MarketPreMarket},
		{"mid pre", clock(9, 0), MarketPreMarket},
		{"market opens", clock(9, 30), MarketRegular},
		{"mid session", clock(12, 0), MarketRegular},
		{"market closes", clock(16, 0), MarketAfterHours},
		{"mid after-hours", clock(17, 0), MarketAfterHours},
		{"post closes", clock(20, 0), MarketClosed},
		{"late night", clock(23, 0), MarketClosed},
	}
	for _, testCase := range cases {
		if got := StateAt(series, testCase.now); got != testCase.want {
			t.Errorf("%s: StateAt = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestStateAtMissingWindows(t *testing.T) {
	// Without a pre window a pre-market instant is just Closed; same for post.
	noPre := extendedSeries()
	noPre.PreStart = time.Time{}
	if got := StateAt(noPre, clock(9, 0)); got != MarketClosed {
		t.Errorf("no pre window: StateAt = %v, want MarketClosed", got)
	}
	noPost := extendedSeries()
	noPost.PostEnd = time.Time{}
	if got := StateAt(noPost, clock(17, 0)); got != MarketClosed {
		t.Errorf("no post window: StateAt = %v, want MarketClosed", got)
	}
	// Without a valid regular window everything is Closed.
	noRegular := extendedSeries()
	noRegular.SessionStart, noRegular.SessionEnd = time.Time{}, time.Time{}
	if got := StateAt(noRegular, clock(12, 0)); got != MarketClosed {
		t.Errorf("no regular window: StateAt = %v, want MarketClosed", got)
	}
}

func TestBuildExtendedDisplayPreMarket(t *testing.T) {
	series := extendedSeries(clock(8, 30), clock(9, 0))
	display := BuildExtendedDisplay(series, clock(9, 15))

	if display.State != MarketPreMarket {
		t.Fatalf("State = %v, want MarketPreMarket", display.State)
	}
	if len(display.Series.Candles) != 2 {
		t.Fatalf("got %d candles, want 2: %+v", len(display.Series.Candles), display.Series.Candles)
	}
	// The display window is the pre session, so the chart fills it gradually.
	if !display.Series.SessionStart.Equal(clock(8, 0)) || !display.Series.SessionEnd.Equal(clock(9, 30)) {
		t.Errorf("display window = %v..%v, want 08:00..09:30",
			display.Series.SessionStart, display.Series.SessionEnd)
	}
	if !SessionWindow(display.Series) {
		t.Error("SessionWindow(display) = false, want true")
	}
	if display.ExtendedPrice != 101 {
		t.Errorf("ExtendedPrice = %v, want 101", display.ExtendedPrice)
	}
	if display.DimFromIdx != -1 || !display.BoundaryTime.IsZero() {
		t.Errorf("unexpected after-hours overlay: dimFrom=%d boundary=%v",
			display.DimFromIdx, display.BoundaryTime)
	}
}

func TestBuildExtendedDisplayPreMarketDropsOutsideCandles(t *testing.T) {
	// A stray candle outside the pre window (e.g. the previous day) is dropped.
	series := extendedSeries(clock(7, 0), clock(8, 30))
	display := BuildExtendedDisplay(series, clock(9, 0))
	if len(display.Series.Candles) != 1 || display.Series.Candles[0].Time != clock(8, 30) {
		t.Errorf("candles = %+v, want only the 08:30 one", display.Series.Candles)
	}
}

func TestBuildExtendedDisplayPreMarketWithoutDataDemotesToClosed(t *testing.T) {
	series := extendedSeries(clock(12, 0)) // no candles inside the pre window
	display := BuildExtendedDisplay(series, clock(9, 0))
	if display.State != MarketClosed {
		t.Fatalf("State = %v, want MarketClosed", display.State)
	}
	if len(display.Series.Candles) != len(series.Candles) {
		t.Errorf("demoted display must leave the series untouched: %+v", display.Series.Candles)
	}
}

func TestBuildExtendedDisplayRegularDropsPreCandles(t *testing.T) {
	series := extendedSeries(clock(9, 0), clock(10, 0), clock(11, 0))
	display := BuildExtendedDisplay(series, clock(12, 0))

	if display.State != MarketRegular {
		t.Fatalf("State = %v, want MarketRegular", display.State)
	}
	// The morning's pre candle is dropped so the open-market chart matches
	// the regular fetch exactly.
	if len(display.Series.Candles) != 2 || display.Series.Candles[0].Time != clock(10, 0) {
		t.Fatalf("candles = %+v, want the two regular ones", display.Series.Candles)
	}
	if !display.Series.SessionStart.Equal(series.SessionStart) || !display.Series.SessionEnd.Equal(series.SessionEnd) {
		t.Errorf("regular display must keep the regular window")
	}
	if display.ExtendedPrice != 0 || display.DimFromIdx != -1 {
		t.Errorf("regular display must carry no extended price/overlay: %+v", display)
	}
}

func TestBuildExtendedDisplayAfterHours(t *testing.T) {
	series := extendedSeries(clock(10, 0), clock(15, 0), clock(16, 30), clock(17, 0))
	display := BuildExtendedDisplay(series, clock(17, 30))

	if display.State != MarketAfterHours {
		t.Fatalf("State = %v, want MarketAfterHours", display.State)
	}
	if len(display.Series.Candles) != 4 {
		t.Fatalf("got %d candles, want 4: %+v", len(display.Series.Candles), display.Series.Candles)
	}
	// The window stretches to the post end so the tail fills in gradually.
	if !display.Series.SessionStart.Equal(series.SessionStart) || !display.Series.SessionEnd.Equal(clock(20, 0)) {
		t.Errorf("display window = %v..%v, want 09:30..20:00",
			display.Series.SessionStart, display.Series.SessionEnd)
	}
	if !SessionWindow(display.Series) {
		t.Error("SessionWindow(display) = false, want true")
	}
	if !display.BoundaryTime.Equal(clock(16, 0)) {
		t.Errorf("BoundaryTime = %v, want 16:00", display.BoundaryTime)
	}
	if display.DimFromIdx != 2 {
		t.Errorf("DimFromIdx = %d, want 2", display.DimFromIdx)
	}
	if display.ExtendedPrice != 103 {
		t.Errorf("ExtendedPrice = %v, want 103", display.ExtendedPrice)
	}
}

func TestBuildExtendedDisplayAfterHoursWithoutPostCandles(t *testing.T) {
	series := extendedSeries(clock(10, 0), clock(15, 0))
	display := BuildExtendedDisplay(series, clock(16, 30))

	if display.State != MarketAfterHours {
		t.Fatalf("State = %v, want MarketAfterHours", display.State)
	}
	// Without post data the completed regular day renders unchanged.
	if len(display.Series.Candles) != 2 {
		t.Fatalf("got %d candles, want 2: %+v", len(display.Series.Candles), display.Series.Candles)
	}
	if !display.Series.SessionEnd.Equal(series.SessionEnd) {
		t.Errorf("SessionEnd = %v, want the regular close", display.Series.SessionEnd)
	}
	if display.DimFromIdx != -1 || display.ExtendedPrice != 0 || !display.BoundaryTime.IsZero() {
		t.Errorf("expected no overlay/extended price: %+v", display)
	}
}

func TestBuildExtendedDisplayClosedWithoutPostCandlesPassesThrough(t *testing.T) {
	// No post candles: nothing to replay, so the caller can fall back to a
	// regular fetch (signalled by the untouched series and DimFromIdx -1).
	series := extendedSeries(clock(10, 0), clock(15, 0))
	display := BuildExtendedDisplay(series, clock(23, 0))
	if display.State != MarketClosed {
		t.Fatalf("State = %v, want MarketClosed", display.State)
	}
	if len(display.Series.Candles) != 2 || !display.Series.SessionEnd.Equal(series.SessionEnd) {
		t.Errorf("closed display without post candles must pass the series through untouched")
	}
	if display.DimFromIdx != -1 || display.ExtendedPrice != 0 || !display.BoundaryTime.IsZero() {
		t.Errorf("expected no overlay/extended price: %+v", display)
	}
}

// shiftWindows moves every session window forward by whole days, mimicking
// the upcoming-session windows a closed-market response carries.
func shiftWindows(series model.Series, days int) model.Series {
	shift := time.Duration(days) * 24 * time.Hour
	series.PreStart = series.PreStart.Add(shift)
	series.SessionStart = series.SessionStart.Add(shift)
	series.SessionEnd = series.SessionEnd.Add(shift)
	series.PostEnd = series.PostEnd.Add(shift)
	return series
}

func TestBuildExtendedDisplayClosedReplaysAfterHours(t *testing.T) {
	// Late night, same-day windows: the completed session replays with its
	// after-hours tail dimmed behind the regular-close divider.
	series := extendedSeries(clock(9, 0), clock(10, 0), clock(15, 0), clock(17, 0))
	display := BuildExtendedDisplay(series, clock(23, 0))

	if display.State != MarketClosed {
		t.Fatalf("State = %v, want MarketClosed", display.State)
	}
	// The 09:00 pre candle is dropped; regular + post remain.
	if len(display.Series.Candles) != 3 || display.Series.Candles[0].Time != clock(10, 0) {
		t.Fatalf("candles = %+v, want the regular + post ones", display.Series.Candles)
	}
	if !display.Series.SessionStart.Equal(clock(9, 30)) || !display.Series.SessionEnd.Equal(clock(20, 0)) {
		t.Errorf("display window = %v..%v, want 09:30..20:00",
			display.Series.SessionStart, display.Series.SessionEnd)
	}
	if !display.BoundaryTime.Equal(clock(16, 0)) {
		t.Errorf("BoundaryTime = %v, want 16:00", display.BoundaryTime)
	}
	if display.DimFromIdx != 2 {
		t.Errorf("DimFromIdx = %d, want 2", display.DimFromIdx)
	}
	if display.ExtendedPrice != 103 {
		t.Errorf("ExtendedPrice = %v, want 103", display.ExtendedPrice)
	}
}

func TestBuildExtendedDisplayClosedShiftsUpcomingWindows(t *testing.T) {
	// Overnight (windows one day ahead) and over a weekend (three days
	// ahead) the windows must translate back onto the candles' day.
	for _, daysAhead := range []int{1, 3} {
		series := shiftWindows(extendedSeries(clock(10, 0), clock(15, 0), clock(17, 0)), daysAhead)
		now := clock(7, 0).Add(time.Duration(daysAhead) * 24 * time.Hour)
		display := BuildExtendedDisplay(series, now)

		if display.State != MarketClosed {
			t.Fatalf("%d days ahead: State = %v, want MarketClosed", daysAhead, display.State)
		}
		if len(display.Series.Candles) != 3 {
			t.Fatalf("%d days ahead: candles = %+v, want all 3", daysAhead, display.Series.Candles)
		}
		if !display.Series.SessionStart.Equal(clock(9, 30)) || !display.Series.SessionEnd.Equal(clock(20, 0)) {
			t.Errorf("%d days ahead: display window = %v..%v, want 09:30..20:00",
				daysAhead, display.Series.SessionStart, display.Series.SessionEnd)
		}
		if !display.BoundaryTime.Equal(clock(16, 0)) {
			t.Errorf("%d days ahead: BoundaryTime = %v, want 16:00", daysAhead, display.BoundaryTime)
		}
		if display.DimFromIdx != 2 || display.ExtendedPrice != 102 {
			t.Errorf("%d days ahead: DimFromIdx = %d, ExtendedPrice = %v, want 2 and 102",
				daysAhead, display.DimFromIdx, display.ExtendedPrice)
		}
	}
}

func TestBuildExtendedDisplayPreMarketWithoutDataReplaysLastSession(t *testing.T) {
	// Early pre-market with no pre candles yet: the response still carries
	// the previous session's candles, so its after-hours replay shows.
	series := shiftWindows(extendedSeries(clock(10, 0), clock(15, 0), clock(17, 0)), 1)
	now := clock(8, 30).Add(24 * time.Hour) // inside the upcoming pre window
	display := BuildExtendedDisplay(series, now)

	if display.State != MarketClosed {
		t.Fatalf("State = %v, want MarketClosed", display.State)
	}
	if display.DimFromIdx != 2 || !display.BoundaryTime.Equal(clock(16, 0)) {
		t.Errorf("expected the replay overlay: %+v", display)
	}
}

func TestBuildExtendedDisplayClosedWindowsTooFarAhead(t *testing.T) {
	// Windows further ahead than the lookback bound cannot be located; the
	// series passes through untouched.
	series := shiftWindows(extendedSeries(clock(10, 0), clock(17, 0)), 10)
	display := BuildExtendedDisplay(series, clock(23, 0))
	if display.State != MarketClosed || display.DimFromIdx != -1 {
		t.Errorf("expected an untouched closed display: %+v", display)
	}
	if len(display.Series.Candles) != 2 {
		t.Errorf("candles must pass through untouched: %+v", display.Series.Candles)
	}
}
