// Shared market-data helpers for views that load price history.

import {Market} from '../../bindings/github.com/MarioStoilov/simplestonks/internal/service';
import {Range} from '../../bindings/github.com/MarioStoilov/simplestonks/internal/model';
import type {Series} from '../../bindings/github.com/MarioStoilov/simplestonks/internal/model';

export type HistoryResult = {series: Series | null; failed: boolean};

// loadHistory fetches a symbol's series at a range, folding fetch errors and
// empty series into a failed result (rendered as "unavailable").
export async function loadHistory(symbol: string, range: Range): Promise<HistoryResult> {
  try {
    const series = await Market.History(symbol, range);
    return {series, failed: (series.Candles ?? []).length === 0};
  } catch {
    return {series: null, failed: true};
  }
}

// RANGES is the ordered set of ranges shown as toggles (model.Ranges).
export const RANGES: Range[] = [
  Range.Range1D,
  Range.Range5D,
  Range.Range1W,
  Range.Range1M,
  Range.RangeYTD,
  Range.Range1Y,
  Range.Range5Y,
  Range.RangeAll,
];

// lastClose returns a series' most recent closing price, or null.
export function lastClose(series: Series | null): number | null {
  const candles = series?.Candles ?? [];
  return candles.length > 0 ? candles[candles.length - 1].Close : null;
}
