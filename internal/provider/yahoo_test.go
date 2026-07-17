package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte(quoteBody))
	}))
	defer srv.Close()

	y := NewYahoo(srv.Client())
	y.baseURL = srv.URL + "/"

	q, err := y.Quote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if q.Symbol != "AAPL" || q.Price != 150.25 || q.Currency != "USD" {
		t.Errorf("unexpected quote: %+v", q)
	}
	// previousCloseRef prefers chartPreviousClose (147.5) over previousClose.
	if q.PreviousClose != 147.5 {
		t.Errorf("PreviousClose = %v, want 147.5", q.PreviousClose)
	}
	if got.URL.Path != "/AAPL" {
		t.Errorf("request path = %q, want /AAPL", got.URL.Path)
	}
	if ua := got.Header.Get("User-Agent"); ua != yahooUserAgent {
		t.Errorf("User-Agent = %q, want %q", ua, yahooUserAgent)
	}
}

const historyBody = `{"chart":{"result":[{"meta":{
  "currency":"USD","symbol":"AAPL","shortName":"Apple","longName":"Apple Inc.",
  "chartPreviousClose":100.0,
  "currentTradingPeriod":{"regular":{"start":1699972200,"end":1699995600}}},
  "timestamp":[1700000000,1700000060,1700000120],
  "indicators":{"quote":[{
    "open":[10.0,null,12.0],"high":[10.5,null,12.5],
    "low":[9.5,null,11.5],"close":[10.0,null,12.0],"volume":[100,null,120]}]}
}],"error":null}}`

func TestYahooHistorySkipsGaps(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte(historyBody))
	}))
	defer srv.Close()

	y := NewYahoo(srv.Client())
	y.baseURL = srv.URL + "/"

	s, err := y.History(context.Background(), "AAPL", model.Range1D)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	// The middle candle (null close) must be skipped.
	if len(s.Candles) != 2 {
		t.Fatalf("got %d candles, want 2 (gap skipped): %+v", len(s.Candles), s.Candles)
	}
	if s.Candles[0].Close != 10.0 || s.Candles[0].Volume != 100 {
		t.Errorf("candle[0] = %+v", s.Candles[0])
	}
	if s.Candles[1].Close != 12.0 || s.Candles[1].Time.Unix() != 1700000120 {
		t.Errorf("candle[1] = %+v", s.Candles[1])
	}
	if s.PreviousClose != 100.0 || s.Range != model.Range1D {
		t.Errorf("series meta wrong: prevClose=%v range=%v", s.PreviousClose, s.Range)
	}
	// The friendly name prefers longName over shortName.
	if s.Name != "Apple Inc." {
		t.Errorf("series name = %q, want %q", s.Name, "Apple Inc.")
	}
	// The regular trading session window is carried through.
	if s.SessionStart.Unix() != 1699972200 || s.SessionEnd.Unix() != 1699995600 {
		t.Errorf("session window = %v..%v, want 1699972200..1699995600",
			s.SessionStart.Unix(), s.SessionEnd.Unix())
	}
	// Range1D must map to range=1d interval=1m on the wire.
	if r, i := got.URL.Query().Get("range"), got.URL.Query().Get("interval"); r != "1d" || i != "1m" {
		t.Errorf("query range=%q interval=%q, want 1d/1m", r, i)
	}
}

func TestYahooAPIError(t *testing.T) {
	body := `{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	y := NewYahoo(srv.Client())
	y.baseURL = srv.URL + "/"

	if _, err := y.Quote(context.Background(), "NOPE"); err == nil {
		t.Fatal("expected error for API error response, got nil")
	}
}

func TestYahooNon200Unparseable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Too Many Requests"))
	}))
	defer srv.Close()

	y := NewYahoo(srv.Client())
	y.baseURL = srv.URL + "/"

	if _, err := y.Quote(context.Background(), "AAPL"); err == nil {
		t.Fatal("expected error for 429 response, got nil")
	}
}

func TestYahooEmptySymbol(t *testing.T) {
	y := NewYahoo(nil)
	if _, err := y.Quote(context.Background(), ""); err == nil {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte(searchBody))
	}))
	defer srv.Close()

	y := NewYahoo(srv.Client())
	y.searchURL = srv.URL

	res, err := y.Search(context.Background(), "apple")
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
	y := NewYahoo(nil)
	y.searchURL = "http://127.0.0.1:0/should-not-be-called"
	res, err := y.Search(context.Background(), "   ")
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
	for r, want := range cases {
		rng, interval := yahooParams(r)
		if rng != want[0] || interval != want[1] {
			t.Errorf("yahooParams(%s) = %q/%q, want %q/%q", r, rng, interval, want[0], want[1])
		}
	}
}
