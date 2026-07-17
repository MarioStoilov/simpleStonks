package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// buildDetail builds the detail screen: a left sidebar of all symbols (current
// one highlighted) and a main area with the expanded chart, price/change header,
// and the range toggles.
func (a *App) buildDetail() fyne.CanvasObject {
	a.rng = a.cfg.DefaultRange

	side := make([]*tile, 0, len(a.cfg.Symbols))
	sideObjs := make([]fyne.CanvasObject, 0, len(a.cfg.Symbols))
	for _, sym := range a.cfg.Symbols {
		sym := sym
		t := newTile(sym, false, func() { a.showDetail(sym) }, nil)
		if sym == a.selected {
			t.SetSelected(true)
		}
		side = append(side, t)
		sideObjs = append(sideObjs, t)
	}
	a.sideTiles = side
	sidebar := container.NewVScroll(container.NewVBox(sideObjs...))
	sidebar.SetMinSize(fyne.NewSize(190, 0))

	a.detailChart = newChart()
	a.detailPrice = canvas.NewText("—", theme.Color(theme.ColorNameForeground))
	a.detailPrice.TextStyle = fyne.TextStyle{Bold: true}
	a.detailPrice.Alignment = fyne.TextAlignTrailing
	a.detailChange = canvas.NewText("", colorNeutral)

	back := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { a.showHome() })
	title := widget.NewLabelWithStyle(a.selected, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewBorder(nil, nil,
		container.NewHBox(back, title),
		container.NewHBox(a.detailPrice, a.detailChange),
	)
	top := container.NewVBox(header, a.buildRangeToggles())
	main := container.NewBorder(top, nil, nil, nil, a.detailChart)

	return container.NewBorder(nil, nil, sidebar, nil, main)
}

// buildRangeToggles builds the 1D..ALL range buttons, highlighting the active one.
func (a *App) buildRangeToggles() fyne.CanvasObject {
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
	return container.NewHScroll(box)
}

// setRange switches the detail chart's range, restyles the toggles, reloads the
// main chart, and restarts the refresh loop (whose behavior depends on whether
// the new range is intraday).
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
	a.stopRefresh()
	a.loadMain()
	a.startRefresh(a.detailTick())
}

// loadDetail kicks off the initial fetches for the detail screen: the main chart
// at the current range and each sidebar cell at 1D.
func (a *App) loadDetail() {
	a.loadMain()
	for _, t := range a.sideTiles {
		loadTile1D(a.provider, t)
	}
}

// detailTick returns the periodic refresh for the detail screen: sidebar cells
// (1D) always, and the main chart only when its range is intraday.
func (a *App) detailTick() func() {
	prov := a.provider
	side := a.sideTiles
	intraday := a.rng.Intraday()
	chart := a.detailChart
	price := a.detailPrice
	change := a.detailChange
	symbol := a.selected
	rng := a.rng
	return func() {
		for _, t := range side {
			loadTile1D(prov, t)
		}
		if intraday {
			loadMainChart(prov, chart, price, change, symbol, rng)
		}
	}
}

// loadMain reloads the main chart for the current selection and range.
func (a *App) loadMain() {
	loadMainChart(a.provider, a.detailChart, a.detailPrice, a.detailChange, a.selected, a.rng)
}

// loadMainChart fetches a symbol's history at a range and applies it to the main
// chart and header on the UI goroutine.
func loadMainChart(prov provider.QuoteProvider, chart *chart, price, change *canvas.Text, symbol string, rng model.Range) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		s, err := prov.History(ctx, symbol, rng)
		fyne.Do(func() {
			if err != nil || len(s.Candles) == 0 {
				price.Text = "—"
				price.Refresh()
				change.Text = "unavailable"
				change.Color = colorNeutral
				change.Refresh()
				chart.SetColor(colorNeutral)
				chart.SetSeries(model.Series{})
				return
			}
			last := s.Candles[len(s.Candles)-1].Close
			col, text := priceChangeText(last, s.PreviousClose)
			price.Text = fmt.Sprintf("%.2f", last)
			price.Refresh()
			change.Text = text
			change.Color = col
			change.Refresh()
			chart.SetColor(col)
			chart.SetSeries(s)
		})
	}()
}
