<script lang="ts">
  import {PRICE_PLACEHOLDER} from './lib/constants';

  // Price displays a price number and flashes its background — green for a
  // rise, red for a drop, fading out — when the value changes after the first
  // display (port of the Fyne priceText widget). Parents that must not flash
  // on a context switch (symbol/range change) remount it with {#key}.
  let {value = null}: {value?: number | null} = $props();

  let shownValue: number | null = null;
  let flashDirection = $state<'up' | 'down' | null>(null);
  let flashKey = $state(0);

  $effect(() => {
    if (value === null) {
      shownValue = null;
      return;
    }
    if (shownValue !== null && value !== shownValue) {
      flashDirection = value > shownValue ? 'up' : 'down';
      flashKey += 1;
    }
    shownValue = value;
  });
</script>

{#key flashKey}
  <span class="price" class:flash-up={flashDirection === 'up'} class:flash-down={flashDirection === 'down'}>
    {value !== null ? value.toFixed(2) : PRICE_PLACEHOLDER}
  </span>
{/key}

<style>
  .price {
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    border-radius: 3px; /* FlashCornerRadius */
    padding: 0 3px; /* FlashPad */
  }
  .flash-up {
    animation: flash-up 900ms ease-out forwards; /* FLASH_DURATION_MS */
  }
  .flash-down {
    animation: flash-down 900ms ease-out forwards;
  }
  @keyframes flash-up {
    from {
      background-color: rgba(38, 166, 91, 0.4); /* COLOR_UP at FlashAlpha */
    }
    to {
      background-color: transparent;
    }
  }
  @keyframes flash-down {
    from {
      background-color: rgba(208, 58, 58, 0.4); /* COLOR_DOWN at FlashAlpha */
    }
    to {
      background-color: transparent;
    }
  }
</style>
