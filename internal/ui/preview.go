package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// showResultPreview opens a small preview of a search result — a compact version
// of the detail view (symbol/name header, price/change, a 1D chart) — with Add
// and Cancel actions. Adding tracks the symbol and calls onAdded (used to also
// close the search dialog).
func (a *App) showResultPreview(r model.SearchResult, onAdded func()) {
	if a.win == nil {
		return
	}

	chartW := newChart()
	price := canvas.NewText("—", theme.Color(theme.ColorNameForeground))
	price.TextStyle = fyne.TextStyle{Bold: true}
	price.Alignment = fyne.TextAlignTrailing
	change := canvas.NewText("", colorNeutral)

	name := r.Symbol
	if r.Name != "" {
		name += "  ·  " + r.Name
	}
	title := widget.NewLabelWithStyle(name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	meta := r.Exchange
	if r.Type != "" {
		if meta != "" {
			meta += " · "
		}
		meta += r.Type
	}
	subtitle := widget.NewLabel(meta)

	header := container.NewVBox(
		container.NewBorder(nil, nil, title, container.NewHBox(price, change)),
		subtitle,
	)
	body := container.NewBorder(header, nil, nil, nil, chartW)

	d := dialog.NewCustomConfirm("Preview", "Add", "Cancel", body, func(add bool) {
		if add {
			a.addSymbol(r.Symbol)
			if onAdded != nil {
				onAdded()
			}
		}
	}, a.win)
	d.Resize(fyne.NewSize(520, 420))
	d.Show()

	// Fetch the 1D series for the preview chart and header.
	loadMainChart(a.provider, chartW, price, change, r.Symbol, model.Range1D)
}
