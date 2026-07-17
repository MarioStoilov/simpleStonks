// Package model holds simpleStonks' core domain types: tracked symbols,
// chart ranges, quotes, and price series. These types are shared across the
// data providers, config, and UI so none of those packages depend on each
// other's internals.
package model

import "time"

// Range is a selectable chart time range, mirroring the Yahoo Finance toggles.
type Range string

const (
	Range1D  Range = "1D"  // intraday, live-ticking default view
	Range5D  Range = "5D"  // trading week
	Range1W  Range = "1W"  // calendar week
	Range1M  Range = "1M"  // one month
	RangeYTD Range = "YTD" // year to date
	Range1Y  Range = "1Y"  // one year
	Range5Y  Range = "5Y"  // five years
	RangeAll Range = "ALL" // full available history
)

// Ranges is the ordered set of ranges shown as toggles in the UI.
var Ranges = []Range{
	Range1D, Range5D, Range1W, Range1M, RangeYTD, Range1Y, Range5Y, RangeAll,
}

// Intraday reports whether a range is fine-grained enough to warrant
// live-tick polling (currently only the 1D view).
func (r Range) Intraday() bool { return r == Range1D }

// Candle is a single OHLC data point in a price series.
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// Series is a symbol's price history over a requested range, plus enough
// metadata to render the current price and percent change.
type Series struct {
	Symbol        string
	Name          string // friendly display name, e.g. "S&P 500"; may be empty
	Range         Range
	Currency      string
	Candles       []Candle
	PreviousClose float64 // reference close used for the range's % change

	// SessionStart/SessionEnd bound the regular trading session the series
	// belongs to (when the provider knows it). An intraday chart is drawn
	// against this full window so a live day fills in gradually.
	SessionStart time.Time
	SessionEnd   time.Time
}

// SearchResult is one instrument returned by a symbol search: enough to show
// the user a disambiguated suggestion (symbol, full name, market/exchange).
type SearchResult struct {
	Symbol   string // e.g. "AAPL", "^GSPC"
	Name     string // full/long name, e.g. "Apple Inc."
	Exchange string // market/exchange display, e.g. "NASDAQ"
	Type     string // instrument type, e.g. "Equity", "Index", "ETF"
}

// Quote is a lightweight snapshot of a symbol's latest price and how it has
// moved relative to a reference close.
type Quote struct {
	Symbol        string
	Price         float64
	PreviousClose float64
	Currency      string
	AsOf          time.Time
}

// Change returns the absolute price change versus the previous close.
func (q Quote) Change() float64 { return q.Price - q.PreviousClose }

// ChangePercent returns the percent change versus the previous close.
// It returns 0 when the previous close is unknown to avoid divide-by-zero.
func (q Quote) ChangePercent() float64 {
	if q.PreviousClose == 0 {
		return 0
	}
	return (q.Price - q.PreviousClose) / q.PreviousClose * 100
}
