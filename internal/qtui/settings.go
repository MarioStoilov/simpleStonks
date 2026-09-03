package qtui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/config"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
	"github.com/MarioStoilov/simplestonks/internal/notify"
)

// logLevels is the ordered set of log levels shown in the settings form.
var logLevels = []config.LogLevel{
	config.LogSilent, config.LogError, config.LogWarn, config.LogInfo, config.LogDebug,
}

// showSettingsDialog runs the sectioned settings editor (General / Appearance
// / Logging). Appearance edits preview live through previewBackground and
// previewChart and nothing persists until Save; the caller restores the
// persisted appearance when the dialog closes unsaved.
func showSettingsDialog(parent *qt.QWidget, store *config.Store, previewBackground func(color.NRGBA, float64), previewChart func(config.Chart)) {
	cfg := store.Get()
	dialog, body := newCardDialog(parent, constants.TitleSettings)
	dialog.Resize(int(constants.SettingsWindowWidth), int(constants.SettingsWindowHeight))
	loaded := loadForm(settingsForm)
	body.AddWidget(loaded.root)
	body.SetStretchFactor(loaded.root, 1)

	// Section navigation on the left, one page per section on the right.
	pages := loaded.stack("pages")
	sectionButtons := []*qt.QPushButton{
		loaded.button("generalButton"), loaded.button("appearanceButton"),
		loaded.button("notificationsButton"), loaded.button("loggingButton"),
	}
	styleSections := func(active int) {
		for idx, button := range sectionButtons {
			setState(button.QWidget, "selected", fmt.Sprint(idx == active))
		}
	}
	for idx, button := range sectionButtons {
		pageIdx := idx // capture per iteration
		button.OnClicked(func() {
			pages.SetCurrentIndex(pageIdx)
			styleSections(pageIdx)
		})
	}
	styleSections(0)

	// --- General ---
	rangeBox := loaded.comboBox("rangeBox")
	for _, rangeOption := range model.Ranges {
		rangeBox.AddItem(string(rangeOption))
	}
	rangeBox.SetCurrentText(string(cfg.DefaultRange))
	refreshEdit := loaded.lineEdit("refreshEdit")
	refreshEdit.SetText(strconv.Itoa(int(cfg.RefreshInterval / time.Second)))
	extendedBox := loaded.checkBox("extendedBox")
	extendedBox.SetChecked(cfg.ExtendedHours)

	// --- Appearance (edits preview live) ---
	background, ok := parseHexColor(cfg.Background.Color)
	if !ok {
		background, _ = parseHexColor(constants.DefaultBackgroundColor)
	}
	chartBackground, ok := parseHexColor(cfg.Chart.Background)
	if !ok {
		chartBackground = constants.ColorChartBg
	}
	gridColor, ok := parseHexColor(cfg.Chart.GridColor)
	if !ok {
		gridColor, _ = parseHexColor(constants.DefaultChartGridColor)
	}

	// The preview closures are wired after all the controls exist.
	var preview func()
	var chartPreview func()

	bindColorSwatch(loaded.button("backgroundSwatch"), dialog.QWidget, background,
		constants.TitleColorPicker, func(picked color.NRGBA) {
			background = picked
			preview()
		})
	opacitySlider := loaded.slider("opacitySlider")
	opacitySlider.SetValue(int(cfg.Background.Opacity*constants.PercentMax + 0.5))

	bindColorSwatch(loaded.button("chartSwatch"), dialog.QWidget, chartBackground,
		constants.TitleChartColorPicker, func(picked color.NRGBA) {
			chartBackground = picked
			chartPreview()
		})
	gridBox := loaded.checkBox("gridBox")
	gridBox.SetChecked(cfg.Chart.Grid)
	gridSizeLabel := loaded.label("gridSizeLabel")
	gridSizeEdit := loaded.lineEdit("gridSizeEdit")
	gridSizeEdit.SetText(strconv.Itoa(cfg.Chart.GridSize))
	gridColorLabel := loaded.label("gridColorLabel")
	gridSwatch := loaded.button("gridSwatch")
	bindColorSwatch(gridSwatch, dialog.QWidget, gridColor,
		constants.TitleGridColorPicker, func(picked color.NRGBA) {
			gridColor = picked
			chartPreview()
		})
	fillBox := loaded.checkBox("fillBox")
	fillBox.SetChecked(cfg.Chart.Fill)
	fillOpacityLabel := loaded.label("fillOpacityLabel")
	fillSlider := loaded.slider("fillSlider")
	fillSlider.SetValue(int(cfg.Chart.FillOpacity*constants.PercentMax + 0.5))

	// A disabled effect grays out the settings that only apply to it.
	applyEffectStates := func() {
		for _, widget := range []*qt.QWidget{
			gridSizeLabel.QWidget, gridSizeEdit.QWidget, gridColorLabel.QWidget, gridSwatch.QWidget,
		} {
			widget.SetEnabled(gridBox.IsChecked())
		}
		fillOpacityLabel.SetEnabled(fillBox.IsChecked())
		fillSlider.SetEnabled(fillBox.IsChecked())
	}
	applyEffectStates()

	preview = func() {
		previewBackground(background, float64(opacitySlider.Value())/constants.PercentMax)
	}
	// gridSizeValue parses the grid square size; -1 flags an invalid entry.
	gridSizeValue := func() int {
		size, err := parseWhole(gridSizeEdit.Text())
		if err != nil || size < constants.MinChartGridSize {
			return -1
		}
		return size
	}
	chartOf := func(gridSize int) config.Chart {
		return config.Chart{
			Background:  hexOf(chartBackground),
			Grid:        gridBox.IsChecked(),
			GridSize:    gridSize,
			GridColor:   hexOf(gridColor),
			Fill:        fillBox.IsChecked(),
			FillOpacity: float64(fillSlider.Value()) / constants.PercentMax,
		}
	}
	chartPreview = func() {
		size := gridSizeValue()
		if size < 0 { // not yet valid: preview with the saved size
			size = cfg.Chart.GridSize
		}
		previewChart(chartOf(size))
	}
	opacitySlider.OnValueChanged(func(value int) { preview() })
	gridBox.OnClicked(func() {
		applyEffectStates()
		chartPreview()
	})
	gridSizeEdit.OnTextChanged(func(text string) { chartPreview() })
	fillBox.OnClicked(func() {
		applyEffectStates()
		chartPreview()
	})
	fillSlider.OnValueChanged(func(value int) { chartPreview() })

	// --- Notifications (price alerts) ---
	notifyBox := loaded.checkBox("notifyBox")
	notifyBox.SetChecked(cfg.Notifications.Enabled)
	durationLabel := loaded.label("durationLabel")
	durationBox := loaded.comboBox("durationBox")
	for _, durationOption := range notify.Durations {
		durationBox.AddItem(string(durationOption))
	}
	durationBox.SetCurrentText(string(cfg.Notifications.Duration))
	// Disabled notifications gray out the setting that only applies to them.
	applyNotifyState := func() {
		durationLabel.SetEnabled(notifyBox.IsChecked())
		durationBox.SetEnabled(notifyBox.IsChecked())
	}
	applyNotifyState()
	notifyBox.OnClicked(func() { applyNotifyState() })
	// Clearing alerts applies immediately (it is not part of Save) after an
	// explicit, irreversible-action confirmation.
	loaded.button("clearButton").OnClicked(func() {
		if showConfirmDialog(dialog.QWidget, constants.TitleClearAlerts,
			constants.MsgConfirmClearAlerts, constants.LabelClear) {
			clearAlerts(store)
		}
	})

	// --- Logging ---
	levelBox := loaded.comboBox("levelBox")
	for _, level := range logLevels {
		levelBox.AddItem(string(level))
	}
	levelBox.SetCurrentText(string(cfg.Logging.Level))
	fileEdit := loaded.lineEdit("fileEdit")
	fileEdit.SetText(cfg.Logging.File)
	fileEdit.SetPlaceholderText(config.DefaultLogPath())
	maxSizeEdit := loaded.lineEdit("maxSizeEdit")
	maxSizeEdit.SetText(strconv.Itoa(cfg.Logging.MaxSizeMB))
	archivesEdit := loaded.lineEdit("archivesEdit")
	archivesEdit.SetText(strconv.Itoa(cfg.Logging.MaxArchives))

	errorLabel := loaded.label("errorLabel")
	loaded.button("cancelButton").OnClicked(func() { dialog.Reject() })
	loaded.button("saveButton").OnClicked(func() {
		seconds, err := parseWhole(refreshEdit.Text())
		if err != nil || seconds < 1 {
			errorLabel.SetText(constants.MsgErrRefreshInterval)
			return
		}
		gridSize := gridSizeValue()
		if gridSize < 0 {
			errorLabel.SetText(constants.MsgErrGridSize)
			return
		}
		maxSize, err := parseWhole(maxSizeEdit.Text())
		if err != nil {
			errorLabel.SetText(constants.MsgErrLogMaxSize)
			return
		}
		archives, err := parseWhole(archivesEdit.Text())
		if err != nil {
			errorLabel.SetText(constants.MsgErrLogArchives)
			return
		}
		updateErr := store.Update(func(conf *config.Config) {
			conf.DefaultRange = model.Range(rangeBox.CurrentText())
			conf.RefreshInterval = time.Duration(seconds) * time.Second
			conf.ExtendedHours = extendedBox.IsChecked()
			conf.Notifications.Enabled = notifyBox.IsChecked()
			conf.Notifications.Duration = notify.Duration(durationBox.CurrentText())
			conf.Background.Color = hexOf(background)
			conf.Background.Opacity = float64(opacitySlider.Value()) / constants.PercentMax
			conf.Chart = chartOf(gridSize)
			conf.Logging.Level = config.LogLevel(levelBox.CurrentText())
			conf.Logging.File = strings.TrimSpace(fileEdit.Text())
			conf.Logging.MaxSizeMB = maxSize
			conf.Logging.MaxArchives = archives
		})
		if updateErr != nil {
			errorLabel.SetText(updateErr.Error())
			return
		}
		dialog.Accept()
	})

	dialog.Exec()
}

// bindColorSwatch turns a form button into a color-picker swatch: it shows
// the current color (the one property the theme cannot know), opens the
// picker titled title, and reports through onPicked.
func bindColorSwatch(swatch *qt.QPushButton, dialogParent *qt.QWidget, initial color.NRGBA, title string, onPicked func(color.NRGBA)) {
	current := initial
	styleSwatch := func() {
		swatch.SetStyleSheet("background-color: " + cssRGB(current) + ";")
	}
	styleSwatch()
	swatch.OnClicked(func() {
		picked := qt.QColorDialog_GetColor3(qColor(current), dialogParent, title)
		if !picked.IsValid() {
			return
		}
		current = color.NRGBA{R: uint8(picked.Red()), G: uint8(picked.Green()), B: uint8(picked.Blue()), A: 0xff}
		styleSwatch()
		onPicked(current)
	})
}

// parseWhole parses a non-negative whole number.
func parseWhole(text string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("not a non-negative whole number: %q", text)
	}
	return value, nil
}

// hexOf renders a color as "#RRGGBB".
func hexOf(src color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", src.R, src.G, src.B)
}
