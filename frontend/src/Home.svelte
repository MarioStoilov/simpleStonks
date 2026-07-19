<script lang="ts">
  import {Range} from '../bindings/github.com/MarioStoilov/simplestonks/internal/model';
  import Tile from './Tile.svelte';
  import {loadHistory} from './lib/market';
  import type {HistoryResult} from './lib/market';

  // Home is the grid of tracked-symbol tiles at 1D, loading each series on
  // mount, on symbol-list changes, and on the periodic refresh tick.
  let {
    symbols,
    refreshMs,
    onopen,
  }: {
    symbols: string[];
    refreshMs: number;
    onopen: (symbol: string) => void;
  } = $props();

  let entries: Record<string, HistoryResult> = $state({});

  function loadAll(): void {
    for (const symbol of symbols) {
      loadHistory(symbol, Range.Range1D).then((result) => {
        entries[symbol] = result;
      });
    }
  }

  // Reload whenever the tracked list changes (also covers the initial load).
  $effect(() => {
    symbols;
    loadAll();
  });

  // Periodic refresh at the configured cadence.
  $effect(() => {
    const timer = setInterval(loadAll, refreshMs);
    return () => clearInterval(timer);
  });
</script>

<div class="grid">
  {#each symbols as symbol (symbol)}
    <Tile
      {symbol}
      series={entries[symbol]?.series ?? null}
      failed={entries[symbol]?.failed ?? false}
      onopen={() => onopen(symbol)}
    />
  {/each}
</div>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); /* GRID_CELL_WIDTH */
    gap: 0.75rem;
    padding: 0.75rem;
    height: 100%;
    overflow-y: auto;
    align-content: start;
  }
</style>
