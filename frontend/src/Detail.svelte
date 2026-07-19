<script lang="ts">
  import {Range} from '../bindings/github.com/MarioStoilov/simplestonks/internal/model';
  import Chart from './Chart.svelte';
  import Price from './Price.svelte';
  import Tile from './Tile.svelte';
  import {priceChange} from './lib/format';
  import {COLOR_DOWN, COLOR_NEUTRAL, COLOR_UP, MSG_UNAVAILABLE} from './lib/constants';
  import {RANGES, lastClose, loadHistory} from './lib/market';
  import type {HistoryResult} from './lib/market';

  // Detail is the expanded view of one symbol: a sidebar of all tracked
  // symbols (current one highlighted), a header with price/change, the range
  // toggles, and the big chart with the hover readout. The parent remounts it
  // (keyed by symbol) when the selection changes, resetting the range to the
  // configured default — mirroring the Fyne detail screen.
  let {
    symbols,
    symbol,
    defaultRange,
    refreshMs,
    onback,
    onselect,
  }: {
    symbols: string[];
    symbol: string;
    defaultRange: Range;
    refreshMs: number;
    onback: () => void;
    onselect: (symbol: string) => void;
  } = $props();

  // svelte-ignore state_referenced_locally -- capturing the initial value is
  // intended: the parent remounts Detail per symbol, resetting the range.
  let rng = $state<Range>(defaultRange);
  let main = $state<HistoryResult | null>(null);
  let sidebar: Record<string, HistoryResult> = $state({});

  async function loadMain(): Promise<void> {
    main = await loadHistory(symbol, rng);
  }

  function loadSidebar(): void {
    for (const tracked of symbols) {
      loadHistory(tracked, Range.Range1D).then((result) => {
        sidebar[tracked] = result;
      });
    }
  }

  // Reload the main chart on range changes (and initially).
  $effect(() => {
    rng;
    loadMain();
  });

  // Load sidebar tiles initially and when the tracked list changes.
  $effect(() => {
    symbols;
    loadSidebar();
  });

  // Periodic refresh: sidebar (1D) always, the main chart only when intraday.
  $effect(() => {
    const timer = setInterval(() => {
      loadSidebar();
      if (rng === Range.Range1D) {
        loadMain();
      }
    }, refreshMs);
    return () => clearInterval(timer);
  });

  const lastPrice = $derived(lastClose(main?.series ?? null));
  const change = $derived(
    lastPrice !== null && main?.series ? priceChange(lastPrice, main.series.PreviousClose) : null,
  );
  const changeColor = $derived(
    change?.direction === 'up' ? COLOR_UP : change?.direction === 'down' ? COLOR_DOWN : COLOR_NEUTRAL,
  );
  const failed = $derived(main?.failed ?? false);
</script>

<div class="detail">
  <aside class="sidebar">
    {#each symbols as tracked (tracked)}
      <Tile
        symbol={tracked}
        series={sidebar[tracked]?.series ?? null}
        failed={sidebar[tracked]?.failed ?? false}
        selected={tracked === symbol}
        compact
        onopen={() => onselect(tracked)}
      />
    {/each}
  </aside>
  <section class="main">
    <header class="head">
      <button class="back" onclick={onback} aria-label="Back">←</button>
      <div class="ident">
        <span class="symbol">{symbol}</span>
        {#if main?.series?.Name}<span class="name">{main.series.Name}</span>{/if}
      </div>
      <div class="quote">
        {#key rng}
          <Price value={lastPrice} />
        {/key}
        <span class="change" style:color={failed ? COLOR_NEUTRAL : changeColor}>
          {failed ? MSG_UNAVAILABLE : (change?.text ?? '')}
        </span>
      </div>
    </header>
    <nav class="ranges">
      {#each RANGES as rangeOption (rangeOption)}
        <button
          class="range"
          class:active={rangeOption === rng}
          onclick={() => {
            rng = rangeOption;
          }}
        >
          {rangeOption}
        </button>
      {/each}
    </nav>
    <div class="chartbox">
      <Chart series={main?.series ?? null} color={failed ? COLOR_NEUTRAL : changeColor} hoverReadout />
    </div>
  </section>
</div>

<style>
  .detail {
    display: flex;
    height: 100%;
    gap: 0.5rem;
    padding: 0.5rem;
  }
  .sidebar {
    width: 190px; /* SidebarMinWidth */
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    overflow-y: auto;
  }
  .main {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    min-width: 0;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }
  .back {
    background: #24262e; /* COLOR_CARD_BG */
    color: inherit;
    border: none;
    border-radius: 4px;
    padding: 0.3rem 0.6rem;
    cursor: pointer;
    font-size: 1rem;
  }
  .back:hover {
    background: #2c303c; /* COLOR_HOVER */
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
  }
  .quote {
    margin-left: auto;
    display: flex;
    align-items: baseline;
    gap: 0.6rem;
  }
  .change {
    font-variant-numeric: tabular-nums;
  }
  .ranges {
    display: flex;
    gap: 0.3rem;
    overflow-x: auto;
  }
  .range {
    background: #24262e; /* COLOR_CARD_BG */
    color: inherit;
    border: none;
    border-radius: 4px;
    padding: 0.3rem 0.7rem;
    cursor: pointer;
  }
  .range:hover {
    background: #2c303c; /* COLOR_HOVER */
  }
  .range.active {
    background: #303a52; /* COLOR_SELECTED */
    font-weight: 600;
  }
  .chartbox {
    flex: 1;
    min-height: 0;
  }
</style>
