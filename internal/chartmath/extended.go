package chartmath

import (
	"time"

	"github.com/MarioStoilov/simplestonks/internal/constants"
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
//   - Closed: a replay of the completed session — its regular + post candles
//     with the divider/dim metadata — so the last after-hours prices stay
//     viewable overnight. A closed market pairs those candles with the
//     UPCOMING session's windows, so the windows are first translated back
//     onto the candles' day. When no replay can be built (no post candles or
//     windows that cannot be located) the input is returned untouched
//     (DimFromIdx -1) and callers should fall back to a regular fetch.
//
// A pre-market moment without any pre candles yet also demotes to the
// closed-market replay.
func BuildExtendedDisplay(series model.Series, now time.Time) ExtendedDisplay {
	display := ExtendedDisplay{Series: series, State: StateAt(series, now), DimFromIdx: -1}
	switch display.State {
	case MarketPreMarket:
		preCandles := candlesBetween(series.Candles, series.PreStart, series.SessionStart)
		if len(preCandles) == 0 {
			display.State = MarketClosed
			return closedReplay(display)
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
		postFrom := firstCandleAtOrAfter(sessionCandles, series.SessionEnd)
		if postFrom < 0 {
			// No post candles yet: render the completed regular day as-is.
			display.Series.Candles = candlesBetween(series.Candles, series.SessionStart, series.SessionEnd)
			return display
		}
		display.Series.SessionEnd = series.PostEnd
		display.BoundaryTime = series.SessionEnd
		display.DimFromIdx = postFrom
		display.ExtendedPrice = sessionCandles[len(sessionCandles)-1].Close
	case MarketClosed:
		return closedReplay(display)
	}
	return display
}

// closedReplay maps a closed-market extended display to a replay of its
// completed session: the regular + post candles against the extended window,
// with the divider/dim metadata set. When there is nothing extended to show
// it leaves the display untouched so the caller can fall back to a regular
// fetch.
func closedReplay(display ExtendedDisplay) ExtendedDisplay {
	series := display.Series
	sessionStart, sessionEnd, postEnd, ok := completedSessionWindows(series)
	if !ok {
		return display
	}
	candles := candlesBetween(series.Candles, sessionStart, postEnd)
	postFrom := firstCandleAtOrAfter(candles, sessionEnd)
	if postFrom < 0 {
		return display
	}
	display.Series.Candles = candles
	display.Series.SessionStart = sessionStart
	display.Series.SessionEnd = postEnd
	display.BoundaryTime = sessionEnd
	display.DimFromIdx = postFrom
	display.ExtendedPrice = candles[len(candles)-1].Close
	return display
}

// completedSessionWindows translates the series' session windows onto the day
// of its last candle: while the market is closed the provider reports the
// UPCOMING session's windows alongside the COMPLETED session's candles, so
// the windows are stepped back in whole days until the regular open no longer
// lies past the last candle (one step overnight, three across a weekend).
// The whole-day step means a DST change between the two days shifts the
// boundary by an hour — a rare, cosmetic inaccuracy. It reports failure when
// the series has no candles, no post window, or windows that stay ahead of
// the candles within the lookback bound.
func completedSessionWindows(series model.Series) (sessionStart, sessionEnd, postEnd time.Time, ok bool) {
	if len(series.Candles) == 0 || !series.PostEnd.After(series.SessionEnd) {
		return time.Time{}, time.Time{}, time.Time{}, false
	}
	lastCandle := series.Candles[len(series.Candles)-1].Time
	sessionStart, sessionEnd, postEnd = series.SessionStart, series.SessionEnd, series.PostEnd
	for shifted := 0; sessionStart.After(lastCandle); shifted++ {
		if shifted >= constants.SessionShiftMaxDays {
			return time.Time{}, time.Time{}, time.Time{}, false
		}
		sessionStart = sessionStart.Add(-constants.SessionShiftDay)
		sessionEnd = sessionEnd.Add(-constants.SessionShiftDay)
		postEnd = postEnd.Add(-constants.SessionShiftDay)
	}
	return sessionStart, sessionEnd, postEnd, true
}

// firstCandleAtOrAfter returns the index of the first candle at or after the
// boundary, or -1 when there is none.
func firstCandleAtOrAfter(candles []model.Candle, boundary time.Time) int {
	for idx, candle := range candles {
		if !candle.Time.Before(boundary) {
			return idx
		}
	}
	return -1
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
