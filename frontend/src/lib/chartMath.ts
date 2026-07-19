// Pure chart math ported from internal/ui/chart.go so the web chart renders
// identically to the original Fyne widget: value→pixel mapping, session-window
// x fractions for gradually-filling intraday charts, and axis tick selection.

import {Range} from '../../bindings/github.com/MarioStoilov/simplestonks/internal/model';
import type {Series} from '../../bindings/github.com/MarioStoilov/simplestonks/internal/model';
import {xAxisLabel} from './format';

export type Point = {x: number; y: number};
export type AxisTick = {frac: number; label: string};

// closesOf extracts the closing prices from a series.
export function closesOf(series: Series): number[] {
  return (series.Candles ?? []).map((candle) => candle.Close);
}

// candleTimes extracts candle timestamps as epoch milliseconds.
export function candleTimes(series: Series): number[] {
  return (series.Candles ?? []).map((candle) => Date.parse(candle.Time));
}

// valueBounds returns the [low, high] value scale for the plot, widened to
// include the previous close (the dashed reference line) when known.
export function valueBounds(values: number[], previousClose: number): {low: number; high: number} {
  let low = values[0];
  let high = values[0];
  for (const value of values) {
    if (value < low) low = value;
    if (value > high) high = value;
  }
  if (previousClose > 0) {
    if (previousClose < low) low = previousClose;
    if (previousClose > high) high = previousClose;
  }
  return {low, high};
}

// sessionWindow reports whether a series should be drawn against its full
// trading-session window: intraday data with known session bounds whose
// candles actually belong to that session. While a market is closed, Yahoo
// pairs the previous day's candles with the upcoming session, which must fall
// back to even spacing (see chart.go sessionWindow for the full story).
export function sessionWindow(series: Series): boolean {
  const candles = series.Candles ?? [];
  if (series.Range !== Range.Range1D || candles.length === 0) {
    return false;
  }
  const start = Date.parse(series.SessionStart);
  const end = Date.parse(series.SessionEnd);
  if (!(end > start)) {
    return false;
  }
  return Date.parse(candles[candles.length - 1].Time) >= start;
}

// evenFracs returns count positions spaced evenly across 0..1.
export function evenFracs(count: number): number[] {
  if (count < 2) {
    return [];
  }
  return Array.from({length: count}, (unused, index) => index / (count - 1));
}

// xFracs returns each candle's horizontal position as a 0..1 fraction: time
// based over the session window when live (so the day fills in gradually),
// evenly spaced otherwise.
export function xFracs(series: Series): number[] {
  const candles = series.Candles ?? [];
  if (!sessionWindow(series)) {
    return evenFracs(candles.length);
  }
  const start = Date.parse(series.SessionStart);
  const span = Date.parse(series.SessionEnd) - start;
  return candles.map((candle) => {
    const frac = (Date.parse(candle.Time) - start) / span;
    return Math.min(1, Math.max(0, frac));
  });
}

// plotPath maps close values to pixel positions within a width×height box
// (inset by pad): x from the given 0..1 fractions, y scaled to [low, high],
// inverted so higher values sit toward the top. Returns null when there is
// nothing meaningful to draw.
export function plotPath(
  values: number[],
  fracs: number[],
  width: number,
  height: number,
  pad: number,
  low: number,
  high: number,
): Point[] | null {
  if (values.length < 2 || fracs.length !== values.length) {
    return null;
  }
  const innerWidth = width - 2 * pad;
  if (innerWidth <= 0 || height - 2 * pad <= 0) {
    return null;
  }
  return values.map((value, index) => ({
    x: pad + innerWidth * fracs[index],
    y: yFor(value, low, high, height, pad),
  }));
}

// yFor maps a value on the [low, high] scale to a y pixel within a height-tall
// box inset by pad, inverted so high sits at the top. A flat scale centers.
export function yFor(value: number, low: number, high: number, height: number, pad: number): number {
  const normalized = high === low ? 0.5 : (value - low) / (high - low);
  return pad + (height - 2 * pad) * (1 - normalized);
}

// yTicks returns max evenly spaced reference values from high (first) down to low.
export function yTicks(low: number, high: number, max: number): number[] {
  if (max < 2 || high <= low) {
    return [];
  }
  return Array.from({length: max}, (unused, index) => high - ((high - low) * index) / (max - 1));
}

// xTicks picks up to max evenly spaced candles from the series and formats
// their timestamps for the series' range. Consecutive duplicate labels — e.g.
// the same year twice on a 5Y chart — are dropped.
export function xTicks(series: Series, max: number): AxisTick[] {
  const times = candleTimes(series);
  const count = times.length;
  if (count < 2 || max < 2) {
    return [];
  }
  const limit = Math.min(max, count);
  const out: AxisTick[] = [];
  let prev = '';
  for (let tickIdx = 0; tickIdx < limit; tickIdx++) {
    const candleIdx = Math.floor((tickIdx * (count - 1)) / (limit - 1));
    const label = xAxisLabel(series.Range, times[candleIdx]);
    if (label === prev) {
      continue;
    }
    prev = label;
    out.push({frac: candleIdx / (count - 1), label});
  }
  return out;
}

// sessionTicks returns up to max evenly spaced intraday time labels spanning a
// trading-session window.
export function sessionTicks(startMs: number, endMs: number, max: number): AxisTick[] {
  if (max < 2 || !(endMs > startMs)) {
    return [];
  }
  const span = endMs - startMs;
  const out: AxisTick[] = [];
  let prev = '';
  for (let tickIdx = 0; tickIdx < max; tickIdx++) {
    const tickTime = startMs + (span * tickIdx) / (max - 1);
    const label = xAxisLabel(Range.Range1D, tickTime);
    if (label === prev) {
      continue;
    }
    prev = label;
    out.push({frac: tickIdx / (max - 1), label});
  }
  return out;
}

// nearestPoint returns the index of the point whose x coordinate is closest to
// targetX. Points must be ordered by non-decreasing x (as plotted points are).
export function nearestPoint(points: Point[], targetX: number): number {
  let best = 0;
  let bestDist = -1;
  for (let index = 0; index < points.length; index++) {
    const dist = Math.abs(points[index].x - targetX);
    if (bestDist < 0 || dist < bestDist) {
      best = index;
      bestDist = dist;
    }
  }
  return best;
}
