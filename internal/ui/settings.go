package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/config"
)

// showAddSymbolDialog prompts for a symbol, appends it to the tracked list, and
// persists the config.
//
// TODO(v1): validate the symbol against the provider before adding, and rebuild
// the grid so the new tile appears without a restart.
func (a *App) showAddSymbolDialog() {
	if a.win == nil {
		return
	}
	entry := widget.NewEntry()
	entry.SetPlaceHolder("e.g. AAPL or ^GSPC")

	form := dialog.NewForm("Add symbol", "Add", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Symbol", entry)},
		func(ok bool) {
			if !ok {
				return
			}
			sym := strings.ToUpper(strings.TrimSpace(entry.Text))
			if sym == "" {
				return
			}
			a.addSymbol(sym)
		},
		a.win,
	)
	form.Resize(fyne.NewSize(320, 120))
	form.Show()
}

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
