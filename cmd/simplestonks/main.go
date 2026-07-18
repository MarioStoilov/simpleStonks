// Command simplestonks is the entrypoint for the simpleStonks stock tracker.
package main

import (
	"log"
	"log/slog"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/logging"
	"github.com/MarioStoilov/simplestonks/internal/provider"
	"github.com/MarioStoilov/simplestonks/internal/ui"
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
	store.Subscribe(func(c config.Config) {
		if err := logger.Reconfigure(c.Logging); err != nil {
			slog.Error("logging reconfigure failed", "err", err)
		}
	})

	slog.Info("simplestonks starting", "symbols", len(store.Get().Symbols))

	p := provider.NewYahoo(nil)

	ui.New(p, store).Run()
}
