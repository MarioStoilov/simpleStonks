package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
)

const quoteBody = `{"chart":{"result":[{"meta":{
  "currency":"USD","symbol":"AAPL","regularMarketPrice":150.25,
  "previousClose":148.0,"chartPreviousClose":147.5,"regularMarketTime":1700000000},
  "timestamp":[1700000000],
  "indicators":{"quote":[{"open":[149.0],"high":[151.0],"low":[148.5],"close":[150.25],"volume":[1000]}]}
}],"error":null}}`

func TestYahooQuote(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, request *http.Request) {
		got = request
		_, _ = respWriter.Write([]byte(quoteBody))
	}))
	defer srv.Close()

	provider := NewYahoo(srv.Client())
	provider.baseURL = srv.URL + "/"

	quote, err := provider.Quote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if quote.Symbol != "AAPL" || quote.Price != 150.25 || quote.Currency != "USD" {
		t.Errorf("unexpected quote: %+v", quote)
	}
	// previousCloseRef prefers chartPreviousClose (147.5) over previousClose.
	if quote.PreviousClose != 147.5 {
		t.Errorf("PreviousClose = %v, want 147.5", quote.PreviousClose)
	}
	if got.URL.Path != "/AAPL" {
		t.Errorf("request path = %q, want /AAPL", got.URL.Path)
	}
	if userAgent := got.Header.Get("User-Agent"); userAgent != constants.YahooUserAgent {
		t.Errorf("User-Agent = %q, want %q", userAgent, constants.YahooUserAgent)
	}
}

const historyBody = `{"chart":{"result":[{"meta":{
  "currency":"USD","symbol":"AAPL","shortName":"Apple","longName":"Apple Inc.",
  "chartPreviousClose":100.0,"regularMarketPrice":12.5,
  "currentTradingPeriod":{
    "pre":{"start":1699952400,"end":1699972200},
    "regular":{"start":1699972200,"end":1699995600},
    "post":{"start":1699995600,"end":1700010000}}},
  "timestamp":[1700000000,1700000060,1700000120],
  "indicators":{"quote":[{
    "open":[10.0,null,12.0],"high":[10.5,null,12.5],
    "low":[9.5,null,11.5],"close":[10.0,null,12.0],"volume":[100,null,120]}]}
}],"error":null}}`

func TestYahooHistorySkipsGaps(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, request *http.Request) {
		got = request
		_, _ = respWriter.Write([]byte(historyBody))
	}))
	defer srv.Close()

	provider := NewYahoo(srv.Client())
	provider.baseURL = srv.URL + "/"

	series, err := provider.History(context.Background(), "AAPL", model.Range1D)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	// The middle candle (null close) must be skipped.
	if len(series.Candles) != 2 {
		t.Fatalf("got %d candles, want 2 (gap skipped): %+v", len(series.Candles), series.Candles)
	}
	if series.Candles[0].Close != 10.0 || series.Candles[0].Volume != 100 {
		t.Errorf("candle[0] = %+v", series.Candles[0])
	}
	if series.Candles[1].Close != 12.0 || series.Candles[1].Time.Unix() != 1700000120 {
		t.Errorf("candle[1] = %+v", series.Candles[1])
	}
	if series.PreviousClose != 100.0 || series.Range != model.Range1D {
		t.Errorf("series meta wrong: prevClose=%v range=%v", series.PreviousClose, series.Range)
	}
	// The friendly name prefers longName over shortName.
	if series.Name != "Apple Inc." {
		t.Errorf("series name = %q, want %q", series.Name, "Apple Inc.")
	}
	// The regular trading session window is carried through.
	if series.SessionStart.Unix() != 1699972200 || series.SessionEnd.Unix() != 1699995600 {
		t.Errorf("session window = %v..%v, want 1699972200..1699995600",
			series.SessionStart.Unix(), series.SessionEnd.Unix())
	}
	// The extended windows and regular price ride along on every fetch.
	if series.PreStart.Unix() != 1699952400 || series.PostEnd.Unix() != 1700010000 {
		t.Errorf("extended windows = %v..%v, want 1699952400/1700010000",
			series.PreStart.Unix(), series.PostEnd.Unix())
	}
	if series.RegularPrice != 12.5 {
		t.Errorf("RegularPrice = %v, want 12.5", series.RegularPrice)
	}
	// Range1D must map to range=1d interval=1m on the wire, and a regular
	// History fetch must never ask for extended-hours candles.
	if rangeQ, intervalQ := got.URL.Query().Get("range"), got.URL.Query().Get("interval"); rangeQ != "1d" || intervalQ != "1m" {
		t.Errorf("query range=%q interval=%q, want 1d/1m", rangeQ, intervalQ)
	}
	if got.URL.Query().Has("includePrePost") {
		t.Error("regular History request must not include includePrePost")
	}
}

const extendedBody = `{"chart":{"result":[{"meta":{
  "currency":"USD","symbol":"AAPL","shortName":"Apple","longName":"Apple Inc.",
  "chartPreviousClose":100.0,"regularMarketPrice":12.0,
  "currentTradingPeriod":{
    "pre":{"start":1699952400,"end":1699972200},
    "regular":{"start":1699972200,"end":1699995600},
    "post":{"start":1699995600,"end":1700010000}}},
  "timestamp":[1699960000,1699972260,1699980000,1700000000],
  "indicators":{"quote":[{
    "open":[9.0,10.0,11.0,12.0],"high":[9.5,10.5,11.5,12.5],
    "low":[8.5,9.5,10.5,11.5],"close":[9.0,10.0,11.0,12.0],"volume":[50,100,110,60]}]}
}],"error":null}}`

func TestYahooHistoryExtended(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, request *http.Request) {
		got = request
		_, _ = respWriter.Write([]byte(extendedBody))
	}))
	defer srv.Close()

	provider := NewYahoo(srv.Client())
	provider.baseURL = srv.URL + "/"

	series, err := provider.HistoryExtended(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("HistoryExtended: %v", err)
	}
	// Extended fetches go out as 1D with includePrePost=true.
	query := got.URL.Query()
	if query.Get("includePrePost") != "true" {
		t.Errorf("includePrePost = %q, want true", query.Get("includePrePost"))
	}
	if query.Get("range") != "1d" || query.Get("interval") != "1m" {
		t.Errorf("query range=%q interval=%q, want 1d/1m", query.Get("range"), query.Get("interval"))
	}
	// All candles — pre, regular, and post — are kept.
	if len(series.Candles) != 4 {
		t.Fatalf("got %d candles, want 4: %+v", len(series.Candles), series.Candles)
	}
	if series.Range != model.Range1D {
		t.Errorf("Range = %v, want %v", series.Range, model.Range1D)
	}
	if series.SessionStart.Unix() != 1699972200 || series.SessionEnd.Unix() != 1699995600 {
		t.Errorf("session window = %v..%v, want 1699972200..1699995600",
			series.SessionStart.Unix(), series.SessionEnd.Unix())
	}
	if series.PreStart.Unix() != 1699952400 || series.PostEnd.Unix() != 1700010000 {
		t.Errorf("extended windows = %v..%v, want 1699952400/1700010000",
			series.PreStart.Unix(), series.PostEnd.Unix())
	}
	if series.RegularPrice != 12.0 {
		t.Errorf("RegularPrice = %v, want 12.0", series.RegularPrice)
	}
}

func TestYahooAPIError(t *testing.T) {
	body := `{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, request *http.Request) {
		respWriter.WriteHeader(http.StatusNotFound)
		_, _ = respWriter.Write([]byte(body))
	}))
	defer srv.Close()

	provider := NewYahoo(srv.Client())
	provider.baseURL = srv.URL + "/"

	if _, err := provider.Quote(context.Background(), "NOPE"); err == nil {
		t.Fatal("expected error for API error response, got nil")
	}
}

func TestYahooNon200Unparseable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, request *http.Request) {
		respWriter.WriteHeader(http.StatusTooManyRequests)
		_, _ = respWriter.Write([]byte("Too Many Requests"))
	}))
	defer srv.Close()

	provider := NewYahoo(srv.Client())
	provider.baseURL = srv.URL + "/"

	if _, err := provider.Quote(context.Background(), "AAPL"); err == nil {
		t.Fatal("expected error for 429 response, got nil")
	}
}

func TestYahooEmptySymbol(t *testing.T) {
	provider := NewYahoo(nil)
	if _, err := provider.Quote(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty symbol, got nil")
	}
}

const searchBody = `{"quotes":[
  {"symbol":"AAPL","shortname":"Apple","longname":"Apple Inc.","exchange":"NMS","exchDisp":"NASDAQ","quoteType":"EQUITY","typeDisp":"Equity"},
  {"symbol":"","shortname":"junk with no symbol"},
  {"symbol":"^GSPC","longname":"S&P 500","exchDisp":"SNP","quoteType":"INDEX","typeDisp":"Index"}
],"news":[]}`

func TestYahooSearch(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, request *http.Request) {
		got = request
		_, _ = respWriter.Write([]byte(searchBody))
	}))
	defer srv.Close()

	provider := NewYahoo(srv.Client())
	provider.searchURL = srv.URL

	res, err := provider.Search(context.Background(), "apple")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	// The entry with an empty symbol must be dropped.
	if len(res) != 2 {
		t.Fatalf("got %d results, want 2: %+v", len(res), res)
	}
	if res[0] != (model.SearchResult{Symbol: "AAPL", Name: "Apple Inc.", Exchange: "NASDAQ", Type: "Equity"}) {
		t.Errorf("result[0] = %+v", res[0])
	}
	// Name falls back are covered; index uses longname + exchDisp.
	if res[1].Symbol != "^GSPC" || res[1].Name != "S&P 500" || res[1].Exchange != "SNP" {
		t.Errorf("result[1] = %+v", res[1])
	}
	if got.URL.Query().Get("q") != "apple" {
		t.Errorf("query q = %q, want apple", got.URL.Query().Get("q"))
	}
}

func TestYahooSearchEmptyQuery(t *testing.T) {
	// Empty query must short-circuit without an HTTP call.
	provider := NewYahoo(nil)
	provider.searchURL = "http://127.0.0.1:0/should-not-be-called"
	res, err := provider.Search(context.Background(), "   ")
	if err != nil || res != nil {
		t.Fatalf("empty query: got (%v, %v), want (nil, nil)", res, err)
	}
}

func TestYahooParams(t *testing.T) {
	cases := map[model.Range][2]string{
		model.Range1D:  {"1d", "1m"},
		model.Range5D:  {"5d", "5m"},
		model.Range1M:  {"1mo", "1d"},
		model.RangeYTD: {"ytd", "1d"},
		model.RangeAll: {"max", "1mo"},
	}
	for chartRange, want := range cases {
		rng, interval := yahooParams(chartRange)
		if rng != want[0] || interval != want[1] {
			t.Errorf("yahooParams(%s) = %q/%q, want %q/%q", chartRange, rng, interval, want[0], want[1])
		}
	}
}
