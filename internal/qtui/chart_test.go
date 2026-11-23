package qtui

import (
	"image/color"
	"testing"

	"github.com/MarioStoilov/simplestonks/internal/config"
)

func TestChartStyleAlertLines(t *testing.T) {
	cfg := config.DefaultChart()
	cfg.AlertLineColor = "#102030"
	cfg.AlertLineOpacity = 0.5
	style := chartStyleOf(cfg)
	if want := (color.NRGBA{R: 0x10, G: 0x20, B: 0x30, A: 0x80}); !style.alertLines || style.alertColor != want {
		t.Errorf("alert lines = %v in %v, want on in %v", style.alertLines, style.alertColor, want)
	}

	// Invalid values (e.g. a hand-edited config) fall back to the defaults.
	cfg.AlertLines = false
	cfg.AlertLineColor = "amber"
	cfg.AlertLineOpacity = 3
	style = chartStyleOf(cfg)
	if want := (color.NRGBA{R: 0xf0, G: 0xb4, B: 0x29, A: 0xcc}); style.alertLines || style.alertColor != want {
		t.Errorf("fallback alert lines = %v in %v, want off in %v", style.alertLines, style.alertColor, want)
	}
}
