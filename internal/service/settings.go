package service

import (
	"github.com/MarioStoilov/simplestonks/internal/config"
)

// Settings exposes the persisted configuration to the frontend. Mutations go
// through the config store, so they persist and notify subscribers (main
// forwards those notifications to the frontend as events).
type Settings struct {
	store *config.Store
}

// NewSettings wraps the config store for frontend consumption.
func NewSettings(store *config.Store) *Settings {
	return &Settings{store: store}
}

// Get returns a snapshot of the current configuration.
func (settings *Settings) Get() config.Config {
	return settings.store.Get()
}

// AddSymbol appends a symbol to the tracked list, ignoring duplicates.
func (settings *Settings) AddSymbol(symbol string) error {
	return settings.store.Update(func(conf *config.Config) {
		for _, tracked := range conf.Symbols {
			if tracked == symbol {
				return
			}
		}
		conf.Symbols = append(conf.Symbols, symbol)
	})
}

// RemoveSymbol drops a symbol from the tracked list.
func (settings *Settings) RemoveSymbol(symbol string) error {
	return settings.store.Update(func(conf *config.Config) {
		kept := make([]string, 0, len(conf.Symbols))
		for _, tracked := range conf.Symbols {
			if tracked != symbol {
				kept = append(kept, tracked)
			}
		}
		conf.Symbols = kept
	})
}

// MoveSymbol swaps the symbol at index with its neighbor delta positions away
// (delta is -1 or +1), persisting the new order. Out-of-range moves are no-ops.
func (settings *Settings) MoveSymbol(index, delta int) error {
	neighbor := index + delta
	return settings.store.Update(func(conf *config.Config) {
		if index < 0 || index >= len(conf.Symbols) || neighbor < 0 || neighbor >= len(conf.Symbols) {
			return
		}
		conf.Symbols[index], conf.Symbols[neighbor] = conf.Symbols[neighbor], conf.Symbols[index]
	})
}
