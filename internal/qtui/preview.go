package qtui

import (
	"context"
	"fmt"
	"slices"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// showPreviewDialog shows a compact look at one search result (preview.ui)
// — title, price/change, a 1D chart — with Cancel and Add actions. Add is
// disabled (with a tooltip) when the symbol is already tracked. Reports
// whether the symbol was added, so the caller can close the search dialog
// too.
func showPreviewDialog(parent *qt.QWidget, quotes provider.QuoteProvider, store *config.Store, match model.SearchResult) bool {
	title := match.Symbol
	if match.Name != "" {
		title += constants.SepTitle + match.Name
	}
	dialog, body := newCardDialog(parent, title)
	dialog.Resize(int(constants.PreviewDialogWidth), int(constants.PreviewDialogHeight))
	loaded := loadForm(previewForm)
	body.AddWidget(loaded.root)
	body.SetStretchFactor(loaded.root, 1)

	loaded.label("metaLabel").SetText(searchResultMeta(match))
	priceLabel := loaded.label("priceLabel")
	changeLabel := loaded.label("changeLabel")
	chart := loaded.chart("chart")

	loaded.button("cancelButton").OnClicked(func() { dialog.Reject() })
	addButton := loaded.button("addButton")
	addButton.OnClicked(func() {
		addSymbol(store, match.Symbol)
		dialog.Accept()
	})
	if slices.Contains(store.Get().Symbols, match.Symbol) {
		addButton.SetDisabled(true)
		addButton.SetToolTip(constants.TipAlreadyTracked)
	}

	// Load the 1D series for the preview chart off the UI thread.
	closed := false
	dialog.OnFinished(func(result int) { closed = true })
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), constants.FetchTimeout)
		defer cancel()
		series, err := quotes.History(ctx, match.Symbol, model.Range1D)
		mainthread.Wait(func() {
			if closed {
				return
			}
			if err != nil || len(series.Candles) == 0 {
				changeLabel.SetText(constants.MsgUnavailable)
				setState(changeLabel.QWidget, "direction", directionFlat)
				return
			}
			last := series.Candles[len(series.Candles)-1].Close
			priceLabel.SetText(chartPrice(last))
			lineColor := constants.ColorNeutral
			if series.PreviousClose > 0 {
				change := last - series.PreviousClose
				percent := change / series.PreviousClose * constants.PercentMax
				col, sign := changeStyle(change)
				lineColor = col
				changeLabel.SetText(fmt.Sprintf(constants.FmtPriceChange, sign, change, sign, percent))
				setState(changeLabel.QWidget, "direction", directionOf(change))
			}
			chart.setSeries(series, lineColor)
		})
	}()

	return dialog.Exec() == int(qt.QDialog__Accepted)
}
