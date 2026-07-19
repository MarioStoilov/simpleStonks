// Command simplestonks (Wails UI) is the web-frontend entrypoint for the
// simpleStonks stock tracker. It bootstraps the same config store, logger,
// and quote provider as the Fyne entrypoint (cmd/simplestonks), then serves
// the Svelte frontend from the embedded frontend/dist bundle.
package main

import (
	"embed"
	"log"
	"log/slog"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

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

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            constants.AppName,
		Width:            constants.MainWindowWidth,
		Height:           constants.MainWindowHeight,
		BackgroundColour: backgroundColour(store.Get()),
		URL:              "/",
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

// backgroundColour converts the configured "#RRGGBB" background color (falling
// back to the default on a malformed value) into the window background.
func backgroundColour(cfg config.Config) application.RGBA {
	packed, ok := parseHexRGB(cfg.Background.Color)
	if !ok {
		packed, _ = parseHexRGB(constants.DefaultBackgroundColor)
	}
	return application.NewRGB(uint8(packed>>16), uint8(packed>>8), uint8(packed))
}

// parseHexRGB parses a "#RRGGBB" string into a packed 0xRRGGBB value.
func parseHexRGB(hex string) (uint32, bool) {
	trimmed := strings.TrimPrefix(hex, "#")
	if len(trimmed) != len("RRGGBB") {
		return 0, false
	}
	packed, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return 0, false
	}
	return uint32(packed), true
}
