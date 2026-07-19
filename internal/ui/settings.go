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
func (app *App) addSymbol(symbol string) {
	app.update(func(conf *config.Config) {
		for _, tracked := range conf.Symbols {
			if tracked == symbol {
				return
			}
		}
		conf.Symbols = append(conf.Symbols, symbol)
	})
}

// moveSymbol swaps the symbol at index idx with its neighbor delta positions
// away (delta is -1 or +1), persisting the new order. Out-of-range moves are
// no-ops.
func (app *App) moveSymbol(idx, delta int) {
	neighbor := idx + delta
	app.update(func(conf *config.Config) {
		if idx < 0 || idx >= len(conf.Symbols) || neighbor < 0 || neighbor >= len(conf.Symbols) {
			return
		}
		conf.Symbols[idx], conf.Symbols[neighbor] = conf.Symbols[neighbor], conf.Symbols[idx]
	})
}

// removeSymbol drops a symbol from the tracked list via the config store.
func (app *App) removeSymbol(symbol string) {
	app.update(func(conf *config.Config) {
		kept := make([]string, 0, len(conf.Symbols))
		for _, tracked := range conf.Symbols {
			if tracked != symbol {
				kept = append(kept, tracked)
			}
		}
		conf.Symbols = kept
	})
}

// update applies a config mutation through the store, surfacing any save error
// on the main window.
func (app *App) update(mutate func(*config.Config)) {
	app.updateOn(app.win, mutate)
}

// updateOn applies a config mutation, surfacing any save error on win.
func (app *App) updateOn(win fyne.Window, mutate func(*config.Config)) {
	if err := app.store.Update(mutate); err != nil && win != nil {
		dialog.ShowError(err, win)
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
func (app *App) showSettingsWindow() {
	cfg := app.cfg // snapshot to populate the forms
	win := app.fyne.NewWindow("simpleStonks — Settings")

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
		app.previewBackground(config.Background{
			Color:   formatHexColor(bgCol),
			Opacity: opacity.Value / 100,
		})
	}
	pick := widget.NewButton("Choose…", func() {
		picker := dialog.NewColorPicker("Background color", "", func(picked color.Color) {
			bgCol = color.NRGBAModel.Convert(picked).(color.NRGBA)
			bgCol.A = 0xff
			swatch.FillColor = bgCol
			swatch.Refresh()
			preview()
		}, win)
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
	selectSection := func(selected int) {
		for btnIdx, btn := range btns {
			if btnIdx == selected {
				btn.Importance = widget.HighImportance
			} else {
				btn.Importance = widget.MediumImportance
			}
			btn.Refresh()
		}
		content.Objects = []fyne.CanvasObject{sections[selected].view}
		content.Refresh()
	}
	sidebar := container.NewVBox()
	for idx, section := range sections {
		idx := idx
		btns[idx] = widget.NewButton(section.name, func() { selectSection(idx) })
		sidebar.Add(btns[idx])
	}
	selectSection(0)

	save := widget.NewButton("Save", func() {
		interval, sizeMB, keep, err := parseSettingsForm(refresh.Text, maxSize.Text, archives.Text)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		app.updateOn(win, func(conf *config.Config) {
			conf.DefaultRange = model.Range(rangeSel.Selected)
			conf.RefreshInterval = interval
			conf.Background = config.Background{Color: formatHexColor(bgCol), Opacity: opacity.Value / 100}
			conf.Logging.Level = config.LogLevel(levelSel.Selected)
			conf.Logging.File = strings.TrimSpace(logFile.Text)
			conf.Logging.MaxSizeMB = sizeMB
			conf.Logging.MaxArchives = keep
		})
		win.Close()
	})
	save.Importance = widget.HighImportance
	cancel := widget.NewButton("Cancel", func() { win.Close() })
	buttons := container.NewHBox(layout.NewSpacer(), cancel, save)

	// Closing without saving reverts any live appearance preview.
	win.SetOnClosed(func() { app.applyBackground() })

	win.SetContent(container.NewBorder(nil, buttons, sidebar, nil, container.NewVScroll(content)))
	win.Resize(fyne.NewSize(600, 420))
	win.Show()
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
	for idx, rng := range model.Ranges {
		out[idx] = string(rng)
	}
	return out
}

func levelOptions() []string {
	out := make([]string, len(logLevels))
	for idx, level := range logLevels {
		out[idx] = string(level)
	}
	return out
}
