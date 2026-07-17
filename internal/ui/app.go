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
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// appID is the Fyne application ID; it also namespaces Fyne's own storage.
const appID = "com.github.mariostoilov.simplestonks"

// App wires together the Fyne application, the live config store, and the data
// provider.
type App struct {
	fyne     fyne.App
	win      fyne.Window
	provider provider.QuoteProvider
	store    *config.Store
	cfg      config.Config // cached snapshot of store.Get()

	// View state (UI goroutine only).
	rng       model.Range
	tiles     []*tile
	rangeBtns map[model.Range]*widget.Button
	stopCh    chan struct{} // closes to stop the current refresh loop
}

// New constructs the application with the given data provider and config store.
func New(p provider.QuoteProvider, store *config.Store) *App {
	// Declare that the app follows Fyne's fyne.Do threading model: every
	// cross-goroutine UI update is marshalled onto the UI thread via fyne.Do.
	// This is a truthful declaration (see load/startData), not a silencer.
	fyneapp.SetMetadata(fyne.AppMetadata{
		ID:         appID,
		Name:       "simpleStonks",
		Migrations: map[string]bool{"fyneDo": true},
	})
	return &App{
		fyne:     fyneapp.NewWithID(appID),
		provider: p,
		store:    store,
		cfg:      store.Get(),
	}
}

// Run builds the main window according to config and starts the event loop.
// It blocks until the window is closed.
func (a *App) Run() {
	a.win = a.fyne.NewWindow("simpleStonks")
	a.win.SetContent(a.buildContent())
	a.win.Resize(fyne.NewSize(900, 600))

	// Rebuild the UI whenever the config changes, whether from our own edits
	// or an external change to the file. Reloads arrive on the watcher
	// goroutine, so marshal the rebuild onto the UI thread.
	a.store.Subscribe(func(cfg config.Config) {
		fyne.Do(func() {
			a.cfg = cfg
			a.win.SetContent(a.buildContent())
			a.startData()
		})
	})

	// Start fetching once the app is running, and stop polling on close.
	a.fyne.Lifecycle().SetOnStarted(func() { a.startData() })
	a.win.SetOnClosed(a.stopRefresh)

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
