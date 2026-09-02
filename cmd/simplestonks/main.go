// Command simplestonks is the entrypoint for the simpleStonks stock tracker:
// it bootstraps the config store, logger, and quote provider, then runs the
// Qt UI.
package main

import (
	"log"
	"log/slog"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/logging"
	"github.com/MarioStoilov/simplestonks/internal/provider"
	"github.com/MarioStoilov/simplestonks/internal/qtui"
)

func main() {
	store, err := config.Open()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	defer store.Close()

	// A bad log destination must never stop the app: fall back to the default
	// path (e.g. when a stale absolute path survives a snap refresh), then to
	// silent logging as a last resort.
	logCfg := store.Get().Logging
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
	defer logger.Close()
	slog.SetDefault(logger.Slog())

	// Apply logging changes live when the config file is reloaded.
	store.Subscribe(func(cfg config.Config) {
		if err := logger.Reconfigure(cfg.Logging); err != nil {
			slog.Error("logging reconfigure failed", "err", err)
		}
	})

	slog.Info("simplestonks starting", "symbols", len(store.Get().Symbols))

	// Identical fetches issued together (e.g. the detail view's main chart
	// and its sidebar tile) share one request, so both show the same price.
	prov := provider.Coalesce(provider.NewYahoo(nil))

	qtui.New(prov, store).Run()
}
