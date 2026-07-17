// Command simplestonks is the entrypoint for the simpleStonks stock tracker.
package main

import (
	"log"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/provider"
	"github.com/MarioStoilov/simplestonks/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("config: falling back to defaults: %v", err)
		cfg = config.Default()
	}

	p := provider.NewYahoo(nil)

	ui.New(p, cfg).Run()
}
