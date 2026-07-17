package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// buildGridView renders one mini-chart tile per tracked symbol in a grid, with
// a toolbar for range toggles and symbol management above it.
//
// TODO(v1): populate tiles with live data from a.provider, wire the range
// toggles, and refresh the 1D view on a.cfg.RefreshInterval.
func (a *App) buildGridView() fyne.CanvasObject {
	tiles := make([]fyne.CanvasObject, 0, len(a.cfg.Symbols))
	for _, sym := range a.cfg.Symbols {
		tiles = append(tiles, a.buildTile(sym))
	}

	grid := container.New(layout.NewGridWrapLayout(fyne.NewSize(260, 200)), tiles...)
	return container.NewBorder(a.buildToolbar(), nil, nil, nil, container.NewVScroll(grid))
}

// buildToolbar builds the range toggles and the add-symbol control.
//
// TODO(v1): make the range buttons switch a.cfg.DefaultRange and refresh tiles;
// wire "＋" to the add-symbol dialog in settings.go.
func (a *App) buildToolbar() fyne.CanvasObject {
	ranges := container.NewHBox()
	for _, r := range rangeButtons() {
		ranges.Add(r)
	}
	add := widget.NewButtonWithIcon("", nil, func() { a.showAddSymbolDialog() })
	add.SetText("＋")
	return container.NewBorder(nil, nil, nil, add, ranges)
}

// buildTile builds a single symbol tile: header (symbol + change) over a chart.
//
// TODO(v1): show real price/percent change with up/down coloring and a real
// chart via newChart.
func (a *App) buildTile(symbol string) fyne.CanvasObject {
	header := widget.NewLabel(symbol)
	chart := newChart()
	return widget.NewCard("", "", container.NewBorder(header, nil, nil, nil, chart))
}

// rangeButtons builds the (currently inert) range toggle buttons.
func rangeButtons() []fyne.CanvasObject {
	var out []fyne.CanvasObject
	for _, r := range rangeLabels() {
		out = append(out, widget.NewButton(r, func() {}))
	}
	return out
}

// rangeLabels returns the range toggle labels in display order.
func rangeLabels() []string {
	out := make([]string, 0, len(model.Ranges))
	for _, r := range model.Ranges {
		out = append(out, string(r))
	}
	return out
}
