<script lang="ts">
  import {Range} from '../bindings/github.com/MarioStoilov/simplestonks/internal/model';
  import {Settings} from '../bindings/github.com/MarioStoilov/simplestonks/internal/service';
  import SearchDialog from './SearchDialog.svelte';
  import Tile from './Tile.svelte';
  import {loadHistory} from './lib/market';
  import type {HistoryResult} from './lib/market';

  // Home is the grid of tracked-symbol tiles at 1D, loading each series on
  // mount, on symbol-list changes, and on the periodic refresh tick. The top
  // bar toggles edit mode (reorder/remove) and opens the add-symbol search.
  let {
    symbols,
    refreshMs,
    onopen,
    onopensettings,
  }: {
    symbols: string[];
    refreshMs: number;
    onopen: (symbol: string) => void;
    onopensettings: () => void;
  } = $props();

  let entries: Record<string, HistoryResult> = $state({});
  let editing = $state(false);
  let searchOpen = $state(false);

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

  // Mutations persist through the config store; the resulting configChanged
  // event updates the symbols prop, which re-renders the grid.
  function moveSymbol(index: number, delta: number): void {
    Settings.MoveSymbol(index, delta).catch((err) => console.error('move symbol failed:', err));
  }

  function removeSymbol(symbol: string): void {
    Settings.RemoveSymbol(symbol).catch((err) => console.error('remove symbol failed:', err));
  }
</script>

<div class="home">
  <div class="topbar">
    <button
      class="bar-btn"
      class:active={editing}
      onclick={() => {
        editing = !editing;
      }}
    >
      {editing ? 'Done' : 'Edit'}
    </button>
    <button
      class="bar-btn"
      onclick={() => {
        searchOpen = true;
      }}
    >
      + Add
    </button>
    <button class="bar-btn" onclick={onopensettings} aria-label="Settings">⚙</button>
  </div>
  <div class="grid">
    {#each symbols as symbol, index (symbol)}
      <Tile
        {symbol}
        series={entries[symbol]?.series ?? null}
        failed={entries[symbol]?.failed ?? false}
        {editing}
        canMoveLeft={index > 0}
        canMoveRight={index < symbols.length - 1}
        onopen={() => onopen(symbol)}
        onmoveleft={() => moveSymbol(index, -1)}
        onmoveright={() => moveSymbol(index, 1)}
        onremove={() => removeSymbol(symbol)}
      />
    {/each}
  </div>
</div>

{#if searchOpen}
  <SearchDialog
    tracked={symbols}
    onclose={() => {
      searchOpen = false;
    }}
  />
{/if}

<style>
  .home {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  .topbar {
    display: flex;
    justify-content: flex-end;
    gap: 0.4rem;
    padding: 0.6rem 0.75rem 0;
  }
  .bar-btn {
    background: #24262e; /* COLOR_CARD_BG */
    color: inherit;
    border: none;
    border-radius: 4px;
    padding: 0.3rem 0.8rem;
    cursor: pointer;
  }
  .bar-btn:hover {
    background: #2c303c; /* COLOR_HOVER */
  }
  .bar-btn.active {
    background: #303a52; /* COLOR_SELECTED */
    font-weight: 600;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); /* GRID_CELL_WIDTH */
    gap: 0.75rem;
    padding: 0.75rem;
    flex: 1;
    overflow-y: auto;
    align-content: start;
  }
</style>
