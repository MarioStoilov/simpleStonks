// Package qtui is the Qt (miqt) user interface for simpleStonks: a frameless,
// translucent widget-style window hosting the tracked-symbol grid. It reuses
// the same config store, provider, and chart math as the other UIs.
package qtui

import (
	"fmt"
	"os"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/provider"
)

// App wires the Qt widgets to the config store and quote provider.
type App struct {
	quotes provider.QuoteProvider
	store  *config.Store

	window  *qt.QWidget
	card    *qt.QWidget
	stack   *qt.QStackedWidget
	home    *homeView
	detail  *detailView
	refresh *qt.QTimer

	refreshInterval time.Duration
}

// New builds the application shell.
func New(quotes provider.QuoteProvider, store *config.Store) *App {
	return &App{quotes: quotes, store: store}
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

	app.home = newHomeView(card, app.quotes)
	// An explicit cursor stops the grid from inheriting the window's resize
	// cursor when the pointer jumps straight from the grip margin onto it.
	app.home.scroll.SetCursor(qt.NewQCursor2(qt.ArrowCursor))
	app.detail = newDetailView(card, app.quotes, app.store, func() {
		app.stack.SetCurrentWidget(app.home.scroll.QWidget)
	})
	app.home.onOpen = func(symbol string) {
		app.detail.showSymbol(symbol)
		app.stack.SetCurrentWidget(app.detail.root)
	}

	app.stack = qt.NewQStackedWidget(card)
	app.stack.AddWidget(app.home.scroll.QWidget)
	app.stack.AddWidget(app.detail.root)
	rootLayout.AddWidget2(app.stack.QWidget, 1)

	// Periodic refresh with price flashes, dispatched to the visible view.
	app.refresh = qt.NewQTimer()
	app.refresh.OnTimeout(func() {
		if app.stack.CurrentWidget() == app.detail.root {
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
	app.card.SetStyleSheet(fmt.Sprintf(
		"#card { background-color: %s; border-radius: %dpx; } QLabel { color: %s; }",
		cssRGBA(background, alphaByte(opacity)),
		int(constants.TileCornerRadius),
		cssRGB(constants.ColorForeground)))

	app.home.setSymbols(cfg.Symbols)
	if app.detail != nil {
		app.detail.setSymbols(cfg.Symbols)
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
	return fmt.Sprintf(
		"QPushButton { background: transparent; color: %s; border: none; border-radius: %dpx; padding: 4px 10px; }"+
			" QPushButton:hover { background-color: %s; color: %s; }",
		cssRGB(constants.ColorNeutral), int(constants.PanelCornerRadius), hoverColor, cssRGB(constants.ColorForeground))
}
