package chartmath

import (
	"time"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// MarketState classifies a moment relative to a symbol's trading sessions.
type MarketState int

const (
	MarketClosed MarketState = iota
	MarketPreMarket
	MarketRegular
	MarketAfterHours
)

// StateAt classifies now against the series' session windows: pre-market is
// [PreStart, SessionStart), the regular session [SessionStart, SessionEnd),
// and after-hours [SessionEnd, PostEnd). Missing or degenerate windows
// collapse to Closed, which callers treat as "regular behavior".
func StateAt(series model.Series, now time.Time) MarketState {
	if !series.SessionEnd.After(series.SessionStart) {
		return MarketClosed
	}
	switch {
	case !now.Before(series.SessionStart) && now.Before(series.SessionEnd):
		return MarketRegular
	case series.PreStart.Before(series.SessionStart) && !series.PreStart.IsZero() &&
		!now.Before(series.PreStart) && now.Before(series.SessionStart):
		return MarketPreMarket
	case series.PostEnd.After(series.SessionEnd) &&
		!now.Before(series.SessionEnd) && now.Before(series.PostEnd):
		return MarketAfterHours
	default:
		return MarketClosed
	}
}

// ExtendedDisplay is what the detail view renders for an extended-hours
// fetch: the display series (session window overridden and candles filtered
// for the current market state) plus the pieces the chart overlay and the
// extended price label need.
type ExtendedDisplay struct {
	Series        model.Series
	State         MarketState
	ExtendedPrice float64   // last pre/post close; 0 when there is none
	BoundaryTime  time.Time // regular close (the after-hours divider); zero otherwise
	DimFromIdx    int       // index of the first after-hours candle; -1 when none
}

// BuildExtendedDisplay maps an extended-hours series (candles spanning pre,
// regular, and post sessions) to what the detail view should draw at the
// given time:
//
//   - Pre-market: only the pre candles, drawn against the pre window.
//   - Regular: only the regular candles — exactly the everyday chart.
//   - After-hours: regular + post candles against the extended window, with
//     the divider/dim metadata set.
//   - Closed: the input untouched; callers should fall back to a regular
//     fetch, since a closed market pairs the previous day's candles with the
//     upcoming session's windows (see SessionWindow).
//
// A pre-market moment without any pre candles yet also demotes to Closed.
func BuildExtendedDisplay(series model.Series, now time.Time) ExtendedDisplay {
	display := ExtendedDisplay{Series: series, State: StateAt(series, now), DimFromIdx: -1}
	switch display.State {
	case MarketPreMarket:
		preCandles := candlesBetween(series.Candles, series.PreStart, series.SessionStart)
		if len(preCandles) == 0 {
			display.State = MarketClosed
			return display
		}
		display.Series.Candles = preCandles
		display.Series.SessionStart = series.PreStart
		display.Series.SessionEnd = series.SessionStart
		display.ExtendedPrice = preCandles[len(preCandles)-1].Close
	case MarketRegular:
		display.Series.Candles = candlesBetween(series.Candles, series.SessionStart, series.SessionEnd)
	case MarketAfterHours:
		sessionCandles := candlesBetween(series.Candles, series.SessionStart, series.PostEnd)
		display.Series.Candles = sessionCandles
		postFrom := -1
		for idx, candle := range sessionCandles {
			if !candle.Time.Before(series.SessionEnd) {
				postFrom = idx
				break
			}
		}
		if postFrom < 0 {
			// No post candles yet: render the completed regular day as-is.
			display.Series.Candles = candlesBetween(series.Candles, series.SessionStart, series.SessionEnd)
			return display
		}
		display.Series.SessionEnd = series.PostEnd
		display.BoundaryTime = series.SessionEnd
		display.DimFromIdx = postFrom
		display.ExtendedPrice = sessionCandles[len(sessionCandles)-1].Close
	}
	return display
}

// candlesBetween returns the candles whose time falls in [from, to).
func candlesBetween(candles []model.Candle, from, to time.Time) []model.Candle {
	out := make([]model.Candle, 0, len(candles))
	for _, candle := range candles {
		if !candle.Time.Before(from) && candle.Time.Before(to) {
			out = append(out, candle)
		}
	}
	return out
}
