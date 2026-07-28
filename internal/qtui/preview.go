package qtui

import (
	"context"
	"fmt"
	"strings"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// showPreviewDialog shows a compact look at one search result — title,
// price/change, a 1D chart — with Cancel and Add actions. Add is disabled
// (with a tooltip) when the symbol is already tracked. Reports whether the
// symbol was added, so the caller can close the search dialog too.
func showPreviewDialog(parent *qt.QWidget, quotes provider.QuoteProvider, store *config.Store, match model.SearchResult) bool {
	title := match.Symbol
	if match.Name != "" {
		title += constants.SepTitle + match.Name
	}
	dialog, body := newCardDialog(parent, title)
	dialog.Resize(int(constants.PreviewDialogWidth), int(constants.PreviewDialogHeight))

	var meta []string
	if match.Exchange != "" {
		meta = append(meta, match.Exchange)
	}
	if match.Type != "" {
		meta = append(meta, match.Type)
	}
	head := qt.NewQHBoxLayout2()
	metaLabel := qt.NewQLabel5(strings.Join(meta, constants.SepMeta), dialog.QWidget)
	metaLabel.SetStyleSheet(fmt.Sprintf(constants.StyleSmallText,
		cssRGB(constants.ColorNeutral), int(constants.NameTextSize)))
	head.AddWidget(metaLabel.QWidget)
	head.AddStretch()
	priceLabel := qt.NewQLabel5(constants.PricePlaceholder, dialog.QWidget)
	priceLabel.SetStyleSheet(priceBaseStyle)
	head.AddWidget(priceLabel.QWidget)
	changeLabel := qt.NewQLabel(dialog.QWidget)
	changeLabel.SetStyleSheet("background: transparent;")
	head.AddWidget(changeLabel.QWidget)
	body.AddLayout(head.QLayout)

	chart := newChartWidget(dialog.QWidget)
	body.AddWidget2(chart.QWidget, 1)

	actions := qt.NewQHBoxLayout2()
	actions.AddStretch()
	cancelButton := qt.NewQPushButton5(constants.LabelCancel, dialog.QWidget)
	cancelButton.SetStyleSheet(dialogButtonStyle(false))
	cancelButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	cancelButton.OnClicked(func() { dialog.Reject() })
	actions.AddWidget(cancelButton.QWidget)
	addButton := qt.NewQPushButton5(constants.LabelAdd, dialog.QWidget)
	addButton.SetStyleSheet(dialogButtonStyle(true))
	addButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	alreadyTracked := false
	for _, tracked := range store.Get().Symbols {
		if tracked == match.Symbol {
			alreadyTracked = true
			break
		}
	}
	if alreadyTracked {
		addButton.SetDisabled(true)
		addButton.SetToolTip(constants.TipAlreadyTracked)
	}
	addButton.OnClicked(func() {
		addSymbol(store, match.Symbol)
		dialog.Accept()
	})
	actions.AddWidget(addButton.QWidget)
	body.AddLayout(actions.QLayout)

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
				changeLabel.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorNeutral) + ";")
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
				changeLabel.SetStyleSheet("background: transparent; color: " + cssRGB(col) + ";")
			}
			chart.setSeries(series, lineColor)
		})
	}()

	return dialog.Exec() == int(qt.QDialog__Accepted)
}
