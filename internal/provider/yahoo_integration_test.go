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

	q, err := NewYahoo(nil).Quote(ctx, "AAPL")
	if err != nil {
		t.Fatalf("live Quote: %v", err)
	}
	if q.Symbol == "" || q.Price <= 0 {
		t.Fatalf("implausible quote: %+v", q)
	}
}

func TestIntegrationYahooHistory(t *testing.T) {
	requireNetwork(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	s, err := NewYahoo(nil).History(ctx, "AAPL", model.Range1M)
	if err != nil {
		t.Fatalf("live History: %v", err)
	}
	if len(s.Candles) == 0 {
		t.Fatal("expected candles for the 1M range")
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
	for _, r := range res {
		if r.Symbol == "AAPL" {
			found = true
			if r.Name == "" {
				t.Errorf("AAPL result missing name: %+v", r)
			}
		}
	}
	if !found {
		t.Errorf("expected AAPL among results, got %+v", res)
	}
}
