// Package service exposes the Go core to the Wails frontend as bound
// services: thin wrappers around the quote provider and the config store
// whose methods are callable from TypeScript through generated bindings.
package service

import (
	"context"

	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// Market serves price history and symbol search to the frontend.
type Market struct {
	provider provider.QuoteProvider
}

// NewMarket wraps a quote provider for frontend consumption.
func NewMarket(prov provider.QuoteProvider) *Market {
	return &Market{provider: prov}
}

// History returns the price series for a symbol over the given range.
func (market *Market) History(ctx context.Context, symbol string, rng model.Range) (model.Series, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.FetchTimeout)
	defer cancel()
	return market.provider.History(ctx, symbol, rng)
}

// Search returns instruments matching a free-text query (name or symbol).
func (market *Market) Search(ctx context.Context, query string) ([]model.SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.FetchTimeout)
	defer cancel()
	return market.provider.Search(ctx, query)
}
