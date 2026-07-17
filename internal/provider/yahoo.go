package provider

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// ErrNotImplemented is returned by stubbed methods during scaffolding.
var ErrNotImplemented = errors.New("provider: not implemented")

// yahooBaseURL is the keyless Yahoo Finance chart endpoint. It returns both a
// meta block (current price, previous close, currency) and OHLC candles, which
// together cover Quote and History.
const yahooBaseURL = "https://query1.finance.yahoo.com/v8/finance/chart/"

// Yahoo is the default, keyless QuoteProvider backed by the Yahoo Finance
// chart endpoint.
type Yahoo struct {
	client *http.Client
}

// NewYahoo returns a Yahoo provider using the given HTTP client, or a client
// with a sane default timeout when nil.
func NewYahoo(client *http.Client) *Yahoo {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Yahoo{client: client}
}

// Name implements QuoteProvider.
func (y *Yahoo) Name() string { return "yahoo" }

// Quote implements QuoteProvider.
//
// TODO(v1): request the 1D chart, read meta.regularMarketPrice /
// meta.previousClose / meta.currency, and map into model.Quote.
func (y *Yahoo) Quote(ctx context.Context, symbol string) (model.Quote, error) {
	return model.Quote{}, ErrNotImplemented
}

// History implements QuoteProvider.
//
// TODO(v1): map model.Range to Yahoo's range+interval params, request the
// chart, and decode timestamps + OHLC arrays into model.Series.
func (y *Yahoo) History(ctx context.Context, symbol string, r model.Range) (model.Series, error) {
	return model.Series{}, ErrNotImplemented
}

// yahooParams maps a model.Range to Yahoo's (range, interval) query parameters.
// Yahoo has no exact 1W range, so it is approximated with a 5d range at a
// coarser interval; revisit during implementation.
func yahooParams(r model.Range) (rng, interval string) {
	switch r {
	case model.Range1D:
		return "1d", "1m"
	case model.Range5D:
		return "5d", "5m"
	case model.Range1W:
		return "5d", "15m"
	case model.Range1M:
		return "1mo", "1d"
	case model.RangeYTD:
		return "ytd", "1d"
	case model.Range1Y:
		return "1y", "1d"
	case model.Range5Y:
		return "5y", "1wk"
	case model.RangeAll:
		return "max", "1mo"
	default:
		return "1d", "1m"
	}
}

// Ensure Yahoo satisfies the interface at compile time.
var _ QuoteProvider = (*Yahoo)(nil)
