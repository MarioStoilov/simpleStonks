// Command simplestonks is the entrypoint for the simpleStonks stock tracker.
package main

import (
	"log"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/provider"
	"github.com/MarioStoilov/simplestonks/internal/ui"
)

func main() {
	store, err := config.Open()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	defer store.Close()

	p := provider.NewYahoo(nil)

	ui.New(p, store).Run()
}
