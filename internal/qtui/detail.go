package qtui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/assets"
	"github.com/MarioStoilov/simplestonks/internal/chartmath"
	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/notify"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// detailView is the expanded view of one symbol: a sidebar of compact tiles
// for every tracked symbol (the current one highlighted), a header with the
// price and change, the range toggles, and the big chart with the hover
// readout.
type detailView struct {
	quotes provider.QuoteProvider
	store  *config.Store
	onBack func()

	// onQuote reports every sidebar fetch outcome (latest close on success)
	// so the app can track connectivity and check the pending price alerts.
	onQuote func(symbol string, price float64, ok bool)

	root           *qt.QWidget
	sidebarContent *qt.QWidget
	sidebarBox     *qt.QVBoxLayout
	symbolLabel    *qt.QLabel
	nameLabel      *qt.QLabel
	priceLabel     *qt.QLabel
	changeLabel    *qt.QLabel
	extendedLabel  *qt.QLabel // separate pre-market/after-hours price
	extendedToggle *qt.QCheckBox
	alertButton    *qt.QPushButton
	alertsBox      *qt.QHBoxLayout // pending-alert pills under the chart
	alertPills     []*qt.QFrame
	chart          *chartWidget
	rangeButtons   map[model.Range]*qt.QPushButton

	symbol        string
	rng           model.Range
	extendedHours bool // shared setting: show pre/post market data on 1D
	generation    int  // drops stale async responses after symbol/range switches

	// notificationsOn mirrors the notifications setting: off hides the
	// alert bell and the pending-alert pills (the alerts stay stored).
	notificationsOn bool

	// Extended-hours capability per symbol, from Yahoo's
	// hasPrePostMarketData flag on every fetched series (indexes report
	// pre/post windows but never trade in them, so the windows cannot tell).
	extendedKnown map[string]bool

	// marketState is the shown symbol's session state as of the last applied
	// fetch; the toggle only shows when the setting has an effect (pre-market
	// or after-hours).
	marketState chartmath.MarketState

	order        []string
	sidebarTiles map[string]*tile

	shownPrice float64
	priceShown bool
	flashTimer *qt.QTimer
}

// newDetailView loads the detail form (detail.ui); showSymbol populates it.
func newDetailView(parent *qt.QWidget, quotes provider.QuoteProvider, store *config.Store, onBack func()) *detailView {
	loaded := loadForm(detailForm)
	loaded.root.SetParent(parent)
	view := &detailView{
		quotes:         quotes,
		store:          store,
		onBack:         onBack,
		root:           loaded.root,
		sidebarContent: loaded.widget("sidebarContent"),
		sidebarBox:     loaded.vbox("sidebarBox"),
		symbolLabel:    loaded.label("symbolLabel"),
		nameLabel:      loaded.label("nameLabel"),
		priceLabel:     loaded.label("priceLabel"),
		changeLabel:    loaded.label("changeLabel"),
		extendedLabel:  loaded.label("extendedLabel"),
		extendedToggle: loaded.checkBox("extendedToggle"),
		alertButton:    loaded.button("alertButton"),
		alertsBox:      loaded.hbox("alertsBox"),
		chart:          loaded.chart("chart"),
		rangeButtons:   map[model.Range]*qt.QPushButton{},
		sidebarTiles:   map[string]*tile{},
		extendedKnown:  map[string]bool{},
	}

	// The sidebar lists every tracked symbol as a compact tile; the loader
	// parents the content widget but leaves the viewport assignment to us.
	view.sidebarBox.AddStretch()
	loaded.scrollArea("sidebarScroll").SetWidget(view.sidebarContent)
	// SetWidget turns on the content's autoFillBackground, which paints the
	// palette color over the translucent window; the theme styles it.
	view.sidebarContent.SetAutoFillBackground(false)

	loaded.button("backButton").OnClicked(func() { onBack() })

	// The alert bell is enabled once the first fetch supplies a current
	// price to measure alerts against. Its icon is an embedded SVG — the
	// bell emoji rendered font-dependently across machines.
	view.alertButton.SetIcon(qt.NewQIcon2(svgPixmap(assets.BellPlusSVG, int(constants.HeaderIconSize))))
	view.alertButton.SetIconSize(qt.NewQSize2(int(constants.HeaderIconSize), int(constants.HeaderIconSize)))
	view.alertButton.OnClicked(func() {
		if view.priceShown {
			showAlertDialog(view.root, view.store, view.symbol, view.shownPrice)
		}
	})

	// The extended-hours toggle writes the shared setting — the reload
	// arrives via the config subscription, the same path the settings
	// dialog and external file edits use. The form keeps it hidden until
	// updateToggleVisibility finds it has an effect.
	view.extendedToggle.OnClicked(func() {
		checked := view.extendedToggle.IsChecked()
		if err := view.store.Update(func(conf *config.Config) { conf.ExtendedHours = checked }); err != nil {
			slog.Error("saving extended-hours toggle failed", "error", err)
		}
	})

	// One range toggle per range, by object name; a range the form has no
	// button for is a programming error and panics in loaded.button.
	for _, rangeOption := range model.Ranges {
		toggled := rangeOption // capture per iteration
		button := loaded.button("range" + string(rangeOption))
		button.OnClicked(func() { view.setRange(toggled) })
		view.rangeButtons[rangeOption] = button
	}

	view.chart.enableHoverReadout()

	view.flashTimer = qt.NewQTimer2(view.root.QObject)
	view.flashTimer.SetSingleShot(true)
	view.flashTimer.OnTimeout(func() {
		setState(view.priceLabel.QWidget, "flash", "")
	})

	return view
}

// showSymbol switches the view to a symbol at the configured default range
// and reloads everything.
func (view *detailView) showSymbol(symbol string) {
	view.symbol = symbol
	view.symbolLabel.SetText(symbol)
	view.nameLabel.SetText("")
	view.priceLabel.SetText(constants.PricePlaceholder)
	view.changeLabel.SetText("")
	view.extendedLabel.SetText("")
	view.priceShown = false
	view.alertButton.SetEnabled(false)
	view.setAlerts(view.store.Get().Alerts)
	view.rng = view.store.Get().DefaultRange
	view.styleRangeButtons()
	// Regular hides the toggle until the first fetch classifies the session.
	view.marketState = chartmath.MarketRegular
	view.updateToggleVisibility()
	view.generation++
	view.loadMain(false)
	view.setSymbols(view.store.Get().Symbols)
	for tracked, cell := range view.sidebarTiles {
		cell.setSelected(tracked == symbol)
	}
	view.loadSidebar(false)
}

// setRange switches the chart range and reloads the main chart.
func (view *detailView) setRange(rng model.Range) {
	if rng == view.rng {
		return
	}
	view.rng = rng
	view.priceShown = false // a range switch must not flash
	view.styleRangeButtons()
	view.updateToggleVisibility()
	view.generation++
	view.loadMain(false)
}

// refresh is the periodic tick: the sidebar tiles (1D) and the main chart
// at whatever range it shows, so the headline price keeps ticking alongside
// the sidebar on every range (a series' last candle is always the latest
// price). On 1D the two fetches of the shown symbol are one shared request
// (provider.Coalesced), so header and sidebar tile agree.
func (view *detailView) refresh() {
	view.loadSidebar(true)
	view.loadMain(true)
}

// setExtendedHours applies the shared extended-hours setting: syncs the
// header toggle and reloads the main chart when it is showing 1D.
func (view *detailView) setExtendedHours(enabled bool) {
	if enabled == view.extendedHours && view.extendedToggle.IsChecked() == enabled {
		return
	}
	view.extendedHours = enabled
	view.extendedToggle.SetChecked(enabled)
	if view.symbol != "" && view.rng.Intraday() {
		view.generation++
		view.loadMain(false)
	}
}

// updateToggleVisibility shows the extended-hours toggle only when it has an
// effect: on the 1D range, for symbols known to trade pre/post, outside the
// regular session — the live pre-market/after-hours views and the
// closed-market replay of the last session. Unknown symbols keep it hidden
// until their first fetch reports the capability flag.
func (view *detailView) updateToggleVisibility() {
	view.extendedToggle.SetVisible(view.extendedKnown[view.symbol] && view.rng.Intraday() &&
		view.marketState != chartmath.MarketRegular)
}

// setNotificationsEnabled shows or hides the alert bell with the
// notifications setting; setAlerts consults the flag too, so callers apply
// it before rebuilding the pills.
func (view *detailView) setNotificationsEnabled(enabled bool) {
	view.notificationsOn = enabled
	view.alertButton.SetVisible(enabled)
}

// setAlerts rebuilds the pending-alert pills under the chart for the shown
// symbol: each pill shows the alert's direction and threshold with a remove
// button. Alert changes (adds, removals, fired alerts, external file edits)
// all land here through the config subscription. With notifications
// disabled the row stays empty.
func (view *detailView) setAlerts(alerts []notify.Alert) {
	for _, pill := range view.alertPills {
		pill.SetParent(nil)
		pill.DeleteLater()
	}
	view.alertPills = nil
	for view.alertsBox.Count() > 0 { // drop the stretch and stale items
		view.alertsBox.TakeAt(0).Delete()
	}
	if !view.notificationsOn {
		view.alertsBox.AddStretch()
		return
	}
	for _, alert := range alerts {
		if alert.Symbol != view.symbol {
			continue
		}
		removed := alert // capture per iteration for the remove callback
		loaded := loadForm(alertPillForm)
		pill := loaded.frame("pill")
		direction := constants.SymbolAlertUp
		if !alert.Above {
			direction = constants.SymbolAlertDown
		}
		priceLabel := loaded.label("priceLabel")
		priceLabel.SetText(fmt.Sprintf(constants.FmtAlertPill, direction, alert.Price))
		setState(priceLabel.QWidget, "direction", directionOfAlert(alert.Above))
		loaded.button("removeButton").OnClicked(func() { removeAlert(view.store, removed) })
		view.alertsBox.AddWidget(pill.QWidget)
		view.alertPills = append(view.alertPills, pill)
	}
	view.alertsBox.AddStretch()
}

// directionOfAlert maps an alert's direction to the theme's states.
func directionOfAlert(above bool) string {
	if above {
		return directionUp
	}
	return directionDown
}

// repaintCharts repaints the main chart and every sidebar tile chart (the
// chart styling changed).
func (view *detailView) repaintCharts() {
	view.chart.Update()
	for _, cell := range view.sidebarTiles {
		cell.chart.Update()
	}
}

// setSymbols reconciles the sidebar with the tracked list: tiles are created
// for new symbols, dropped for removed ones, and the whole column is re-laid
// in tracked order so a reorder on the home grid shows here immediately.
func (view *detailView) setSymbols(symbols []string) {
	next := map[string]*tile{}
	for _, symbol := range symbols {
		if existing, ok := view.sidebarTiles[symbol]; ok {
			next[symbol] = existing
			continue
		}
		opened := symbol // capture per iteration for the click callback
		next[symbol] = newTile(view.sidebarContent, opened, true, func() {
			view.showSymbol(opened)
		})
	}
	for symbol, cell := range view.sidebarTiles {
		if _, keep := next[symbol]; !keep {
			cell.frame.SetParent(nil)
			cell.frame.DeleteLater()
		}
	}
	view.order = append([]string{}, symbols...)
	view.sidebarTiles = next
	for view.sidebarBox.Count() > 0 { // drop the stretch and stale items
		view.sidebarBox.TakeAt(0).Delete()
	}
	for _, symbol := range view.order {
		cell := next[symbol]
		cell.setSelected(symbol == view.symbol)
		view.sidebarBox.AddWidget(cell.frame.QWidget)
	}
	view.sidebarBox.AddStretch()
}

// loadMain fetches the main chart's series off the UI thread. On 1D with the
// extended-hours setting on it fetches pre/post market data too; a closed
// market replays the completed session's after-hours tail, falling back to
// the regular fetch when no replay can be built (see
// chartmath.BuildExtendedDisplay).
func (view *detailView) loadMain(flash bool) {
	symbol, rng, requestGen := view.symbol, view.rng, view.generation
	capable, known := view.extendedKnown[symbol]
	extended := view.extendedHours && rng == model.Range1D && (!known || capable)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), constants.FetchTimeout)
		defer cancel()
		var display chartmath.ExtendedDisplay
		var series model.Series
		var err error
		if extended {
			series, err = view.quotes.HistoryExtended(ctx, symbol)
			if err == nil {
				display = chartmath.BuildExtendedDisplay(series, time.Now())
				if display.State == chartmath.MarketClosed && display.DimFromIdx < 0 {
					// A closed market pairs the previous day's candles with
					// the upcoming session's windows; when no after-hours
					// replay could be built from them, only the regular
					// fetch renders those correctly.
					extended = false
				} else {
					series = display.Series
				}
			}
		}
		if !extended && err == nil {
			series, err = view.quotes.History(ctx, symbol, rng)
		}
		mainthread.Wait(func() {
			if err == nil {
				// Every chart response carries the pre/post capability flag.
				view.extendedKnown[symbol] = series.HasExtendedHours
			}
			if requestGen != view.generation { // superseded by a newer switch
				return
			}
			if err != nil || len(series.Candles) == 0 {
				// Keep the last rendered chart on refresh failures (e.g.
				// network loss); only a view that never had data for this
				// symbol/range shows the unavailable state.
				if !view.priceShown {
					view.priceLabel.SetText(constants.PricePlaceholder)
					view.changeLabel.SetText(constants.MsgUnavailable)
					setState(view.changeLabel.QWidget, "direction", directionFlat)
					view.extendedLabel.SetText("")
					view.chart.setSeries(model.Series{}, constants.ColorNeutral)
					view.alertButton.SetEnabled(false)
				}
				return
			}
			if extended {
				view.applyExtended(display, flash)
			} else {
				view.applyMain(series, flash)
			}
		})
	}()
}

// applyMain renders a fetched series into the header and chart.
func (view *detailView) applyMain(series model.Series, flash bool) {
	// Regular fetches carry the session windows too; while the market is
	// closed they belong to the upcoming session, which StateAt classifies
	// as closed — the toggle then offers the last session's replay.
	view.marketState = chartmath.StateAt(series, time.Now())
	view.updateToggleVisibility()
	view.nameLabel.SetText(series.Name)
	view.extendedLabel.SetText("")
	last := series.Candles[len(series.Candles)-1].Close
	view.priceLabel.SetText(chartPrice(last))
	view.changeLabel.SetText("")

	lineColor := constants.ColorNeutral
	if series.PreviousClose > 0 {
		change := last - series.PreviousClose
		percent := change / series.PreviousClose * constants.PercentMax
		col, sign := changeStyle(change)
		lineColor = col
		view.changeLabel.SetText(fmt.Sprintf(constants.FmtPriceChange, sign, change, sign, percent))
		setState(view.changeLabel.QWidget, "direction", directionOf(change))
	}
	view.chart.setSeries(series, lineColor)
	view.flashPrice(last, flash)
}

// applyExtended renders an extended-hours display: the state-filtered chart,
// the regular price in the headline, and the separate pre/post price label.
func (view *detailView) applyExtended(display chartmath.ExtendedDisplay, flash bool) {
	series := display.Series
	view.marketState = display.State
	view.updateToggleVisibility()
	view.nameLabel.SetText(series.Name)

	// Headline: the regular-session price. During pre-market the series has
	// no regular candles, so the meta price is the only source; during the
	// session it equals the latest close.
	headline := series.RegularPrice
	if headline <= 0 {
		headline = series.Candles[len(series.Candles)-1].Close
	}
	view.priceLabel.SetText(chartPrice(headline))
	view.changeLabel.SetText("")
	if series.PreviousClose > 0 {
		change := headline - series.PreviousClose
		percent := change / series.PreviousClose * constants.PercentMax
		_, sign := changeStyle(change)
		view.changeLabel.SetText(fmt.Sprintf(constants.FmtPriceChange, sign, change, sign, percent))
		setState(view.changeLabel.QWidget, "direction", directionOf(change))
	}

	// Separate extended-hours price, measured against the regular price.
	if display.ExtendedPrice > 0 && series.RegularPrice > 0 && display.State != chartmath.MarketRegular {
		prefix := constants.LabelPreMarket
		if display.State != chartmath.MarketPreMarket {
			// The after-hours live view or the closed-market replay.
			prefix = constants.LabelAfterHours
		}
		change := display.ExtendedPrice - series.RegularPrice
		percent := change / series.RegularPrice * constants.PercentMax
		_, sign := changeStyle(change)
		view.extendedLabel.SetText(fmt.Sprintf(constants.FmtExtendedQuote, prefix, display.ExtendedPrice, sign, percent))
		setState(view.extendedLabel.QWidget, "direction", directionOf(change))
	} else {
		view.extendedLabel.SetText("")
	}

	// The pre-market chart reads against the regular close, Yahoo-style:
	// the dashed reference sits at yesterday's close and the line, hover
	// percentage, and color follow the pre-market change. The header math
	// above keeps the session's own previous close.
	if display.State == chartmath.MarketPreMarket && series.RegularPrice > 0 {
		series.PreviousClose = series.RegularPrice
	}

	// The line color follows what the chart shows: the last plotted close
	// against the dashed previous-close reference.
	lineColor := constants.ColorNeutral
	if series.PreviousClose > 0 {
		col, _ := changeStyle(series.Candles[len(series.Candles)-1].Close - series.PreviousClose)
		lineColor = col
	}
	view.chart.setSeries(series, lineColor)
	if display.DimFromIdx >= 0 { // after-hours live view or closed replay
		view.chart.setAfterHoursOverlay(display.BoundaryTime, display.DimFromIdx)
	}
	view.flashPrice(headline, flash)
}

// flashPrice flashes the headline price backdrop when a live refresh moved
// it, and records the shown price for the next comparison.
func (view *detailView) flashPrice(last float64, flash bool) {
	if flash && view.priceShown && last != view.shownPrice {
		setState(view.priceLabel.QWidget, "flash", directionOf(last-view.shownPrice))
		view.flashTimer.Start(int(constants.FlashDuration / time.Millisecond))
	}
	view.shownPrice = last
	view.priceShown = true
	view.alertButton.SetEnabled(true)
}

// loadSidebar refreshes every sidebar tile at 1D.
func (view *detailView) loadSidebar(flash bool) {
	for _, symbol := range view.order {
		tracked := symbol // capture per iteration
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), constants.FetchTimeout)
			defer cancel()
			series, err := view.quotes.History(ctx, tracked, model.Range1D)
			mainthread.Wait(func() {
				cell, ok := view.sidebarTiles[tracked]
				if !ok {
					return
				}
				if err != nil || len(series.Candles) == 0 {
					// Keep showing the last data (e.g. network loss); only
					// a tile that never had any shows the failure state.
					if !cell.priceShown {
						cell.setFailed()
					}
					if view.onQuote != nil {
						view.onQuote(tracked, 0, false)
					}
					return
				}
				cell.setSeries(series, flash)
				if view.onQuote != nil {
					view.onQuote(tracked, series.Candles[len(series.Candles)-1].Close, true)
				}
			})
		}()
	}
}

// styleRangeButtons highlights the active range toggle.
func (view *detailView) styleRangeButtons() {
	for rangeOption, button := range view.rangeButtons {
		setState(button.QWidget, "selected", fmt.Sprint(rangeOption == view.rng))
	}
}
