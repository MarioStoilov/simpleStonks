package qtui

import (
	"log/slog"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/notify"
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

// removeSymbol drops a symbol from the tracked list, along with its pending
// price alerts — an untracked symbol is never fetched, so they could neither
// fire nor be managed.
func removeSymbol(store *config.Store, symbol string) {
	err := store.Update(func(conf *config.Config) {
		kept := make([]string, 0, len(conf.Symbols))
		for _, tracked := range conf.Symbols {
			if tracked != symbol {
				kept = append(kept, tracked)
			}
		}
		conf.Symbols = kept
		keptAlerts := make([]notify.Alert, 0, len(conf.Alerts))
		for _, alert := range conf.Alerts {
			if alert.Symbol != symbol {
				keptAlerts = append(keptAlerts, alert)
			}
		}
		conf.Alerts = keptAlerts
	})
	if err != nil {
		slog.Error("remove symbol failed", "symbol", symbol, "err", err)
	}
}

// clearAlerts drops every pending price alert.
func clearAlerts(store *config.Store) {
	err := store.Update(func(conf *config.Config) {
		conf.Alerts = nil
	})
	if err != nil {
		slog.Error("clear alerts failed", "err", err)
	}
}

// removeAlert drops one pending price alert.
func removeAlert(store *config.Store, alert notify.Alert) {
	err := store.Update(func(conf *config.Config) {
		kept := make([]notify.Alert, 0, len(conf.Alerts))
		for _, existing := range conf.Alerts {
			if existing != alert {
				kept = append(kept, existing)
			}
		}
		conf.Alerts = kept
	})
	if err != nil {
		slog.Error("remove alert failed", "symbol", alert.Symbol, "price", alert.Price, "err", err)
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
