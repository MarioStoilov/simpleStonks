// Formatting helpers mirroring the Go side's time layouts and number formats
// (internal/constants constants.go: TimeFmt*/Fmt*), using explicit en-GB
// formatting so labels match the original UI ("02 Jan", "Mon 02", ...).

import {Range} from '../../bindings/github.com/MarioStoilov/simplestonks/internal/model';

const LOCALE = 'en-GB';

const clockFormat = new Intl.DateTimeFormat(LOCALE, {hour: '2-digit', minute: '2-digit', hour12: false});
const weekdayDayFormat = new Intl.DateTimeFormat(LOCALE, {weekday: 'short', day: '2-digit'});
const dayMonthFormat = new Intl.DateTimeFormat(LOCALE, {day: '2-digit', month: 'short'});
const monthFormat = new Intl.DateTimeFormat(LOCALE, {month: 'short'});
const yearFormat = new Intl.DateTimeFormat(LOCALE, {year: 'numeric'});
const weekdayDateFormat = new Intl.DateTimeFormat(LOCALE, {weekday: 'short', day: '2-digit', month: 'short'});
const fullDateFormat = new Intl.DateTimeFormat(LOCALE, {day: '2-digit', month: 'short', year: 'numeric'});

export function formatPrice(value: number): string {
  return value.toFixed(2);
}

// xAxisLabel maps a chart range to its x-axis label format: hours intraday,
// then days, dates, months, and years as the span grows (chart.go xAxisFormat).
export function xAxisLabel(range: Range, timeMs: number): string {
  const when = new Date(timeMs);
  switch (range) {
    case Range.Range1D:
      return clockFormat.format(when);
    case Range.Range5D:
    case Range.Range1W:
      return weekdayDayFormat.format(when);
    case Range.Range1M:
      return dayMonthFormat.format(when);
    case Range.RangeYTD:
    case Range.Range1Y:
      return monthFormat.format(when);
    default: // 5Y, ALL
      return yearFormat.format(when);
  }
}

// hoverTimeLabel maps a chart range to the hover readout's time label: clock
// time intraday, calendar dates beyond (chart.go hoverTimeFormat).
export function hoverTimeLabel(range: Range, timeMs: number): string {
  const when = new Date(timeMs);
  switch (range) {
    case Range.Range1D:
      return clockFormat.format(when);
    case Range.Range5D:
    case Range.Range1W:
    case Range.Range1M:
      return weekdayDateFormat.format(when);
    default: // YTD, 1Y, 5Y, ALL
      return fullDateFormat.format(when);
  }
}

export type ChangeDirection = 'up' | 'down' | 'flat';

// priceChange renders "+1.23 (+0.45%)" versus the previous close, with the
// movement direction for coloring (ui priceChangeText/changeStyle).
export function priceChange(last: number, previousClose: number): {text: string; direction: ChangeDirection} {
  if (previousClose <= 0) {
    return {text: '', direction: 'flat'};
  }
  const change = last - previousClose;
  const percent = (change / previousClose) * 100;
  const sign = change > 0 ? '+' : '';
  const direction: ChangeDirection = change > 0 ? 'up' : change < 0 ? 'down' : 'flat';
  return {text: `${sign}${change.toFixed(2)} (${sign}${percent.toFixed(2)}%)`, direction};
}

// percentChange renders a signed percent change, e.g. "+1.23%".
export function percentChange(value: number, reference: number): {text: string; direction: ChangeDirection} {
  const percent = ((value - reference) / reference) * 100;
  const sign = percent > 0 ? '+' : '';
  const direction: ChangeDirection = percent > 0 ? 'up' : percent < 0 ? 'down' : 'flat';
  return {text: `${sign}${percent.toFixed(2)}%`, direction};
}
