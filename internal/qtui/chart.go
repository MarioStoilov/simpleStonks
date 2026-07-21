package qtui

import (
	"fmt"
	"image/color"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/chartmath"
	"github.com/MarioStoilov/simplestonks/internal/constants"
	"github.com/MarioStoilov/simplestonks/internal/model"
)

// chartWidget plots a price series as a line, colored by the caller (up/down),
// with y price labels, range-aware x time labels, and a dashed previous-close
// reference line — the QPainter port of the Fyne chart widget, driven by the
// shared chartmath package.
type chartWidget struct {
	*qt.QWidget
	series    model.Series
	lineColor color.NRGBA
	font      *qt.QFont
	metrics   *qt.QFontMetrics

	// After-hours overlay (detail view only): a dashed vertical divider at
	// the regular close and a dimmed line from dimFromIdx on. Cleared by
	// every setSeries so regular charts can never inherit it.
	dividerTime time.Time
	dimFromIdx  int

	// Hover readout state (only used with hoverReadout enabled): pointer
	// tracking plus the plotted geometry cached by the last paint.
	hoverReadout bool
	hovering     bool
	hoverX       float32
	hoverPts     []chartmath.Point // absolute coordinates
	hoverVals    []float64
	hoverPlotH   float32
}

// newChartWidget constructs an empty chart.
func newChartWidget(parent *qt.QWidget) *chartWidget {
	font := qt.NewQFont()
	font.SetPixelSize(int(constants.AxisTextSize))
	chart := &chartWidget{
		QWidget:    qt.NewQWidget(parent),
		lineColor:  constants.ColorNeutral,
		font:       font,
		metrics:    qt.NewQFontMetrics(font),
		dimFromIdx: -1,
	}
	chart.SetMinimumSize2(int(constants.ChartMinWidth), int(constants.ChartMinHeight))
	chart.SetStyleSheet("background: " + cssRGB(constants.ColorChartBg) + ";")
	chart.OnPaintEvent(func(super func(event *qt.QPaintEvent), event *qt.QPaintEvent) {
		chart.paint()
	})
	return chart
}

// enableHoverReadout turns on the hover marker/guide/tooltip (the detail
// view's expanded chart).
func (chart *chartWidget) enableHoverReadout() {
	chart.hoverReadout = true
	chart.SetMouseTracking(true)
	// Mouse tracking swallows the moves the frameless window relies on to
	// reset its resize cursor, so the chart must carry its own cursor or a
	// stale resize cursor sticks when entering from an edge grip.
	chart.SetCursor(qt.NewQCursor2(qt.ArrowCursor))
	chart.OnMouseMoveEvent(func(super func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		chart.hovering = true
		chart.hoverX = float32(event.Position().X())
		chart.Update()
	})
	chart.OnLeaveEvent(func(super func(event *qt.QEvent), event *qt.QEvent) {
		chart.hovering = false
		chart.Update()
		super(event)
	})
}

// setSeries updates the plotted data and line color, and repaints. Any
// after-hours overlay is cleared; callers wanting one re-apply it afterwards.
func (chart *chartWidget) setSeries(series model.Series, lineColor color.NRGBA) {
	chart.series = series
	chart.lineColor = lineColor
	chart.dividerTime = time.Time{}
	chart.dimFromIdx = -1
	chart.Update()
}

// setAfterHoursOverlay marks the regular-close boundary on an extended-hours
// chart: a dashed vertical divider at boundary and a dimmed line from the
// dimFromIdx-th candle on. Call after setSeries, which clears the overlay.
func (chart *chartWidget) setAfterHoursOverlay(boundary time.Time, dimFromIdx int) {
	chart.dividerTime = boundary
	chart.dimFromIdx = dimFromIdx
	chart.Update()
}

// paint renders the chart for the current size: plot background, dashed
// previous-close line, the series line, and the axis labels. Margins collapse
// to a bare line when the widget is too small (mirrors the Fyne renderer).
func (chart *chartWidget) paint() {
	painter := qt.NewQPainter2(chart.QPaintDevice)
	defer painter.End()

	width := float32(chart.Width())
	height := float32(chart.Height())
	painter.FillRect(qt.NewQRectF4(0, 0, float64(width), float64(height)), qt.NewQBrush3(qColor(constants.ColorChartBg)))

	chart.hoverPts, chart.hoverVals = nil, nil
	series := chart.series
	values := chartmath.ClosesOf(series)
	if len(values) < 2 {
		return
	}
	painter.SetRenderHint(qt.QPainter__Antialiasing)
	painter.SetFont(chart.font)

	prevClose := series.PreviousClose
	low, high := chartmath.ValueBounds(values, prevClose)

	labelH := float32(chart.metrics.Height())
	bottomMargin := labelH + constants.AxisGap
	plotH := height - bottomMargin

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
		labels[idx].width = float32(chart.metrics.HorizontalAdvance(labels[idx].text))
		if labels[idx].width > leftMargin {
			leftMargin = labels[idx].width
		}
	}
	leftMargin += constants.AxisGap

	plotW := width - leftMargin
	bare := false
	pts := chartmath.PlotPath(values, chartmath.XFracs(series), plotW, plotH, constants.ChartPadding, low, high)
	if pts == nil { // too small for margins: fall back to the bare line
		bare = true
		leftMargin, bottomMargin = 0, 0
		plotW, plotH = width, height
		pts = chartmath.PlotPath(values, chartmath.XFracs(series), plotW, plotH, constants.ChartPadding, low, high)
		if pts == nil {
			return
		}
	}

	// Cache the plotted points (absolute coordinates) for the hover readout,
	// which draws last so it sits on top of everything.
	chart.hoverPlotH = plotH
	if chart.hoverReadout {
		chart.hoverVals = values
		chart.hoverPts = make([]chartmath.Point, len(pts))
		for idx, point := range pts {
			chart.hoverPts[idx] = chartmath.Point{X: point.X + leftMargin, Y: point.Y}
		}
	}
	defer chart.drawHover(painter, width, height)

	// Dashed reference line at the previous interval's close (under the series).
	if prevClose > 0 {
		axisPen := qt.NewQPen3(qColor(constants.ColorAxis))
		axisPen.SetWidthF(float64(constants.HairlineWidth))
		axisPen.SetDashPattern([]float64{float64(constants.DashLen), float64(constants.DashGap)})
		painter.SetPenWithPen(axisPen)
		posY := float64(chartmath.YFor(prevClose, low, high, plotH, constants.ChartPadding))
		painter.DrawLine(qt.NewQLineF3(
			float64(leftMargin+constants.ChartPadding), posY,
			float64(leftMargin+plotW-constants.ChartPadding), posY,
		))
	}

	linePen := qt.NewQPen3(qColor(chart.lineColor))
	linePen.SetWidthF(float64(constants.ChartLineWidth))
	dimColor := chart.lineColor
	dimColor.A = constants.AfterHoursDimAlpha
	dimPen := qt.NewQPen3(qColor(dimColor))
	dimPen.SetWidthF(float64(constants.ChartLineWidth))
	for idx := 1; idx < len(pts); idx++ {
		// The after-hours tail — including the segment crossing the regular
		// close — draws dimmed against the divider.
		if chart.dimFromIdx >= 0 && idx >= chart.dimFromIdx {
			painter.SetPenWithPen(dimPen)
		} else {
			painter.SetPenWithPen(linePen)
		}
		painter.DrawLine(qt.NewQLineF3(
			float64(pts[idx-1].X+leftMargin), float64(pts[idx-1].Y),
			float64(pts[idx].X+leftMargin), float64(pts[idx].Y),
		))
	}
	if bare {
		return
	}

	// Dashed vertical divider at the regular close of an after-hours chart.
	if !chart.dividerTime.IsZero() && chartmath.SessionWindow(series) {
		frac := float32(chart.dividerTime.Sub(series.SessionStart)) / float32(series.SessionEnd.Sub(series.SessionStart))
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		dividerPen := qt.NewQPen3(qColor(constants.ColorAxis))
		dividerPen.SetWidthF(float64(constants.HairlineWidth))
		dividerPen.SetDashPattern([]float64{float64(constants.DashLen), float64(constants.DashGap)})
		painter.SetPenWithPen(dividerPen)
		posX := float64(leftMargin + constants.ChartPadding + frac*(plotW-2*constants.ChartPadding))
		painter.DrawLine(qt.NewQLineF3(
			posX, float64(constants.ChartPadding),
			posX, float64(plotH-constants.ChartPadding),
		))
	}

	painter.SetPen(qColor(constants.ColorAxis))
	ascent := float32(chart.metrics.Ascent())
	drawText := func(text string, posX, posY float32) {
		// QPainter draws text at the baseline; our layout positions the top.
		painter.DrawText(qt.NewQPointF3(float64(posX), float64(posY+ascent)), text)
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
		drawText(lbl.text, leftMargin-constants.AxisGap-lbl.width, posY)
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
		tickWidth := float32(chart.metrics.HorizontalAdvance(tick.Label))
		posX := leftMargin + constants.ChartPadding + tick.Frac*(plotW-2*constants.ChartPadding) - tickWidth/2
		if posX < leftMargin {
			posX = leftMargin
		}
		if posX+tickWidth > width {
			posX = width - tickWidth
		}
		drawText(tick.Label, posX, plotH+constants.AxisGap)
	}
}

// drawHover renders the hover readout on top of the finished chart: a dashed
// vertical guide through the nearest data point, a marker dot, the point's
// time boxed on the x axis, and a tooltip with the price and its % change
// versus the previous close (port of the Fyne renderer's updateHover).
func (chart *chartWidget) drawHover(painter *qt.QPainter, width, height float32) {
	if !chart.hoverReadout || !chart.hovering || len(chart.hoverPts) == 0 {
		return
	}
	idx := chartmath.NearestPoint(chart.hoverPts, chart.hoverX)
	point := chart.hoverPts[idx]
	series := chart.series
	plotH := chart.hoverPlotH

	// Vertical dashed guide through the hovered point, spanning the plot.
	guidePen := qt.NewQPen3(qColor(constants.ColorAxis))
	guidePen.SetWidthF(float64(constants.HairlineWidth))
	guidePen.SetDashPattern([]float64{float64(constants.DashLen), float64(constants.DashGap)})
	painter.SetPenWithPen(guidePen)
	painter.DrawLine(qt.NewQLineF3(
		float64(point.X), float64(constants.ChartPadding),
		float64(point.X), float64(plotH-constants.ChartPadding),
	))

	// Marker dot on the hovered data point.
	dotPen := qt.NewQPen3(qColor(constants.ColorForeground))
	dotPen.SetWidthF(float64(constants.HairlineWidth))
	painter.SetPenWithPen(dotPen)
	painter.SetBrush(qt.NewQBrush3(qColor(chart.lineColor)))
	painter.DrawEllipse(qt.NewQRectF4(
		float64(point.X-constants.DotRadius), float64(point.Y-constants.DotRadius),
		float64(2*constants.DotRadius), float64(2*constants.DotRadius),
	))

	labelH := float32(chart.metrics.Height())
	ascent := float32(chart.metrics.Ascent())
	panelBrush := qt.NewQBrush3(qColor(constants.ColorHover))

	// Time/date of the hovered point, boxed on the x axis under the guide.
	timeText := series.Candles[idx].Time.In(time.Local).Format(chartmath.HoverTimeFormat(series.Range))
	timeWidth := float32(chart.metrics.HorizontalAdvance(timeText)) + 2*constants.TipPad
	timeHeight := labelH + 2
	timeX := point.X - timeWidth/2
	if timeX < 0 {
		timeX = 0
	}
	if timeX+timeWidth > width {
		timeX = width - timeWidth
	}
	timeY := plotH + (height-plotH-timeHeight)/2 // centered in the axis strip
	painter.SetPenWithStyle(qt.NoPen)
	painter.SetBrush(panelBrush)
	painter.DrawRoundedRect(qt.NewQRectF4(float64(timeX), float64(timeY), float64(timeWidth), float64(timeHeight)),
		float64(constants.PanelCornerRadius), float64(constants.PanelCornerRadius))
	painter.SetPen(qColor(constants.ColorForeground))
	painter.DrawText(qt.NewQPointF3(float64(timeX+constants.TipPad), float64(timeY+1+ascent)), timeText)

	// Tooltip: the price, and under it the % change versus the previous close
	// (the dashed reference line), green for above and red for below.
	priceText := chartmath.FormatAxisPrice(chart.hoverVals[idx])
	tipWidth := float32(chart.metrics.HorizontalAdvance(priceText))
	tipHeight := labelH
	showPct := series.PreviousClose > 0
	var pctText string
	var pctColor = constants.ColorNeutral
	if showPct {
		pct := (chart.hoverVals[idx] - series.PreviousClose) / series.PreviousClose * constants.PercentMax
		col, sign := changeStyle(pct)
		pctText = fmt.Sprintf(constants.FmtPercentChange, sign, pct)
		pctColor = col
		if pctWidth := float32(chart.metrics.HorizontalAdvance(pctText)); pctWidth > tipWidth {
			tipWidth = pctWidth
		}
		tipHeight += labelH
	}
	boxWidth := tipWidth + 2*constants.TipPad
	boxHeight := tipHeight + 2*constants.TipPad
	boxX := point.X - boxWidth/2
	if boxX < 0 {
		boxX = 0
	}
	if boxX+boxWidth > width {
		boxX = width - boxWidth
	}
	boxY := point.Y - constants.DotRadius - constants.TipGap - boxHeight // above the dot ...
	if boxY < 0 {
		boxY = point.Y + constants.DotRadius + constants.TipGap // ... or below when clipped at the top
	}
	painter.SetPenWithStyle(qt.NoPen)
	painter.SetBrush(panelBrush)
	painter.DrawRoundedRect(qt.NewQRectF4(float64(boxX), float64(boxY), float64(boxWidth), float64(boxHeight)),
		float64(constants.PanelCornerRadius), float64(constants.PanelCornerRadius))
	painter.SetPen(qColor(constants.ColorForeground))
	painter.DrawText(qt.NewQPointF3(float64(boxX+constants.TipPad), float64(boxY+constants.TipPad+ascent)), priceText)
	if showPct {
		painter.SetPen(qColor(pctColor))
		painter.DrawText(qt.NewQPointF3(float64(boxX+constants.TipPad), float64(boxY+constants.TipPad+labelH+ascent)), pctText)
	}
}
