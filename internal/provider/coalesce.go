package provider

import (
	"context"
	"sync"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// Coalesced wraps a QuoteProvider so that identical history requests made
// while one is already in flight share that single fetch and its result.
// The UI asks for the same series from several places on one refresh tick
// (the detail view's main chart and its sidebar tile, for instance), and
// separate requests can come back with different latest prices; sharing one
// keeps every place showing the same number. Quote and Search pass through.
type Coalesced struct {
	QuoteProvider

	mutex    sync.Mutex
	inflight map[historyKey]*historyCall
}

// historyKey identifies one distinct history request.
type historyKey struct {
	symbol   string
	rng      model.Range
	extended bool
}

// historyCall is a fetch in progress; done closes once the result is set.
type historyCall struct {
	done   chan struct{}
	series model.Series
	err    error
}

// Coalesce wraps a provider with in-flight request sharing.
func Coalesce(inner QuoteProvider) *Coalesced {
	return &Coalesced{QuoteProvider: inner, inflight: map[historyKey]*historyCall{}}
}

// History shares an identical in-flight History fetch, or starts one.
func (prov *Coalesced) History(ctx context.Context, symbol string, rng model.Range) (model.Series, error) {
	return prov.shared(ctx, historyKey{symbol: symbol, rng: rng}, func(fetchCtx context.Context) (model.Series, error) {
		return prov.QuoteProvider.History(fetchCtx, symbol, rng)
	})
}

// HistoryExtended shares an identical in-flight HistoryExtended fetch, or
// starts one.
func (prov *Coalesced) HistoryExtended(ctx context.Context, symbol string) (model.Series, error) {
	return prov.shared(ctx, historyKey{symbol: symbol, rng: model.Range1D, extended: true}, func(fetchCtx context.Context) (model.Series, error) {
		return prov.QuoteProvider.HistoryExtended(fetchCtx, symbol)
	})
}

// shared runs fetch for key unless one is already in flight, in which case
// it waits for that one's result. The leader's fetch is detached from its
// own caller's cancellation (joiners depend on it) but keeps the deadline;
// each waiter still honours its own context.
func (prov *Coalesced) shared(ctx context.Context, key historyKey, fetch func(context.Context) (model.Series, error)) (model.Series, error) {
	prov.mutex.Lock()
	call, joined := prov.inflight[key]
	if !joined {
		call = &historyCall{done: make(chan struct{})}
		prov.inflight[key] = call
	}
	prov.mutex.Unlock()

	if joined {
		select {
		case <-call.done:
			return call.series, call.err
		case <-ctx.Done():
			return model.Series{}, ctx.Err()
		}
	}

	fetchCtx := context.WithoutCancel(ctx)
	if deadline, ok := ctx.Deadline(); ok {
		var cancel context.CancelFunc
		fetchCtx, cancel = context.WithDeadline(fetchCtx, deadline)
		defer cancel()
	}
	call.series, call.err = fetch(fetchCtx)
	prov.mutex.Lock()
	delete(prov.inflight, key)
	prov.mutex.Unlock()
	close(call.done)
	return call.series, call.err
}
