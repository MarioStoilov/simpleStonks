// Package qtui is the Qt (miqt) user interface for simpleStonks: a frameless,
// translucent widget-style window hosting the tracked-symbol grid. It reuses
// the same config store, provider, and chart math as the other UIs.
package qtui

import (
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"slices"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/assets"
	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/notify"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// App wires the Qt widgets to the config store and quote provider.
type App struct {
	quotes provider.QuoteProvider
	store  *config.Store

	window       *qt.QWidget
	card         *qt.QWidget
	stack        *qt.QStackedWidget
	home         *homeView
	detail       *detailView
	refresh      *qt.QTimer
	offlineLabel *qt.QLabel

	// fetchOK records each tracked symbol's latest fetch outcome; the
	// offline indicator shows once they have all failed.
	fetchOK map[string]bool

	refreshInterval time.Duration
}

// New builds the application shell.
func New(quotes provider.QuoteProvider, store *config.Store) *App {
	return &App{quotes: quotes, store: store, fetchOK: map[string]bool{}}
}

// Run builds the window and blocks until the app exits.
func (app *App) Run() {
	qt.NewQApplication(os.Args)

	// Frameless translucent window: the inner card paints the configured
	// background color at the configured opacity — the widget look that
	// motivated the UI rewrite. Dragging anywhere on the background moves
	// the window; the top bar supplies minimise/close in place of the
	// missing decorations.
	window := qt.NewQWidget2()
	window.SetWindowTitle(constants.AppName)
	window.SetWindowFlags(qt.FramelessWindowHint)
	window.SetAttribute(qt.WA_TranslucentBackground)
	window.Resize(int(constants.MainWindowWidth), int(constants.MainWindowHeight))
	app.window = window

	card := qt.NewQWidget(window)
	card.SetObjectName(*qt.NewQAnyStringView3("card"))
	card.SetGeometry(0, 0, int(constants.MainWindowWidth), int(constants.MainWindowHeight))
	app.card = card

	rootLayout := qt.NewQVBoxLayout(card)

	topBar := qt.NewQHBoxLayout2()
	// Connection-lost indicator: shown while every tracked symbol's latest
	// fetch failed; the views keep showing their last data meanwhile.
	offlineLabel := qt.NewQLabel(card)
	offlineLabel.SetPixmap(svgPixmap(assets.OfflineSVG, int(constants.HeaderIconSize)))
	offlineLabel.SetToolTip(constants.TipOffline)
	offlineLabel.SetStyleSheet("background: transparent;")
	offlineLabel.SetVisible(false)
	app.offlineLabel = offlineLabel
	topBar.AddWidget(offlineLabel.QWidget)
	topBar.AddStretch()
	minimiseButton := qt.NewQPushButton5("—", card)
	minimiseButton.SetStyleSheet(windowButtonStyle(cssRGB(constants.ColorHover)))
	minimiseButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	minimiseButton.OnClicked(func() { window.ShowMinimized() })
	topBar.AddWidget(minimiseButton.QWidget)
	closeButton := qt.NewQPushButton5("✕", card)
	closeButton.SetStyleSheet(windowButtonStyle(cssRGB(constants.ColorDown)))
	closeButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	closeButton.OnClicked(func() { window.Close() })
	topBar.AddWidget(closeButton.QWidget)
	rootLayout.AddLayout(topBar.QLayout)

	app.home = newHomeView(card, app.quotes, app.store)
	app.home.onOpenSettings = func() {
		showSettingsDialog(window, app.store, app.applyBackgroundStyle, app.previewChartStyle)
		// Revert any unsaved appearance preview to the persisted values (a
		// save also lands here, harmlessly re-applying the new config).
		app.applyConfig(app.store.Get())
	}
	app.detail = newDetailView(card, app.quotes, app.store, func() {
		app.stack.SetCurrentWidget(app.home.root)
	})
	app.home.onOpen = func(symbol string) {
		app.detail.showSymbol(symbol)
		app.stack.SetCurrentWidget(app.detail.root)
	}
	// Every fetch outcome feeds the offline indicator, and every fresh
	// quote runs through the price-alert check; the home tiles and the
	// detail sidebar each cover all tracked symbols every tick, so both
	// work whichever view is showing.
	app.home.onQuote = app.handleQuote
	app.detail.onQuote = app.handleQuote
	// A clicked alert notification brings the window up on the alert's
	// symbol; the callback arrives on a D-Bus goroutine.
	notify.OnActivate(func(alert notify.Alert) {
		mainthread.Wait(func() { app.openAlertSymbol(alert.Symbol) })
	})

	app.stack = qt.NewQStackedWidget(card)
	app.stack.AddWidget(app.home.root)
	app.stack.AddWidget(app.detail.root)
	rootLayout.AddWidget2(app.stack.QWidget, 1)

	// Periodic refresh with price flashes, dispatched to the visible view.
	// Compare stack indexes, not widgets: miqt's CurrentWidget() returns a
	// fresh Go wrapper each call, so wrapper identity never matches.
	app.refresh = qt.NewQTimer()
	app.refresh.OnTimeout(func() {
		if app.stack.CurrentIndex() == app.stack.IndexOf(app.detail.root) {
			app.detail.refresh()
			return
		}
		app.home.loadAll(true)
	})

	app.applyConfig(app.store.Get())
	app.home.loadAll(false)

	// Config changes (UI edits or external file edits) apply live; the
	// subscription fires on a watcher goroutine, so hop to the UI thread.
	app.store.Subscribe(func(cfg config.Config) {
		mainthread.Wait(func() { app.applyConfig(cfg) })
	})

	window.OnResizeEvent(func(super func(event *qt.QResizeEvent), event *qt.QResizeEvent) {
		super(event)
		card.SetGeometry(0, 0, window.Width(), window.Height())
	})

	// Frameless windows have no native borders: the outer few pixels act as
	// an invisible resize grip (with matching cursors), everything else drags
	// the window. Mouse tracking on the card lets hover moves reach the
	// window handler for the cursor updates.
	window.SetMouseTracking(true)
	card.SetMouseTracking(true)
	window.OnMouseMoveEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		position := event.Position()
		if edges := app.edgeAt(position.X(), position.Y()); edges != 0 {
			window.SetCursor(qt.NewQCursor2(resizeCursor(edges)))
		} else {
			window.UnsetCursor()
		}
		super(event)
	})
	window.OnLeaveEvent(func(super func(event *qt.QEvent), event *qt.QEvent) {
		window.UnsetCursor()
		super(event)
	})
	window.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		position := event.Position()
		if edges := app.edgeAt(position.X(), position.Y()); edges != 0 {
			window.WindowHandle().StartSystemResize(edges)
			return
		}
		window.WindowHandle().StartSystemMove()
	})

	window.Show()
	qt.QApplication_Exec()
}

// applyBackgroundStyle paints the card with a background color at an opacity
// (also used by the settings dialog's live preview).
func (app *App) applyBackgroundStyle(background color.NRGBA, opacity float64) {
	app.card.SetStyleSheet(fmt.Sprintf(constants.StyleWindowCard,
		cssRGBA(background, alphaByte(opacity)),
		int(constants.TileCornerRadius),
		cssRGB(constants.ColorForeground)))
}

// handleQuote digests one fetch outcome on the UI thread: the result feeds
// the offline indicator, and successful quotes run the price-alert check.
func (app *App) handleQuote(symbol string, price float64, ok bool) {
	app.fetchOK[symbol] = ok
	app.updateOfflineIndicator()
	if ok {
		app.handlePrice(symbol, price)
	}
}

// updateOfflineIndicator shows the header's connection-lost icon while no
// tracked symbol is fetchable.
func (app *App) updateOfflineIndicator() {
	if app.offlineLabel != nil {
		app.offlineLabel.SetVisible(offlineNow(app.fetchOK))
	}
}

// offlineNow reports whether every recorded fetch result is a failure: one
// unreachable symbol keeps the app "online" (a bad ticker must not raise the
// indicator), but all of them failing means the network is gone.
func offlineNow(results map[string]bool) bool {
	if len(results) == 0 {
		return false
	}
	for _, fetched := range results {
		if fetched {
			return false
		}
	}
	return true
}

// handlePrice checks the pending price alerts against a fresh quote: fired
// alerts raise a desktop notification and are removed (one-shot). With
// notifications disabled the alerts stay pending instead of firing
// silently. Runs on the UI thread via the fetch callbacks; the alert-list
// re-render follows through the config subscription.
func (app *App) handlePrice(symbol string, price float64) {
	cfg := app.store.Get()
	if !cfg.Notifications.Enabled {
		return
	}
	fired, _ := notify.Triggered(cfg.Alerts, symbol, price)
	if len(fired) == 0 {
		return
	}
	if err := app.store.Update(func(conf *config.Config) {
		_, conf.Alerts = notify.Triggered(conf.Alerts, symbol, price)
	}); err != nil {
		slog.Error("removing fired alerts failed", "symbol", symbol, "err", err)
	}
	for _, alert := range fired {
		triggered := alert // capture per iteration for the goroutine
		// Off the UI thread: the session-bus call may block briefly.
		go func() {
			if err := notify.SendAlert(triggered, price, cfg.Notifications.Duration); err != nil {
				slog.Error("desktop notification failed",
					"symbol", triggered.Symbol, "price", triggered.Price, "err", err)
			}
		}()
	}
}

// openAlertSymbol brings the window to the front and opens the detail view
// of the symbol whose alert notification was clicked (when it is still
// tracked). Wayland compositors may only flash the taskbar entry instead of
// stealing focus — that is compositor policy.
func (app *App) openAlertSymbol(symbol string) {
	app.window.ShowNormal()
	app.window.Raise()
	app.window.ActivateWindow()
	for _, tracked := range app.store.Get().Symbols {
		if tracked == symbol {
			app.detail.showSymbol(symbol)
			app.stack.SetCurrentWidget(app.detail.root)
			return
		}
	}
}

// previewChartStyle applies chart styling immediately (the settings dialog's
// live preview); unsaved edits are reverted by the applyConfig that runs when
// the dialog closes.
func (app *App) previewChartStyle(cfg config.Chart) {
	setChartStyle(cfg)
	app.repaintCharts()
}

// repaintCharts repaints every live chart after a chart-style change.
func (app *App) repaintCharts() {
	app.home.repaintCharts()
	if app.detail != nil {
		app.detail.repaintCharts()
	}
}

// applyConfig applies a config snapshot: background styling, tracked symbols,
// and the refresh cadence. Runs on the UI thread.
func (app *App) applyConfig(cfg config.Config) {
	background, ok := parseHexColor(cfg.Background.Color)
	if !ok {
		background, _ = parseHexColor(constants.DefaultBackgroundColor)
	}
	opacity := cfg.Background.Opacity
	if opacity < 0 || opacity > 1 {
		opacity = constants.DefaultBackgroundOpacity
	}
	app.applyBackgroundStyle(background, opacity)
	setChartStyle(cfg.Chart)
	app.repaintCharts()

	// Forget fetch results of untracked symbols so stale entries cannot
	// skew the offline indicator.
	for symbol := range app.fetchOK {
		if !slices.Contains(cfg.Symbols, symbol) {
			delete(app.fetchOK, symbol)
		}
	}
	app.updateOfflineIndicator()

	app.home.setSymbols(cfg.Symbols)
	if app.detail != nil {
		app.detail.setSymbols(cfg.Symbols)
		app.detail.setExtendedHours(cfg.ExtendedHours)
		// The notifications flag gates the alert pills, so apply it first.
		app.detail.setNotificationsEnabled(cfg.Notifications.Enabled)
		app.detail.setAlerts(cfg.Alerts)
	}

	interval := cfg.RefreshInterval
	if interval <= 0 {
		interval = constants.DefaultRefreshInterval
	}
	if interval != app.refreshInterval {
		app.refreshInterval = interval
		app.refresh.Start(int(interval / time.Millisecond))
	}
}

// edgeAt returns the window edges within the resize-grip margin of a point,
// for frameless-resize hit testing (zero means no edge).
func (app *App) edgeAt(posX, posY float64) qt.Edge {
	var edges qt.Edge
	if posX < constants.ResizeGripMargin {
		edges |= qt.LeftEdge
	}
	if posX > float64(app.window.Width())-constants.ResizeGripMargin {
		edges |= qt.RightEdge
	}
	if posY < constants.ResizeGripMargin {
		edges |= qt.TopEdge
	}
	if posY > float64(app.window.Height())-constants.ResizeGripMargin {
		edges |= qt.BottomEdge
	}
	return edges
}

// resizeCursor maps resize edges to the matching cursor shape.
func resizeCursor(edges qt.Edge) qt.CursorShape {
	switch edges {
	case qt.LeftEdge, qt.RightEdge:
		return qt.SizeHorCursor
	case qt.TopEdge, qt.BottomEdge:
		return qt.SizeVerCursor
	case qt.TopEdge | qt.LeftEdge, qt.BottomEdge | qt.RightEdge:
		return qt.SizeFDiagCursor
	case qt.TopEdge | qt.RightEdge, qt.BottomEdge | qt.LeftEdge:
		return qt.SizeBDiagCursor
	default:
		return qt.ArrowCursor
	}
}

// windowButtonStyle styles a window-control button with the given hover color.
func windowButtonStyle(hoverColor string) string {
	return fmt.Sprintf(constants.StyleWindowButton,
		cssRGB(constants.ColorNeutral), int(constants.PanelCornerRadius), hoverColor, cssRGB(constants.ColorForeground))
}
