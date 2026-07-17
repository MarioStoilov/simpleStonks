package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// buildHome builds the home screen: a top bar plus a grid of 1D-only cells, one
// per tracked symbol. Clicking a cell opens that symbol's detail view.
func (a *App) buildHome() fyne.CanvasObject {
	tiles := make([]*tile, 0, len(a.cfg.Symbols))
	objs := make([]fyne.CanvasObject, 0, len(a.cfg.Symbols))
	for _, sym := range a.cfg.Symbols {
		sym := sym
		t := newTile(sym, true, func() { a.showDetail(sym) }, func() { a.removeSymbol(sym) })
		tiles = append(tiles, t)
		objs = append(objs, t)
	}
	a.homeTiles = tiles

	grid := container.New(layout.NewGridWrapLayout(fyne.NewSize(300, 240)), objs...)
	return container.NewBorder(a.buildTopBar(), nil, nil, nil, container.NewVScroll(grid))
}

// buildTopBar is the home title with an Add button and a settings cog on the
// right. The full edit-mode flow and the settings window are separate, pending
// work; for now Add opens a simple dialog and the cog is a placeholder.
func (a *App) buildTopBar() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("simpleStonks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	add := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() { a.showAddSymbolDialog() })
	settings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() { a.showSettingsPlaceholder() })
	return container.NewBorder(nil, nil, title, container.NewHBox(add, settings))
}

// loadHome kicks off the initial 1D fetch for every home tile.
func (a *App) loadHome() {
	for _, t := range a.homeTiles {
		loadTile1D(a.provider, t)
	}
}

// homeTick returns the periodic refresh for the home screen (all cells are 1D).
func (a *App) homeTick() func() {
	prov := a.provider
	tiles := a.homeTiles
	return func() {
		for _, t := range tiles {
			loadTile1D(prov, t)
		}
	}
}
