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
// nothing persists until Save; the caller restores the persisted background
// when the dialog closes unsaved.
func showSettingsDialog(parent *qt.QWidget, store *config.Store, previewBackground func(color.NRGBA, float64)) {
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
		label.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorNeutral) + ";")
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
	generalLayout.AddStretch()
	pages.AddWidget(generalPage)

	// --- Appearance (edits preview live) ---
	appearancePage, appearanceLayout := newPage()
	background, ok := parseHexColor(cfg.Background.Color)
	if !ok {
		background, _ = parseHexColor(constants.DefaultBackgroundColor)
	}
	opacityPercent := int(cfg.Background.Opacity*constants.PercentMax + 0.5)
	appearanceLayout.AddWidget(fieldLabel(appearancePage, constants.LabelBackgroundColor).QWidget)
	swatch := qt.NewQPushButton(appearancePage)
	swatch.SetFixedSize2(int(constants.SwatchWidth), int(constants.SwatchHeight))
	swatch.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	styleSwatch := func() {
		swatch.SetStyleSheet(fmt.Sprintf(
			"QPushButton { background-color: %s; border: 1px solid %s; border-radius: %dpx; }",
			cssRGB(background), cssRGB(constants.ColorAxis), int(constants.PanelCornerRadius)))
	}
	styleSwatch()
	appearanceLayout.AddWidget(swatch.QWidget)
	appearanceLayout.AddWidget(fieldLabel(appearancePage, constants.LabelBackgroundOpacity).QWidget)
	opacitySlider := qt.NewQSlider4(qt.Horizontal, appearancePage)
	opacitySlider.SetMinimum(0)
	opacitySlider.SetMaximum(int(constants.PercentMax))
	opacitySlider.SetValue(opacityPercent)
	appearanceLayout.AddWidget(opacitySlider.QWidget)
	appearanceLayout.AddStretch()
	pages.AddWidget(appearancePage)

	preview := func() {
		previewBackground(background, float64(opacitySlider.Value())/constants.PercentMax)
	}
	swatch.OnClicked(func() {
		picked := qt.QColorDialog_GetColor3(qColor(background), dialog.QWidget, constants.TitleColorPicker)
		if !picked.IsValid() {
			return
		}
		background = color.NRGBA{R: uint8(picked.Red()), G: uint8(picked.Green()), B: uint8(picked.Blue()), A: 0xff}
		styleSwatch()
		preview()
	})
	opacitySlider.OnValueChanged(func(value int) { preview() })

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
			conf.Background.Color = hexOf(background)
			conf.Background.Opacity = float64(opacitySlider.Value()) / constants.PercentMax
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
