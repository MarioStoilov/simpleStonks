package qtui

import (
	"fmt"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/assets"
	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// showAboutDialog shows the About card (about.ui): centered logo, app name
// and version, the short description, source/license links, and the data
// disclaimer.
func showAboutDialog(parent *qt.QWidget) {
	dialog, body := newCardDialog(parent, constants.TitleAbout)
	dialog.SetFixedWidth(int(constants.AboutDialogWidth))
	loaded := loadForm(aboutForm)
	body.AddWidget(loaded.root)

	// Logo, rendered from the embedded SVG through Qt\'s SVG image plugin;
	// the label stays laid out but empty when the plugin is missing.
	logo := qt.NewQPixmap()
	if logo.LoadFromDataWithData(assets.IconSVG) {
		loaded.label("logoLabel").SetPixmap(logo.Scaled3(
			int(constants.AboutLogoSize), int(constants.AboutLogoSize),
			qt.KeepAspectRatio, qt.SmoothTransformation))
	}

	loaded.label("nameLabel").SetText(constants.AppName)
	loaded.label("versionLabel").SetText(constants.LabelVersionPrefix + constants.AppVersion)
	loaded.label("descriptionLabel").SetText(constants.AboutDescription)
	loaded.label("disclaimerLabel").SetText(constants.AboutDisclaimer)

	// The links carry their own color: a QLabel\'s rich text ignores the
	// theme\'s color for anchors.
	linkColor := cssRGB(constants.ColorSelected)
	loaded.label("linksLabel").SetText(fmt.Sprintf(
		`<a style="color: %s;" href="%s">%s</a> &nbsp;·&nbsp; <a style="color: %s;" href="%s">%s</a>`,
		linkColor, constants.RepoURL, constants.LabelSourceLink,
		linkColor, constants.LicenseURL, constants.LabelLicenseLink))

	loaded.button("closeButton").OnClicked(func() { dialog.Accept() })
	dialog.Exec()
}
