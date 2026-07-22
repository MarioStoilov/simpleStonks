package qtui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/chartmath"
	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
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

	root           *qt.QWidget
	sidebarContent *qt.QWidget
	sidebarBox     *qt.QVBoxLayout
	symbolLabel    *qt.QLabel
	nameLabel      *qt.QLabel
	priceLabel     *qt.QLabel
	changeLabel    *qt.QLabel
	extendedLabel  *qt.QLabel // separate pre-market/after-hours price
	extendedToggle *qt.QCheckBox
	chart          *chartWidget
	rangeButtons   map[model.Range]*qt.QPushButton

	symbol        string
	rng           model.Range
	extendedHours bool // shared setting: show pre/post market data on 1D
	generation    int  // drops stale async responses after symbol/range switches

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

// newDetailView builds the detail screen; showSymbol populates it.
func newDetailView(parent *qt.QWidget, quotes provider.QuoteProvider, store *config.Store, onBack func()) *detailView {
	root := qt.NewQWidget(parent)
	root.SetStyleSheet("background: transparent;")
	view := &detailView{
		quotes:        quotes,
		store:         store,
		onBack:        onBack,
		root:          root,
		rangeButtons:  map[model.Range]*qt.QPushButton{},
		sidebarTiles:  map[string]*tile{},
		extendedKnown: map[string]bool{},
	}

	outer := qt.NewQHBoxLayout(root)

	// Sidebar: every tracked symbol as a compact tile.
	sidebarScroll := qt.NewQScrollArea(root)
	sidebarScroll.SetWidgetResizable(true)
	// The scrollbar rides inside the fixed width, so leave room for it and
	// never scroll horizontally (tiles shrink to the viewport instead).
	sidebarScroll.SetFixedWidth(int(constants.SidebarMinWidth) + int(constants.ScrollBarWidth))
	sidebarScroll.SetHorizontalScrollBarPolicy(qt.ScrollBarAlwaysOff)
	sidebarScroll.SetStyleSheet(scrollAreaStyle())
	sidebarScroll.SetCursor(qt.NewQCursor2(qt.ArrowCursor))
	sidebarContent := qt.NewQWidget2()
	sidebarContent.SetStyleSheet("background: transparent;")
	view.sidebarContent = sidebarContent
	view.sidebarBox = qt.NewQVBoxLayout(sidebarContent)
	view.sidebarBox.AddStretch()
	sidebarScroll.SetWidget(sidebarContent)
	outer.AddWidget(sidebarScroll.QWidget)

	main := qt.NewQVBoxLayout2()

	// Header: back, symbol + friendly name, price + change.
	head := qt.NewQHBoxLayout2()
	backButton := qt.NewQPushButton5("←", root)
	backButton.SetStyleSheet(windowButtonStyle(cssRGB(constants.ColorHover)))
	backButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	backButton.OnClicked(func() { onBack() })
	head.AddWidget(backButton.QWidget)
	ident := qt.NewQVBoxLayout2()
	view.symbolLabel = qt.NewQLabel(root)
	view.symbolLabel.SetStyleSheet("background: transparent; font-weight: 600;")
	view.nameLabel = qt.NewQLabel(root)
	view.nameLabel.SetStyleSheet(fmt.Sprintf("background: transparent; color: %s; font-size: %dpx;",
		cssRGB(constants.ColorAxis), int(constants.NameTextSize)))
	ident.AddWidget(view.symbolLabel.QWidget)
	ident.AddWidget(view.nameLabel.QWidget)
	head.AddLayout(ident.QLayout)
	// Extended-hours toggle, anchored left of the stretch so the varying
	// width of the price block never shifts it. Only visible on the 1D range
	// for symbols with pre/post trading; writes the shared setting — the
	// reload arrives via the config subscription, the same path the settings
	// dialog and external file edits use.
	view.extendedToggle = qt.NewQCheckBox4(constants.LabelExtendedToggle, root)
	view.extendedToggle.SetStyleSheet(checkBoxStyle())
	view.extendedToggle.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	view.extendedToggle.SetToolTip(constants.TipExtendedHours)
	view.extendedToggle.SetVisible(false)
	view.extendedToggle.OnClicked(func() {
		checked := view.extendedToggle.IsChecked()
		if err := view.store.Update(func(conf *config.Config) { conf.ExtendedHours = checked }); err != nil {
			slog.Error("saving extended-hours toggle failed", "error", err)
		}
	})
	head.AddWidget(view.extendedToggle.QWidget)
	head.AddStretch()
	quote := qt.NewQVBoxLayout2()
	view.priceLabel = qt.NewQLabel5(constants.PricePlaceholder, root)
	view.priceLabel.SetStyleSheet(priceBaseStyle)
	view.priceLabel.SetAlignment(qt.AlignRight)
	view.changeLabel = qt.NewQLabel(root)
	view.changeLabel.SetStyleSheet("background: transparent;")
	view.changeLabel.SetAlignment(qt.AlignRight)
	// Always present (empty when inactive) so showing/hiding the extended
	// price never changes the header height and resizes the chart.
	view.extendedLabel = qt.NewQLabel(root)
	view.extendedLabel.SetStyleSheet(extendedLabelStyle(constants.ColorNeutral))
	view.extendedLabel.SetAlignment(qt.AlignRight)
	quote.AddWidget(view.priceLabel.QWidget)
	quote.AddWidget(view.changeLabel.QWidget)
	quote.AddWidget(view.extendedLabel.QWidget)
	head.AddLayout(quote.QLayout)
	main.AddLayout(head.QLayout)

	// Range toggles.
	ranges := qt.NewQHBoxLayout2()
	for _, rangeOption := range model.Ranges {
		toggled := rangeOption // capture per iteration
		button := qt.NewQPushButton5(string(rangeOption), root)
		button.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
		button.OnClicked(func() { view.setRange(toggled) })
		view.rangeButtons[rangeOption] = button
		ranges.AddWidget(button.QWidget)
	}
	ranges.AddStretch()
	main.AddLayout(ranges.QLayout)

	view.chart = newChartWidget(root)
	view.chart.enableHoverReadout()
	main.AddWidget2(view.chart.QWidget, 1)

	outer.AddLayout2(main.QLayout, 1)

	view.flashTimer = qt.NewQTimer2(root.QObject)
	view.flashTimer.SetSingleShot(true)
	view.flashTimer.OnTimeout(func() {
		view.priceLabel.SetStyleSheet(priceBaseStyle)
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

// refresh is the periodic tick: sidebar tiles always (1D), the main chart
// only when it shows an intraday range.
func (view *detailView) refresh() {
	view.loadSidebar(true)
	if view.rng.Intraday() {
		view.loadMain(true)
	}
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

// setSymbols reconciles the sidebar with the tracked list.
func (view *detailView) setSymbols(symbols []string) {
	next := map[string]*tile{}
	for _, symbol := range symbols {
		if existing, ok := view.sidebarTiles[symbol]; ok {
			next[symbol] = existing
			continue
		}
		opened := symbol // capture per iteration for the click callback
		cell := newTile(view.sidebarContent, opened, true, func() {
			view.showSymbol(opened)
		})
		view.sidebarBox.InsertWidget(view.sidebarBox.Count()-1, cell.frame.QWidget) // before the stretch
		next[symbol] = cell
	}
	for symbol, cell := range view.sidebarTiles {
		if _, keep := next[symbol]; !keep {
			cell.frame.SetParent(nil)
			cell.frame.DeleteLater()
		}
	}
	view.order = append([]string{}, symbols...)
	view.sidebarTiles = next
	for tracked, cell := range next {
		cell.setSelected(tracked == view.symbol)
	}
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
				view.priceLabel.SetText(constants.PricePlaceholder)
				view.changeLabel.SetText(constants.MsgUnavailable)
				view.changeLabel.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorNeutral) + ";")
				view.extendedLabel.SetText("")
				view.chart.setSeries(model.Series{}, constants.ColorNeutral)
				view.priceShown = false
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
		view.changeLabel.SetStyleSheet("background: transparent; color: " + cssRGB(col) + ";")
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
		col, sign := changeStyle(change)
		view.changeLabel.SetText(fmt.Sprintf(constants.FmtPriceChange, sign, change, sign, percent))
		view.changeLabel.SetStyleSheet("background: transparent; color: " + cssRGB(col) + ";")
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
		col, sign := changeStyle(change)
		view.extendedLabel.SetText(fmt.Sprintf(constants.FmtExtendedQuote, prefix, display.ExtendedPrice, sign, percent))
		view.extendedLabel.SetStyleSheet(extendedLabelStyle(col))
	} else {
		view.extendedLabel.SetText("")
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
		flashColor := constants.ColorUp
		if last < view.shownPrice {
			flashColor = constants.ColorDown
		}
		view.priceLabel.SetStyleSheet(fmt.Sprintf("%s background-color: %s; border-radius: %dpx;",
			priceBaseStyle, cssRGBA(flashColor, constants.FlashAlpha), int(constants.FlashCornerRadius)))
		view.flashTimer.Start(int(constants.FlashDuration / time.Millisecond))
	}
	view.shownPrice = last
	view.priceShown = true
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
					cell.setFailed()
					return
				}
				cell.setSeries(series, flash)
			})
		}()
	}
}

// styleRangeButtons highlights the active range toggle.
func (view *detailView) styleRangeButtons() {
	for rangeOption, button := range view.rangeButtons {
		button.SetStyleSheet(toggleButtonStyle(rangeOption == view.rng))
	}
}

// extendedLabelStyle styles the separate pre/post price label; the label is
// permanently laid out (empty when inactive), so the font size stays fixed to
// keep the header height — and with it the chart size — stable.
func extendedLabelStyle(textColor color.NRGBA) string {
	return fmt.Sprintf("background: transparent; color: %s; font-size: %dpx;",
		cssRGB(textColor), int(constants.NameTextSize))
}

// toggleButtonStyle is the pill style of the range toggles, highlighted when
// selected.
func toggleButtonStyle(selected bool) string {
	background := constants.ColorCardBg
	if selected {
		background = constants.ColorSelected
	}
	return fmt.Sprintf(
		"QPushButton { background-color: %s; color: %s; border: none; border-radius: %dpx; padding: 4px 10px; }"+
			" QPushButton:hover { background-color: %s; }",
		cssRGB(background), cssRGB(constants.ColorForeground),
		int(constants.PanelCornerRadius), cssRGB(constants.ColorHover))
}
