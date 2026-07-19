package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// buildHome builds the home screen: a top bar plus a grid of 1D-only cells, one
// per tracked symbol. In view mode, clicking a cell opens its detail view; in
// edit mode, cells expose reorder/remove controls instead of navigating.
func (app *App) buildHome() fyne.CanvasObject {
	tiles := make([]*tile, 0, len(app.cfg.Symbols))
	objs := make([]fyne.CanvasObject, 0, len(app.cfg.Symbols))
	for idx, symbol := range app.cfg.Symbols {
		idx, symbol := idx, symbol
		if app.editMode {
			cell := newTile(symbol, true, nil, nil) // no navigation while editing
			tiles = append(tiles, cell)
			objs = append(objs, container.NewBorder(nil, app.editControls(idx, symbol), nil, nil, cell))
		} else {
			cell := newTile(symbol, true, func() { app.showDetail(symbol) }, nil)
			tiles = append(tiles, cell)
			objs = append(objs, cell)
		}
	}
	app.homeTiles = tiles

	cellSize := fyne.NewSize(constants.GridCellWidth, constants.GridCellHeight)
	if app.editMode {
		cellSize = fyne.NewSize(constants.GridCellWidth, constants.GridCellEditHeight) // room for the control row
	}
	grid := container.New(layout.NewGridWrapLayout(cellSize), objs...)
	return container.NewBorder(app.buildTopBar(), nil, nil, nil, container.NewVScroll(grid))
}

// buildTopBar shows the mode-appropriate actions: Edit + settings in view mode;
// Add (live search) + Done in edit mode.
func (app *App) buildTopBar() fyne.CanvasObject {
	var right *fyne.Container
	if app.editMode {
		add := widget.NewButtonWithIcon(constants.LabelAdd, theme.ContentAddIcon(), func() { app.showSearchDialog() })
		add.Importance = widget.HighImportance
		done := widget.NewButtonWithIcon(constants.LabelDone, theme.ConfirmIcon(), func() { app.setEditMode(false) })
		right = container.NewHBox(add, done)
	} else {
		edit := widget.NewButtonWithIcon(constants.LabelEdit, theme.DocumentCreateIcon(), func() { app.setEditMode(true) })
		info := widget.NewButtonWithIcon("", theme.InfoIcon(), func() { app.showAboutDialog() })
		settings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() { app.showSettingsWindow() })
		right = container.NewHBox(edit, info, settings)
	}
	return container.NewBorder(nil, nil, nil, right)
}

// editControls is the per-cell reorder/remove row shown in edit mode.
func (app *App) editControls(idx int, symbol string) fyne.CanvasObject {
	upBtn := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() { app.moveSymbol(idx, -1) })
	if idx == 0 {
		upBtn.Disable()
	}
	downBtn := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() { app.moveSymbol(idx, +1) })
	if idx == len(app.cfg.Symbols)-1 {
		downBtn.Disable()
	}
	remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() { app.removeSymbol(symbol) })
	remove.Importance = widget.DangerImportance
	return container.NewHBox(upBtn, downBtn, layout.NewSpacer(), remove)
}

// loadHome kicks off the initial 1D fetch for every home tile.
func (app *App) loadHome() {
	for _, cell := range app.homeTiles {
		loadTile1D(app.provider, cell)
	}
}

// homeTick returns the periodic refresh for the home screen (all cells are 1D).
func (app *App) homeTick() func() {
	prov := app.provider
	tiles := app.homeTiles
	return func() {
		for _, cell := range tiles {
			loadTile1D(prov, cell)
		}
	}
}
