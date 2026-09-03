package qtui

import (
	"context"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// homeView is the scrollable grid of tracked-symbol tiles at the 1D range,
// under a bar with the Edit (reorder/remove) toggle and the Add search. Tiles
// reflow to the available width, reload when the tracked list changes, and
// refresh (with price flashes) on the periodic tick.
type homeView struct {
	quotes         provider.QuoteProvider
	store          *config.Store
	onOpen         func(symbol string) // opens the detail view
	onOpenSettings func()

	// onQuote reports every tile fetch outcome (latest close on success) so
	// the app can track connectivity and check the pending price alerts.
	onQuote func(symbol string, price float64, ok bool)

	root       *qt.QWidget
	editButton *qt.QPushButton
	scroll     *qt.QScrollArea
	content    *qt.QWidget
	grid       *qt.QGridLayout

	order   []string
	tiles   map[string]*tile
	columns int
	editing bool
}

// newHomeView loads the home form (home.ui); symbols arrive via setSymbols.
func newHomeView(parent *qt.QWidget, quotes provider.QuoteProvider, store *config.Store) *homeView {
	loaded := loadForm(homeForm)
	loaded.root.SetParent(parent)

	home := &homeView{
		quotes:     quotes,
		store:      store,
		root:       loaded.root,
		editButton: loaded.button("editButton"),
		scroll:     loaded.scrollArea("scroll"),
		content:    loaded.widget("gridContent"),
		grid:       loaded.grid("grid"),
		tiles:      map[string]*tile{},
		columns:    1,
	}

	home.editButton.OnClicked(func() { home.toggleEditing() })
	loaded.button("addButton").OnClicked(func() {
		showSearchDialog(home.root, home.quotes, home.store)
	})
	loaded.button("settingsButton").OnClicked(func() {
		if home.onOpenSettings != nil {
			home.onOpenSettings()
		}
	})
	loaded.button("aboutButton").OnClicked(func() {
		showAboutDialog(home.root)
	})

	// The scroll area owns the grid content: the loader parents it but
	// leaves the viewport assignment to us.
	home.scroll.SetWidget(home.content)
	// SetWidget turns on the content's autoFillBackground, which paints the
	// palette color over the translucent window; the theme styles it.
	home.content.SetAutoFillBackground(false)
	home.scroll.OnResizeEvent(func(super func(event *qt.QResizeEvent), event *qt.QResizeEvent) {
		super(event)
		home.reflow()
	})
	return home
}

// toggleEditing flips edit mode: tiles show reorder/remove controls and stop
// opening the detail view.
func (home *homeView) toggleEditing() {
	home.editing = !home.editing
	if home.editing {
		home.editButton.SetText(labelDone)
	} else {
		home.editButton.SetText(labelEdit)
	}
	home.applyEditState()
}

// applyEditState pushes the edit flag and reorder bounds to every tile.
func (home *homeView) applyEditState() {
	for idx, symbol := range home.order {
		cell, ok := home.tiles[symbol]
		if !ok {
			continue
		}
		cell.setEditing(home.editing)
		cell.setMoveBounds(idx > 0, idx < len(home.order)-1)
	}
}

// indexOf returns a symbol's position in the tracked order, or -1.
func (home *homeView) indexOf(symbol string) int {
	for idx, tracked := range home.order {
		if tracked == symbol {
			return idx
		}
	}
	return -1
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
		opened := symbol // capture per iteration for the callbacks
		cell := newTile(home.content, opened, false, func() {
			if !home.editing && home.onOpen != nil {
				home.onOpen(opened)
			}
		})
		cell.onMoveLeft = func() { moveSymbol(home.store, home.indexOf(opened), -1) }
		cell.onMoveRight = func() { moveSymbol(home.store, home.indexOf(opened), 1) }
		cell.onRemove = func() { removeSymbol(home.store, opened) }
		next[symbol] = cell
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

// repaintCharts repaints every tile chart (the chart styling changed).
func (home *homeView) repaintCharts() {
	for _, cell := range home.tiles {
		cell.chart.Update()
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
				// Keep showing the last data (e.g. network loss); only a
				// tile that never had any shows the failure state.
				if !cell.priceShown {
					cell.setFailed()
				}
				if home.onQuote != nil {
					home.onQuote(symbol, 0, false)
				}
				return
			}
			cell.setSeries(series, flash)
			if home.onQuote != nil {
				home.onQuote(symbol, series.Candles[len(series.Candles)-1].Close, true)
			}
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

// relayout re-places all tiles into the grid in tracked order and refreshes
// the edit-mode state.
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
	home.applyEditState()
}

// The Edit button's two labels; the form supplies the resting one.
const (
	labelEdit = "Edit"
	labelDone = "Done"
)
