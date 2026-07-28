package chartmath

import (
	"testing"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

func TestNearestPoint(t *testing.T) {
	pts := []Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 30, Y: 0}}
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
		if got := NearestPoint(pts, testCase.x); got != testCase.want {
			t.Errorf("NearestPoint(%v) = %d, want %d", testCase.x, got, testCase.want)
		}
	}
}

func TestFillRegionsSingleSide(t *testing.T) {
	// Everything above the reference: one region, closed along it.
	pts := []Point{{X: 0, Y: 10}, {X: 10, Y: 20}, {X: 20, Y: 15}}
	regions := FillRegions(pts, 30)
	if len(regions) != 1 {
		t.Fatalf("got %d regions, want 1: %+v", len(regions), regions)
	}
	region := regions[0]
	if !region.Above {
		t.Errorf("region.Above = false, want true")
	}
	want := []Point{{X: 0, Y: 10}, {X: 10, Y: 20}, {X: 20, Y: 15}, {X: 20, Y: 30}, {X: 0, Y: 30}}
	if len(region.Points) != len(want) {
		t.Fatalf("polygon = %+v, want %+v", region.Points, want)
	}
	for idx := range want {
		if region.Points[idx] != want[idx] {
			t.Errorf("vertex %d = %+v, want %+v", idx, region.Points[idx], want[idx])
		}
	}
}

func TestFillRegionsSplitsAtCrossing(t *testing.T) {
	// One segment crossing the reference at (5,20): a triangle above and a
	// triangle below, split at the interpolated crossing.
	pts := []Point{{X: 0, Y: 10}, {X: 10, Y: 30}}
	regions := FillRegions(pts, 20)
	if len(regions) != 2 {
		t.Fatalf("got %d regions, want 2: %+v", len(regions), regions)
	}
	if !regions[0].Above || regions[1].Above {
		t.Errorf("sides = %v/%v, want above/below", regions[0].Above, regions[1].Above)
	}
	crossing := Point{X: 5, Y: 20}
	if regions[0].Points[1] != crossing {
		t.Errorf("above region crossing = %+v, want %+v", regions[0].Points[1], crossing)
	}
	if regions[1].Points[0] != crossing {
		t.Errorf("below region crossing = %+v, want %+v", regions[1].Points[0], crossing)
	}
}

func TestFillRegionsSplitsAtTouchPoint(t *testing.T) {
	// A point exactly on the reference splits two same-side regions.
	pts := []Point{{X: 0, Y: 10}, {X: 10, Y: 20}, {X: 20, Y: 10}}
	regions := FillRegions(pts, 20)
	if len(regions) != 2 {
		t.Fatalf("got %d regions, want 2: %+v", len(regions), regions)
	}
	for idx, region := range regions {
		if !region.Above {
			t.Errorf("region %d Above = false, want true", idx)
		}
	}
}

func TestSegmentCrossing(t *testing.T) {
	refY := float32(20)
	crossing, ok := SegmentCrossing(Point{X: 0, Y: 10}, Point{X: 10, Y: 30}, refY)
	if !ok || crossing != (Point{X: 5, Y: 20}) {
		t.Errorf("crossing = %+v/%v, want {5 20}/true", crossing, ok)
	}
	sameSide := [][2]Point{
		{{X: 0, Y: 10}, {X: 10, Y: 15}}, // both above
		{{X: 0, Y: 25}, {X: 10, Y: 30}}, // both below
		{{X: 0, Y: 20}, {X: 10, Y: 30}}, // starts on the reference
		{{X: 0, Y: 10}, {X: 10, Y: 20}}, // ends on the reference
	}
	for _, segment := range sameSide {
		if _, crossed := SegmentCrossing(segment[0], segment[1], refY); crossed {
			t.Errorf("SegmentCrossing(%+v, %+v) = true, want false", segment[0], segment[1])
		}
	}
}

func TestFillRegionsFlatOnReference(t *testing.T) {
	// A line lying on the reference encloses nothing.
	pts := []Point{{X: 0, Y: 20}, {X: 10, Y: 20}, {X: 20, Y: 20}}
	if regions := FillRegions(pts, 20); len(regions) != 0 {
		t.Errorf("got %d regions, want 0: %+v", len(regions), regions)
	}
	if regions := FillRegions([]Point{{X: 0, Y: 10}}, 20); regions != nil {
		t.Errorf("single point must yield nil, got %+v", regions)
	}
}

func TestPlotPathMapsExtremes(t *testing.T) {
	// values: min=10 (should map to bottom), max=20 (top).
	pts := PlotPath([]float64{10, 20, 15}, EvenFracs(3), 100, 50, 0, 10, 20)
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
	pts := PlotPath([]float64{1, 2}, EvenFracs(2), 100, 50, 5, 1, 2)
	if pts[0].X != 5 || pts[1].X != 95 {
		t.Errorf("x range = [%v, %v], want [5, 95]", pts[0].X, pts[1].X)
	}
}

func TestPlotPathFlatSeriesCentered(t *testing.T) {
	pts := PlotPath([]float64{5, 5, 5}, EvenFracs(3), 100, 50, 0, 5, 5)
	for idx, point := range pts {
		if point.Y != 25 {
			t.Errorf("flat series point %d y = %v, want 25 (mid)", idx, point.Y)
		}
	}
}

func TestPlotPathDegenerate(t *testing.T) {
	if PlotPath([]float64{1}, EvenFracs(1), 100, 50, 0, 1, 1) != nil {
		t.Error("expected nil for a single point")
	}
	if PlotPath(nil, nil, 100, 50, 0, 0, 0) != nil {
		t.Error("expected nil for no points")
	}
	if PlotPath([]float64{1, 2}, EvenFracs(2), 4, 50, 4, 1, 2) != nil {
		t.Error("expected nil when padding leaves no width")
	}
	if PlotPath([]float64{1, 2}, EvenFracs(3), 100, 50, 0, 1, 2) != nil {
		t.Error("expected nil for mismatched fracs")
	}
}

func TestPlotPathWidenedScale(t *testing.T) {
	// With the scale widened below the series (e.g. a previous close of 5 under
	// values 10..20), the series floats above the bottom: value 10 maps to the
	// scale fraction (10-5)/(20-5)=1/3 up, i.e. y = 50 * 2/3.
	pts := PlotPath([]float64{10, 20}, EvenFracs(2), 100, 50, 0, 5, 20)
	want := float32(50) * 2 / 3
	if diff := pts[0].Y - want; diff < -0.001 || diff > 0.001 {
		t.Errorf("widened-scale min y = %v, want ~%v", pts[0].Y, want)
	}
	if pts[1].Y != 0 {
		t.Errorf("max y = %v, want 0 (top)", pts[1].Y)
	}
}

func TestYFor(t *testing.T) {
	if posY := YFor(10, 10, 20, 50, 0); posY != 50 {
		t.Errorf("lo maps to y=%v, want 50 (bottom)", posY)
	}
	if posY := YFor(20, 10, 20, 50, 0); posY != 0 {
		t.Errorf("hi maps to y=%v, want 0 (top)", posY)
	}
	if posY := YFor(15, 10, 20, 50, 0); posY != 25 {
		t.Errorf("mid maps to y=%v, want 25", posY)
	}
	if posY := YFor(7, 7, 7, 50, 0); posY != 25 {
		t.Errorf("flat scale maps to y=%v, want 25 (centered)", posY)
	}
	if posY := YFor(20, 10, 20, 50, 5); posY != 5 {
		t.Errorf("hi with padding maps to y=%v, want 5", posY)
	}
}

func TestYTicks(t *testing.T) {
	got := YTicks(10, 20, 3)
	want := []float64{20, 15, 10}
	if len(got) != len(want) {
		t.Fatalf("got %d ticks, want %d", len(got), len(want))
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Errorf("tick %d = %v, want %v", idx, got[idx], want[idx])
		}
	}
	if YTicks(10, 10, 3) != nil {
		t.Error("expected nil for a flat scale")
	}
	if YTicks(10, 20, 1) != nil {
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
	fracs := XFracs(series)
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
	fracs := XFracs(series)
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
	if SessionWindow(series) {
		t.Fatal("sessionWindow = true for previous-day candles, want false")
	}
	fracs := XFracs(series)
	want := []float32{0, 0.5, 1}
	for idx := range want {
		if fracs[idx] != want[idx] {
			t.Errorf("frac %d = %v, want %v", idx, fracs[idx], want[idx])
		}
	}
}

func TestXFracsFallsBackToEvenSpacing(t *testing.T) {
	// No session window (e.g. a 1M series): candles are spaced evenly.
	fracs := XFracs(hourlySeries(model.Range1M, 3))
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
	ticks := SessionTicks(start, end, 3, time.UTC)
	if len(ticks) != 3 {
		t.Fatalf("got %d ticks, want 3", len(ticks))
	}
	want := []string{"09:30", "12:45", "16:00"}
	fracs := []float32{0, 0.5, 1}
	for idx, tick := range ticks {
		if tick.Label != want[idx] {
			t.Errorf("tick %d label = %q, want %q", idx, tick.Label, want[idx])
		}
		if tick.Frac != fracs[idx] {
			t.Errorf("tick %d frac = %v, want %v", idx, tick.Frac, fracs[idx])
		}
	}
}

func TestSessionTicksDegenerate(t *testing.T) {
	when := time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC)
	if SessionTicks(when, when, 3, time.UTC) != nil {
		t.Error("expected nil for an empty window")
	}
	if SessionTicks(when, when.Add(time.Hour), 1, time.UTC) != nil {
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
	ticks := XTicks(hourlySeries(model.Range1D, 7), 3, time.UTC)
	if len(ticks) != 3 {
		t.Fatalf("got %d ticks, want 3", len(ticks))
	}
	want := []string{"09:30", "12:30", "15:30"}
	fracs := []float32{0, 0.5, 1}
	for idx, tick := range ticks {
		if tick.Label != want[idx] {
			t.Errorf("tick %d label = %q, want %q", idx, tick.Label, want[idx])
		}
		if tick.Frac != fracs[idx] {
			t.Errorf("tick %d frac = %v, want %v", idx, tick.Frac, fracs[idx])
		}
	}
}

func TestXTicksSpansEndpoints(t *testing.T) {
	ticks := XTicks(hourlySeries(model.Range1D, 100), 4, time.UTC)
	if len(ticks) == 0 {
		t.Fatal("no ticks")
	}
	if ticks[0].Frac != 0 {
		t.Errorf("first frac = %v, want 0", ticks[0].Frac)
	}
	if last := ticks[len(ticks)-1].Frac; last != 1 {
		t.Errorf("last frac = %v, want 1", last)
	}
}

func TestXTicksRespectsLocation(t *testing.T) {
	loc := time.FixedZone("UTC+2", 2*3600)
	ticks := XTicks(hourlySeries(model.Range1D, 2), 2, loc)
	if ticks[0].Label != "11:30" {
		t.Errorf("label = %q, want %q (UTC 09:30 shifted +2h)", ticks[0].Label, "11:30")
	}
}

func TestXTicksDropsDuplicateLabels(t *testing.T) {
	// All candles within the same year: a 5Y-style year format collapses to one.
	ticks := XTicks(hourlySeries(model.Range5Y, 10), 4, time.UTC)
	if len(ticks) != 1 {
		t.Fatalf("got %d ticks, want 1 (duplicate years dropped)", len(ticks))
	}
	if ticks[0].Label != "2026" {
		t.Errorf("label = %q, want %q", ticks[0].Label, "2026")
	}
}

func TestXTicksDegenerate(t *testing.T) {
	if XTicks(hourlySeries(model.Range1D, 1), 3, time.UTC) != nil {
		t.Error("expected nil for a single candle")
	}
	if XTicks(hourlySeries(model.Range1D, 5), 1, time.UTC) != nil {
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
		if got := when.Format(XAxisFormat(testCase.rng)); got != testCase.want {
			t.Errorf("range %s: label = %q, want %q", testCase.rng, got, testCase.want)
		}
	}
}

func TestFormatAxisPrice(t *testing.T) {
	if got := FormatAxisPrice(5432.1); got != "5432.10" {
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
		if got := when.Format(HoverTimeFormat(testCase.rng)); got != testCase.want {
			t.Errorf("range %s: label = %q, want %q", testCase.rng, got, testCase.want)
		}
	}
}
