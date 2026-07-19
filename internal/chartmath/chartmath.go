// Package chartmath holds the pure geometry and axis-tick logic behind the
// price charts: value→pixel mapping, session-window x fractions for the
// gradually-filling intraday view, and axis tick selection/formatting. It is
// toolkit-neutral so every UI (Fyne, Qt, ...) renders identical charts from
// the same math.
package chartmath

import (
	"fmt"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
)

// Point is a plotted pixel position.
type Point struct {
	X, Y float32
}

// AxisTick is one x-axis label at a fractional position along the plot width.
type AxisTick struct {
	Frac  float32 // 0 (left edge) .. 1 (right edge)
	Label string
}

// XTicks picks up to max evenly spaced candles from the series and formats
// their timestamps (in loc) for the series' range. Consecutive duplicate
// labels — e.g. the same year twice on a 5Y chart — are dropped.
func XTicks(series model.Series, max int, loc *time.Location) []AxisTick {
	count := len(series.Candles)
	if count < 2 || max < 2 {
		return nil
	}
	if max > count {
		max = count
	}
	layout := XAxisFormat(series.Range)
	out := make([]AxisTick, 0, max)
	prev := ""
	for tickIdx := 0; tickIdx < max; tickIdx++ {
		candleIdx := tickIdx * (count - 1) / (max - 1)
		label := series.Candles[candleIdx].Time.In(loc).Format(layout)
		if label == prev {
			continue
		}
		prev = label
		out = append(out, AxisTick{Frac: float32(candleIdx) / float32(count-1), Label: label})
	}
	return out
}

// XAxisFormat maps a chart range to the time layout of its x-axis labels:
// hours intraday, then days, dates, months, and years as the span grows.
func XAxisFormat(rng model.Range) string {
	switch rng {
	case model.Range1D:
		return constants.TimeFmtClock
	case model.Range5D, model.Range1W:
		return constants.TimeFmtWeekdayDay
	case model.Range1M:
		return constants.TimeFmtDayMonth
	case model.RangeYTD, model.Range1Y:
		return constants.TimeFmtMonth
	default: // 5Y, ALL
		return constants.TimeFmtYear
	}
}

// HoverTimeFormat maps a chart range to the format of the hover guide's
// x-axis label: clock time intraday, calendar dates beyond, with the year for
// multi-year spans.
func HoverTimeFormat(rng model.Range) string {
	switch rng {
	case model.Range1D:
		return constants.TimeFmtClock
	case model.Range5D, model.Range1W, model.Range1M:
		return constants.TimeFmtWeekdayDate
	default: // YTD, 1Y, 5Y, ALL
		return constants.TimeFmtFullDate
	}
}

// FormatAxisPrice formats a y-axis price label.
func FormatAxisPrice(value float64) string { return fmt.Sprintf(constants.FmtPrice, value) }

// ClosesOf extracts the closing prices from a series.
func ClosesOf(series model.Series) []float64 {
	out := make([]float64, len(series.Candles))
	for idx, candle := range series.Candles {
		out[idx] = candle.Close
	}
	return out
}

// PlotPath maps close values to pixel positions within a width×height box
// (inset by pad): x from the given 0..1 fractions, y scaled to the [low, high]
// value scale, inverted so higher values sit toward the top. It returns nil
// when there is nothing meaningful to draw (fewer than two points, mismatched
// fractions, or no room after padding).
func PlotPath(values []float64, fracs []float32, width, height, pad float32, low, high float64) []Point {
	if len(values) < 2 || len(fracs) != len(values) {
		return nil
	}
	innerWidth := width - 2*pad
	if innerWidth <= 0 || height-2*pad <= 0 {
		return nil
	}
	points := make([]Point, len(values))
	for idx, value := range values {
		points[idx] = Point{X: pad + innerWidth*fracs[idx], Y: YFor(value, low, high, height, pad)}
	}
	return points
}

// EvenFracs returns count positions spaced evenly across 0..1.
func EvenFracs(count int) []float32 {
	if count < 2 {
		return nil
	}
	out := make([]float32, count)
	for idx := range out {
		out[idx] = float32(idx) / float32(count-1)
	}
	return out
}

// SessionWindow reports whether a series should be drawn against its full
// trading-session window: intraday data with known session bounds whose
// candles actually belong to that session. While a market is closed, Yahoo
// pairs the *previous* day's candles with the *upcoming* session in
// currentTradingPeriod; drawing those against the future window would
// collapse them onto its left edge, so they fall back to even spacing (the
// completed day spans the full width). Once the session opens and its first
// candles arrive, the window mode kicks in and the new chart starts filling
// in from the left.
func SessionWindow(series model.Series) bool {
	if !series.Range.Intraday() || !series.SessionEnd.After(series.SessionStart) || len(series.Candles) == 0 {
		return false
	}
	last := series.Candles[len(series.Candles)-1].Time
	return !last.Before(series.SessionStart)
}

// XFracs returns each candle's horizontal position as a 0..1 fraction. Within
// a session window the fraction is time-based over the whole session, so a
// live trading day fills in gradually (Yahoo-style); otherwise candles are
// spaced evenly.
func XFracs(series model.Series) []float32 {
	if !SessionWindow(series) {
		return EvenFracs(len(series.Candles))
	}
	span := float32(series.SessionEnd.Sub(series.SessionStart))
	out := make([]float32, len(series.Candles))
	for idx, candle := range series.Candles {
		frac := float32(candle.Time.Sub(series.SessionStart)) / span
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		out[idx] = frac
	}
	return out
}

// SessionTicks returns up to max evenly spaced intraday time labels spanning a
// trading-session window.
func SessionTicks(start, end time.Time, max int, loc *time.Location) []AxisTick {
	if max < 2 || !end.After(start) {
		return nil
	}
	layout := XAxisFormat(model.Range1D)
	span := end.Sub(start)
	out := make([]AxisTick, 0, max)
	prev := ""
	for tickIdx := 0; tickIdx < max; tickIdx++ {
		tickTime := start.Add(span * time.Duration(tickIdx) / time.Duration(max-1))
		label := tickTime.In(loc).Format(layout)
		if label == prev {
			continue
		}
		prev = label
		out = append(out, AxisTick{Frac: float32(tickIdx) / float32(max-1), Label: label})
	}
	return out
}

// YTicks returns max evenly spaced reference values from high (first) down to
// low.
func YTicks(low, high float64, max int) []float64 {
	if max < 2 || high <= low {
		return nil
	}
	out := make([]float64, max)
	for idx := range out {
		out[idx] = high - (high-low)*float64(idx)/float64(max-1)
	}
	return out
}

// YFor maps a value on the [low, high] scale to a y pixel within a height-tall
// box inset by pad, inverted so high sits at the top. A zero-width scale
// centers.
func YFor(value, low, high float64, height, pad float32) float32 {
	normalized := 0.5 // flat scale sits in the middle
	if high != low {
		normalized = (value - low) / (high - low)
	}
	return pad + (height-2*pad)*float32(1-normalized)
}

// NearestPoint returns the index of the point whose x coordinate is closest to
// targetX. Points must be ordered by non-decreasing x (as plotted points are).
func NearestPoint(points []Point, targetX float32) int {
	best := 0
	bestDist := float32(-1)
	for idx, point := range points {
		dist := point.X - targetX
		if dist < 0 {
			dist = -dist
		}
		if bestDist < 0 || dist < bestDist {
			best, bestDist = idx, dist
		}
	}
	return best
}
