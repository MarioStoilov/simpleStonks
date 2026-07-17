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

// addSymbol appends a symbol (ignoring duplicates) and persists the config.
func (a *App) addSymbol(symbol string) {
	for _, s := range a.cfg.Symbols {
		if s == symbol {
			return
		}
	}
	a.cfg.Symbols = append(a.cfg.Symbols, symbol)
	a.persist()
}

// removeSymbol drops a symbol from the tracked list and persists the config.
func (a *App) removeSymbol(symbol string) {
	kept := a.cfg.Symbols[:0]
	for _, s := range a.cfg.Symbols {
		if s != symbol {
			kept = append(kept, s)
		}
	}
	a.cfg.Symbols = kept
	a.persist()
}

// persist saves the current config, surfacing any error to the user.
func (a *App) persist() {
	if err := config.Save(a.cfg); err != nil && a.win != nil {
		dialog.ShowError(err, a.win)
	}
}
