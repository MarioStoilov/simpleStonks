package provider

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// gatedProvider counts history fetches and holds each one until released,
// so a test can line up concurrent callers deterministically.
type gatedProvider struct {
	release chan struct{}
	calls   atomic.Int32
	err     error
}

func (fake *gatedProvider) Name() string { return "gated" }

func (fake *gatedProvider) Quote(context.Context, string) (model.Quote, error) {
	return model.Quote{}, nil
}

func (fake *gatedProvider) Search(context.Context, string) ([]model.SearchResult, error) {
	return nil, nil
}

func (fake *gatedProvider) History(ctx context.Context, symbol string, rng model.Range) (model.Series, error) {
	return fake.fetch(ctx, symbol, rng)
}

func (fake *gatedProvider) HistoryExtended(ctx context.Context, symbol string) (model.Series, error) {
	return fake.fetch(ctx, symbol, model.Range1D)
}

func (fake *gatedProvider) fetch(ctx context.Context, symbol string, rng model.Range) (model.Series, error) {
	call := fake.calls.Add(1)
	select {
	case <-fake.release:
	case <-ctx.Done():
		return model.Series{}, ctx.Err()
	}
	if fake.err != nil {
		return model.Series{}, fake.err
	}
	// The call number as the price tells callers which fetch they got.
	return model.Series{Symbol: symbol, Range: rng, Candles: []model.Candle{{Close: float64(call)}}}, nil
}

// waitForCalls blocks until the fake has seen the expected number of fetches.
func waitForCalls(t *testing.T, fake *gatedProvider, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for fake.calls.Load() < want {
		if time.Now().After(deadline) {
			t.Fatalf("fetch count = %d, want %d", fake.calls.Load(), want)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestCoalescedSharesInflightHistory(t *testing.T) {
	fake := &gatedProvider{release: make(chan struct{})}
	prov := Coalesce(fake)

	const callers = 4
	results := make([]model.Series, callers)
	errs := make([]error, callers)
	var group sync.WaitGroup
	for idx := range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			results[idx], errs[idx] = prov.History(context.Background(), "AAPL", model.Range1D)
		}()
	}
	waitForCalls(t, fake, 1)
	// Only the leader reached the inner provider; the rest are waiting.
	time.Sleep(10 * time.Millisecond)
	if got := fake.calls.Load(); got != 1 {
		t.Fatalf("inner fetches while in flight = %d, want 1", got)
	}
	close(fake.release)
	group.Wait()

	for idx := range callers {
		if errs[idx] != nil {
			t.Fatalf("caller %d: %v", idx, errs[idx])
		}
		if got := results[idx].Candles[0].Close; got != 1 {
			t.Errorf("caller %d got fetch #%v, want the shared fetch #1", idx, got)
		}
	}

	// After completion a new request fetches again.
	series, err := prov.History(context.Background(), "AAPL", model.Range1D)
	if err != nil {
		t.Fatal(err)
	}
	if got := series.Candles[0].Close; got != 2 {
		t.Errorf("follow-up got fetch #%v, want a fresh fetch #2", got)
	}
}

func TestCoalescedKeepsDistinctRequestsApart(t *testing.T) {
	fake := &gatedProvider{release: make(chan struct{})}
	prov := Coalesce(fake)

	var group sync.WaitGroup
	for _, run := range []func(){
		func() { _, _ = prov.History(context.Background(), "AAPL", model.Range1D) },
		func() { _, _ = prov.History(context.Background(), "AAPL", model.Range1M) },
		func() { _, _ = prov.History(context.Background(), "MSFT", model.Range1D) },
		func() { _, _ = prov.HistoryExtended(context.Background(), "AAPL") },
	} {
		group.Add(1)
		go func() {
			defer group.Done()
			run()
		}()
	}
	waitForCalls(t, fake, 4)
	close(fake.release)
	group.Wait()
}

func TestCoalescedPropagatesErrorToAllCallers(t *testing.T) {
	fetchErr := errors.New("boom")
	fake := &gatedProvider{release: make(chan struct{}), err: fetchErr}
	prov := Coalesce(fake)

	errs := make([]error, 2)
	var group sync.WaitGroup
	for idx := range errs {
		group.Add(1)
		go func() {
			defer group.Done()
			_, errs[idx] = prov.History(context.Background(), "AAPL", model.Range1D)
		}()
	}
	waitForCalls(t, fake, 1)
	close(fake.release)
	group.Wait()
	for idx, err := range errs {
		if !errors.Is(err, fetchErr) {
			t.Errorf("caller %d err = %v, want %v", idx, err, fetchErr)
		}
	}
}

func TestCoalescedJoinerHonoursOwnContext(t *testing.T) {
	fake := &gatedProvider{release: make(chan struct{})}
	prov := Coalesce(fake)

	leaderDone := make(chan error, 1)
	go func() {
		_, err := prov.History(context.Background(), "AAPL", model.Range1D)
		leaderDone <- err
	}()
	waitForCalls(t, fake, 1)

	ctx, cancel := context.WithCancel(context.Background())
	joinerDone := make(chan error, 1)
	go func() {
		_, err := prov.History(ctx, "AAPL", model.Range1D)
		joinerDone <- err
	}()
	cancel()
	select {
	case err := <-joinerDone:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("joiner err = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled joiner did not return")
	}

	// The leader's fetch is unaffected by the joiner's cancellation.
	close(fake.release)
	if err := <-leaderDone; err != nil {
		t.Errorf("leader err = %v", err)
	}
}

func TestCoalescedLeaderSurvivesOwnCallerCancel(t *testing.T) {
	fake := &gatedProvider{release: make(chan struct{})}
	prov := Coalesce(fake)

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderDone := make(chan struct{})
	go func() {
		defer close(leaderDone)
		_, _ = prov.History(leaderCtx, "AAPL", model.Range1D)
	}()
	waitForCalls(t, fake, 1)

	joinerDone := make(chan model.Series, 1)
	go func() {
		series, _ := prov.History(context.Background(), "AAPL", model.Range1D)
		joinerDone <- series
	}()
	// Give the joiner a moment to attach, then cancel the leader's caller:
	// the shared fetch must keep going for the joiner's sake.
	time.Sleep(10 * time.Millisecond)
	cancelLeader()
	time.Sleep(10 * time.Millisecond)
	close(fake.release)
	<-leaderDone
	select {
	case series := <-joinerDone:
		if len(series.Candles) == 0 {
			t.Error("joiner got no data after the leader's caller cancelled")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("joiner did not return")
	}
}
