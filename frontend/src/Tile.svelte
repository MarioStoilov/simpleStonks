<script lang="ts">
  import type {Series} from '../bindings/github.com/MarioStoilov/simplestonks/internal/model';
  import Chart from './Chart.svelte';
  import Price from './Price.svelte';
  import {priceChange} from './lib/format';
  import {COLOR_DOWN, COLOR_NEUTRAL, COLOR_UP, MSG_UNAVAILABLE} from './lib/constants';
  import {lastClose} from './lib/market';

  // Tile is one tracked-symbol card: symbol + friendly name, latest price with
  // change (colored) and a live flash, and a mini 1D chart. compact shrinks it
  // for the detail view's sidebar; editing shows a reorder/remove control row
  // and disables opening the detail view.
  let {
    symbol,
    series = null,
    failed = false,
    selected = false,
    compact = false,
    editing = false,
    canMoveLeft = false,
    canMoveRight = false,
    onopen,
    onmoveleft,
    onmoveright,
    onremove,
  }: {
    symbol: string;
    series?: Series | null;
    failed?: boolean;
    selected?: boolean;
    compact?: boolean;
    editing?: boolean;
    canMoveLeft?: boolean;
    canMoveRight?: boolean;
    onopen?: () => void;
    onmoveleft?: () => void;
    onmoveright?: () => void;
    onremove?: () => void;
  } = $props();

  const lastPrice = $derived(lastClose(series));
  const change = $derived(lastPrice !== null && series ? priceChange(lastPrice, series.PreviousClose) : null);
  const changeColor = $derived(
    change?.direction === 'up' ? COLOR_UP : change?.direction === 'down' ? COLOR_DOWN : COLOR_NEUTRAL,
  );
</script>

<div
  class="tile"
  class:selected
  class:compact
  role="button"
  tabindex="0"
  onclick={() => {
    if (!editing) onopen?.();
  }}
  onkeydown={(event) => {
    if (event.key === 'Enter' && !editing) onopen?.();
  }}
>
  <div class="head">
    <div class="ident">
      <span class="symbol">{symbol}</span>
      {#if series?.Name}<span class="name">{series.Name}</span>{/if}
    </div>
    <div class="quote">
      <Price value={lastPrice} />
      <span class="change" style:color={failed ? COLOR_NEUTRAL : changeColor}>
        {failed ? MSG_UNAVAILABLE : (change?.text ?? '')}
      </span>
    </div>
  </div>
  <div class="chartbox">
    <Chart {series} color={failed ? COLOR_NEUTRAL : changeColor} />
  </div>
  {#if editing}
    <div class="controls">
      <button class="ctrl" disabled={!canMoveLeft} onclick={onmoveleft} aria-label="Move left">◀</button>
      <button class="ctrl" disabled={!canMoveRight} onclick={onmoveright} aria-label="Move right">▶</button>
      <button class="ctrl remove" onclick={onremove} aria-label="Remove">✕</button>
    </div>
  {/if}
</div>

<style>
  .tile {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.65rem;
    background: #24262e; /* COLOR_CARD_BG */
    border-radius: 6px; /* TILE_CORNER_RADIUS */
    cursor: pointer;
    min-height: 200px;
  }
  .tile.compact {
    min-height: 150px;
    padding: 0.5rem;
  }
  .tile:hover {
    background: #2c303c; /* COLOR_HOVER */
  }
  .tile.selected {
    background: #303a52; /* COLOR_SELECTED */
  }
  .head {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 0.5rem;
  }
  .ident {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .symbol {
    font-weight: 600;
  }
  .name {
    color: #6e727e; /* COLOR_AXIS */
    font-size: 11px; /* NameTextSize */
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .quote {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    flex-shrink: 0;
  }
  .change {
    font-size: 0.85rem;
    font-variant-numeric: tabular-nums;
  }
  .compact .change {
    font-size: 0.75rem;
  }
  .chartbox {
    flex: 1;
    min-height: 80px; /* CHART_MIN_HEIGHT */
  }
  .compact .chartbox {
    min-height: 60px;
  }
  .controls {
    display: flex;
    gap: 0.3rem;
  }
  .ctrl {
    background: #2c303c; /* COLOR_HOVER */
    color: inherit;
    border: none;
    border-radius: 4px;
    padding: 0.2rem 0.55rem;
    cursor: pointer;
    font-size: 0.8rem;
  }
  .ctrl:hover:not(:disabled) {
    background: #303a52; /* COLOR_SELECTED */
  }
  .ctrl:disabled {
    color: #808080; /* COLOR_DISABLED_FG */
    cursor: not-allowed;
  }
  .ctrl.remove {
    margin-left: auto;
    color: #d03a3a; /* COLOR_DOWN */
  }
</style>
