// Command qt-spike is a throwaway capability testbed for the Qt UI rewrite.
// It exercises everything that was broken or risky on the previous stacks,
// in one frameless translucent window on KWin/Wayland + NVIDIA:
//
//   - real window translucency (rounded card over the desktop)
//   - LIVE opacity changes via a slider (broken on Wails/WebKitGTK)
//   - an animated, antialiased QPainter line chart repainting 10x/second
//   - a scroll-heavy list (scrolling triggered stale-buffer glitches on Wails)
//   - native window drag (StartSystemMove) on a frameless window
//
// Drag the title area to move, Esc to quit. Delete once the verdict is in.
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"

	qt "github.com/mappu/miqt/qt6"
)

const (
	windowWidth    = 460
	windowHeight   = 560
	chartHeight    = 160
	listRows       = 100
	tickMillis     = 100
	maxPoints      = 120
	initialOpacity = 60
)

func cardStyle(opacityPercent int) string {
	return fmt.Sprintf("#card { background-color: rgba(23, 23, 24, %d%%); border-radius: 12px; }", opacityPercent)
}

func main() {
	qt.NewQApplication(os.Args)

	window := qt.NewQWidget2()
	window.SetWindowFlags(qt.FramelessWindowHint)
	window.SetAttribute(qt.WA_TranslucentBackground)
	window.Resize(windowWidth, windowHeight)

	card := qt.NewQWidget(window)
	card.SetObjectName(*qt.NewQAnyStringView3("card"))
	card.SetGeometry(0, 0, windowWidth, windowHeight)
	card.SetStyleSheet(cardStyle(initialOpacity))

	layout := qt.NewQVBoxLayout(card)

	title := qt.NewQLabel5("simpleStonks Qt capability spike — drag me, Esc quits", card)
	title.SetStyleSheet("color: #e6e6e6; font-weight: bold; background: transparent;")
	layout.AddWidget(title.QWidget)

	// Animated antialiased chart: repaints on a 10 Hz timer.
	values := make([]float64, 0, maxPoints)
	phase := 0.0
	chart := qt.NewQWidget(card)
	chart.SetMinimumHeight(chartHeight)
	chart.SetStyleSheet("background: rgba(30, 30, 36, 60%); border-radius: 6px;")
	lineColor := qt.NewQColor3(0x26, 0xa6, 0x5b)
	pen := qt.NewQPen3(lineColor)
	pen.SetWidthF(1.5)
	chart.OnPaintEvent(func(super func(event *qt.QPaintEvent), event *qt.QPaintEvent) {
		if len(values) < 2 {
			return
		}
		painter := qt.NewQPainter2(chart.QPaintDevice)
		defer painter.End()
		painter.SetRenderHint(qt.QPainter__Antialiasing)
		painter.SetPenWithPen(pen)
		width := float64(chart.Width())
		height := float64(chart.Height())
		stepX := width / float64(maxPoints-1)
		for idx := 1; idx < len(values); idx++ {
			lineSeg := qt.NewQLineF3(
				float64(idx-1)*stepX, height*(0.5-0.4*values[idx-1]),
				float64(idx)*stepX, height*(0.5-0.4*values[idx]),
			)
			painter.DrawLine(lineSeg)
		}
	})
	layout.AddWidget(chart)

	ticker := qt.NewQTimer()
	ticker.OnTimeout(func() {
		phase += 0.15
		values = append(values, math.Sin(phase)+0.3*rand.Float64())
		if len(values) > maxPoints {
			values = values[1:]
		}
		chart.Update()
	})
	ticker.Start(tickMillis)

	// Scroll stress: flinging this list is what glitched WebKitGTK.
	list := qt.NewQListWidget(card)
	list.SetStyleSheet("background: rgba(30, 30, 36, 60%); color: #e6e6e6; border-radius: 6px;")
	for row := 1; row <= listRows; row++ {
		list.AddItem(fmt.Sprintf("Row %03d — scroll me hard and watch for artifacts", row))
	}
	layout.AddWidget(list.QWidget)

	// Live translucency: the capability WebKitGTK could not deliver.
	sliderLabel := qt.NewQLabel5(fmt.Sprintf("Card opacity: %d%%", initialOpacity), card)
	sliderLabel.SetStyleSheet("color: #8a8a8a; background: transparent;")
	layout.AddWidget(sliderLabel.QWidget)
	slider := qt.NewQSlider4(qt.Horizontal, card)
	slider.SetMinimum(10)
	slider.SetMaximum(100)
	slider.SetValue(initialOpacity)
	slider.OnValueChanged(func(value int) {
		card.SetStyleSheet(cardStyle(value))
		sliderLabel.SetText(fmt.Sprintf("Card opacity: %d%%", value))
	})
	layout.AddWidget(slider.QWidget)

	window.OnKeyPressEvent(func(super func(event *qt.QKeyEvent), event *qt.QKeyEvent) {
		if event.Key() == int(qt.Key_Escape) {
			window.Close()
			return
		}
		super(event)
	})
	window.OnMousePressEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		window.WindowHandle().StartSystemMove()
	})
	window.OnResizeEvent(func(super func(event *qt.QResizeEvent), event *qt.QResizeEvent) {
		card.SetGeometry(0, 0, window.Width(), window.Height())
		super(event)
	})

	window.Show()
	qt.QApplication_Exec()
}
