package qtui

import (
	"fmt"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/assets"
	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// showAboutDialog shows the About card, mirroring the Fyne layout: centered
// logo, app name and version, the short description, source/license links,
// and the data disclaimer.
func showAboutDialog(parent *qt.QWidget) {
	dialog, body := newCardDialog(parent, constants.TitleAbout)
	dialog.SetFixedWidth(int(constants.AboutDialogWidth))

	centered := func(label *qt.QLabel) *qt.QLabel {
		label.SetAlignment(qt.AlignHCenter)
		label.SetWordWrap(true)
		return label
	}

	// Logo, rendered from the embedded SVG (Qt's SVG image plugin).
	logo := qt.NewQPixmap()
	if logo.LoadFromDataWithData(assets.IconSVG) {
		logoLabel := qt.NewQLabel(dialog.QWidget)
		logoLabel.SetPixmap(logo.Scaled3(
			int(constants.AboutLogoSize), int(constants.AboutLogoSize),
			qt.KeepAspectRatio, qt.SmoothTransformation))
		logoLabel.SetAlignment(qt.AlignHCenter)
		logoLabel.SetStyleSheet("background: transparent;")
		body.AddWidget(logoLabel.QWidget)
	}

	nameLabel := qt.NewQLabel5(constants.AppName, dialog.QWidget)
	nameLabel.SetStyleSheet("background: transparent; font-weight: 600; font-size: 18px;")
	body.AddWidget(centered(nameLabel).QWidget)

	versionLabel := qt.NewQLabel5(constants.LabelVersionPrefix+constants.AppVersion, dialog.QWidget)
	versionLabel.SetStyleSheet("background: transparent; color: " + cssRGB(constants.ColorNeutral) + ";")
	body.AddWidget(centered(versionLabel).QWidget)

	descriptionLabel := qt.NewQLabel5(constants.AboutDescription, dialog.QWidget)
	descriptionLabel.SetStyleSheet("background: transparent;")
	body.AddWidget(centered(descriptionLabel).QWidget)

	linkStyle := cssRGB(constants.ColorSelected)
	linksLabel := qt.NewQLabel5(fmt.Sprintf(
		`<a style="color: %s;" href="%s">%s</a> &nbsp;·&nbsp; <a style="color: %s;" href="%s">%s</a>`,
		linkStyle, constants.RepoURL, constants.LabelSourceLink,
		linkStyle, constants.LicenseURL, constants.LabelLicenseLink), dialog.QWidget)
	linksLabel.SetTextFormat(qt.RichText)
	linksLabel.SetOpenExternalLinks(true)
	linksLabel.SetStyleSheet("background: transparent;")
	body.AddWidget(centered(linksLabel).QWidget)

	disclaimerLabel := qt.NewQLabel5(constants.AboutDisclaimer, dialog.QWidget)
	disclaimerLabel.SetStyleSheet(fmt.Sprintf(constants.StyleSmallText,
		cssRGB(constants.ColorNeutral), int(constants.NameTextSize)))
	body.AddWidget(centered(disclaimerLabel).QWidget)

	actions := qt.NewQHBoxLayout2()
	actions.AddStretch()
	closeButton := qt.NewQPushButton5(constants.LabelClose, dialog.QWidget)
	closeButton.SetStyleSheet(dialogButtonStyle(true))
	closeButton.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	closeButton.OnClicked(func() { dialog.Accept() })
	actions.AddWidget(closeButton.QWidget)
	actions.AddStretch()
	body.AddLayout(actions.QLayout)

	dialog.Exec()
}
