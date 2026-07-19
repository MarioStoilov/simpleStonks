package ui

import (
	"context"

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
func (app *App) buildDetail() fyne.CanvasObject {
	app.rng = app.cfg.DefaultRange

	side := make([]*tile, 0, len(app.cfg.Symbols))
	sideObjs := make([]fyne.CanvasObject, 0, len(app.cfg.Symbols))
	for _, symbol := range app.cfg.Symbols {
		symbol := symbol
		cell := newTile(symbol, false, func() { app.showDetail(symbol) }, nil)
		if symbol == app.selected {
			cell.SetSelected(true)
		}
		side = append(side, cell)
		sideObjs = append(sideObjs, cell)
	}
	app.sideTiles = side
	sidebar := container.NewVScroll(container.NewVBox(sideObjs...))
	sidebar.SetMinSize(fyne.NewSize(190, 0))

	app.detailChart = newChart()
	app.detailChart.hoverReadout = true // price/time readout only on the big chart
	app.detailName = canvas.NewText("", colorAxis)
	app.detailName.TextSize = nameTextSize
	app.detailPrice = newPriceText()
	app.detailChange = canvas.NewText("", colorNeutral)

	back := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { app.showHome() })
	title := widget.NewLabelWithStyle(app.selected, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewBorder(nil, nil,
		container.NewHBox(back, container.NewVBox(title, app.detailName)),
		container.NewHBox(app.detailPrice, app.detailChange),
	)
	top := container.NewVBox(header, app.buildRangeToggles())
	main := container.NewBorder(top, nil, nil, nil, app.detailChart)

	return container.NewBorder(nil, nil, sidebar, nil, main)
}

// buildRangeToggles builds the 1D..ALL range buttons, highlighting the active one.
func (app *App) buildRangeToggles() fyne.CanvasObject {
	app.rangeBtns = make(map[model.Range]*widget.Button, len(model.Ranges))
	box := container.NewHBox()
	for _, rng := range model.Ranges {
		rng := rng
		btn := widget.NewButton(string(rng), func() { app.setRange(rng) })
		if rng == app.rng {
			btn.Importance = widget.HighImportance
		}
		app.rangeBtns[rng] = btn
		box.Add(btn)
	}
	return container.NewHScroll(box)
}

// setRange switches the detail chart's range, restyles the toggles, reloads the
// main chart, and restarts the refresh loop (whose behavior depends on whether
// the new range is intraday).
func (app *App) setRange(rng model.Range) {
	if app.rng == rng {
		return
	}
	app.rng = rng
	for btnRange, btn := range app.rangeBtns {
		if btnRange == rng {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.MediumImportance
		}
		btn.Refresh()
	}
	app.stopRefresh()
	app.loadMain()
	app.startRefresh(app.detailTick())
}

// loadDetail kicks off the initial fetches for the detail screen: the main chart
// at the current range and each sidebar cell at 1D.
func (app *App) loadDetail() {
	app.loadMain()
	for _, cell := range app.sideTiles {
		loadTile1D(app.provider, cell)
	}
}

// detailTick returns the periodic refresh for the detail screen: sidebar cells
// (1D) always, and the main chart only when its range is intraday.
func (app *App) detailTick() func() {
	prov := app.provider
	side := app.sideTiles
	intraday := app.rng.Intraday()
	chart := app.detailChart
	name := app.detailName
	price := app.detailPrice
	change := app.detailChange
	symbol := app.selected
	rng := app.rng
	return func() {
		for _, cell := range side {
			loadTile1D(prov, cell)
		}
		if intraday {
			loadMainChart(prov, chart, name, price, change, symbol, rng, true)
		}
	}
}

// loadMain reloads the main chart for the current selection and range.
func (app *App) loadMain() {
	loadMainChart(app.provider, app.detailChart, app.detailName, app.detailPrice, app.detailChange, app.selected, app.rng, false)
}

// loadMainChart fetches a symbol's history at a range and applies it to the
// main chart and header on the UI goroutine. name is optional (nil skips it);
// flash marks a live refresh, letting a changed price flash its background.
func loadMainChart(prov provider.QuoteProvider, chart *chart, name *canvas.Text, price *priceText, change *canvas.Text, symbol string, rng model.Range, flash bool) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		series, err := prov.History(ctx, symbol, rng)
		fyne.Do(func() {
			if err != nil || len(series.Candles) == 0 {
				price.SetUnavailable()
				change.Text = "unavailable"
				change.Color = colorNeutral
				change.Refresh()
				chart.SetColor(colorNeutral)
				chart.SetSeries(model.Series{})
				return
			}
			last := series.Candles[len(series.Candles)-1].Close
			col, text := priceChangeText(last, series.PreviousClose)
			if name != nil && series.Name != "" && series.Name != name.Text {
				name.Text = series.Name
				name.Refresh()
			}
			price.SetPrice(last, flash)
			change.Text = text
			change.Color = col
			change.Refresh()
			chart.SetColor(col)
			chart.SetSeries(series)
		})
	}()
}
