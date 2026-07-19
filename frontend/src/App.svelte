<script lang="ts">
  import {onMount} from 'svelte';
  import {Events} from '@wailsio/runtime';
  import {Settings} from '../bindings/github.com/MarioStoilov/simplestonks/internal/service';
  import type {Config} from '../bindings/github.com/MarioStoilov/simplestonks/internal/config';
  import Home from './Home.svelte';
  import Detail from './Detail.svelte';
  import SettingsDialog from './SettingsDialog.svelte';
  import {applyBackground} from './lib/background';
  import {DEFAULT_REFRESH_MS, EVENT_CONFIG_CHANGED} from './lib/constants';

  // App is the view shell: it owns the config snapshot (kept live through the
  // configChanged event) and switches between the home grid and the detail
  // view of one symbol.
  type View = {kind: 'home'} | {kind: 'detail'; symbol: string};

  let cfg = $state<Config | null>(null);
  let loadError: string = $state('');
  let view = $state<View>({kind: 'home'});
  let settingsOpen = $state(false);

  // Apply the persisted background whenever the config changes.
  $effect(() => {
    if (cfg) {
      applyBackground(cfg.background);
    }
  });

  const NS_PER_MS = 1_000_000; // config.refreshInterval is a Go time.Duration (ns)
  const refreshMs = $derived(
    cfg && cfg.refreshInterval > 0 ? cfg.refreshInterval / NS_PER_MS : DEFAULT_REFRESH_MS,
  );

  onMount(() => {
    Settings.Get()
      .then((loaded) => {
        cfg = loaded;
      })
      .catch((err) => {
        loadError = String(err);
      });
    Events.On(EVENT_CONFIG_CHANGED, (event: any) => {
      if (event.data) {
        cfg = event.data;
      }
    });
  });

  function openDetail(symbol: string): void {
    view = {kind: 'detail', symbol};
  }
</script>

{#if cfg}
  {#if view.kind === 'detail'}
    {#key view.symbol}
      <Detail
        symbols={cfg.symbols ?? []}
        symbol={view.symbol}
        defaultRange={cfg.defaultRange}
        {refreshMs}
        onback={() => {
          view = {kind: 'home'};
        }}
        onselect={openDetail}
      />
    {/key}
  {:else}
    <Home
      symbols={cfg.symbols ?? []}
      {refreshMs}
      onopen={openDetail}
      onopensettings={() => {
        settingsOpen = true;
      }}
    />
  {/if}
  {#if settingsOpen}
    <SettingsDialog
      {cfg}
      onclose={() => {
        settingsOpen = false;
        // Revert any unsaved appearance preview to the persisted values.
        applyBackground(cfg?.background);
      }}
    />
  {/if}
{:else if loadError}
  <p class="error">{loadError}</p>
{/if}

<style>
  .error {
    padding: 1rem;
    color: #d03a3a; /* COLOR_DOWN */
  }
</style>
