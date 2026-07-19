package ui

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// showAboutDialog opens a modal with the app's logo, version, source and
// license links, and the data-source disclaimer.
func (app *App) showAboutDialog() {
	logo := canvas.NewImageFromResource(appIcon)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(constants.AboutLogoSize, constants.AboutLogoSize))

	name := widget.NewLabelWithStyle(constants.AppName, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	version := widget.NewLabelWithStyle(constants.LabelVersionPrefix+constants.AppVersion, fyne.TextAlignCenter, fyne.TextStyle{})

	desc := widget.NewLabel(constants.AboutDescription)
	desc.Alignment = fyne.TextAlignCenter

	repo, _ := url.Parse(constants.RepoURL)
	license, _ := url.Parse(constants.LicenseURL)
	links := container.NewHBox(
		layout.NewSpacer(),
		widget.NewHyperlink(constants.LabelSourceLink, repo),
		widget.NewHyperlink(constants.LabelLicenseLink, license),
		layout.NewSpacer(),
	)

	disclaimer := widget.NewLabel(constants.AboutDisclaimer)
	disclaimer.Wrapping = fyne.TextWrapWord
	disclaimer.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		container.NewCenter(logo),
		name,
		version,
		desc,
		links,
		disclaimer,
	)
	dlg := dialog.NewCustom(constants.TitleAbout, constants.LabelClose, content, app.win)
	dlg.Resize(fyne.NewSize(constants.AboutDialogWidth, 0))
	dlg.Show()
}
