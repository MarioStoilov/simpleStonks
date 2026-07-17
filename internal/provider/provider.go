// Package provider defines the data-source abstraction for simpleStonks and
// its concrete implementations.
//
// QuoteProvider is the seam that lets the data source be swapped without
// touching the UI or config. The shipping implementation is keyless
// (see yahoo.go); this interface exists so alternative or paid sources can be
// added later. That swap capability is an internal design affordance, not an
// advertised or committed feature.
package provider

import (
	"context"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// QuoteProvider fetches market data for a single symbol. Implementations must
// be safe for concurrent use by multiple goroutines, since the UI refreshes
// tiles in parallel.
type QuoteProvider interface {
	// Name identifies the provider (e.g. for settings/diagnostics).
	Name() string

	// Quote returns the latest price snapshot for a symbol.
	Quote(ctx context.Context, symbol string) (model.Quote, error)

	// History returns the price series for a symbol over the given range.
	History(ctx context.Context, symbol string, r model.Range) (model.Series, error)

	// Search returns instruments matching a free-text query (name or symbol),
	// for the add-symbol live search. An empty query yields no results.
	Search(ctx context.Context, query string) ([]model.SearchResult, error)
}
