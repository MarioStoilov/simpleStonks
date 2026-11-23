package qtui

import (
	"image/color"
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
)

// tooltipBackground shows a tooltip for a button under a themed card in the
// given color scheme and samples the tooltip's background color. Tooltips
// run on timers, so the work happens on the UI thread.
func tooltipBackground(t *testing.T, scheme string) (background color.NRGBA, shown bool) {
	t.Helper()
	mainthread.Wait(func() {
		card := qt.NewQFrame2()
		defer card.Delete()
		card.SetStyleSheet(themeStyleSheet)
		setState(card.QWidget, colorSchemeProperty, scheme)
		button := qt.NewQPushButton(card.QWidget)
		card.Show()

		qt.QToolTip_ShowText2(qt.NewQPoint2(10, 10), "tooltip", button.QWidget)
		defer qt.QToolTip_HideText()
		for _, top := range qt.QApplication_TopLevelWidgets() {
			if top.MetaObject().ClassName() != "QTipLabel" || !top.IsVisible() {
				continue
			}
			pixel := top.Grab().ToImage().PixelColor(3, 3)
			background = color.NRGBA{R: uint8(pixel.Red()), G: uint8(pixel.Green()), B: uint8(pixel.Blue()), A: 0xff}
			shown = true
			return
		}
	})
	return background, shown
}

// TestTooltipFollowsTheColorScheme guards the tooltip looks: white in the
// light scheme (the default) and gray in the dark one — never the desktop's
// own tooltip background, which the theme's light label text is unreadable
// on.
func TestTooltipFollowsTheColorScheme(t *testing.T) {
	cases := []struct {
		scheme string
		want   color.NRGBA
	}{
		{colorSchemeLight, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
		{colorSchemeDark, color.NRGBA{R: 0x3a, G: 0x3a, B: 0x40, A: 0xff}},
	}
	for _, testCase := range cases {
		got, shown := tooltipBackground(t, testCase.scheme)
		if !shown {
			t.Fatalf("%s: no tooltip window was shown", testCase.scheme)
		}
		if got != testCase.want {
			t.Errorf("%s tooltip background = %v, want %v", testCase.scheme, got, testCase.want)
		}
	}
}
