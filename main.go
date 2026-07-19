// Command simplestonks (Wails UI) is the web-frontend entrypoint for the
// simpleStonks stock tracker. It bootstraps the same config store, logger,
// and quote provider as the Fyne entrypoint (cmd/simplestonks), then serves
// the Svelte frontend from the embedded frontend/dist bundle.
package main

import (
	"embed"
	"log"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/logging"
	"github.com/MarioStoilov/simplestonks/internal/provider"
	"github.com/MarioStoilov/simplestonks/internal/service"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Registered events get strongly typed TS bindings generated for them.
	application.RegisterEvent[config.Config](constants.EventConfigChanged)
}

func main() {
	store, err := config.Open()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	defer store.Close()

	logger := openLogger(store.Get().Logging)
	defer logger.Close()
	slog.SetDefault(logger.Slog())

	prov := provider.NewYahoo(nil)

	app := application.New(application.Options{
		Name: constants.AppName,
		Services: []application.Service{
			application.NewService(service.NewMarket(prov)),
			application.NewService(service.NewSettings(store)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// Live config propagation: reconfigure the logger and notify the frontend
	// whenever the config changes (UI edit or external file edit).
	store.Subscribe(func(cfg config.Config) {
		if err := logger.Reconfigure(cfg.Logging); err != nil {
			slog.Error("logging reconfigure failed", "err", err)
		}
		app.Event.Emit(constants.EventConfigChanged, cfg)
	})

	// Frameless + transparent window: the page's body background (the config
	// color at the config opacity, applied by the frontend) is all that
	// renders behind the UI, giving the translucent widget look that
	// motivated the Wails rewrite. The frontend supplies drag regions and
	// window controls in place of the missing decorations.
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            constants.AppName,
		Width:            constants.MainWindowWidth,
		Height:           constants.MainWindowHeight,
		Frameless:        true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		Linux: application.LinuxWindow{
			// Software rendering for the webview: WebKitGTK's GPU path
			// produces stale-buffer artifacts (and unreliable window alpha)
			// on NVIDIA; this app's UI is light enough to paint on the CPU.
			WebviewGpuPolicy: application.WebviewGpuPolicyNever,
		},
		URL: "/",
	})

	// Workaround: the GTK4/WebKitGTK backend of Wails v3-alpha only starts
	// honoring the transparent window background after the surface has been
	// resized once (observed on KWin/Wayland + NVIDIA). Nudge the size right
	// after startup so transparency applies without a manual resize.
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		go func() {
			time.Sleep(constants.WindowTransparencyNudgeDelay)
			width, height := window.Size()
			window.SetSize(width+1, height)
			window.SetSize(width, height)
		}()
	})

	slog.Info("simplestonks starting", "symbols", len(store.Get().Symbols))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// openLogger builds the file logger with the same fallback chain as the Fyne
// entrypoint: a bad destination falls back to the default log path, then to
// silent logging, so logging problems never stop the app.
func openLogger(logCfg config.Logging) *logging.Logger {
	logger, err := logging.New(logCfg)
	if err != nil {
		log.Printf("logging: %v; falling back to the default log path", err)
		logCfg.File = ""
		if logger, err = logging.New(logCfg); err != nil {
			log.Printf("logging: %v; continuing without file logging", err)
			logCfg.Level = config.LogSilent
			logger, _ = logging.New(logCfg) // silent logging cannot fail
		}
	}
	return logger
}
