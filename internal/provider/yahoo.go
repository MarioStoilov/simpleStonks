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
func (y *Yahoo) get(ctx context.Context, endpoint string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", yahooUserAgent)
	return y.client.Do(req)
}

// Name implements QuoteProvider.
func (y *Yahoo) Name() string { return "yahoo" }

// Quote implements QuoteProvider. It reads the latest price and previous close
// from the 1D chart's meta block.
func (y *Yahoo) Quote(ctx context.Context, symbol string) (model.Quote, error) {
	res, err := y.fetchChart(ctx, symbol, "1d", "1m")
	if err != nil {
		return model.Quote{}, err
	}
	m := res.Meta
	return model.Quote{
		Symbol:        symbolOr(m.Symbol, symbol),
		Price:         m.RegularMarketPrice,
		PreviousClose: m.previousCloseRef(),
		Currency:      m.Currency,
		AsOf:          time.Unix(m.RegularMarketTime, 0).UTC(),
	}, nil
}

// History implements QuoteProvider. It maps the range to Yahoo's range+interval
// parameters and decodes the returned timestamps and OHLC arrays.
func (y *Yahoo) History(ctx context.Context, symbol string, r model.Range) (model.Series, error) {
	rng, interval := yahooParams(r)
	res, err := y.fetchChart(ctx, symbol, rng, interval)
	if err != nil {
		return model.Series{}, err
	}
	candles, err := res.candles()
	if err != nil {
		return model.Series{}, fmt.Errorf("yahoo: %s: %w", symbol, err)
	}
	return model.Series{
		Symbol:        symbolOr(res.Meta.Symbol, symbol),
		Range:         r,
		Currency:      res.Meta.Currency,
		Candles:       candles,
		PreviousClose: res.Meta.previousCloseRef(),
	}, nil
}

// Search implements QuoteProvider using Yahoo's search endpoint.
func (y *Yahoo) Search(ctx context.Context, query string) ([]model.SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	endpoint := y.searchURL + "?" + url.Values{
		"q":           {query},
		"quotesCount": {"10"},
		"newsCount":   {"0"},
	}.Encode()

	resp, err := y.get(ctx, endpoint)
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
	for _, q := range body.Quotes {
		if q.Symbol == "" {
			continue
		}
		out = append(out, model.SearchResult{
			Symbol:   q.Symbol,
			Name:     firstNonEmpty(q.LongName, q.ShortName),
			Exchange: firstNonEmpty(q.ExchDisp, q.Exchange),
			Type:     firstNonEmpty(q.TypeDisp, q.QuoteType),
		})
	}
	return out, nil
}

// fetchChart requests the chart endpoint for a symbol and returns the first
// result, mapping transport and API-level failures to errors.
func (y *Yahoo) fetchChart(ctx context.Context, symbol, rng, interval string) (*yahooResult, error) {
	if symbol == "" {
		return nil, fmt.Errorf("yahoo: empty symbol")
	}
	endpoint := y.baseURL + url.PathEscape(symbol) + "?" + url.Values{
		"range":    {rng},
		"interval": {interval},
	}.Encode()

	resp, err := y.get(ctx, endpoint)
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
	Currency           string  `json:"currency"`
	Symbol             string  `json:"symbol"`
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	PreviousClose      float64 `json:"previousClose"`
	ChartPreviousClose float64 `json:"chartPreviousClose"`
	RegularMarketTime  int64   `json:"regularMarketTime"`
}

// previousCloseRef is the close used as the % change reference for a range.
// chartPreviousClose is the close just before the range starts; previousClose
// is the fallback for endpoints that omit it.
func (m yahooMeta) previousCloseRef() float64 {
	if m.ChartPreviousClose != 0 {
		return m.ChartPreviousClose
	}
	return m.PreviousClose
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
	q := res.Indicators.Quote[0]
	out := make([]model.Candle, 0, len(res.Timestamp))
	for i := range res.Timestamp {
		c := fptrAt(q.Close, i)
		if c == nil {
			continue // gap in the data
		}
		out = append(out, model.Candle{
			Time:   time.Unix(res.Timestamp[i], 0).UTC(),
			Open:   fvalAt(q.Open, i),
			High:   fvalAt(q.High, i),
			Low:    fvalAt(q.Low, i),
			Close:  *c,
			Volume: ivalAt(q.Volume, i),
		})
	}
	return out, nil
}

// yahooParams maps a model.Range to Yahoo's (range, interval) query parameters.
// Yahoo has no exact 1W range, so it is approximated with a 5d range at a
// coarser interval.
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

// symbolOr returns meta if non-empty, otherwise the requested symbol.
func symbolOr(meta, requested string) string {
	if meta != "" {
		return meta
	}
	return requested
}

// firstNonEmpty returns the first non-empty string, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// fptrAt returns the *float64 at index i, or nil if out of range.
func fptrAt(s []*float64, i int) *float64 {
	if i < len(s) {
		return s[i]
	}
	return nil
}

// fvalAt dereferences the float at index i, treating missing/null as 0.
func fvalAt(s []*float64, i int) float64 {
	if p := fptrAt(s, i); p != nil {
		return *p
	}
	return 0
}

// ivalAt dereferences the int at index i, treating missing/null as 0.
func ivalAt(s []*int64, i int) int64 {
	if i < len(s) && s[i] != nil {
		return *s[i]
	}
	return 0
}

// Ensure Yahoo satisfies the interface at compile time.
var _ QuoteProvider = (*Yahoo)(nil)
