package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// yahooBaseURL is the keyless Yahoo Finance chart endpoint. It returns both a
// meta block (current price, previous close, currency) and OHLC candles, which
// together cover Quote and History.
const yahooBaseURL = "https://query1.finance.yahoo.com/v8/finance/chart/"

// yahooSearchURL is the keyless Yahoo Finance search endpoint, used to back the
// add-symbol live search.
const yahooSearchURL = "https://query1.finance.yahoo.com/v1/finance/search"

// yahooUserAgent is sent with every request; Yahoo tends to reject requests
// carrying Go's default user agent.
const yahooUserAgent = "simplestonks/0.1 (+https://github.com/MarioStoilov/simpleStonks)"

// Yahoo is the default, keyless QuoteProvider backed by the Yahoo Finance
// chart and search endpoints.
type Yahoo struct {
	client    *http.Client
	baseURL   string
	searchURL string
}

// NewYahoo returns a Yahoo provider using the given HTTP client, or a client
// with a sane default timeout when nil.
func NewYahoo(client *http.Client) *Yahoo {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Yahoo{client: client, baseURL: yahooBaseURL, searchURL: yahooSearchURL}
}

// get issues a GET with the Yahoo-friendly user agent.
func (yahoo *Yahoo) get(ctx context.Context, endpoint string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", yahooUserAgent)
	return yahoo.client.Do(req)
}

// Name implements QuoteProvider.
func (yahoo *Yahoo) Name() string { return "yahoo" }

// Quote implements QuoteProvider. It reads the latest price and previous close
// from the 1D chart's meta block.
func (yahoo *Yahoo) Quote(ctx context.Context, symbol string) (model.Quote, error) {
	res, err := yahoo.fetchChart(ctx, symbol, "1d", "1m")
	if err != nil {
		return model.Quote{}, err
	}
	meta := res.Meta
	return model.Quote{
		Symbol:        symbolOr(meta.Symbol, symbol),
		Price:         meta.RegularMarketPrice,
		PreviousClose: meta.previousCloseRef(),
		Currency:      meta.Currency,
		AsOf:          time.Unix(meta.RegularMarketTime, 0).UTC(),
	}, nil
}

// History implements QuoteProvider. It maps the range to Yahoo's range+interval
// parameters and decodes the returned timestamps and OHLC arrays.
func (yahoo *Yahoo) History(ctx context.Context, symbol string, rng model.Range) (model.Series, error) {
	chartRange, interval := yahooParams(rng)
	res, err := yahoo.fetchChart(ctx, symbol, chartRange, interval)
	if err != nil {
		return model.Series{}, err
	}
	candles, err := res.candles()
	if err != nil {
		return model.Series{}, fmt.Errorf("yahoo: %s: %w", symbol, err)
	}
	series := model.Series{
		Symbol:        symbolOr(res.Meta.Symbol, symbol),
		Name:          res.Meta.displayName(),
		Range:         rng,
		Currency:      res.Meta.Currency,
		Candles:       candles,
		PreviousClose: res.Meta.previousCloseRef(),
	}
	if period := res.Meta.CurrentTradingPeriod.Regular; period.Start > 0 && period.End > period.Start {
		series.SessionStart = time.Unix(period.Start, 0).UTC()
		series.SessionEnd = time.Unix(period.End, 0).UTC()
	}
	return series, nil
}

// Search implements QuoteProvider using Yahoo's search endpoint.
func (yahoo *Yahoo) Search(ctx context.Context, query string) ([]model.SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	endpoint := yahoo.searchURL + "?" + url.Values{
		"q":           {query},
		"quotesCount": {"10"},
		"newsCount":   {"0"},
	}.Encode()

	resp, err := yahoo.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("yahoo: search %q: %w", query, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo: search %q: unexpected status %s", query, resp.Status)
	}
	var body yahooSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("yahoo: search %q: %w", query, err)
	}

	out := make([]model.SearchResult, 0, len(body.Quotes))
	for _, quote := range body.Quotes {
		if quote.Symbol == "" {
			continue
		}
		out = append(out, model.SearchResult{
			Symbol:   quote.Symbol,
			Name:     firstNonEmpty(quote.LongName, quote.ShortName),
			Exchange: firstNonEmpty(quote.ExchDisp, quote.Exchange),
			Type:     firstNonEmpty(quote.TypeDisp, quote.QuoteType),
		})
	}
	return out, nil
}

// fetchChart requests the chart endpoint for a symbol and returns the first
// result, mapping transport and API-level failures to errors.
func (yahoo *Yahoo) fetchChart(ctx context.Context, symbol, rng, interval string) (*yahooResult, error) {
	if symbol == "" {
		return nil, fmt.Errorf("yahoo: empty symbol")
	}
	endpoint := yahoo.baseURL + url.PathEscape(symbol) + "?" + url.Values{
		"range":    {rng},
		"interval": {interval},
	}.Encode()

	resp, err := yahoo.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("yahoo: %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	var body yahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		// A non-2xx status with an unparseable body is most useful reported as
		// the status itself.
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("yahoo: %s: unexpected status %s", symbol, resp.Status)
		}
		return nil, fmt.Errorf("yahoo: decode %s: %w", symbol, err)
	}
	if body.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo: %s: %s", symbol, body.Chart.Error.Description)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo: %s: unexpected status %s", symbol, resp.Status)
	}
	if len(body.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo: %s: no data returned", symbol)
	}
	return &body.Chart.Result[0], nil
}

// yahooResponse mirrors the parts of the chart endpoint response we consume.
type yahooResponse struct {
	Chart struct {
		Result []yahooResult `json:"result"`
		Error  *yahooError   `json:"error"`
	} `json:"chart"`
}

type yahooError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// yahooSearchResponse mirrors the parts of the search endpoint response we use.
type yahooSearchResponse struct {
	Quotes []struct {
		Symbol    string `json:"symbol"`
		ShortName string `json:"shortname"`
		LongName  string `json:"longname"`
		Exchange  string `json:"exchange"`
		ExchDisp  string `json:"exchDisp"`
		QuoteType string `json:"quoteType"`
		TypeDisp  string `json:"typeDisp"`
	} `json:"quotes"`
}

type yahooResult struct {
	Meta       yahooMeta `json:"meta"`
	Timestamp  []int64   `json:"timestamp"`
	Indicators struct {
		Quote []yahooQuote `json:"quote"`
	} `json:"indicators"`
}

type yahooMeta struct {
	Currency             string  `json:"currency"`
	Symbol               string  `json:"symbol"`
	ShortName            string  `json:"shortName"`
	LongName             string  `json:"longName"`
	RegularMarketPrice   float64 `json:"regularMarketPrice"`
	PreviousClose        float64 `json:"previousClose"`
	ChartPreviousClose   float64 `json:"chartPreviousClose"`
	RegularMarketTime    int64   `json:"regularMarketTime"`
	CurrentTradingPeriod struct {
		Regular struct {
			Start int64 `json:"start"`
			End   int64 `json:"end"`
		} `json:"regular"`
	} `json:"currentTradingPeriod"`
}

// displayName is the friendly instrument name, preferring the long form.
func (meta yahooMeta) displayName() string {
	if meta.LongName != "" {
		return meta.LongName
	}
	return meta.ShortName
}

// previousCloseRef is the close used as the % change reference for a range.
// chartPreviousClose is the close just before the range starts; previousClose
// is the fallback for endpoints that omit it.
func (meta yahooMeta) previousCloseRef() float64 {
	if meta.ChartPreviousClose != 0 {
		return meta.ChartPreviousClose
	}
	return meta.PreviousClose
}

// yahooQuote holds the OHLCV arrays. Pointers distinguish a genuine gap (null,
// common in intraday data) from a real zero value.
type yahooQuote struct {
	Open   []*float64 `json:"open"`
	High   []*float64 `json:"high"`
	Low    []*float64 `json:"low"`
	Close  []*float64 `json:"close"`
	Volume []*int64   `json:"volume"`
}

// candles builds the OHLC series, skipping timestamps whose close is null.
func (res *yahooResult) candles() ([]model.Candle, error) {
	if len(res.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote indicators")
	}
	quote := res.Indicators.Quote[0]
	out := make([]model.Candle, 0, len(res.Timestamp))
	for idx := range res.Timestamp {
		closeVal := fptrAt(quote.Close, idx)
		if closeVal == nil {
			continue // gap in the data
		}
		out = append(out, model.Candle{
			Time:   time.Unix(res.Timestamp[idx], 0).UTC(),
			Open:   fvalAt(quote.Open, idx),
			High:   fvalAt(quote.High, idx),
			Low:    fvalAt(quote.Low, idx),
			Close:  *closeVal,
			Volume: ivalAt(quote.Volume, idx),
		})
	}
	return out, nil
}

// yahooParams maps a model.Range to Yahoo's (range, interval) query parameters.
// Yahoo has no exact 1W range, so it is approximated with a 5d range at a
// coarser interval.
func yahooParams(chartRange model.Range) (rng, interval string) {
	switch chartRange {
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

// symbolOr returns meta if non-empty, otherwise the requested symbol.
func symbolOr(meta, requested string) string {
	if meta != "" {
		return meta
	}
	return requested
}

// firstNonEmpty returns the first non-empty string, or "".
func firstNonEmpty(vals ...string) string {
	for _, value := range vals {
		if value != "" {
			return value
		}
	}
	return ""
}

// fptrAt returns the *float64 at index idx, or nil if out of range.
func fptrAt(values []*float64, idx int) *float64 {
	if idx < len(values) {
		return values[idx]
	}
	return nil
}

// fvalAt dereferences the float at index idx, treating missing/null as 0.
func fvalAt(values []*float64, idx int) float64 {
	if ptr := fptrAt(values, idx); ptr != nil {
		return *ptr
	}
	return 0
}

// ivalAt dereferences the int at index idx, treating missing/null as 0.
func ivalAt(values []*int64, idx int) int64 {
	if idx < len(values) && values[idx] != nil {
		return *values[idx]
	}
	return 0
}

// Ensure Yahoo satisfies the interface at compile time.
var _ QuoteProvider = (*Yahoo)(nil)
