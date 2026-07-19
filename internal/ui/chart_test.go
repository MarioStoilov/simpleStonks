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
	for _, c := range cases {
		if got := nearestPoint(pts, c.x); got != c.want {
			t.Errorf("nearestPoint(%v) = %d, want %d", c.x, got, c.want)
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
	for i, p := range pts {
		if p.Y != 25 {
			t.Errorf("flat series point %d y = %v, want 25 (mid)", i, p.Y)
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
	if d := pts[0].Y - want; d < -0.001 || d > 0.001 {
		t.Errorf("widened-scale min y = %v, want ~%v", pts[0].Y, want)
	}
	if pts[1].Y != 0 {
		t.Errorf("max y = %v, want 0 (top)", pts[1].Y)
	}
}

func TestYFor(t *testing.T) {
	if y := yFor(10, 10, 20, 50, 0); y != 50 {
		t.Errorf("lo maps to y=%v, want 50 (bottom)", y)
	}
	if y := yFor(20, 10, 20, 50, 0); y != 0 {
		t.Errorf("hi maps to y=%v, want 0 (top)", y)
	}
	if y := yFor(15, 10, 20, 50, 0); y != 25 {
		t.Errorf("mid maps to y=%v, want 25", y)
	}
	if y := yFor(7, 7, 7, 50, 0); y != 25 {
		t.Errorf("flat scale maps to y=%v, want 25 (centered)", y)
	}
	if y := yFor(20, 10, 20, 50, 5); y != 5 {
		t.Errorf("hi with padding maps to y=%v, want 5", y)
	}
}

func TestYTicks(t *testing.T) {
	got := yTicks(10, 20, 3)
	want := []float64{20, 15, 10}
	if len(got) != len(want) {
		t.Fatalf("got %d ticks, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tick %d = %v, want %v", i, got[i], want[i])
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
	s := model.Series{
		Range:        model.Range1D,
		SessionStart: day.Add(9*time.Hour + 30*time.Minute),
		SessionEnd:   day.Add(16 * time.Hour),
	}
	for _, at := range candleTimes {
		s.Candles = append(s.Candles, model.Candle{Time: at, Close: 100})
	}
	return s
}

func TestXFracsSessionScalesToFullWindow(t *testing.T) {
	// Session is 6.5h; candles cover only the first 3.25h → last frac is 0.5.
	s := sessionSeries(
		time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 7, 17, 12, 45, 0, 0, time.UTC),
	)
	fr := xFracs(s)
	if len(fr) != 2 {
		t.Fatalf("got %d fracs, want 2", len(fr))
	}
	if fr[0] != 0 {
		t.Errorf("session-start frac = %v, want 0", fr[0])
	}
	if fr[1] != 0.5 {
		t.Errorf("mid-session frac = %v, want 0.5 (half the window)", fr[1])
	}
}

func TestXFracsClampsOutsideSession(t *testing.T) {
	s := sessionSeries(
		time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC),   // pre-market
		time.Date(2026, 7, 17, 16, 30, 0, 0, time.UTC), // post-market
	)
	fr := xFracs(s)
	if fr[0] != 0 || fr[1] != 1 {
		t.Errorf("fracs = %v, want [0 1] (clamped to the session)", fr)
	}
}

func TestXFracsPreviousDayFallsBackToEvenSpacing(t *testing.T) {
	// Market closed: Yahoo pairs the previous day's candles with the upcoming
	// session in currentTradingPeriod. The completed day must span the full
	// width instead of collapsing onto the left edge of the future window —
	// once the session opens and its candles arrive, window mode takes over
	// again (covered by TestXFracsSessionScalesToFullWindow).
	s := sessionSeries(
		time.Date(2026, 7, 16, 9, 30, 0, 0, time.UTC),
		time.Date(2026, 7, 16, 12, 45, 0, 0, time.UTC),
		time.Date(2026, 7, 16, 16, 0, 0, 0, time.UTC),
	)
	if sessionWindow(s) {
		t.Fatal("sessionWindow = true for previous-day candles, want false")
	}
	fr := xFracs(s)
	want := []float32{0, 0.5, 1}
	for i := range want {
		if fr[i] != want[i] {
			t.Errorf("frac %d = %v, want %v", i, fr[i], want[i])
		}
	}
}

func TestXFracsFallsBackToEvenSpacing(t *testing.T) {
	// No session window (e.g. a 1M series): candles are spaced evenly.
	fr := xFracs(hourlySeries(model.Range1M, 3))
	want := []float32{0, 0.5, 1}
	for i := range want {
		if fr[i] != want[i] {
			t.Errorf("frac %d = %v, want %v", i, fr[i], want[i])
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
	for i, tk := range ticks {
		if tk.label != want[i] {
			t.Errorf("tick %d label = %q, want %q", i, tk.label, want[i])
		}
		if tk.frac != fracs[i] {
			t.Errorf("tick %d frac = %v, want %v", i, tk.frac, fracs[i])
		}
	}
}

func TestSessionTicksDegenerate(t *testing.T) {
	at := time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC)
	if sessionTicks(at, at, 3, time.UTC) != nil {
		t.Error("expected nil for an empty window")
	}
	if sessionTicks(at, at.Add(time.Hour), 1, time.UTC) != nil {
		t.Error("expected nil when fewer than two ticks fit")
	}
}

// hourlySeries builds a series with n candles spaced an hour apart (UTC).
func hourlySeries(r model.Range, n int) model.Series {
	start := time.Date(2026, 7, 17, 9, 30, 0, 0, time.UTC)
	s := model.Series{Range: r, Candles: make([]model.Candle, n)}
	for i := range s.Candles {
		s.Candles[i] = model.Candle{Time: start.Add(time.Duration(i) * time.Hour), Close: 100}
	}
	return s
}

func TestXTicks1DShowsHours(t *testing.T) {
	ticks := xTicks(hourlySeries(model.Range1D, 7), 3, time.UTC)
	if len(ticks) != 3 {
		t.Fatalf("got %d ticks, want 3", len(ticks))
	}
	want := []string{"09:30", "12:30", "15:30"}
	fracs := []float32{0, 0.5, 1}
	for i, tk := range ticks {
		if tk.label != want[i] {
			t.Errorf("tick %d label = %q, want %q", i, tk.label, want[i])
		}
		if tk.frac != fracs[i] {
			t.Errorf("tick %d frac = %v, want %v", i, tk.frac, fracs[i])
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
	at := time.Date(2026, 7, 17, 15, 4, 0, 0, time.UTC) // a Friday
	cases := []struct {
		r    model.Range
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
	for _, c := range cases {
		if got := at.Format(xAxisFormat(c.r)); got != c.want {
			t.Errorf("range %s: label = %q, want %q", c.r, got, c.want)
		}
	}
}

func TestFormatAxisPrice(t *testing.T) {
	if got := formatAxisPrice(5432.1); got != "5432.10" {
		t.Errorf("got %q, want %q", got, "5432.10")
	}
}

func TestHoverTimeFormatPerRange(t *testing.T) {
	at := time.Date(2026, 7, 17, 15, 4, 0, 0, time.UTC) // a Friday
	cases := []struct {
		r    model.Range
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
	for _, c := range cases {
		if got := at.Format(hoverTimeFormat(c.r)); got != c.want {
			t.Errorf("range %s: label = %q, want %q", c.r, got, c.want)
		}
	}
}
