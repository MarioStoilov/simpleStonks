<script lang="ts">
  import {Range} from '../bindings/github.com/MarioStoilov/simplestonks/internal/model';
  import type {SearchResult} from '../bindings/github.com/MarioStoilov/simplestonks/internal/model';
  import Chart from './Chart.svelte';
  import Modal from './Modal.svelte';
  import Price from './Price.svelte';
  import {priceChange} from './lib/format';
  import {
    COLOR_DOWN,
    COLOR_NEUTRAL,
    COLOR_UP,
    MSG_UNAVAILABLE,
    SEP_META,
    SEP_TITLE,
    TIP_ALREADY_TRACKED,
  } from './lib/constants';
  import {lastClose, loadHistory} from './lib/market';
  import type {HistoryResult} from './lib/market';

  // Preview is a compact look at one search result — symbol/name header,
  // price/change, a 1D chart — with Add and Cancel actions. Add is disabled
  // (with a tooltip) when the index is already tracked.
  let {
    result,
    tracked,
    onadd,
    oncancel,
  }: {
    result: SearchResult;
    tracked: string[];
    onadd: (symbol: string) => void;
    oncancel: () => void;
  } = $props();

  let loaded = $state<HistoryResult | null>(null);
  $effect(() => {
    loadHistory(result.Symbol, Range.Range1D).then((history) => {
      loaded = history;
    });
  });

  const title = $derived(result.Name ? `${result.Symbol}${SEP_TITLE}${result.Name}` : result.Symbol);
  const subtitle = $derived([result.Exchange, result.Type].filter(Boolean).join(SEP_META));
  const alreadyTracked = $derived(tracked.includes(result.Symbol));

  const lastPrice = $derived(lastClose(loaded?.series ?? null));
  const change = $derived(
    lastPrice !== null && loaded?.series ? priceChange(lastPrice, loaded.series.PreviousClose) : null,
  );
  const changeColor = $derived(
    change?.direction === 'up' ? COLOR_UP : change?.direction === 'down' ? COLOR_DOWN : COLOR_NEUTRAL,
  );
  const failed = $derived(loaded?.failed ?? false);
</script>

<Modal title="Preview" onclose={oncancel}>
  <div class="head">
    <div class="ident">
      <span class="title">{title}</span>
      {#if subtitle}<span class="subtitle">{subtitle}</span>{/if}
    </div>
    <div class="quote">
      <Price value={lastPrice} />
      <span class="change" style:color={failed ? COLOR_NEUTRAL : changeColor}>
        {failed ? MSG_UNAVAILABLE : (change?.text ?? '')}
      </span>
    </div>
  </div>
  <div class="chartbox">
    <Chart series={loaded?.series ?? null} color={failed ? COLOR_NEUTRAL : changeColor} />
  </div>
  <footer class="actions">
    <button class="btn" onclick={oncancel}>Cancel</button>
    <button
      class="btn primary"
      disabled={alreadyTracked}
      title={alreadyTracked ? TIP_ALREADY_TRACKED : undefined}
      onclick={() => onadd(result.Symbol)}
    >
      Add
    </button>
  </footer>
</Modal>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
    width: 480px;
    max-width: 86vw;
  }
  .ident {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .title {
    font-weight: 600;
  }
  .subtitle {
    color: #8a8a8a; /* COLOR_NEUTRAL */
    font-size: 0.8rem;
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
  .chartbox {
    height: 260px;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }
  .btn {
    background: #2c303c; /* COLOR_HOVER */
    color: inherit;
    border: none;
    border-radius: 4px;
    padding: 0.35rem 0.9rem;
    cursor: pointer;
  }
  .btn:hover:not(:disabled) {
    background: #303a52; /* COLOR_SELECTED */
  }
  .btn.primary {
    background: #303a52; /* COLOR_SELECTED */
    font-weight: 600;
  }
  .btn:disabled {
    background: #3a3a40; /* COLOR_DISABLED_BG */
    color: #808080; /* COLOR_DISABLED_FG */
    cursor: not-allowed;
  }
</style>
