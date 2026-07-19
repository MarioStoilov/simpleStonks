package qtui

import (
	"context"
	"fmt"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

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
	chart          *chartWidget
	rangeButtons   map[model.Range]*qt.QPushButton

	symbol     string
	rng        model.Range
	generation int // drops stale async responses after symbol/range switches

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
		quotes:       quotes,
		store:        store,
		onBack:       onBack,
		root:         root,
		rangeButtons: map[model.Range]*qt.QPushButton{},
		sidebarTiles: map[string]*tile{},
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
	head.AddStretch()
	quote := qt.NewQVBoxLayout2()
	view.priceLabel = qt.NewQLabel5(constants.PricePlaceholder, root)
	view.priceLabel.SetStyleSheet(priceBaseStyle)
	view.priceLabel.SetAlignment(qt.AlignRight)
	view.changeLabel = qt.NewQLabel(root)
	view.changeLabel.SetStyleSheet("background: transparent;")
	view.changeLabel.SetAlignment(qt.AlignRight)
	quote.AddWidget(view.priceLabel.QWidget)
	quote.AddWidget(view.changeLabel.QWidget)
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
	view.priceShown = false
	view.rng = view.store.Get().DefaultRange
	view.styleRangeButtons()
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

// loadMain fetches the main chart's series off the UI thread.
func (view *detailView) loadMain(flash bool) {
	symbol, rng, requestGen := view.symbol, view.rng, view.generation
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), constants.FetchTimeout)
		defer cancel()
		series, err := view.quotes.History(ctx, symbol, rng)
		mainthread.Wait(func() {
			if requestGen != view.generation { // superseded by a newer switch
				return
			}
			if err != nil || len(series.Candles) == 0 {
				view.priceLabel.SetText(constants.PricePlaceholder)
				view.changeLabel.SetText(constants.MsgUnavailable)
				view.changeLabel.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorNeutral) + ";")
				view.chart.setSeries(model.Series{}, constants.ColorNeutral)
				view.priceShown = false
				return
			}
			view.applyMain(series, flash)
		})
	}()
}

// applyMain renders a fetched series into the header and chart.
func (view *detailView) applyMain(series model.Series, flash bool) {
	view.nameLabel.SetText(series.Name)
	last := series.Candles[len(series.Candles)-1].Close
	view.priceLabel.SetText(chartPrice(last))

	lineColor := constants.ColorNeutral
	if series.PreviousClose > 0 {
		change := last - series.PreviousClose
		percent := change / series.PreviousClose * constants.PercentMax
		col, sign := changeStyle(change)
		lineColor = col
		view.changeLabel.SetText(fmt.Sprintf(constants.FmtPriceChange, sign, change, sign, percent))
		view.changeLabel.SetStyleSheet("background: transparent; color: " + cssRGB(col) + ";")
	} else {
		view.changeLabel.SetText("")
	}
	view.chart.setSeries(series, lineColor)

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
		background := constants.ColorCardBg
		if rangeOption == view.rng {
			background = constants.ColorSelected
		}
		button.SetStyleSheet(fmt.Sprintf(
			"QPushButton { background-color: %s; color: %s; border: none; border-radius: %dpx; padding: 4px 10px; }"+
				" QPushButton:hover { background-color: %s; }",
			cssRGB(background), cssRGB(constants.ColorForeground),
			int(constants.PanelCornerRadius), cssRGB(constants.ColorHover)))
	}
}
