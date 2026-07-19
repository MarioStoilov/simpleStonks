<script lang="ts">
  import type {Series} from '../bindings/github.com/MarioStoilov/simplestonks/internal/model';
  import {
    closesOf,
    nearestPoint,
    plotPath,
    sessionTicks,
    sessionWindow,
    valueBounds,
    xFracs,
    xTicks,
    yFor,
    yTicks,
  } from './lib/chartMath';
  import {formatPrice, hoverTimeLabel, percentChange} from './lib/format';
  import {
    AXIS_GAP,
    AXIS_TEXT_SIZE,
    CHART_LINE_WIDTH,
    CHART_PADDING,
    COLOR_AXIS,
    COLOR_CHART_BG,
    COLOR_DOWN,
    COLOR_FOREGROUND,
    COLOR_HOVER,
    COLOR_NEUTRAL,
    COLOR_UP,
    DASH_ARRAY,
    DOT_RADIUS,
    HAIRLINE_WIDTH,
    MAX_Y_TICKS,
    PANEL_CORNER_RADIUS,
    TIP_GAP,
    TIP_PAD,
    X_TICK_SPACING,
    Y_TICK_SPACING,
  } from './lib/constants';

  // Chart plots a price series as a line, colored by the caller (up/down),
  // with y price labels, range-aware x time labels, and a dashed previous-close
  // reference line. With hoverReadout enabled (the detail view's expanded
  // chart), hovering marks the nearest data point with a dot, a dashed vertical
  // guide, its time on the x axis, and a price/% tooltip. Port of the Fyne
  // chart widget (internal/ui/chart.go).
  let {
    series = null,
    color = COLOR_NEUTRAL,
    hoverReadout = false,
  }: {
    series?: Series | null;
    color?: string;
    hoverReadout?: boolean;
  } = $props();

  let width = $state(0);
  let height = $state(0);

  // Text measurement for axis label layout (Fyne's fyne.MeasureText).
  const measureContext = document.createElement('canvas').getContext('2d')!;
  function textWidth(text: string): number {
    measureContext.font = `${AXIS_TEXT_SIZE}px system-ui, sans-serif`;
    return measureContext.measureText(text).width;
  }
  const labelHeight = Math.ceil(AXIS_TEXT_SIZE * 1.3);

  type PlacedLabel = {text: string; x: number; y: number};

  const geometry = $derived.by(() => {
    if (!series || width <= 0 || height <= 0) {
      return null;
    }
    const values = closesOf(series);
    if (values.length === 0) {
      return null;
    }
    const prevClose = series.PreviousClose;
    const {low, high} = valueBounds(values, prevClose);

    let bottomMargin = labelHeight + AXIS_GAP;
    let plotH = height - bottomMargin;

    // Price (y) labels: as many evenly spaced reference values as the height
    // allows, plus the previous close; a flat scale gets a single label. The
    // left margin fits the widest label.
    type RawLabel = {value: number; text: string; width: number};
    let rawLabels: RawLabel[];
    if (low === high) {
      rawLabels = [{value: low, text: formatPrice(low), width: 0}];
    } else {
      let count = Math.floor(plotH / Y_TICK_SPACING) + 1;
      count = Math.max(2, Math.min(MAX_Y_TICKS, count));
      rawLabels = yTicks(low, high, count).map((value) => ({value, text: formatPrice(value), width: 0}));
    }
    let prevIdx = -1; // index of the previous-close label, which wins collisions
    if (prevClose > 0 && low !== high) {
      rawLabels.push({value: prevClose, text: formatPrice(prevClose), width: 0});
      prevIdx = rawLabels.length - 1;
    }
    let leftMargin = 0;
    for (const label of rawLabels) {
      label.width = textWidth(label.text);
      if (label.width > leftMargin) {
        leftMargin = label.width;
      }
    }
    leftMargin += AXIS_GAP;

    let plotW = width - leftMargin;
    let bare = false;
    let path = plotPath(values, xFracs(series), plotW, plotH, CHART_PADDING, low, high);
    if (!path) {
      // Too small for margins: fall back to the bare line.
      bare = true;
      leftMargin = 0;
      bottomMargin = 0;
      plotW = width;
      plotH = height;
      path = plotPath(values, xFracs(series), plotW, plotH, CHART_PADDING, low, high);
      if (!path) {
        return null;
      }
    }
    const points = path.map((point) => ({x: point.x + leftMargin, y: point.y}));

    // Dashed reference line at the previous interval's close.
    const prevCloseY = prevClose > 0 ? yFor(prevClose, low, high, plotH, CHART_PADDING) : null;

    // y labels, right-aligned against the plot's left edge and vertically
    // centered on their value; ticks colliding with the previous-close label
    // make way for it.
    const labelYFor = (value: number) => {
      const posY = yFor(value, low, high, plotH, CHART_PADDING) - labelHeight / 2;
      return Math.min(Math.max(posY, 0), plotH - labelHeight);
    };
    const yLabels: PlacedLabel[] = [];
    if (!bare) {
      const prevY = prevIdx >= 0 ? labelYFor(rawLabels[prevIdx].value) : 0;
      rawLabels.forEach((label, index) => {
        const posY = labelYFor(label.value);
        if (prevIdx >= 0 && index !== prevIdx) {
          const delta = posY - prevY;
          if (delta > -labelHeight && delta < labelHeight) {
            return;
          }
        }
        yLabels.push({text: label.text, x: leftMargin - AXIS_GAP - label.width, y: posY});
      });
    }

    // Time (x) labels along the bottom, formatted per the series' range. An
    // intraday series with a known session is labeled across the full window.
    const xLabels: PlacedLabel[] = [];
    if (!bare) {
      const maxTicks = Math.floor(plotW / X_TICK_SPACING);
      const ticks = sessionWindow(series)
        ? sessionTicks(Date.parse(series.SessionStart), Date.parse(series.SessionEnd), maxTicks)
        : xTicks(series, maxTicks);
      for (const tick of ticks) {
        const tickWidth = textWidth(tick.label);
        let posX = leftMargin + CHART_PADDING + tick.frac * (plotW - 2 * CHART_PADDING) - tickWidth / 2;
        posX = Math.max(leftMargin, Math.min(posX, width - tickWidth));
        xLabels.push({text: tick.label, x: posX, y: plotH + AXIS_GAP});
      }
    }

    return {values, points, leftMargin, plotW, plotH, prevCloseY, yLabels, xLabels};
  });

  // Hover readout state: the nearest point index, or null when not hovering.
  let hoverIdx: number | null = $state(null);

  function handleMouseMove(event: MouseEvent): void {
    if (!hoverReadout || !geometry) {
      hoverIdx = null;
      return;
    }
    const bounds = (event.currentTarget as SVGSVGElement).getBoundingClientRect();
    hoverIdx = nearestPoint(geometry.points, event.clientX - bounds.left);
  }

  function handleMouseLeave(): void {
    hoverIdx = null;
  }

  const hover = $derived.by(() => {
    if (!hoverReadout || hoverIdx === null || !geometry || !series) {
      return null;
    }
    const index = Math.min(hoverIdx, geometry.points.length - 1);
    const point = geometry.points[index];
    const value = geometry.values[index];

    // Tooltip: the price, and under it the % change versus the previous close.
    const priceText = formatPrice(value);
    const change = series.PreviousClose > 0 ? percentChange(value, series.PreviousClose) : null;
    let tipWidth = textWidth(priceText);
    let tipHeight = labelHeight;
    if (change) {
      tipWidth = Math.max(tipWidth, textWidth(change.text));
      tipHeight += labelHeight;
    }
    const boxWidth = tipWidth + 2 * TIP_PAD;
    const boxHeight = tipHeight + 2 * TIP_PAD;
    let boxX = point.x - boxWidth / 2;
    boxX = Math.max(0, Math.min(boxX, width - boxWidth));
    let boxY = point.y - DOT_RADIUS - TIP_GAP - boxHeight; // above the dot ...
    if (boxY < 0) {
      boxY = point.y + DOT_RADIUS + TIP_GAP; // ... or below when clipped at the top
    }

    // Time of the hovered point, boxed on the x axis under the guide.
    const timeText = hoverTimeLabel(series.Range, Date.parse((series.Candles ?? [])[index].Time));
    const timeWidth = textWidth(timeText) + 2 * TIP_PAD;
    const timeHeight = labelHeight + 2;
    let timeX = point.x - timeWidth / 2;
    timeX = Math.max(0, Math.min(timeX, width - timeWidth));
    const timeY = geometry.plotH + (height - geometry.plotH - timeHeight) / 2;

    return {point, priceText, change, boxX, boxY, boxWidth, boxHeight, timeText, timeX, timeY, timeWidth, timeHeight};
  });

  function directionColor(direction: 'up' | 'down' | 'flat'): string {
    return direction === 'up' ? COLOR_UP : direction === 'down' ? COLOR_DOWN : COLOR_NEUTRAL;
  }
</script>

<div class="chart" bind:clientWidth={width} bind:clientHeight={height}>
  {#if geometry}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <svg {width} {height} role="img" onmousemove={handleMouseMove} onmouseleave={handleMouseLeave}>
      <rect x="0" y="0" {width} {height} fill={COLOR_CHART_BG} />
      {#if geometry.prevCloseY !== null}
        <line
          x1={geometry.leftMargin + CHART_PADDING}
          y1={geometry.prevCloseY}
          x2={geometry.leftMargin + geometry.plotW - CHART_PADDING}
          y2={geometry.prevCloseY}
          stroke={COLOR_AXIS}
          stroke-width={HAIRLINE_WIDTH}
          stroke-dasharray={DASH_ARRAY}
        />
      {/if}
      <polyline
        points={geometry.points.map((point) => `${point.x},${point.y}`).join(' ')}
        fill="none"
        stroke={color}
        stroke-width={CHART_LINE_WIDTH}
      />
      {#each geometry.yLabels as label}
        <text x={label.x} y={label.y} fill={COLOR_AXIS} font-size={AXIS_TEXT_SIZE} dominant-baseline="hanging">
          {label.text}
        </text>
      {/each}
      {#each geometry.xLabels as label}
        <text x={label.x} y={label.y} fill={COLOR_AXIS} font-size={AXIS_TEXT_SIZE} dominant-baseline="hanging">
          {label.text}
        </text>
      {/each}
      {#if hover}
        <line
          x1={hover.point.x}
          y1={CHART_PADDING}
          x2={hover.point.x}
          y2={geometry.plotH - CHART_PADDING}
          stroke={COLOR_AXIS}
          stroke-width={HAIRLINE_WIDTH}
          stroke-dasharray={DASH_ARRAY}
        />
        <circle
          cx={hover.point.x}
          cy={hover.point.y}
          r={DOT_RADIUS}
          fill={color}
          stroke={COLOR_FOREGROUND}
          stroke-width={HAIRLINE_WIDTH}
        />
        <rect
          x={hover.timeX}
          y={hover.timeY}
          width={hover.timeWidth}
          height={hover.timeHeight}
          rx={PANEL_CORNER_RADIUS}
          fill={COLOR_HOVER}
        />
        <text
          x={hover.timeX + TIP_PAD}
          y={hover.timeY + 1}
          fill={COLOR_FOREGROUND}
          font-size={AXIS_TEXT_SIZE}
          dominant-baseline="hanging"
        >
          {hover.timeText}
        </text>
        <rect
          x={hover.boxX}
          y={hover.boxY}
          width={hover.boxWidth}
          height={hover.boxHeight}
          rx={PANEL_CORNER_RADIUS}
          fill={COLOR_HOVER}
        />
        <text
          x={hover.boxX + TIP_PAD}
          y={hover.boxY + TIP_PAD}
          fill={COLOR_FOREGROUND}
          font-size={AXIS_TEXT_SIZE}
          dominant-baseline="hanging"
        >
          {hover.priceText}
        </text>
        {#if hover.change}
          <text
            x={hover.boxX + TIP_PAD}
            y={hover.boxY + TIP_PAD + labelHeight}
            fill={directionColor(hover.change.direction)}
            font-size={AXIS_TEXT_SIZE}
            dominant-baseline="hanging"
          >
            {hover.change.text}
          </text>
        {/if}
      {/if}
    </svg>
  {/if}
</div>

<style>
  .chart {
    width: 100%;
    height: 100%;
    min-height: 80px;
    background: #1e1e24; /* COLOR_CHART_BG */
    overflow: hidden;
  }
  svg {
    display: block;
  }
</style>
