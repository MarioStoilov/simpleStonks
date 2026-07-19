package qtui

import (
	"log/slog"

	"github.com/MarioStoilov/simplestonks/internal/config"
)

// Tracked-list mutations persist through the config store; the resulting
// change notification re-renders the views. Failures are logged — the UI
// keeps showing the last-good state.

// addSymbol appends a symbol to the tracked list, ignoring duplicates.
func addSymbol(store *config.Store, symbol string) {
	err := store.Update(func(conf *config.Config) {
		for _, tracked := range conf.Symbols {
			if tracked == symbol {
				return
			}
		}
		conf.Symbols = append(conf.Symbols, symbol)
	})
	if err != nil {
		slog.Error("add symbol failed", "symbol", symbol, "err", err)
	}
}

// removeSymbol drops a symbol from the tracked list.
func removeSymbol(store *config.Store, symbol string) {
	err := store.Update(func(conf *config.Config) {
		kept := make([]string, 0, len(conf.Symbols))
		for _, tracked := range conf.Symbols {
			if tracked != symbol {
				kept = append(kept, tracked)
			}
		}
		conf.Symbols = kept
	})
	if err != nil {
		slog.Error("remove symbol failed", "symbol", symbol, "err", err)
	}
}

// moveSymbol swaps the symbol at index with its neighbor delta positions away
// (delta is -1 or +1). Out-of-range moves are no-ops.
func moveSymbol(store *config.Store, index, delta int) {
	neighbor := index + delta
	err := store.Update(func(conf *config.Config) {
		if index < 0 || index >= len(conf.Symbols) || neighbor < 0 || neighbor >= len(conf.Symbols) {
			return
		}
		conf.Symbols[index], conf.Symbols[neighbor] = conf.Symbols[neighbor], conf.Symbols[index]
	})
	if err != nil {
		slog.Error("move symbol failed", "index", index, "delta", delta, "err", err)
	}
}
