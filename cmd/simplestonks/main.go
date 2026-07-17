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

	logger, err := logging.New(store.Get().Logging)
	if err != nil {
		log.Fatalf("logging: %v", err)
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
