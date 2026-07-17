// Package ui contains the Fyne front end for simpleStonks.
//
// The UI has two screens: a home grid of 1D-only cells, and a detail view for a
// selected symbol (expanded chart with range toggles plus a sidebar of all
// symbols). The form factor (normal window vs. always-on-top widget) is chosen
// from config; v1 implements the normal window, with a seam reserved for the
// widget mode.
package ui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// appID is the Fyne application ID; it also namespaces Fyne's own storage.
const appID = "com.github.mariostoilov.simplestonks"

// fetchTimeout bounds a single provider request.
const fetchTimeout = 15 * time.Second

type screenKind int

const (
	screenHome screenKind = iota
	screenDetail
)

// App wires together the Fyne application, the live config store, and the data
// provider, and holds the current navigation/view state.
type App struct {
	fyne     fyne.App
	win      fyne.Window
	provider provider.QuoteProvider
	store    *config.Store
	cfg      config.Config // cached snapshot of store.Get()

	// View state (UI goroutine only).
	screen   screenKind
	selected string      // symbol shown in the detail view
	rng      model.Range // active range in the detail view
	editMode bool        // home-grid edit mode (gates add/remove/reorder)

	homeTiles []*tile
	sideTiles []*tile
	rangeBtns map[model.Range]*widget.Button

	detailChart  *chart
	detailPrice  *canvas.Text
	detailChange *canvas.Text

	stopCh chan struct{} // closes to stop the current refresh loop
}

// New constructs the application with the given data provider and config store.
func New(p provider.QuoteProvider, store *config.Store) *App {
	// Declare that the app follows Fyne's fyne.Do threading model: every
	// cross-goroutine UI update is marshalled onto the UI thread via fyne.Do.
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

// Run builds the home screen and starts the event loop. It blocks until close.
func (a *App) Run() {
	a.win = a.fyne.NewWindow("simpleStonks")
	a.screen = screenHome
	a.win.SetContent(a.buildHome())
	a.win.Resize(fyne.NewSize(1000, 640))

	// Rebuild the current screen whenever the config changes (our edits or an
	// external file edit). Reloads arrive on the watcher goroutine, so marshal
	// onto the UI thread.
	a.store.Subscribe(func(cfg config.Config) {
		fyne.Do(func() {
			a.cfg = cfg
			a.rebuildCurrent()
		})
	})

	// Start fetching once the app is running, and stop polling on close.
	a.fyne.Lifecycle().SetOnStarted(func() { a.startData() })
	a.win.SetOnClosed(a.stopRefresh)

	a.win.ShowAndRun()
}

// showHome switches to the home grid.
func (a *App) showHome() {
	a.screen = screenHome
	a.win.SetContent(a.buildHome())
	a.startData()
}

// setEditMode toggles home-grid edit mode and rebuilds the home screen.
func (a *App) setEditMode(on bool) {
	a.editMode = on
	a.screen = screenHome
	a.win.SetContent(a.buildHome())
	a.startData()
}

// showDetail switches to (or re-selects within) the detail view for a symbol.
func (a *App) showDetail(symbol string) {
	a.screen = screenDetail
	a.selected = symbol
	a.win.SetContent(a.buildDetail())
	a.startData()
}

// rebuildCurrent rebuilds whichever screen is active after a config change,
// falling back to home if the detailed symbol is no longer tracked.
func (a *App) rebuildCurrent() {
	if a.screen == screenDetail && containsStr(a.cfg.Symbols, a.selected) {
		a.win.SetContent(a.buildDetail())
	} else {
		a.screen = screenHome
		a.win.SetContent(a.buildHome())
	}
	a.startData()
}

// startData (re)starts the initial load and refresh loop for the current screen.
// Must be called on the UI goroutine.
func (a *App) startData() {
	a.stopRefresh()
	switch a.screen {
	case screenDetail:
		a.loadDetail()
		a.startRefresh(a.detailTick())
	default:
		a.loadHome()
		a.startRefresh(a.homeTick())
	}
}

// startRefresh runs tick on the configured interval until stopped. tick captures
// everything it needs, so it never reads shared App state from the goroutine.
func (a *App) startRefresh(tick func()) {
	if tick == nil {
		return
	}
	interval := a.cfg.RefreshInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
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
				tick()
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

// loadTile1D fetches a tile's 1D history and applies it on the UI goroutine.
func loadTile1D(prov provider.QuoteProvider, t *tile) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		s, err := prov.History(ctx, t.symbol, model.Range1D)
		fyne.Do(func() {
			if err != nil {
				t.setError(err)
				return
			}
			t.setSeries(s)
		})
	}()
}

// containsStr reports whether ss contains s.
func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
