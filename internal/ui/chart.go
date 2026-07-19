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

	"github.com/MarioStoilov/simplestonks/internal/model"
)

const (
	chartPadding = 4
	axisTextSize = 12  // font size of the axis labels
	axisGap      = 4   // gap between axis labels and the plot area
	xTickSpacing = 80  // minimum horizontal pixels per time label
	yTickSpacing = 48  // minimum vertical pixels per price label
	maxYTicks    = 8   // cap on the number of y-axis reference values
	dashLen      = 4   // dash length of the previous-close line
	dashGap      = 4   // gap between dashes of the previous-close line
	dotRadius    = 3.5 // radius of the hover marker dot
	tipPad       = 4   // padding inside the hover tooltip
	tipGap       = 8   // gap between the hover dot and its tooltip
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
	c := &chart{line: colorNeutral}
	c.ExtendBaseWidget(c)
	return c
}

// SetSeries updates the plotted data and redraws.
func (c *chart) SetSeries(s model.Series) {
	c.series = s
	c.Refresh()
}

// SetColor sets the line color used on the next redraw.
func (c *chart) SetColor(col color.Color) { c.line = col }

// MouseIn implements desktop.Hoverable.
func (c *chart) MouseIn(ev *desktop.MouseEvent) { c.MouseMoved(ev) }

// MouseMoved implements desktop.Hoverable: it tracks the pointer and updates
// the hover marker without rebuilding the chart. Since the chart is the
// topmost hoverable, it also forwards enter/leave to onHover so an enclosing
// tile can render its own hover effect.
func (c *chart) MouseMoved(ev *desktop.MouseEvent) {
	if !c.hovering && c.onHover != nil {
		c.onHover(true)
	}
	c.hovering = true
	c.hoverAt = ev.Position
	if c.renderer != nil {
		c.renderer.updateHover()
	}
}

// MouseOut implements desktop.Hoverable.
func (c *chart) MouseOut() {
	c.hovering = false
	if c.onHover != nil {
		c.onHover(false)
	}
	if c.renderer != nil {
		c.renderer.updateHover()
	}
}

// CreateRenderer implements fyne.Widget.
func (c *chart) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.NRGBA{R: 0x1e, G: 0x1e, B: 0x24, A: 0xff})
	dot := canvas.NewCircle(colorNeutral)
	dot.StrokeColor = theme.Color(theme.ColorNameForeground)
	dot.StrokeWidth = 1
	tipBg := canvas.NewRectangle(colorHover)
	tipBg.CornerRadius = 4
	tipText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	tipText.TextSize = axisTextSize
	tipPct := canvas.NewText("", colorNeutral)
	tipPct.TextSize = axisTextSize
	timeBg := canvas.NewRectangle(colorHover)
	timeBg.CornerRadius = 4
	timeLbl := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	timeLbl.TextSize = axisTextSize
	r := &chartRenderer{
		chart: c, bg: bg, dot: dot, tipBg: tipBg, tipText: tipText,
		tipPct: tipPct, timeBg: timeBg, timeLbl: timeLbl,
		objects: []fyne.CanvasObject{bg},
	}
	c.renderer = r
	return r
}

type chartRenderer struct {
	chart   *chart
	bg      *canvas.Rectangle
	objects []fyne.CanvasObject

	// Hover readout state (only populated when chart.hoverReadout is set):
	// the plotted points (absolute coordinates) and their values, cached by
	// rebuild, plus the marker/guide/tooltip canvas objects.
	hoverPts   []fyne.Position
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

func (r *chartRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.rebuild(size)
}

func (r *chartRenderer) MinSize() fyne.Size { return fyne.NewSize(120, 80) }

func (r *chartRenderer) Refresh() {
	r.rebuild(r.chart.Size())
	canvas.Refresh(r.chart)
}

func (r *chartRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *chartRenderer) Destroy() {}

// rebuild regenerates the line segments and axis labels for the current series
// and size. The plot is inset by a left margin (price labels) and a bottom
// margin (time labels); margins collapse to zero when there is nothing to draw.
func (r *chartRenderer) rebuild(size fyne.Size) {
	objs := []fyne.CanvasObject{r.bg}
	r.hoverPts, r.hoverVals, r.vDashes = nil, nil, nil
	defer func() {
		if len(r.hoverPts) > 0 { // keep the hover guide/marker/tooltip on top
			for _, ln := range r.vDashes {
				objs = append(objs, ln)
			}
			objs = append(objs, r.timeBg, r.timeLbl, r.tipBg, r.tipText, r.tipPct, r.dot)
		}
		r.objects = objs
	}()
	defer r.updateHover() // runs first: re-aim the marker at the new geometry

	s := r.chart.series
	values := closesOf(s)
	if len(values) < 2 {
		return
	}
	lo, hi := values[0], values[0]
	for _, v := range values {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	// Widen the scale to keep the previous-close reference line in view.
	prevClose := s.PreviousClose
	if prevClose > 0 {
		if prevClose < lo {
			lo = prevClose
		}
		if prevClose > hi {
			hi = prevClose
		}
	}

	labelH := fyne.MeasureText("0", axisTextSize, fyne.TextStyle{}).Height
	bottomMargin := labelH + axisGap
	plotH := size.Height - bottomMargin

	// Price (y) labels: as many evenly spaced reference values as the height
	// allows, plus the previous close; a flat scale gets a single label. The
	// left margin fits the widest label.
	type yLabel struct {
		v    float64
		text string
		w    float32
	}
	var yls []yLabel
	if lo == hi {
		yls = []yLabel{{v: lo, text: formatAxisPrice(lo)}}
	} else {
		n := int(plotH/yTickSpacing) + 1
		if n < 2 {
			n = 2
		}
		if n > maxYTicks {
			n = maxYTicks
		}
		for _, v := range yTicks(lo, hi, n) {
			yls = append(yls, yLabel{v: v, text: formatAxisPrice(v)})
		}
	}
	prevIdx := -1 // index of the previous-close label, which wins collisions
	if prevClose > 0 && lo != hi {
		yls = append(yls, yLabel{v: prevClose, text: formatAxisPrice(prevClose)})
		prevIdx = len(yls) - 1
	}
	var leftMargin float32
	for i := range yls {
		yls[i].w = fyne.MeasureText(yls[i].text, axisTextSize, fyne.TextStyle{}).Width
		if yls[i].w > leftMargin {
			leftMargin = yls[i].w
		}
	}
	leftMargin += axisGap

	plotW := size.Width - leftMargin
	pts := plotPath(values, xFracs(s), plotW, plotH, chartPadding, lo, hi)
	if pts == nil { // too small for margins: fall back to the bare line
		leftMargin, bottomMargin = 0, 0
		plotW, plotH = size.Width, size.Height
		pts = plotPath(values, xFracs(s), plotW, plotH, chartPadding, lo, hi)
	}

	// Cache the plotted points (absolute coordinates) for the hover readout
	// and pool the vertical guide's dash segments for the plot height.
	r.plotH = plotH
	if pts != nil && r.chart.hoverReadout {
		r.hoverVals = values
		r.hoverPts = make([]fyne.Position, len(pts))
		for i, p := range pts {
			r.hoverPts[i] = fyne.NewPos(p.X+leftMargin, p.Y)
		}
		n := int((plotH-2*chartPadding)/(dashLen+dashGap)) + 1
		r.vDashes = make([]*canvas.Line, 0, n)
		for k := 0; k < n; k++ {
			ln := canvas.NewLine(colorAxis)
			ln.StrokeWidth = 1
			ln.Hide()
			r.vDashes = append(r.vDashes, ln)
		}
	}

	// Dashed reference line at the previous interval's close (under the series).
	if prevClose > 0 && pts != nil {
		y := yFor(prevClose, lo, hi, plotH, chartPadding)
		right := leftMargin + plotW - chartPadding
		for x := leftMargin + chartPadding; x < right; x += dashLen + dashGap {
			end := x + dashLen
			if end > right {
				end = right
			}
			ln := canvas.NewLine(colorAxis)
			ln.StrokeWidth = 1
			ln.Position1 = fyne.NewPos(x, y)
			ln.Position2 = fyne.NewPos(end, y)
			objs = append(objs, ln)
		}
	}

	for i := 1; i < len(pts); i++ {
		ln := canvas.NewLine(r.chart.line)
		ln.StrokeWidth = 1.5
		ln.Position1 = fyne.NewPos(pts[i-1].X+leftMargin, pts[i-1].Y)
		ln.Position2 = fyne.NewPos(pts[i].X+leftMargin, pts[i].Y)
		objs = append(objs, ln)
	}
	if leftMargin == 0 {
		return
	}

	newLabel := func(text string) *canvas.Text {
		t := canvas.NewText(text, colorAxis)
		t.TextSize = axisTextSize
		return t
	}
	place := func(t *canvas.Text, x, y float32) {
		t.Move(fyne.NewPos(x, y))
		objs = append(objs, t)
	}

	// y labels, right-aligned against the plot's left edge and vertically
	// centered on their value; ticks that would collide with the previous-close
	// label make way for it.
	labelY := func(v float64) float32 {
		y := yFor(v, lo, hi, plotH, chartPadding) - labelH/2
		if y < 0 {
			y = 0
		}
		if m := plotH - labelH; y > m {
			y = m
		}
		return y
	}
	var prevY float32
	if prevIdx >= 0 {
		prevY = labelY(yls[prevIdx].v)
	}
	for i, l := range yls {
		y := labelY(l.v)
		if prevIdx >= 0 && i != prevIdx {
			if d := y - prevY; d > -labelH && d < labelH {
				continue
			}
		}
		place(newLabel(l.text), leftMargin-axisGap-l.w, y)
	}

	// Time (x) labels along the bottom, formatted per the series' range. An
	// intraday series with a known session is labeled across the full window.
	maxTicks := int(plotW / xTickSpacing)
	var ticks []axisTick
	if sessionWindow(s) {
		ticks = sessionTicks(s.SessionStart, s.SessionEnd, maxTicks, time.Local)
	} else {
		ticks = xTicks(s, maxTicks, time.Local)
	}
	for _, tick := range ticks {
		t := newLabel(tick.label)
		w := fyne.MeasureText(tick.label, axisTextSize, fyne.TextStyle{}).Width
		x := leftMargin + chartPadding + tick.frac*(plotW-2*chartPadding) - w/2
		if x < leftMargin {
			x = leftMargin
		}
		if x+w > size.Width {
			x = size.Width - w
		}
		place(t, x, plotH+axisGap)
	}
}

// updateHover repositions the hover readout — marker dot, vertical dashed
// guide, x-axis time label, and the price/% tooltip — for the current pointer
// position. It is a lightweight path that avoids rebuilding the chart, and
// hides everything when the pointer leaves or there is nothing plotted.
func (r *chartRenderer) updateHover() {
	c := r.chart
	if !c.hovering || len(r.hoverPts) == 0 {
		if !r.hoverShown {
			return
		}
		r.hoverShown = false
		r.dot.Hide()
		r.tipBg.Hide()
		r.tipText.Hide()
		r.tipPct.Hide()
		r.timeBg.Hide()
		r.timeLbl.Hide()
		for _, ln := range r.vDashes {
			ln.Hide()
		}
		canvas.Refresh(c)
		return
	}
	i := nearestPoint(r.hoverPts, c.hoverAt.X)
	p := r.hoverPts[i]
	s := c.series
	size := c.Size()

	// Vertical dashed guide through the hovered point, spanning the plot.
	k := 0
	for y := float32(chartPadding); y < r.plotH-chartPadding && k < len(r.vDashes); y += dashLen + dashGap {
		end := y + dashLen
		if m := r.plotH - chartPadding; end > m {
			end = m
		}
		ln := r.vDashes[k]
		ln.Position1 = fyne.NewPos(p.X, y)
		ln.Position2 = fyne.NewPos(p.X, end)
		ln.Show()
		k++
	}
	for ; k < len(r.vDashes); k++ {
		r.vDashes[k].Hide()
	}

	// Marker dot on the hovered data point.
	r.dot.FillColor = c.line
	r.dot.Position1 = fyne.NewPos(p.X-dotRadius, p.Y-dotRadius)
	r.dot.Position2 = fyne.NewPos(p.X+dotRadius, p.Y+dotRadius)
	r.dot.Show()

	// Time/date of the hovered point, boxed on the x axis under the guide.
	r.timeLbl.Text = s.Candles[i].Time.In(time.Local).Format(hoverTimeFormat(s.Range))
	ls := fyne.MeasureText(r.timeLbl.Text, axisTextSize, fyne.TextStyle{})
	lw, lh := ls.Width+2*tipPad, ls.Height+2
	lx := p.X - lw/2
	if lx < 0 {
		lx = 0
	}
	if lx+lw > size.Width {
		lx = size.Width - lw
	}
	ly := r.plotH + (size.Height-r.plotH-lh)/2 // centered in the axis strip
	r.timeBg.Resize(fyne.NewSize(lw, lh))
	r.timeBg.Move(fyne.NewPos(lx, ly))
	r.timeBg.Show()
	r.timeLbl.Move(fyne.NewPos(lx+tipPad, ly+1))
	r.timeLbl.Show()

	// Tooltip: the price, and under it the % change versus the previous close
	// (the dashed reference line), green for above and red for below.
	r.tipText.Text = formatAxisPrice(r.hoverVals[i])
	ts := fyne.MeasureText(r.tipText.Text, axisTextSize, fyne.TextStyle{})
	tw, th := ts.Width, ts.Height
	showPct := s.PreviousClose > 0
	if showPct {
		pct := (r.hoverVals[i] - s.PreviousClose) / s.PreviousClose * 100
		col, sign := changeStyle(pct)
		r.tipPct.Text = fmt.Sprintf("%s%.2f%%", sign, pct)
		r.tipPct.Color = col
		ps := fyne.MeasureText(r.tipPct.Text, axisTextSize, fyne.TextStyle{})
		if ps.Width > tw {
			tw = ps.Width
		}
		th += ps.Height
	}
	w := tw + 2*tipPad
	h := th + 2*tipPad
	x := p.X - w/2
	if x < 0 {
		x = 0
	}
	if x+w > size.Width {
		x = size.Width - w
	}
	y := p.Y - dotRadius - tipGap - h // above the dot ...
	if y < 0 {
		y = p.Y + dotRadius + tipGap // ... or below when clipped at the top
	}
	r.tipBg.Resize(fyne.NewSize(w, h))
	r.tipBg.Move(fyne.NewPos(x, y))
	r.tipBg.Show()
	r.tipText.Move(fyne.NewPos(x+tipPad, y+tipPad))
	r.tipText.Show()
	if showPct {
		r.tipPct.Move(fyne.NewPos(x+tipPad, y+tipPad+ts.Height))
		r.tipPct.Show()
	} else {
		r.tipPct.Hide()
	}

	r.hoverShown = true
	canvas.Refresh(c)
}

// hoverTimeFormat maps a chart range to the format of the hover guide's
// x-axis label: clock time intraday, calendar dates beyond, with the year for
// multi-year spans.
func hoverTimeFormat(r model.Range) string {
	switch r {
	case model.Range1D:
		return "15:04"
	case model.Range5D, model.Range1W, model.Range1M:
		return "Mon, 02 Jan"
	default: // YTD, 1Y, 5Y, ALL
		return "02 Jan 2006"
	}
}

// nearestPoint returns the index of the point whose x coordinate is closest
// to x. Points must be ordered by non-decreasing x (as plotted points are).
func nearestPoint(pts []fyne.Position, x float32) int {
	best := 0
	bestDist := float32(-1)
	for i, p := range pts {
		d := p.X - x
		if d < 0 {
			d = -d
		}
		if bestDist < 0 || d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

// axisTick is one x-axis label at a fractional position along the plot width.
type axisTick struct {
	frac  float32 // 0 (left edge) .. 1 (right edge)
	label string
}

// xTicks picks up to max evenly spaced candles from the series and formats
// their timestamps (in loc) for the series' range. Consecutive duplicate
// labels — e.g. the same year twice on a 5Y chart — are dropped.
func xTicks(s model.Series, max int, loc *time.Location) []axisTick {
	n := len(s.Candles)
	if n < 2 || max < 2 {
		return nil
	}
	if max > n {
		max = n
	}
	layout := xAxisFormat(s.Range)
	out := make([]axisTick, 0, max)
	prev := ""
	for k := 0; k < max; k++ {
		i := k * (n - 1) / (max - 1)
		label := s.Candles[i].Time.In(loc).Format(layout)
		if label == prev {
			continue
		}
		prev = label
		out = append(out, axisTick{frac: float32(i) / float32(n-1), label: label})
	}
	return out
}

// xAxisFormat maps a chart range to the time layout of its x-axis labels:
// hours intraday, then days, dates, months, and years as the span grows.
func xAxisFormat(r model.Range) string {
	switch r {
	case model.Range1D:
		return "15:04"
	case model.Range5D, model.Range1W:
		return "Mon 02"
	case model.Range1M:
		return "02 Jan"
	case model.RangeYTD, model.Range1Y:
		return "Jan"
	default: // 5Y, ALL
		return "2006"
	}
}

// formatAxisPrice formats a y-axis price label.
func formatAxisPrice(v float64) string { return fmt.Sprintf("%.2f", v) }

// closesOf extracts the closing prices from a series.
func closesOf(s model.Series) []float64 {
	out := make([]float64, len(s.Candles))
	for i, c := range s.Candles {
		out[i] = c.Close
	}
	return out
}

// plotPath maps close values to pixel positions within a w×h box (inset by
// pad): x from the given 0..1 fractions, y scaled to the [lo, hi] value scale,
// inverted so higher values sit toward the top. It returns nil when there is
// nothing meaningful to draw (fewer than two points, mismatched fractions, or
// no room after padding).
func plotPath(values []float64, fracs []float32, w, h, pad float32, lo, hi float64) []fyne.Position {
	if len(values) < 2 || len(fracs) != len(values) {
		return nil
	}
	iw := w - 2*pad
	if iw <= 0 || h-2*pad <= 0 {
		return nil
	}
	pts := make([]fyne.Position, len(values))
	for i, v := range values {
		pts[i] = fyne.NewPos(pad+iw*fracs[i], yFor(v, lo, hi, h, pad))
	}
	return pts
}

// evenFracs returns n positions spaced evenly across 0..1.
func evenFracs(n int) []float32 {
	if n < 2 {
		return nil
	}
	out := make([]float32, n)
	for i := range out {
		out[i] = float32(i) / float32(n-1)
	}
	return out
}

// sessionWindow reports whether a series should be drawn against its full
// trading-session window: intraday data with known session bounds whose
// candles actually belong to that session. While a market is closed, Yahoo
// pairs the *previous* day's candles with the *upcoming* session in
// currentTradingPeriod; drawing those against the future window would
// collapse them onto its left edge, so they fall back to even spacing (the
// completed day spans the full width). Once the session opens and its first
// candles arrive, the window mode kicks in and the new chart starts filling
// in from the left.
func sessionWindow(s model.Series) bool {
	if !s.Range.Intraday() || !s.SessionEnd.After(s.SessionStart) || len(s.Candles) == 0 {
		return false
	}
	last := s.Candles[len(s.Candles)-1].Time
	return !last.Before(s.SessionStart)
}

// xFracs returns each candle's horizontal position as a 0..1 fraction. Within
// a session window the fraction is time-based over the whole session, so a
// live trading day fills in gradually (Yahoo-style); otherwise candles are
// spaced evenly.
func xFracs(s model.Series) []float32 {
	if !sessionWindow(s) {
		return evenFracs(len(s.Candles))
	}
	span := float32(s.SessionEnd.Sub(s.SessionStart))
	out := make([]float32, len(s.Candles))
	for i, c := range s.Candles {
		f := float32(c.Time.Sub(s.SessionStart)) / span
		if f < 0 {
			f = 0
		}
		if f > 1 {
			f = 1
		}
		out[i] = f
	}
	return out
}

// sessionTicks returns up to max evenly spaced intraday time labels spanning a
// trading-session window.
func sessionTicks(start, end time.Time, max int, loc *time.Location) []axisTick {
	if max < 2 || !end.After(start) {
		return nil
	}
	layout := xAxisFormat(model.Range1D)
	span := end.Sub(start)
	out := make([]axisTick, 0, max)
	prev := ""
	for k := 0; k < max; k++ {
		at := start.Add(span * time.Duration(k) / time.Duration(max-1))
		label := at.In(loc).Format(layout)
		if label == prev {
			continue
		}
		prev = label
		out = append(out, axisTick{frac: float32(k) / float32(max-1), label: label})
	}
	return out
}

// yTicks returns max evenly spaced reference values from hi (first) down to lo.
func yTicks(lo, hi float64, max int) []float64 {
	if max < 2 || hi <= lo {
		return nil
	}
	out := make([]float64, max)
	for k := range out {
		out[k] = hi - (hi-lo)*float64(k)/float64(max-1)
	}
	return out
}

// yFor maps a value on the [lo, hi] scale to a y pixel within an h-tall box
// inset by pad, inverted so hi sits at the top. A zero-width scale centers.
func yFor(v, lo, hi float64, h, pad float32) float32 {
	yn := 0.5 // flat scale sits in the middle
	if hi != lo {
		yn = (v - lo) / (hi - lo)
	}
	return pad + (h-2*pad)*float32(1-yn)
}
