//go:build integration

// Integration tests exercise the real Yahoo Finance endpoint. They run on push
// (see .githooks/pre-push) via `go test -tags=integration`, and skip themselves
// when there is no network so offline pushes are not blocked.

package provider

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// requireNetwork skips the test when the Yahoo endpoint is unreachable.
func requireNetwork(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "query1.finance.yahoo.com:443", 3*time.Second)
	if err != nil {
		t.Skipf("skipping: no network to Yahoo: %v", err)
	}
	_ = conn.Close()
}

func TestIntegrationYahooQuote(t *testing.T) {
	requireNetwork(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	quote, err := NewYahoo(nil).Quote(ctx, "AAPL")
	if err != nil {
		t.Fatalf("live Quote: %v", err)
	}
	if quote.Symbol == "" || quote.Price <= 0 {
		t.Fatalf("implausible quote: %+v", quote)
	}
}

func TestIntegrationYahooHistory(t *testing.T) {
	requireNetwork(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	series, err := NewYahoo(nil).History(ctx, "AAPL", model.Range1M)
	if err != nil {
		t.Fatalf("live History: %v", err)
	}
	if len(series.Candles) == 0 {
		t.Fatal("expected candles for the 1M range")
	}
}

func TestIntegrationYahooHistoryExtended(t *testing.T) {
	requireNetwork(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	series, err := NewYahoo(nil).HistoryExtended(ctx, "AAPL")
	if err != nil {
		t.Fatalf("live HistoryExtended: %v", err)
	}
	if len(series.Candles) == 0 {
		t.Fatal("expected candles for the extended 1D range")
	}
	if series.RegularPrice <= 0 {
		t.Fatalf("implausible regular price: %v", series.RegularPrice)
	}
	if !series.HasExtendedHours {
		t.Error("HasExtendedHours = false for AAPL, want true (hasPrePostMarketData)")
	}
	// AAPL has pre/post sessions; when Yahoo reports them they must bracket
	// the regular session.
	if !series.PreStart.IsZero() && !series.PreStart.Before(series.SessionStart) {
		t.Errorf("PreStart %v not before SessionStart %v", series.PreStart, series.SessionStart)
	}
	if !series.PostEnd.IsZero() && !series.PostEnd.After(series.SessionEnd) {
		t.Errorf("PostEnd %v not after SessionEnd %v", series.PostEnd, series.SessionEnd)
	}
}

func TestIntegrationYahooSearch(t *testing.T) {
	requireNetwork(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := NewYahoo(nil).Search(ctx, "apple")
	if err != nil {
		t.Fatalf("live Search: %v", err)
	}
	if len(res) == 0 {
		t.Fatal("expected search results for 'apple'")
	}
	found := false
	for _, result := range res {
		if result.Symbol == "AAPL" {
			found = true
			if result.Name == "" {
				t.Errorf("AAPL result missing name: %+v", result)
			}
		}
	}
	if !found {
		t.Errorf("expected AAPL among results, got %+v", res)
	}
}
