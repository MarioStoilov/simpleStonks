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

	split := qt.NewQHBoxLayout2()

	// Section navigation on the left, one form page per section on the right.
	sections := qt.NewQVBoxLayout2()
	pages := qt.NewQStackedWidget(dialog.QWidget)
	sectionNames := []string{constants.SectionGeneral, constants.SectionAppearance, constants.SectionLogging}
	sectionButtons := make([]*qt.QPushButton, 0, len(sectionNames))
	styleSections := func(active int) {
		for idx, button := range sectionButtons {
			button.SetStyleSheet(dialogButtonStyle(idx == active))
		}
	}
	for idx, name := range sectionNames {
		pageIdx := idx // capture per iteration
		button := qt.NewQPushButton5(name, dialog.QWidget)
		button.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
		button.OnClicked(func() {
			pages.SetCurrentIndex(pageIdx)
			styleSections(pageIdx)
		})
		sectionButtons = append(sectionButtons, button)
		sections.AddWidget(button.QWidget)
	}
	sections.AddStretch()
	split.AddLayout(sections.QLayout)
	split.AddWidget2(pages.QWidget, 1)
	body.AddLayout2(split.QLayout, 1)

	newPage := func() (*qt.QWidget, *qt.QVBoxLayout) {
		page := qt.NewQWidget(dialog.QWidget)
		page.SetStyleSheet("background: transparent;")
		pageLayout := qt.NewQVBoxLayout(page)
		return page, pageLayout
	}
	fieldLabel := func(page *qt.QWidget, text string) *qt.QLabel {
		label := qt.NewQLabel5(text, page)
		label.SetStyleSheet(fmt.Sprintf(
			"QLabel { background: transparent; color: %s; } QLabel:disabled { color: %s; }",
			cssRGB(constants.ColorNeutral), cssRGBA(constants.ColorNeutral, constants.SwatchDisabledAlpha)))
		return label
	}

	// --- General ---
	generalPage, generalLayout := newPage()
	generalLayout.AddWidget(fieldLabel(generalPage, constants.LabelDefaultRange).QWidget)
	rangeBox := qt.NewQComboBox(generalPage)
	rangeBox.SetStyleSheet(inputStyle())
	for _, rangeOption := range model.Ranges {
		rangeBox.AddItem(string(rangeOption))
	}
	rangeBox.SetCurrentText(string(cfg.DefaultRange))
	generalLayout.AddWidget(rangeBox.QWidget)
	generalLayout.AddWidget(fieldLabel(generalPage, constants.LabelRefreshInterval).QWidget)
	refreshEdit := qt.NewQLineEdit4(strconv.Itoa(int(cfg.RefreshInterval/time.Second)), generalPage)
	refreshEdit.SetStyleSheet(inputStyle())
	generalLayout.AddWidget(refreshEdit.QWidget)
	separator := qt.NewQFrame(generalPage)
	separator.SetFixedHeight(int(constants.HairlineWidth))
	separator.SetStyleSheet("background-color: " + cssRGB(constants.ColorAxis) + ";")
	generalLayout.AddWidget(separator.QWidget)
	extendedBox := qt.NewQCheckBox4(constants.LabelExtendedHours, generalPage)
	extendedBox.SetStyleSheet(checkBoxStyle())
	extendedBox.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	extendedBox.SetToolTip(constants.TipExtendedHours)
	extendedBox.SetChecked(cfg.ExtendedHours)
	generalLayout.AddWidget(extendedBox.QWidget)
	generalLayout.AddStretch()
	pages.AddWidget(generalPage)

	// --- Appearance (edits preview live) ---
	appearancePage, appearanceLayout := newPage()
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

	// Window background.
	opacityPercent := int(cfg.Background.Opacity*constants.PercentMax + 0.5)
	appearanceLayout.AddWidget(fieldLabel(appearancePage, constants.LabelBackgroundColor).QWidget)
	swatch := colorSwatch(appearancePage, dialog.QWidget, background, constants.TitleColorPicker, func(picked color.NRGBA) {
		background = picked
		preview()
	})
	appearanceLayout.AddWidget(swatch.QWidget)
	appearanceLayout.AddWidget(fieldLabel(appearancePage, constants.LabelBackgroundOpacity).QWidget)
	opacitySlider := qt.NewQSlider4(qt.Horizontal, appearancePage)
	opacitySlider.SetMinimum(0)
	opacitySlider.SetMaximum(int(constants.PercentMax))
	opacitySlider.SetValue(opacityPercent)
	appearanceLayout.AddWidget(opacitySlider.QWidget)

	chartSeparator := qt.NewQFrame(appearancePage)
	chartSeparator.SetFixedHeight(int(constants.HairlineWidth))
	chartSeparator.SetStyleSheet("background-color: " + cssRGB(constants.ColorAxis) + ";")
	appearanceLayout.AddWidget(chartSeparator.QWidget)

	// Chart plot styling: background, checkered grid, up/down area fill.
	appearanceLayout.AddWidget(fieldLabel(appearancePage, constants.LabelChartBackground).QWidget)
	chartSwatch := colorSwatch(appearancePage, dialog.QWidget, chartBackground, constants.TitleChartColorPicker, func(picked color.NRGBA) {
		chartBackground = picked
		chartPreview()
	})
	appearanceLayout.AddWidget(chartSwatch.QWidget)
	gridBox := qt.NewQCheckBox4(constants.LabelChartGrid, appearancePage)
	gridBox.SetStyleSheet(checkBoxStyle())
	gridBox.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	gridBox.SetToolTip(constants.TipChartGrid)
	gridBox.SetChecked(cfg.Chart.Grid)
	appearanceLayout.AddWidget(gridBox.QWidget)
	gridSizeLabel := fieldLabel(appearancePage, constants.LabelChartGridSize)
	appearanceLayout.AddWidget(gridSizeLabel.QWidget)
	gridSizeEdit := qt.NewQLineEdit4(strconv.Itoa(cfg.Chart.GridSize), appearancePage)
	gridSizeEdit.SetStyleSheet(inputStyle())
	appearanceLayout.AddWidget(gridSizeEdit.QWidget)
	gridColorLabel := fieldLabel(appearancePage, constants.LabelChartGridColor)
	appearanceLayout.AddWidget(gridColorLabel.QWidget)
	gridSwatch := colorSwatch(appearancePage, dialog.QWidget, gridColor, constants.TitleGridColorPicker, func(picked color.NRGBA) {
		gridColor = picked
		chartPreview()
	})
	appearanceLayout.AddWidget(gridSwatch.QWidget)
	fillBox := qt.NewQCheckBox4(constants.LabelChartFill, appearancePage)
	fillBox.SetStyleSheet(checkBoxStyle())
	fillBox.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	fillBox.SetToolTip(constants.TipChartFill)
	fillBox.SetChecked(cfg.Chart.Fill)
	appearanceLayout.AddWidget(fillBox.QWidget)
	fillOpacityLabel := fieldLabel(appearancePage, constants.LabelChartFillOpacity)
	appearanceLayout.AddWidget(fillOpacityLabel.QWidget)
	fillSlider := qt.NewQSlider4(qt.Horizontal, appearancePage)
	fillSlider.SetMinimum(0)
	fillSlider.SetMaximum(int(constants.PercentMax))
	fillSlider.SetValue(int(cfg.Chart.FillOpacity*constants.PercentMax + 0.5))
	appearanceLayout.AddWidget(fillSlider.QWidget)
	appearanceLayout.AddStretch()
	pages.AddWidget(appearancePage)

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

	// --- Logging ---
	loggingPage, loggingLayout := newPage()
	loggingLayout.AddWidget(fieldLabel(loggingPage, constants.LabelLogLevel).QWidget)
	levelBox := qt.NewQComboBox(loggingPage)
	levelBox.SetStyleSheet(inputStyle())
	for _, level := range logLevels {
		levelBox.AddItem(string(level))
	}
	levelBox.SetCurrentText(string(cfg.Logging.Level))
	loggingLayout.AddWidget(levelBox.QWidget)
	loggingLayout.AddWidget(fieldLabel(loggingPage, constants.LabelLogFile).QWidget)
	fileEdit := qt.NewQLineEdit4(cfg.Logging.File, loggingPage)
	fileEdit.SetPlaceholderText(config.DefaultLogPath())
	fileEdit.SetStyleSheet(inputStyle())
	loggingLayout.AddWidget(fileEdit.QWidget)
	loggingLayout.AddWidget(fieldLabel(loggingPage, constants.LabelLogMaxSize).QWidget)
	maxSizeEdit := qt.NewQLineEdit4(strconv.Itoa(cfg.Logging.MaxSizeMB), loggingPage)
	maxSizeEdit.SetStyleSheet(inputStyle())
	loggingLayout.AddWidget(maxSizeEdit.QWidget)
	loggingLayout.AddWidget(fieldLabel(loggingPage, constants.LabelLogArchives).QWidget)
	archivesEdit := qt.NewQLineEdit4(strconv.Itoa(cfg.Logging.MaxArchives), loggingPage)
	archivesEdit.SetStyleSheet(inputStyle())
	loggingLayout.AddWidget(archivesEdit.QWidget)
	loggingLayout.AddStretch()
	pages.AddWidget(loggingPage)

	styleSections(0)

	errorLabel := qt.NewQLabel(dialog.QWidget)
	errorLabel.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorDown) + ";")
	body.AddWidget(errorLabel.QWidget)

	actions := qt.NewQHBoxLayout2()
	actions.AddStretch()
	cancelButton := qt.NewQPushButton5(constants.LabelCancel, dialog.QWidget)
	cancelButton.SetStyleSheet(dialogButtonStyle(false))
	cancelButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	cancelButton.OnClicked(func() { dialog.Reject() })
	actions.AddWidget(cancelButton.QWidget)
	saveButton := qt.NewQPushButton5(constants.LabelSave, dialog.QWidget)
	saveButton.SetStyleSheet(dialogButtonStyle(true))
	saveButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	saveButton.OnClicked(func() {
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
	actions.AddWidget(saveButton.QWidget)
	body.AddLayout(actions.QLayout)

	dialog.Exec()
}

// colorSwatch builds a color-picker button: a swatch showing the current
// color that opens the picker (titled title, parented to dialogParent) and
// restyles itself and reports through onPicked when a color is chosen.
func colorSwatch(parent *qt.QWidget, dialogParent *qt.QWidget, initial color.NRGBA, title string, onPicked func(color.NRGBA)) *qt.QPushButton {
	swatch := qt.NewQPushButton(parent)
	swatch.SetFixedSize2(int(constants.SwatchWidth), int(constants.SwatchHeight))
	swatch.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	current := initial
	styleSwatch := func() {
		swatch.SetStyleSheet(fmt.Sprintf(
			"QPushButton { background-color: %s; border: 1px solid %s; border-radius: %dpx; }"+
				" QPushButton:disabled { background-color: %s; border-color: %s; }",
			cssRGB(current), cssRGB(constants.ColorAxis), int(constants.PanelCornerRadius),
			cssRGBA(current, constants.SwatchDisabledAlpha), cssRGB(constants.ColorDisabledBg)))
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
	return swatch
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
