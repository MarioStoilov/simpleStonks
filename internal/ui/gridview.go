package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// fetchTimeout bounds a single provider request.
const fetchTimeout = 15 * time.Second

// tile is one symbol's card: symbol, latest price, change, and a mini chart.
type tile struct {
	symbol string
	card   *widget.Card
	price  *canvas.Text
	change *canvas.Text
	chart  *chart
}

// newTile builds a tile for a symbol; onRemove is invoked by its remove button.
func newTile(symbol string, onRemove func()) *tile {
	sym := widget.NewLabelWithStyle(symbol, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	price := canvas.NewText("—", theme.Color(theme.ColorNameForeground))
	price.TextStyle = fyne.TextStyle{Bold: true}
	price.Alignment = fyne.TextAlignTrailing

	change := canvas.NewText("", colorNeutral)

	remove := widget.NewButtonWithIcon("", theme.ContentClearIcon(), onRemove)
	remove.Importance = widget.LowImportance

	ch := newChart()

	header := container.NewBorder(nil, nil, sym, container.NewHBox(price, remove))
	top := container.NewVBox(header, change)
	content := container.NewBorder(top, nil, nil, nil, ch)

	return &tile{
		symbol: symbol,
		card:   widget.NewCard("", "", content),
		price:  price,
		change: change,
		chart:  ch,
	}
}

// setSeries renders a fetched series: latest price, absolute and percent change
// versus the range's reference close, with up/down coloring.
func (tl *tile) setSeries(s model.Series) {
	if len(s.Candles) == 0 {
		tl.setError(fmt.Errorf("no data"))
		return
	}
	last := s.Candles[len(s.Candles)-1].Close
	prev := s.PreviousClose
	delta := last - prev
	pct := 0.0
	if prev != 0 {
		pct = delta / prev * 100
	}

	col := colorNeutral
	sign := ""
	switch {
	case delta > 0:
		col, sign = colorUp, "+"
	case delta < 0:
		col = colorDown
	}

	tl.price.Text = fmt.Sprintf("%.2f", last)
	tl.price.Refresh()
	tl.change.Text = fmt.Sprintf("%s%.2f (%s%.2f%%)", sign, delta, sign, pct)
	tl.change.Color = col
	tl.change.Refresh()
	tl.chart.SetColor(col)
	tl.chart.SetSeries(s)
}

// setError puts the tile into an unavailable state and logs the cause.
func (tl *tile) setError(err error) {
	tl.price.Text = "—"
	tl.price.Refresh()
	tl.change.Text = "unavailable"
	tl.change.Color = colorNeutral
	tl.change.Refresh()
	tl.chart.SetColor(colorNeutral)
	tl.chart.SetSeries(model.Series{})
	slog.Warn("tile update failed", "symbol", tl.symbol, "err", err)
}

// buildGridView builds the toolbar and one tile per tracked symbol. Data loading
// is kicked off separately by startData once the app is running.
func (a *App) buildGridView() fyne.CanvasObject {
	a.rng = a.cfg.DefaultRange

	tiles := make([]*tile, 0, len(a.cfg.Symbols))
	objs := make([]fyne.CanvasObject, 0, len(a.cfg.Symbols))
	for _, sym := range a.cfg.Symbols {
		sym := sym
		tl := newTile(sym, func() { a.removeSymbol(sym) })
		tiles = append(tiles, tl)
		objs = append(objs, tl.card)
	}
	a.tiles = tiles

	grid := container.New(layout.NewGridWrapLayout(fyne.NewSize(300, 240)), objs...)
	return container.NewBorder(a.buildToolbar(), nil, nil, nil, container.NewVScroll(grid))
}

// buildToolbar builds the range toggles and the add-symbol button.
func (a *App) buildToolbar() fyne.CanvasObject {
	a.rangeBtns = make(map[model.Range]*widget.Button, len(model.Ranges))
	box := container.NewHBox()
	for _, r := range model.Ranges {
		r := r
		b := widget.NewButton(string(r), func() { a.setRange(r) })
		if r == a.rng {
			b.Importance = widget.HighImportance
		}
		a.rangeBtns[r] = b
		box.Add(b)
	}
	add := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() { a.showAddSymbolDialog() })
	return container.NewBorder(nil, nil, nil, add, container.NewHScroll(box))
}

// setRange switches the active range, updates the toggle highlight, and reloads.
func (a *App) setRange(r model.Range) {
	if a.rng == r {
		return
	}
	a.rng = r
	for rr, b := range a.rangeBtns {
		if rr == r {
			b.Importance = widget.HighImportance
		} else {
			b.Importance = widget.MediumImportance
		}
		b.Refresh()
	}
	a.startData()
}

// startData (re)starts loading and the refresh loop for the current tiles and
// range. Must be called on the UI goroutine.
func (a *App) startData() {
	a.stopRefresh()
	a.load(a.tiles, a.rng)
	a.startRefresh()
}

// load fetches each tile's history concurrently and applies the result on the
// UI goroutine.
func (a *App) load(tiles []*tile, rng model.Range) {
	prov := a.provider
	for _, tl := range tiles {
		tl := tl
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
			defer cancel()
			s, err := prov.History(ctx, tl.symbol, rng)
			fyne.Do(func() {
				if err != nil {
					tl.setError(err)
					return
				}
				tl.setSeries(s)
			})
		}()
	}
}

// startRefresh starts a polling loop for the live 1D view. Non-intraday ranges
// are static, so no loop is started for them.
func (a *App) startRefresh() {
	if !a.rng.Intraday() {
		return
	}
	interval := a.cfg.RefreshInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	tiles := a.tiles
	rng := a.rng
	stop := make(chan struct{})
	a.stopCh = stop
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				a.load(tiles, rng)
			}
		}
	}()
}

// stopRefresh stops the polling loop if one is running.
func (a *App) stopRefresh() {
	if a.stopCh != nil {
		close(a.stopCh)
		a.stopCh = nil
	}
}
