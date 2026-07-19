package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

func TestNearestPoint(t *testing.T) {
	pts := []fyne.Position{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 30, Y: 0}}
	cases := []struct {
		x    float32
		want int
	}{
		{-5, 0},  // left of the first point clamps to it
		{4, 0},   // closer to 0 than 10
		{6, 1},   // closer to 10
		{19, 1},  // just left of the 10/30 midpoint
		{21, 2},  // just right of it
		{100, 2}, // right of the last point clamps to it
	}
	for _, testCase := range cases {
		if got := nearestPoint(pts, testCase.x); got != testCase.want {
			t.Errorf("nearestPoint(%v) = %d, want %d", testCase.x, got, testCase.want)
		}
	}
}

func TestPlotPathMapsExtremes(t *testing.T) {
	// values: min=10 (should map to bottom), max=20 (top).
	pts := plotPath([]float64{10, 20, 15}, evenFracs(3), 100, 50, 0, 10, 20)
	if len(pts) != 3 {
		t.Fatalf("got %d points, want 3", len(pts))
	}
	if pts[0].X != 0 {
		t.Errorf("first x = %v, want 0", pts[0].X)
	}
	if pts[2].X != 100 {
		t.Errorf("last x = %v, want 100 (spans full width)", pts[2].X)
	}
	if pts[0].Y != 50 {
		t.Errorf("min value y = %v, want 50 (bottom)", pts[0].Y)
	}
	if pts[1].Y != 0 {
		t.Errorf("max value y = %v, want 0 (top)", pts[1].Y)
	}
}

func TestPlotPathPadding(t *testing.T) {
	// With padding 5 in a 100-wide box, x spans [5, 95].
	pts := plotPath([]float64{1, 2}, evenFracs(2), 100, 50, 5, 1, 2)
	if pts[0].X != 5 || pts[1].X != 95 {
		t.Errorf("x range = [%v, %v], want [5, 95]", pts[0].X, pts[1].X)
	}
}

func TestPlotPathFlatSeriesCentered(t *testing.T) {
	pts := plotPath([]float64{5, 5, 5}, evenFracs(3), 100, 50, 0, 5, 5)
	for idx, point := range pts {
		if point.Y != 25 {
			t.Errorf("flat series point %d y = %v, want 25 (mid)", idx, point.Y)
		}
	}
}

func TestPlotPathDegenerate(t *testing.T) {
	if plotPath([]float64{1}, evenFracs(1), 100, 50, 0, 1, 1) != nil {
		t.Error("expected nil for a single point")
	}
	if plotPath(nil, nil, 100, 50, 0, 0, 0) != nil {
		t.Error("expected nil for no points")
	}
	if plotPath([]float64{1, 2}, evenFracs(2), 4, 50, 4, 1, 2) != nil {
		t.Error("expected nil when padding leaves no width")
	}
	if plotPath([]float64{1, 2}, evenFracs(3), 100, 50, 0, 1, 2) != nil {
		t.Error("expected nil for mismatched fracs")
	}
}

func TestPlotPathWidenedScale(t *testing.T) {
	// With the scale widened below the series (e.g. a previous close of 5 under
	// values 10..20), the series floats above the bottom: value 10 maps to the
	// scale fraction (10-5)/(20-5)=1/3 up, i.e. y = 50 * 2/3.
	pts := plotPath([]float64{10, 20}, evenFracs(2), 100, 50, 0, 5, 20)
	want := float32(50) * 2 / 3
	if diff := pts[0].Y - want; diff < -0.001 || diff > 0.001 {
		t.Errorf("widened-scale min y = %v, want ~%v", pts[0].Y, want)
	}
	if pts[1].Y != 0 {
		t.Errorf("max y = %v, want 0 (top)", pts[1].Y)
	}
}

func TestYFor(t *testing.T) {
	if posY := yFor(10, 10, 20, 50, 0); posY != 50 {
		t.Errorf("lo maps to y=%v, want 50 (bottom)", posY)
	}
	if posY := yFor(20, 10, 20, 50, 0); posY != 0 {
		t.Errorf("hi maps to y=%v, want 0 (top)", posY)
	}
	if posY := yFor(15, 10, 20, 50, 0); posY != 25 {
		t.Errorf("mid maps to y=%v, want 25", posY)
	}
	if posY := yFor(7, 7, 7, 50, 0); posY != 25 {
		t.Errorf("flat scale maps to y=%v, want 25 (centered)", posY)
	}
	if posY := yFor(20, 10, 20, 50, 5); posY != 5 {
		t.Errorf("hi with padding maps to y=%v, want 5", posY)
	}
}

func TestYTicks(t *testing.T) {
	got := yTicks(10, 20, 3)
	want := []float64{20, 15, 10}
	if len(got) != len(want) {
		t.Fatalf("got %d ticks, want %d", len(got), len(want))
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Errorf("tick %d = %v, want %v", idx, got[idx], want[idx])
		}
	}
	if yTicks(10, 10, 3) != nil {
		t.Error("expected nil for a flat scale")
	}
	if yTicks(10, 20, 1) != nil {
		t.Error("expected nil for fewer than two ticks")
	}
}

// sessionSeries builds a 1D series over a 09:30–16:00 UTC session whose candles
// cover only the first part of the day (a live, partially traded session).
func sessionSeries(candleTimes ...time.Time) model.Series {
	day := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	series := model.Series{
		Range:        model.Range1D,
		SessionStart: day.Add(9*time.Hour + 30*time.Minute),
		SessionEnd:   day.Add(16 * time.Hour),
	}
	for _, candleTime := range candleTimes {
		series.Candles = append(series.Candles, model.Candle{Time: candleTime, Close: 100})
	}
	return series
}

func TestXFracsSessionScalesToFullWindow(t *testing.T) {
	// Session is 6.5h; candles cover only the first 3.25h → last frac is 0.5.
	series := sessionSeries(
		time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 7, 17, 12, 45, 0, 0, time.UTC),
	)
	fracs := xFracs(series)
	if len(fracs) != 2 {
		t.Fatalf("got %d fracs, want 2", len(fracs))
	}
	if fracs[0] != 0 {
		t.Errorf("session-start frac = %v, want 0", fracs[0])
	}
	if fracs[1] != 0.5 {
		t.Errorf("mid-session frac = %v, want 0.5 (half the window)", fracs[1])
	}
}

func TestXFracsClampsOutsideSession(t *testing.T) {
	series := sessionSeries(
		time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC),   // pre-market
		time.Date(2026, 7, 17, 16, 30, 0, 0, time.UTC), // post-market
	)
	fracs := xFracs(series)
	if fracs[0] != 0 || fracs[1] != 1 {
		t.Errorf("fracs = %v, want [0 1] (clamped to the session)", fracs)
	}
}

func TestXFracsPreviousDayFallsBackToEvenSpacing(t *testing.T) {
	// Market closed: Yahoo pairs the previous day's candles with the upcoming
	// session in currentTradingPeriod. The completed day must span the full
	// width instead of collapsing onto the left edge of the future window —
	// once the session opens and its candles arrive, window mode takes over
	// again (covered by TestXFracsSessionScalesToFullWindow).
	series := sessionSeries(
		time.Date(2026, 7, 16, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 7, 16, 12, 45, 0, 0, time.UTC),
		time.Date(2026, 7, 16, 16, 0, 0, 0, time.UTC),
	)
	if sessionWindow(series) {
		t.Fatal("sessionWindow = true for previous-day candles, want false")
	}
	fracs := xFracs(series)
	want := []float32{0, 0.5, 1}
	for idx := range want {
		if fracs[idx] != want[idx] {
			t.Errorf("frac %d = %v, want %v", idx, fracs[idx], want[idx])
		}
	}
}

func TestXFracsFallsBackToEvenSpacing(t *testing.T) {
	// No session window (e.g. a 1M series): candles are spaced evenly.
	fracs := xFracs(hourlySeries(model.Range1M, 3))
	want := []float32{0, 0.5, 1}
	for idx := range want {
		if fracs[idx] != want[idx] {
			t.Errorf("frac %d = %v, want %v", idx, fracs[idx], want[idx])
		}
	}
}

func TestSessionTicksSpanFullWindow(t *testing.T) {
	start := time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC)
	end := time.Date(2026, 7, 17, 16, 0, 0, 0, time.UTC)
	ticks := sessionTicks(start, end, 3, time.UTC)
	if len(ticks) != 3 {
		t.Fatalf("got %d ticks, want 3", len(ticks))
	}
	want := []string{"09:30", "12:45", "16:00"}
	fracs := []float32{0, 0.5, 1}
	for idx, tick := range ticks {
		if tick.label != want[idx] {
			t.Errorf("tick %d label = %q, want %q", idx, tick.label, want[idx])
		}
		if tick.frac != fracs[idx] {
			t.Errorf("tick %d frac = %v, want %v", idx, tick.frac, fracs[idx])
		}
	}
}

func TestSessionTicksDegenerate(t *testing.T) {
	when := time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC)
	if sessionTicks(when, when, 3, time.UTC) != nil {
		t.Error("expected nil for an empty window")
	}
	if sessionTicks(when, when.Add(time.Hour), 1, time.UTC) != nil {
		t.Error("expected nil when fewer than two ticks fit")
	}
}

// hourlySeries builds a series with n candles spaced an hour apart (UTC).
func hourlySeries(rng model.Range, count int) model.Series {
	start := time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC)
	series := model.Series{Range: rng, Candles: make([]model.Candle, count)}
	for idx := range series.Candles {
		series.Candles[idx] = model.Candle{Time: start.Add(time.Duration(idx) * time.Hour), Close: 100}
	}
	return series
}

func TestXTicks1DShowsHours(t *testing.T) {
	ticks := xTicks(hourlySeries(model.Range1D, 7), 3, time.UTC)
	if len(ticks) != 3 {
		t.Fatalf("got %d ticks, want 3", len(ticks))
	}
	want := []string{"09:30", "12:30", "15:30"}
	fracs := []float32{0, 0.5, 1}
	for idx, tick := range ticks {
		if tick.label != want[idx] {
			t.Errorf("tick %d label = %q, want %q", idx, tick.label, want[idx])
		}
		if tick.frac != fracs[idx] {
			t.Errorf("tick %d frac = %v, want %v", idx, tick.frac, fracs[idx])
		}
	}
}

func TestXTicksSpansEndpoints(t *testing.T) {
	ticks := xTicks(hourlySeries(model.Range1D, 100), 4, time.UTC)
	if len(ticks) == 0 {
		t.Fatal("no ticks")
	}
	if ticks[0].frac != 0 {
		t.Errorf("first frac = %v, want 0", ticks[0].frac)
	}
	if last := ticks[len(ticks)-1].frac; last != 1 {
		t.Errorf("last frac = %v, want 1", last)
	}
}

func TestXTicksRespectsLocation(t *testing.T) {
	loc := time.FixedZone("UTC+2", 2*3600)
	ticks := xTicks(hourlySeries(model.Range1D, 2), 2, loc)
	if ticks[0].label != "11:30" {
		t.Errorf("label = %q, want %q (UTC 09:30 shifted +2h)", ticks[0].label, "11:30")
	}
}

func TestXTicksDropsDuplicateLabels(t *testing.T) {
	// All candles within the same year: a 5Y-style year format collapses to one.
	ticks := xTicks(hourlySeries(model.Range5Y, 10), 4, time.UTC)
	if len(ticks) != 1 {
		t.Fatalf("got %d ticks, want 1 (duplicate years dropped)", len(ticks))
	}
	if ticks[0].label != "2026" {
		t.Errorf("label = %q, want %q", ticks[0].label, "2026")
	}
}

func TestXTicksDegenerate(t *testing.T) {
	if xTicks(hourlySeries(model.Range1D, 1), 3, time.UTC) != nil {
		t.Error("expected nil for a single candle")
	}
	if xTicks(hourlySeries(model.Range1D, 5), 1, time.UTC) != nil {
		t.Error("expected nil when fewer than two ticks fit")
	}
}

func TestXAxisFormatPerRange(t *testing.T) {
	when := time.Date(2026, 7, 17, 15, 4, 0, 0, time.UTC) // a Friday
	cases := []struct {
		rng  model.Range
		want string
	}{
		{model.Range1D, "15:04"},
		{model.Range5D, "Fri 17"},
		{model.Range1W, "Fri 17"},
		{model.Range1M, "17 Jul"},
		{model.RangeYTD, "Jul"},
		{model.Range1Y, "Jul"},
		{model.Range5Y, "2026"},
		{model.RangeAll, "2026"},
	}
	for _, testCase := range cases {
		if got := when.Format(xAxisFormat(testCase.rng)); got != testCase.want {
			t.Errorf("range %s: label = %q, want %q", testCase.rng, got, testCase.want)
		}
	}
}

func TestFormatAxisPrice(t *testing.T) {
	if got := formatAxisPrice(5432.1); got != "5432.10" {
		t.Errorf("got %q, want %q", got, "5432.10")
	}
}

func TestHoverTimeFormatPerRange(t *testing.T) {
	when := time.Date(2026, 7, 17, 15, 4, 0, 0, time.UTC) // a Friday
	cases := []struct {
		rng  model.Range
		want string
	}{
		{model.Range1D, "15:04"},
		{model.Range5D, "Fri, 17 Jul"},
		{model.Range1W, "Fri, 17 Jul"},
		{model.Range1M, "Fri, 17 Jul"},
		{model.RangeYTD, "17 Jul 2026"},
		{model.Range1Y, "17 Jul 2026"},
		{model.Range5Y, "17 Jul 2026"},
		{model.RangeAll, "17 Jul 2026"},
	}
	for _, testCase := range cases {
		if got := when.Format(hoverTimeFormat(testCase.rng)); got != testCase.want {
			t.Errorf("range %s: label = %q, want %q", testCase.rng, got, testCase.want)
		}
	}
}
