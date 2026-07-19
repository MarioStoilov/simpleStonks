package ui

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// repoURL is the public home of the project, linked from the About dialog.
const repoURL = "https://github.com/MarioStoilov/simpleStonks"

// showAboutDialog opens a modal with the app's logo, version, source and
// license links, and the data-source disclaimer.
func (app *App) showAboutDialog() {
	logo := canvas.NewImageFromResource(appIcon)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(72, 72))

	name := widget.NewLabelWithStyle("simpleStonks", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	version := widget.NewLabelWithStyle("Version "+appVersion, fyne.TextAlignCenter, fyne.TextStyle{})

	desc := widget.NewLabel("A bare-bones, Yahoo-Finance-style stock tracker —\nvibe coded out of boredom.")
	desc.Alignment = fyne.TextAlignCenter

	repo, _ := url.Parse(repoURL)
	license, _ := url.Parse(repoURL + "/blob/main/LICENSE")
	links := container.NewHBox(
		layout.NewSpacer(),
		widget.NewHyperlink("Source on GitHub", repo),
		widget.NewHyperlink("MIT license", license),
		layout.NewSpacer(),
	)

	disclaimer := widget.NewLabel("Market data comes from unofficial, keyless Yahoo Finance " +
		"endpoints and can lag, break, or disappear. Nothing here is financial advice.")
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
	dlg := dialog.NewCustom("About simpleStonks", "Close", content, app.win)
	dlg.Resize(fyne.NewSize(460, 0))
	dlg.Show()
}
