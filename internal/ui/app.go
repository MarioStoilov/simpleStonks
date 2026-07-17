// Package ui contains the Fyne front end for simpleStonks.
//
// The UI is structured so presentation modes are pluggable: the layout
// (grid vs. list+detail) and form factor (normal window vs. always-on-top
// widget) are chosen from config rather than hardcoded. v1 implements the grid
// layout in a normal window; the other modes have seams reserved here and are
// filled in later.
package ui

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// appID is the Fyne application ID; it also namespaces Fyne's own storage.
const appID = "com.github.mariostoilov.simplestonks"

// App wires together the Fyne application, the loaded config, and the data
// provider.
type App struct {
	fyne     fyne.App
	win      fyne.Window
	provider provider.QuoteProvider
	cfg      config.Config
}

// New constructs the application with the given data provider and config.
func New(p provider.QuoteProvider, cfg config.Config) *App {
	return &App{
		fyne:     fyneapp.NewWithID(appID),
		provider: p,
		cfg:      cfg,
	}
}

// Run builds the main window according to config and starts the event loop.
// It blocks until the window is closed.
func (a *App) Run() {
	a.win = a.fyne.NewWindow("simpleStonks")
	a.win.SetContent(a.buildContent())
	a.win.Resize(fyne.NewSize(900, 600))
	a.win.ShowAndRun()
}

// buildContent selects the presentation for the configured layout. Only the
// grid layout is implemented in v1; other layouts fall back to it for now.
func (a *App) buildContent() fyne.CanvasObject {
	switch a.cfg.Layout {
	case config.LayoutListDetail:
		// TODO(v2): list + detail layout. Falls back to grid until implemented.
		return a.buildGridView()
	default:
		return a.buildGridView()
	}
}
