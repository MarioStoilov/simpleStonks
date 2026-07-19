<script lang="ts">
  import type {Snippet} from 'svelte';

  // Modal is a centered dialog over a dimmed backdrop. Clicking the backdrop
  // or the ✕ closes it; content is provided as a snippet.
  let {
    title,
    onclose,
    children,
  }: {
    title: string;
    onclose: () => void;
    children: Snippet;
  } = $props();
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div
  class="backdrop"
  onclick={(event) => {
    if (event.target === event.currentTarget) onclose();
  }}
>
  <div class="panel" role="dialog" aria-label={title}>
    <header class="bar">
      <span class="title">{title}</span>
      <button class="close" onclick={onclose} aria-label="Close">✕</button>
    </header>
    {@render children()}
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10;
  }
  .panel {
    background: #24262e; /* COLOR_CARD_BG */
    border-radius: 6px; /* TILE_CORNER_RADIUS */
    padding: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    max-height: 90vh;
    max-width: 92vw;
  }
  .bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
  }
  .title {
    font-weight: 600;
  }
  .close {
    background: none;
    border: none;
    color: #8a8a8a; /* COLOR_NEUTRAL */
    cursor: pointer;
    font-size: 0.9rem;
    padding: 0.2rem;
  }
  .close:hover {
    color: #e6e6e6;
  }
</style>
