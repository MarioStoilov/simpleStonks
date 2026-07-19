<script lang="ts">
  import {Market, Settings} from '../bindings/github.com/MarioStoilov/simplestonks/internal/service';
  import type {SearchResult} from '../bindings/github.com/MarioStoilov/simplestonks/internal/model';
  import Modal from './Modal.svelte';
  import Preview from './Preview.svelte';
  import {
    MSG_NO_MATCHES,
    MSG_SEARCHING,
    MSG_SEARCH_FAILED,
    MSG_SEARCH_PROMPT,
    PLACEHOLDER_SEARCH,
    SEARCH_DEBOUNCE_MS,
    SEP_META,
    SEP_TITLE,
  } from './lib/constants';

  // SearchDialog is the live-search add dialog: typing queries the provider
  // (debounced, with a generation counter dropping stale responses) and shows
  // suggestions; picking one opens a Preview, and adding from there closes the
  // whole dialog (port of the Fyne search dialog + controller).
  let {
    tracked,
    onclose,
  }: {
    tracked: string[];
    onclose: () => void;
  } = $props();

  let query = $state('');
  let status = $state(MSG_SEARCH_PROMPT);
  let results = $state<SearchResult[]>([]);
  let preview = $state<SearchResult | null>(null);

  let generation = 0;
  let timer: ReturnType<typeof setTimeout> | null = null;

  function onQueryChanged(): void {
    const trimmed = query.trim();
    generation += 1;
    const requestGen = generation;
    if (timer) {
      clearTimeout(timer);
    }
    if (trimmed === '') {
      status = MSG_SEARCH_PROMPT;
      results = [];
      return;
    }
    status = MSG_SEARCHING;
    timer = setTimeout(async () => {
      try {
        const found = await Market.Search(trimmed);
        if (requestGen !== generation) {
          return; // superseded by a newer keystroke
        }
        results = found ?? [];
        status = results.length === 0 ? MSG_NO_MATCHES : `${results.length} result(s)`;
      } catch {
        if (requestGen !== generation) {
          return;
        }
        results = [];
        status = MSG_SEARCH_FAILED;
      }
    }, SEARCH_DEBOUNCE_MS);
  }

  async function addSymbol(symbol: string): Promise<void> {
    try {
      await Settings.AddSymbol(symbol);
    } catch (err) {
      console.error('add symbol failed:', err);
    }
    onclose();
  }

  function rowTitle(result: SearchResult): string {
    return result.Name ? `${result.Symbol}${SEP_TITLE}${result.Name}` : result.Symbol;
  }

  function rowSubtitle(result: SearchResult): string {
    return [result.Exchange, result.Type].filter(Boolean).join(SEP_META);
  }

  function focusInput(node: HTMLInputElement): void {
    node.focus();
  }
</script>

<Modal title="Add index" {onclose}>
  <input
    class="query"
    type="text"
    placeholder={PLACEHOLDER_SEARCH}
    bind:value={query}
    oninput={onQueryChanged}
    use:focusInput
  />
  <div class="results">
    {#each results as result (result.Symbol)}
      <div
        class="row"
        role="button"
        tabindex="0"
        onclick={() => {
          preview = result;
        }}
        onkeydown={(event) => {
          if (event.key === 'Enter') preview = result;
        }}
      >
        <span class="row-title">{rowTitle(result)}</span>
        <span class="row-subtitle">{rowSubtitle(result)}</span>
      </div>
    {/each}
  </div>
  <p class="status">{status}</p>
</Modal>

{#if preview}
  <Preview
    result={preview}
    {tracked}
    onadd={addSymbol}
    oncancel={() => {
      preview = null;
    }}
  />
{/if}

<style>
  .query {
    width: 460px;
    max-width: 84vw;
    background: #1e1e24; /* COLOR_CHART_BG */
    color: inherit;
    border: 1px solid #6e727e; /* COLOR_AXIS */
    border-radius: 4px;
    padding: 0.45rem 0.6rem;
    font-size: 0.9rem;
    outline: none;
  }
  .query:focus {
    border-color: #303a52; /* COLOR_SELECTED */
  }
  .results {
    height: 280px; /* SearchScrollMinHeight */
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }
  .row {
    display: flex;
    flex-direction: column;
    padding: 0.4rem 0.5rem;
    border-radius: 4px; /* PANEL_CORNER_RADIUS */
    cursor: pointer;
  }
  .row:hover {
    background: #2c303c; /* COLOR_HOVER */
  }
  .row-title {
    font-weight: 600;
    font-size: 0.9rem;
  }
  .row-subtitle {
    color: #8a8a8a; /* COLOR_NEUTRAL */
    font-size: 0.8rem;
  }
  .status {
    margin: 0;
    color: #8a8a8a; /* COLOR_NEUTRAL */
    font-size: 0.8rem;
  }
</style>
