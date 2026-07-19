package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/chartmath"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
)

// chart is a custom Fyne widget that plots a price series as a line, colored by
// the caller (up/down). With hoverReadout enabled (the detail view's expanded
// chart), hovering the plot marks the nearest data point with a dot, a dashed
// vertical guide, its time/date on the x axis, and a tooltip showing the price
// plus its % change versus the previous close.
type chart struct {
	widget.BaseWidget
	series model.Series
	line   color.Color

	hoverReadout bool // enables the hover marker/guide/tooltip

	renderer *chartRenderer // for lightweight hover updates
	hovering bool
	hoverAt  fyne.Position
	onHover  func(bool) // optional: notifies the enclosing widget (e.g. a tile)
}

// newChart constructs an empty chart widget.
func newChart() *chart {
	chartWidget := &chart{line: constants.ColorNeutral}
	chartWidget.ExtendBaseWidget(chartWidget)
	return chartWidget
}

// SetSeries updates the plotted data and redraws.
func (chart *chart) SetSeries(series model.Series) {
	chart.series = series
	chart.Refresh()
}

// SetColor sets the line color used on the next redraw.
func (chart *chart) SetColor(col color.Color) { chart.line = col }

// MouseIn implements desktop.Hoverable.
func (chart *chart) MouseIn(event *desktop.MouseEvent) { chart.MouseMoved(event) }

// MouseMoved implements desktop.Hoverable: it tracks the pointer and updates
// the hover marker without rebuilding the chart. Since the chart is the
// topmost hoverable, it also forwards enter/leave to onHover so an enclosing
// tile can render its own hover effect.
func (chart *chart) MouseMoved(event *desktop.MouseEvent) {
	if !chart.hovering && chart.onHover != nil {
		chart.onHover(true)
	}
	chart.hovering = true
	chart.hoverAt = event.Position
	if chart.renderer != nil {
		chart.renderer.updateHover()
	}
}

// MouseOut implements desktop.Hoverable.
func (chart *chart) MouseOut() {
	chart.hovering = false
	if chart.onHover != nil {
		chart.onHover(false)
	}
	if chart.renderer != nil {
		chart.renderer.updateHover()
	}
}

// CreateRenderer implements fyne.Widget.
func (chart *chart) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(constants.ColorChartBg)
	dot := canvas.NewCircle(constants.ColorNeutral)
	dot.StrokeColor = theme.Color(theme.ColorNameForeground)
	dot.StrokeWidth = constants.HairlineWidth
	tipBg := canvas.NewRectangle(constants.ColorHover)
	tipBg.CornerRadius = constants.PanelCornerRadius
	tipText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	tipText.TextSize = constants.AxisTextSize
	tipPct := canvas.NewText("", constants.ColorNeutral)
	tipPct.TextSize = constants.AxisTextSize
	timeBg := canvas.NewRectangle(constants.ColorHover)
	timeBg.CornerRadius = constants.PanelCornerRadius
	timeLbl := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	timeLbl.TextSize = constants.AxisTextSize
	renderer := &chartRenderer{
		chart: chart, background: background, dot: dot, tipBg: tipBg, tipText: tipText,
		tipPct: tipPct, timeBg: timeBg, timeLbl: timeLbl,
		objects: []fyne.CanvasObject{background},
	}
	chart.renderer = renderer
	return renderer
}

type chartRenderer struct {
	chart      *chart
	background *canvas.Rectangle
	objects    []fyne.CanvasObject

	// Hover readout state (only populated when chart.hoverReadout is set):
	// the plotted points (absolute coordinates) and their values, cached by
	// rebuild, plus the marker/guide/tooltip canvas objects.
	hoverPts   []chartmath.Point
	hoverVals  []float64
	plotH      float32        // plot-area height of the last rebuild
	vDashes    []*canvas.Line // pooled segments of the vertical guide
	dot        *canvas.Circle
	tipBg      *canvas.Rectangle
	tipText    *canvas.Text // hovered price
	tipPct     *canvas.Text // % change versus the previous close
	timeBg     *canvas.Rectangle
	timeLbl    *canvas.Text // hovered time/date on the x axis
	hoverShown bool
}

func (renderer *chartRenderer) Layout(size fyne.Size) {
	renderer.background.Resize(size)
	renderer.rebuild(size)
}

func (renderer *chartRenderer) MinSize() fyne.Size {
	return fyne.NewSize(constants.ChartMinWidth, constants.ChartMinHeight)
}

func (renderer *chartRenderer) Refresh() {
	renderer.rebuild(renderer.chart.Size())
	canvas.Refresh(renderer.chart)
}

func (renderer *chartRenderer) Objects() []fyne.CanvasObject { return renderer.objects }

func (renderer *chartRenderer) Destroy() {}

// rebuild regenerates the line segments and axis labels for the current series
// and size. The plot is inset by a left margin (price labels) and a bottom
// margin (time labels); margins collapse to zero when there is nothing to draw.
func (renderer *chartRenderer) rebuild(size fyne.Size) {
	objs := []fyne.CanvasObject{renderer.background}
	renderer.hoverPts, renderer.hoverVals, renderer.vDashes = nil, nil, nil
	defer func() {
		if len(renderer.hoverPts) > 0 { // keep the hover guide/marker/tooltip on top
			for _, segment := range renderer.vDashes {
				objs = append(objs, segment)
			}
			objs = append(objs, renderer.timeBg, renderer.timeLbl,
				renderer.tipBg, renderer.tipText, renderer.tipPct, renderer.dot)
		}
		renderer.objects = objs
	}()
	defer renderer.updateHover() // runs first: re-aim the marker at the new geometry

	series := renderer.chart.series
	values := chartmath.ClosesOf(series)
	if len(values) < 2 {
		return
	}
	low, high := values[0], values[0]
	for _, value := range values {
		if value < low {
			low = value
		}
		if value > high {
			high = value
		}
	}
	// Widen the scale to keep the previous-close reference line in view.
	prevClose := series.PreviousClose
	if prevClose > 0 {
		if prevClose < low {
			low = prevClose
		}
		if prevClose > high {
			high = prevClose
		}
	}

	labelH := fyne.MeasureText("0", constants.AxisTextSize, fyne.TextStyle{}).Height
	bottomMargin := labelH + constants.AxisGap
	plotH := size.Height - bottomMargin

	// Price (y) labels: as many evenly spaced reference values as the height
	// allows, plus the previous close; a flat scale gets a single label. The
	// left margin fits the widest label.
	type yLabel struct {
		value float64
		text  string
		width float32
	}
	var labels []yLabel
	if low == high {
		labels = []yLabel{{value: low, text: chartmath.FormatAxisPrice(low)}}
	} else {
		count := int(plotH/constants.YTickSpacing) + 1
		if count < 2 {
			count = 2
		}
		if count > constants.MaxYTicks {
			count = constants.MaxYTicks
		}
		for _, value := range chartmath.YTicks(low, high, count) {
			labels = append(labels, yLabel{value: value, text: chartmath.FormatAxisPrice(value)})
		}
	}
	prevIdx := -1 // index of the previous-close label, which wins collisions
	if prevClose > 0 && low != high {
		labels = append(labels, yLabel{value: prevClose, text: chartmath.FormatAxisPrice(prevClose)})
		prevIdx = len(labels) - 1
	}
	var leftMargin float32
	for idx := range labels {
		labels[idx].width = fyne.MeasureText(labels[idx].text, constants.AxisTextSize, fyne.TextStyle{}).Width
		if labels[idx].width > leftMargin {
			leftMargin = labels[idx].width
		}
	}
	leftMargin += constants.AxisGap

	plotW := size.Width - leftMargin
	pts := chartmath.PlotPath(values, chartmath.XFracs(series), plotW, plotH, constants.ChartPadding, low, high)
	if pts == nil { // too small for margins: fall back to the bare line
		leftMargin, bottomMargin = 0, 0
		plotW, plotH = size.Width, size.Height
		pts = chartmath.PlotPath(values, chartmath.XFracs(series), plotW, plotH, constants.ChartPadding, low, high)
	}

	// Cache the plotted points (absolute coordinates) for the hover readout
	// and pool the vertical guide's dash segments for the plot height.
	renderer.plotH = plotH
	if pts != nil && renderer.chart.hoverReadout {
		renderer.hoverVals = values
		renderer.hoverPts = make([]chartmath.Point, len(pts))
		for idx, point := range pts {
			renderer.hoverPts[idx] = chartmath.Point{X: point.X + leftMargin, Y: point.Y}
		}
		segments := int((plotH-2*constants.ChartPadding)/(constants.DashLen+constants.DashGap)) + 1
		renderer.vDashes = make([]*canvas.Line, 0, segments)
		for segIdx := 0; segIdx < segments; segIdx++ {
			segment := canvas.NewLine(constants.ColorAxis)
			segment.StrokeWidth = constants.HairlineWidth
			segment.Hide()
			renderer.vDashes = append(renderer.vDashes, segment)
		}
	}

	// Dashed reference line at the previous interval's close (under the series).
	if prevClose > 0 && pts != nil {
		posY := chartmath.YFor(prevClose, low, high, plotH, constants.ChartPadding)
		right := leftMargin + plotW - constants.ChartPadding
		for posX := leftMargin + constants.ChartPadding; posX < right; posX += constants.DashLen + constants.DashGap {
			segEnd := posX + constants.DashLen
			if segEnd > right {
				segEnd = right
			}
			segment := canvas.NewLine(constants.ColorAxis)
			segment.StrokeWidth = constants.HairlineWidth
			segment.Position1 = fyne.NewPos(posX, posY)
			segment.Position2 = fyne.NewPos(segEnd, posY)
			objs = append(objs, segment)
		}
	}

	for idx := 1; idx < len(pts); idx++ {
		segment := canvas.NewLine(renderer.chart.line)
		segment.StrokeWidth = constants.ChartLineWidth
		segment.Position1 = fyne.NewPos(pts[idx-1].X+leftMargin, pts[idx-1].Y)
		segment.Position2 = fyne.NewPos(pts[idx].X+leftMargin, pts[idx].Y)
		objs = append(objs, segment)
	}
	if leftMargin == 0 {
		return
	}

	newLabel := func(text string) *canvas.Text {
		txt := canvas.NewText(text, constants.ColorAxis)
		txt.TextSize = constants.AxisTextSize
		return txt
	}
	place := func(txt *canvas.Text, posX, posY float32) {
		txt.Move(fyne.NewPos(posX, posY))
		objs = append(objs, txt)
	}

	// y labels, right-aligned against the plot's left edge and vertically
	// centered on their value; ticks that would collide with the previous-close
	// label make way for it.
	labelY := func(value float64) float32 {
		posY := chartmath.YFor(value, low, high, plotH, constants.ChartPadding) - labelH/2
		if posY < 0 {
			posY = 0
		}
		if maxY := plotH - labelH; posY > maxY {
			posY = maxY
		}
		return posY
	}
	var prevY float32
	if prevIdx >= 0 {
		prevY = labelY(labels[prevIdx].value)
	}
	for idx, lbl := range labels {
		posY := labelY(lbl.value)
		if prevIdx >= 0 && idx != prevIdx {
			if delta := posY - prevY; delta > -labelH && delta < labelH {
				continue
			}
		}
		place(newLabel(lbl.text), leftMargin-constants.AxisGap-lbl.width, posY)
	}

	// Time (x) labels along the bottom, formatted per the series' range. An
	// intraday series with a known session is labeled across the full window.
	maxTicks := int(plotW / constants.XTickSpacing)
	var ticks []chartmath.AxisTick
	if chartmath.SessionWindow(series) {
		ticks = chartmath.SessionTicks(series.SessionStart, series.SessionEnd, maxTicks, time.Local)
	} else {
		ticks = chartmath.XTicks(series, maxTicks, time.Local)
	}
	for _, tick := range ticks {
		txt := newLabel(tick.Label)
		width := fyne.MeasureText(tick.Label, constants.AxisTextSize, fyne.TextStyle{}).Width
		posX := leftMargin + constants.ChartPadding + tick.Frac*(plotW-2*constants.ChartPadding) - width/2
		if posX < leftMargin {
			posX = leftMargin
		}
		if posX+width > size.Width {
			posX = size.Width - width
		}
		place(txt, posX, plotH+constants.AxisGap)
	}
}

// updateHover repositions the hover readout — marker dot, vertical dashed
// guide, x-axis time label, and the price/% tooltip — for the current pointer
// position. It is a lightweight path that avoids rebuilding the chart, and
// hides everything when the pointer leaves or there is nothing plotted.
func (renderer *chartRenderer) updateHover() {
	owner := renderer.chart
	if !owner.hovering || len(renderer.hoverPts) == 0 {
		if !renderer.hoverShown {
			return
		}
		renderer.hoverShown = false
		renderer.dot.Hide()
		renderer.tipBg.Hide()
		renderer.tipText.Hide()
		renderer.tipPct.Hide()
		renderer.timeBg.Hide()
		renderer.timeLbl.Hide()
		for _, segment := range renderer.vDashes {
			segment.Hide()
		}
		canvas.Refresh(owner)
		return
	}
	idx := chartmath.NearestPoint(renderer.hoverPts, owner.hoverAt.X)
	point := renderer.hoverPts[idx]
	series := owner.series
	size := owner.Size()

	// Vertical dashed guide through the hovered point, spanning the plot.
	segIdx := 0
	for posY := float32(constants.ChartPadding); posY < renderer.plotH-constants.ChartPadding && segIdx < len(renderer.vDashes); posY += constants.DashLen + constants.DashGap {
		segEnd := posY + constants.DashLen
		if maxY := renderer.plotH - constants.ChartPadding; segEnd > maxY {
			segEnd = maxY
		}
		segment := renderer.vDashes[segIdx]
		segment.Position1 = fyne.NewPos(point.X, posY)
		segment.Position2 = fyne.NewPos(point.X, segEnd)
		segment.Show()
		segIdx++
	}
	for ; segIdx < len(renderer.vDashes); segIdx++ {
		renderer.vDashes[segIdx].Hide()
	}

	// Marker dot on the hovered data point.
	renderer.dot.FillColor = owner.line
	renderer.dot.Position1 = fyne.NewPos(point.X-constants.DotRadius, point.Y-constants.DotRadius)
	renderer.dot.Position2 = fyne.NewPos(point.X+constants.DotRadius, point.Y+constants.DotRadius)
	renderer.dot.Show()

	// Time/date of the hovered point, boxed on the x axis under the guide.
	renderer.timeLbl.Text = series.Candles[idx].Time.In(time.Local).Format(chartmath.HoverTimeFormat(series.Range))
	labelSize := fyne.MeasureText(renderer.timeLbl.Text, constants.AxisTextSize, fyne.TextStyle{})
	lblWidth, lblHeight := labelSize.Width+2*constants.TipPad, labelSize.Height+2
	lblX := point.X - lblWidth/2
	if lblX < 0 {
		lblX = 0
	}
	if lblX+lblWidth > size.Width {
		lblX = size.Width - lblWidth
	}
	lblY := renderer.plotH + (size.Height-renderer.plotH-lblHeight)/2 // centered in the axis strip
	renderer.timeBg.Resize(fyne.NewSize(lblWidth, lblHeight))
	renderer.timeBg.Move(fyne.NewPos(lblX, lblY))
	renderer.timeBg.Show()
	renderer.timeLbl.Move(fyne.NewPos(lblX+constants.TipPad, lblY+1))
	renderer.timeLbl.Show()

	// Tooltip: the price, and under it the % change versus the previous close
	// (the dashed reference line), green for above and red for below.
	renderer.tipText.Text = chartmath.FormatAxisPrice(renderer.hoverVals[idx])
	priceSize := fyne.MeasureText(renderer.tipText.Text, constants.AxisTextSize, fyne.TextStyle{})
	tipWidth, tipHeight := priceSize.Width, priceSize.Height
	showPct := series.PreviousClose > 0
	if showPct {
		pct := (renderer.hoverVals[idx] - series.PreviousClose) / series.PreviousClose * 100
		col, sign := changeStyle(pct)
		renderer.tipPct.Text = fmt.Sprintf(constants.FmtPercentChange, sign, pct)
		renderer.tipPct.Color = col
		pctSize := fyne.MeasureText(renderer.tipPct.Text, constants.AxisTextSize, fyne.TextStyle{})
		if pctSize.Width > tipWidth {
			tipWidth = pctSize.Width
		}
		tipHeight += pctSize.Height
	}
	boxWidth := tipWidth + 2*constants.TipPad
	boxHeight := tipHeight + 2*constants.TipPad
	boxX := point.X - boxWidth/2
	if boxX < 0 {
		boxX = 0
	}
	if boxX+boxWidth > size.Width {
		boxX = size.Width - boxWidth
	}
	boxY := point.Y - constants.DotRadius - constants.TipGap - boxHeight // above the dot ...
	if boxY < 0 {
		boxY = point.Y + constants.DotRadius + constants.TipGap // ... or below when clipped at the top
	}
	renderer.tipBg.Resize(fyne.NewSize(boxWidth, boxHeight))
	renderer.tipBg.Move(fyne.NewPos(boxX, boxY))
	renderer.tipBg.Show()
	renderer.tipText.Move(fyne.NewPos(boxX+constants.TipPad, boxY+constants.TipPad))
	renderer.tipText.Show()
	if showPct {
		renderer.tipPct.Move(fyne.NewPos(boxX+constants.TipPad, boxY+constants.TipPad+priceSize.Height))
		renderer.tipPct.Show()
	} else {
		renderer.tipPct.Hide()
	}

	renderer.hoverShown = true
	canvas.Refresh(owner)
}
