package qtui

import (
	"context"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// homeView is the scrollable grid of tracked-symbol tiles at the 1D range.
// Tiles reflow to the available width, reload when the tracked list changes,
// and refresh (with price flashes) on the periodic tick.
type homeView struct {
	quotes provider.QuoteProvider
	onOpen func(symbol string) // opens the detail view

	scroll  *qt.QScrollArea
	content *qt.QWidget
	grid    *qt.QGridLayout

	order   []string
	tiles   map[string]*tile
	columns int
}

// newHomeView builds an empty grid; symbols arrive via setSymbols.
func newHomeView(parent *qt.QWidget, quotes provider.QuoteProvider) *homeView {
	scroll := qt.NewQScrollArea(parent)
	scroll.SetWidgetResizable(true)
	scroll.SetStyleSheet(scrollAreaStyle())

	content := qt.NewQWidget2()
	content.SetStyleSheet("background: transparent;")
	grid := qt.NewQGridLayout(content)
	scroll.SetWidget(content)

	home := &homeView{
		quotes:  quotes,
		scroll:  scroll,
		content: content,
		grid:    grid,
		tiles:   map[string]*tile{},
		columns: 1,
	}
	scroll.OnResizeEvent(func(super func(event *qt.QResizeEvent), event *qt.QResizeEvent) {
		super(event)
		home.reflow()
	})
	return home
}

// setSymbols reconciles the tile set with the tracked list and reloads data
// for any newly added symbols.
func (home *homeView) setSymbols(symbols []string) {
	next := map[string]*tile{}
	var added []string
	for _, symbol := range symbols {
		if existing, ok := home.tiles[symbol]; ok {
			next[symbol] = existing
			continue
		}
		opened := symbol // capture per iteration for the click callback
		next[symbol] = newTile(home.content, symbol, false, func() {
			if home.onOpen != nil {
				home.onOpen(opened)
			}
		})
		added = append(added, symbol)
	}
	for symbol, cell := range home.tiles {
		if _, keep := next[symbol]; !keep {
			cell.frame.SetParent(nil)
			cell.frame.DeleteLater()
		}
	}
	home.order = append([]string{}, symbols...)
	home.tiles = next
	home.relayout()
	for _, symbol := range added {
		home.loadSymbol(symbol, false)
	}
}

// loadAll refreshes every tile; flash enables the price-change flash.
func (home *homeView) loadAll(flash bool) {
	for _, symbol := range home.order {
		home.loadSymbol(symbol, flash)
	}
}

// loadSymbol fetches the 1D series off the UI thread and applies the result
// on it.
func (home *homeView) loadSymbol(symbol string, flash bool) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), constants.FetchTimeout)
		defer cancel()
		series, err := home.quotes.History(ctx, symbol, model.Range1D)
		mainthread.Wait(func() {
			cell, ok := home.tiles[symbol]
			if !ok { // removed while the fetch was in flight
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

// reflow recomputes the column count for the current width and relays the
// grid when it changes.
func (home *homeView) reflow() {
	columns := int(float32(home.scroll.Width()) / constants.GridCellWidth)
	if columns < 1 {
		columns = 1
	}
	if columns == home.columns {
		return
	}
	home.columns = columns
	home.relayout()
}

// relayout re-places all tiles into the grid in tracked order.
func (home *homeView) relayout() {
	for home.grid.Count() > 0 {
		item := home.grid.TakeAt(0)
		item.Delete()
	}
	for idx, symbol := range home.order {
		cell, ok := home.tiles[symbol]
		if !ok {
			continue
		}
		home.grid.AddWidget2(cell.frame.QWidget, idx/home.columns, idx%home.columns)
	}
}
