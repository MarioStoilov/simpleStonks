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

	"github.com/MarioStoilov/simplestonks/internal/assets"
	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// appIcon is the logo as a Fyne resource, used as the icon of every window.
var appIcon = fyne.NewStaticResource("icon.svg", assets.IconSVG)

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

	bgApplied config.Background // last background applied to the theme
	bgSet     bool              // whether bgApplied has been applied at all

	detailChart  *chart
	detailName   *canvas.Text
	detailPrice  *priceText
	detailChange *canvas.Text

	stopCh chan struct{} // closes to stop the current refresh loop
}

// New constructs the application with the given data provider and config store.
func New(prov provider.QuoteProvider, store *config.Store) *App {
	// Declare that the app follows Fyne's fyne.Do threading model: every
	// cross-goroutine UI update is marshalled onto the UI thread via fyne.Do.
	fyneapp.SetMetadata(fyne.AppMetadata{
		ID:         constants.AppID,
		Name:       constants.AppName,
		Version:    constants.AppVersion,
		Icon:       appIcon,
		Migrations: map[string]bool{"fyneDo": true},
	})
	return &App{
		fyne:     fyneapp.NewWithID(constants.AppID),
		provider: prov,
		store:    store,
		cfg:      store.Get(),
	}
}

// Run builds the home screen and starts the event loop. It blocks until close.
func (app *App) Run() {
	app.applyBackground()
	app.win = app.fyne.NewWindow(constants.AppName)
	app.screen = screenHome
	app.win.SetContent(app.buildHome())
	app.win.Resize(fyne.NewSize(constants.MainWindowWidth, constants.MainWindowHeight))

	// Rebuild the current screen whenever the config changes (our edits or an
	// external file edit). Reloads arrive on the watcher goroutine, so marshal
	// onto the UI thread.
	app.store.Subscribe(func(cfg config.Config) {
		fyne.Do(func() {
			app.cfg = cfg
			app.applyBackground()
			app.rebuildCurrent()
		})
	})

	// Start fetching once the app is running, and stop polling on close.
	app.fyne.Lifecycle().SetOnStarted(func() { app.startData() })
	app.win.SetOnClosed(app.stopRefresh)

	app.win.ShowAndRun()
}

// showHome switches to the home grid.
func (app *App) showHome() {
	app.screen = screenHome
	app.win.SetContent(app.buildHome())
	app.startData()
}

// setEditMode toggles home-grid edit mode and rebuilds the home screen.
func (app *App) setEditMode(enabled bool) {
	app.editMode = enabled
	app.screen = screenHome
	app.win.SetContent(app.buildHome())
	app.startData()
}

// showDetail switches to (or re-selects within) the detail view for a symbol.
func (app *App) showDetail(symbol string) {
	app.screen = screenDetail
	app.selected = symbol
	app.win.SetContent(app.buildDetail())
	app.startData()
}

// rebuildCurrent rebuilds whichever screen is active after a config change,
// falling back to home if the detailed symbol is no longer tracked.
func (app *App) rebuildCurrent() {
	if app.screen == screenDetail && containsStr(app.cfg.Symbols, app.selected) {
		app.win.SetContent(app.buildDetail())
	} else {
		app.screen = screenHome
		app.win.SetContent(app.buildHome())
	}
	app.startData()
}

// startData (re)starts the initial load and refresh loop for the current screen.
// Must be called on the UI goroutine.
func (app *App) startData() {
	app.stopRefresh()
	switch app.screen {
	case screenDetail:
		app.loadDetail()
		app.startRefresh(app.detailTick())
	default:
		app.loadHome()
		app.startRefresh(app.homeTick())
	}
}

// startRefresh runs tick on the configured interval until stopped. tick captures
// everything it needs, so it never reads shared App state from the goroutine.
func (app *App) startRefresh(tick func()) {
	if tick == nil {
		return
	}
	interval := app.cfg.RefreshInterval
	if interval <= 0 {
		interval = constants.DefaultRefreshInterval
	}
	stop := make(chan struct{})
	app.stopCh = stop
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				tick()
			}
		}
	}()
}

// stopRefresh stops the polling loop if one is running.
func (app *App) stopRefresh() {
	if app.stopCh != nil {
		close(app.stopCh)
		app.stopCh = nil
	}
}

// loadTile1D fetches a tile's 1D history and applies it on the UI goroutine.
func loadTile1D(prov provider.QuoteProvider, cell *tile) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), constants.FetchTimeout)
		defer cancel()
		series, err := prov.History(ctx, cell.symbol, model.Range1D)
		fyne.Do(func() {
			if err != nil {
				cell.setError(err)
				return
			}
			cell.setSeries(series)
		})
	}()
}

// containsStr reports whether candidates contains target.
func containsStr(candidates []string, target string) bool {
	for _, candidate := range candidates {
		if candidate == target {
			return true
		}
	}
	return false
}
