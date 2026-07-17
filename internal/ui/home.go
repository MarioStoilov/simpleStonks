package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// buildHome builds the home screen: a top bar plus a grid of 1D-only cells, one
// per tracked symbol. In view mode, clicking a cell opens its detail view; in
// edit mode, cells expose reorder/remove controls instead of navigating.
func (a *App) buildHome() fyne.CanvasObject {
	tiles := make([]*tile, 0, len(a.cfg.Symbols))
	objs := make([]fyne.CanvasObject, 0, len(a.cfg.Symbols))
	for i, sym := range a.cfg.Symbols {
		i, sym := i, sym
		if a.editMode {
			t := newTile(sym, true, nil, nil) // no navigation while editing
			tiles = append(tiles, t)
			objs = append(objs, container.NewBorder(nil, a.editControls(i, sym), nil, nil, t))
		} else {
			t := newTile(sym, true, func() { a.showDetail(sym) }, nil)
			tiles = append(tiles, t)
			objs = append(objs, t)
		}
	}
	a.homeTiles = tiles

	cellSize := fyne.NewSize(300, 240)
	if a.editMode {
		cellSize = fyne.NewSize(300, 280) // room for the control row
	}
	grid := container.New(layout.NewGridWrapLayout(cellSize), objs...)
	return container.NewBorder(a.buildTopBar(), nil, nil, nil, container.NewVScroll(grid))
}

// buildTopBar shows the mode-appropriate actions: Edit + settings in view mode;
// Add (live search) + Done in edit mode.
func (a *App) buildTopBar() fyne.CanvasObject {
	var right *fyne.Container
	if a.editMode {
		add := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() { a.showSearchDialog() })
		add.Importance = widget.HighImportance
		done := widget.NewButtonWithIcon("Done", theme.ConfirmIcon(), func() { a.setEditMode(false) })
		right = container.NewHBox(add, done)
	} else {
		edit := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() { a.setEditMode(true) })
		settings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() { a.showSettingsWindow() })
		right = container.NewHBox(edit, settings)
	}
	return container.NewBorder(nil, nil, nil, right)
}

// editControls is the per-cell reorder/remove row shown in edit mode.
func (a *App) editControls(i int, symbol string) fyne.CanvasObject {
	up := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() { a.moveSymbol(i, -1) })
	if i == 0 {
		up.Disable()
	}
	down := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() { a.moveSymbol(i, +1) })
	if i == len(a.cfg.Symbols)-1 {
		down.Disable()
	}
	remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() { a.removeSymbol(symbol) })
	remove.Importance = widget.DangerImportance
	return container.NewHBox(up, down, layout.NewSpacer(), remove)
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
