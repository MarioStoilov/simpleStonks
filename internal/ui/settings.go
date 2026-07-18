package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/model"
)

// addSymbol appends a symbol (ignoring duplicates) via the config store, which
// persists it and triggers a UI rebuild through the subscription.
func (a *App) addSymbol(symbol string) {
	a.update(func(c *config.Config) {
		for _, s := range c.Symbols {
			if s == symbol {
				return
			}
		}
		c.Symbols = append(c.Symbols, symbol)
	})
}

// moveSymbol swaps the symbol at index i with its neighbor delta positions away
// (delta is -1 or +1), persisting the new order. Out-of-range moves are no-ops.
func (a *App) moveSymbol(i, delta int) {
	j := i + delta
	a.update(func(c *config.Config) {
		if i < 0 || i >= len(c.Symbols) || j < 0 || j >= len(c.Symbols) {
			return
		}
		c.Symbols[i], c.Symbols[j] = c.Symbols[j], c.Symbols[i]
	})
}

// removeSymbol drops a symbol from the tracked list via the config store.
func (a *App) removeSymbol(symbol string) {
	a.update(func(c *config.Config) {
		kept := make([]string, 0, len(c.Symbols))
		for _, s := range c.Symbols {
			if s != symbol {
				kept = append(kept, s)
			}
		}
		c.Symbols = kept
	})
}

// update applies a config mutation through the store, surfacing any save error
// on the main window.
func (a *App) update(mutate func(*config.Config)) {
	a.updateOn(a.win, mutate)
}

// updateOn applies a config mutation, surfacing any save error on w.
func (a *App) updateOn(w fyne.Window, mutate func(*config.Config)) {
	if err := a.store.Update(mutate); err != nil && w != nil {
		dialog.ShowError(err, w)
	}
}

// logLevels is the ordered set of logging levels offered in settings.
var logLevels = []config.LogLevel{
	config.LogSilent, config.LogError, config.LogWarn, config.LogInfo, config.LogDebug,
}

// showSettingsWindow opens the app configuration in a separate window, divided
// into sections picked from a sidebar. Saving writes through the config store,
// so changes persist and apply live (including the logger, which main
// reconfigures from the same store subscription). Appearance edits preview
// live while the window is open and are reverted unless saved.
func (a *App) showSettingsWindow() {
	cfg := a.cfg // snapshot to populate the forms
	w := a.fyne.NewWindow("simpleStonks — Settings")

	// General.
	rangeSel := widget.NewSelect(rangeOptions(), nil)
	rangeSel.SetSelected(string(cfg.DefaultRange))
	refresh := widget.NewEntry()
	refresh.SetText(strconv.Itoa(int(cfg.RefreshInterval / time.Second)))
	general := widget.NewForm(
		widget.NewFormItem("Default range", rangeSel),
		widget.NewFormItem("Refresh interval (s)", refresh),
	)

	// Appearance: background color (swatch + picker dialog) and opacity,
	// previewed live. Transparency is the opacity slider's job, so the picked
	// color is always taken fully opaque.
	bgCol, ok := parseHexColor(cfg.Background.Color)
	if !ok {
		bgCol, _ = parseHexColor(config.DefaultBackground().Color)
	}
	swatch := canvas.NewRectangle(bgCol)
	swatch.CornerRadius = 4
	swatch.StrokeColor = colorAxis
	swatch.StrokeWidth = 1
	swatch.SetMinSize(fyne.NewSize(48, 28))
	opacity := widget.NewSlider(0, 100)
	opacity.Step = 1
	opacity.SetValue(cfg.Background.Opacity * 100)
	preview := func() {
		a.previewBackground(config.Background{
			Color:   formatHexColor(bgCol),
			Opacity: opacity.Value / 100,
		})
	}
	pick := widget.NewButton("Choose…", func() {
		picker := dialog.NewColorPicker("Background color", "", func(c color.Color) {
			bgCol = color.NRGBAModel.Convert(c).(color.NRGBA)
			bgCol.A = 0xff
			swatch.FillColor = bgCol
			swatch.Refresh()
			preview()
		}, w)
		picker.Advanced = true
		picker.SetColor(bgCol)
		picker.Show()
	})
	opacity.OnChanged = func(float64) { preview() }
	appearance := widget.NewForm(
		widget.NewFormItem("Background color", container.NewHBox(swatch, pick)),
		widget.NewFormItem("Background opacity (%)", opacity),
	)

	// Logging.
	levelSel := widget.NewSelect(levelOptions(), nil)
	levelSel.SetSelected(string(cfg.Logging.Level))
	logFile := widget.NewEntry()
	logFile.SetPlaceHolder(config.DefaultLogPath())
	logFile.SetText(cfg.Logging.File)
	maxSize := widget.NewEntry()
	maxSize.SetText(strconv.Itoa(cfg.Logging.MaxSizeMB))
	archives := widget.NewEntry()
	archives.SetText(strconv.Itoa(cfg.Logging.MaxArchives))
	logging := widget.NewForm(
		widget.NewFormItem("Log level", levelSel),
		widget.NewFormItem("Log file (blank = default)", logFile),
		widget.NewFormItem("Log max size (MB)", maxSize),
		widget.NewFormItem("Log archives kept", archives),
	)

	// Sidebar of sections; clicking one swaps the content pane.
	sections := []struct {
		name string
		view fyne.CanvasObject
	}{
		{"General", general},
		{"Appearance", appearance},
		{"Logging", logging},
	}
	content := container.NewStack()
	btns := make([]*widget.Button, len(sections))
	selectSection := func(i int) {
		for j, b := range btns {
			if j == i {
				b.Importance = widget.HighImportance
			} else {
				b.Importance = widget.MediumImportance
			}
			b.Refresh()
		}
		content.Objects = []fyne.CanvasObject{sections[i].view}
		content.Refresh()
	}
	sidebar := container.NewVBox()
	for i, s := range sections {
		i := i
		btns[i] = widget.NewButton(s.name, func() { selectSection(i) })
		sidebar.Add(btns[i])
	}
	selectSection(0)

	save := widget.NewButton("Save", func() {
		interval, sizeMB, keep, err := parseSettingsForm(refresh.Text, maxSize.Text, archives.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		a.updateOn(w, func(c *config.Config) {
			c.DefaultRange = model.Range(rangeSel.Selected)
			c.RefreshInterval = interval
			c.Background = config.Background{Color: formatHexColor(bgCol), Opacity: opacity.Value / 100}
			c.Logging.Level = config.LogLevel(levelSel.Selected)
			c.Logging.File = strings.TrimSpace(logFile.Text)
			c.Logging.MaxSizeMB = sizeMB
			c.Logging.MaxArchives = keep
		})
		w.Close()
	})
	save.Importance = widget.HighImportance
	cancel := widget.NewButton("Cancel", func() { w.Close() })
	buttons := container.NewHBox(layout.NewSpacer(), cancel, save)

	// Closing without saving reverts any live appearance preview.
	w.SetOnClosed(func() { a.applyBackground() })

	w.SetContent(container.NewBorder(nil, buttons, sidebar, nil, container.NewVScroll(content)))
	w.Resize(fyne.NewSize(600, 420))
	w.Show()
}

// parseSettingsForm validates the free-text settings fields, returning the
// refresh interval and log rotation numbers or a descriptive error.
func parseSettingsForm(refreshSecs, maxSizeMB, archives string) (time.Duration, int, int, error) {
	secs, err := strconv.Atoi(strings.TrimSpace(refreshSecs))
	if err != nil || secs < 1 {
		return 0, 0, 0, fmt.Errorf("refresh interval must be a whole number of seconds ≥ 1")
	}
	size, err := strconv.Atoi(strings.TrimSpace(maxSizeMB))
	if err != nil || size < 0 {
		return 0, 0, 0, fmt.Errorf("log max size must be a non-negative whole number of MB")
	}
	keep, err := strconv.Atoi(strings.TrimSpace(archives))
	if err != nil || keep < 0 {
		return 0, 0, 0, fmt.Errorf("log archives kept must be a non-negative whole number")
	}
	return time.Duration(secs) * time.Second, size, keep, nil
}

func rangeOptions() []string {
	out := make([]string, len(model.Ranges))
	for i, r := range model.Ranges {
		out[i] = string(r)
	}
	return out
}

func levelOptions() []string {
	out := make([]string, len(logLevels))
	for i, l := range logLevels {
		out[i] = string(l)
	}
	return out
}
