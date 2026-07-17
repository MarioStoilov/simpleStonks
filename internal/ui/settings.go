package ui

import (
	"fyne.io/fyne/v2/dialog"

	"github.com/MarioStoilov/simplestonks/internal/config"
)

// addSymbol appends a symbol (ignoring duplicates) via the config store, which
// persists it and triggers a UI rebuild through the subscription.
func (a *App) addSymbol(symbol string) {
	a.update(func(c *config.Config) {
		for _, s := range c.Symbols {
			if s == symbol {
				return
			}
		}
		c.Symbols = append(c.Symbols, symbol)
	})
}

// moveSymbol swaps the symbol at index i with its neighbor delta positions away
// (delta is -1 or +1), persisting the new order. Out-of-range moves are no-ops.
func (a *App) moveSymbol(i, delta int) {
	j := i + delta
	a.update(func(c *config.Config) {
		if i < 0 || i >= len(c.Symbols) || j < 0 || j >= len(c.Symbols) {
			return
		}
		c.Symbols[i], c.Symbols[j] = c.Symbols[j], c.Symbols[i]
	})
}

// removeSymbol drops a symbol from the tracked list via the config store.
func (a *App) removeSymbol(symbol string) {
	a.update(func(c *config.Config) {
		kept := make([]string, 0, len(c.Symbols))
		for _, s := range c.Symbols {
			if s != symbol {
				kept = append(kept, s)
			}
		}
		c.Symbols = kept
	})
}

// update applies a config mutation through the store, surfacing any save error.
func (a *App) update(mutate func(*config.Config)) {
	if err := a.store.Update(mutate); err != nil && a.win != nil {
		dialog.ShowError(err, a.win)
	}
}

// showSettingsPlaceholder stands in for the not-yet-built settings window that
// the top-right cog will eventually open (see docs/REQUIREMENTS.md).
func (a *App) showSettingsPlaceholder() {
	if a.win == nil {
		return
	}
	dialog.ShowInformation("Settings", "The settings window is not implemented yet.", a.win)
}
