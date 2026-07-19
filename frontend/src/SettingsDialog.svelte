<script lang="ts">
  import {onMount} from 'svelte';
  import {LogLevel} from '../bindings/github.com/MarioStoilov/simplestonks/internal/config';
  import type {Config} from '../bindings/github.com/MarioStoilov/simplestonks/internal/config';
  import {Settings} from '../bindings/github.com/MarioStoilov/simplestonks/internal/service';
  import Modal from './Modal.svelte';
  import {applyBackgroundValues} from './lib/background';
  import {RANGES} from './lib/market';
  import {
    DEFAULT_BACKGROUND_COLOR,
    LABEL_BACKGROUND_COLOR,
    LABEL_BACKGROUND_OPACITY,
    LABEL_DEFAULT_RANGE,
    LABEL_LOG_ARCHIVES,
    LABEL_LOG_FILE,
    LABEL_LOG_LEVEL,
    LABEL_LOG_MAX_SIZE,
    LABEL_REFRESH_SECONDS,
    MSG_ERR_LOG_ARCHIVES,
    MSG_ERR_LOG_MAX_SIZE,
    MSG_ERR_REFRESH_INTERVAL,
    SECTION_APPEARANCE,
    SECTION_GENERAL,
    SECTION_LOGGING,
    TITLE_SETTINGS,
  } from './lib/constants';

  // SettingsDialog is the sectioned settings editor (General / Appearance /
  // Logging), mirroring the Fyne settings window: appearance edits preview
  // live, and nothing persists until Save. The parent re-applies the persisted
  // background when the dialog closes, reverting an unsaved preview.
  let {
    cfg,
    onclose,
  }: {
    cfg: Config;
    onclose: () => void;
  } = $props();

  const NS_PER_SECOND = 1_000_000_000;
  const LOG_LEVELS: LogLevel[] = [
    LogLevel.LogSilent,
    LogLevel.LogError,
    LogLevel.LogWarn,
    LogLevel.LogInfo,
    LogLevel.LogDebug,
  ];

  // svelte-ignore state_referenced_locally -- the form intentionally captures
  // the config snapshot at open; live external edits are not merged mid-edit.
  let section = $state(SECTION_GENERAL);
  // svelte-ignore state_referenced_locally
  let defaultRange = $state(cfg.defaultRange);
  // svelte-ignore state_referenced_locally
  let refreshSeconds = $state(String(Math.round(cfg.refreshInterval / NS_PER_SECOND)));
  // svelte-ignore state_referenced_locally
  let backgroundColor = $state(cfg.background?.color || DEFAULT_BACKGROUND_COLOR);
  // svelte-ignore state_referenced_locally
  let opacityPercent = $state(Math.round((cfg.background?.opacity ?? 1) * 100));
  // svelte-ignore state_referenced_locally
  let logLevel = $state(cfg.logging?.level ?? LogLevel.LogInfo);
  // svelte-ignore state_referenced_locally
  let logFile = $state(cfg.logging?.file ?? '');
  // svelte-ignore state_referenced_locally
  let logMaxSize = $state(String(cfg.logging?.maxSizeMB ?? 0));
  // svelte-ignore state_referenced_locally
  let logArchives = $state(String(cfg.logging?.maxArchives ?? 0));

  let errorText = $state('');
  let defaultLogPath = $state('');

  onMount(() => {
    Settings.DefaultLogPath()
      .then((path) => {
        defaultLogPath = path;
      })
      .catch(() => {});
  });

  // Live preview of appearance edits; the parent reverts on close-without-save.
  $effect(() => {
    applyBackgroundValues(backgroundColor, opacityPercent / 100);
  });

  function parseWhole(text: string): number | null {
    const trimmed = text.trim();
    if (!/^\d+$/.test(trimmed)) {
      return null;
    }
    return parseInt(trimmed, 10);
  }

  async function save(): Promise<void> {
    const seconds = parseWhole(refreshSeconds);
    if (seconds === null || seconds < 1) {
      errorText = MSG_ERR_REFRESH_INTERVAL;
      return;
    }
    const maxSize = parseWhole(logMaxSize);
    if (maxSize === null) {
      errorText = MSG_ERR_LOG_MAX_SIZE;
      return;
    }
    const archives = parseWhole(logArchives);
    if (archives === null) {
      errorText = MSG_ERR_LOG_ARCHIVES;
      return;
    }
    try {
      await Settings.Save({
        defaultRange,
        refreshSeconds: seconds,
        background: {color: backgroundColor, opacity: opacityPercent / 100},
        logging: {level: logLevel, file: logFile.trim(), maxSizeMB: maxSize, maxArchives: archives},
      });
      onclose();
    } catch (err) {
      errorText = String(err);
    }
  }
</script>

<Modal title={TITLE_SETTINGS} {onclose}>
  <div class="body">
    <nav class="sections">
      {#each [SECTION_GENERAL, SECTION_APPEARANCE, SECTION_LOGGING] as name (name)}
        <button
          class="section-btn"
          class:active={section === name}
          onclick={() => {
            section = name;
          }}
        >
          {name}
        </button>
      {/each}
    </nav>
    <div class="pane">
      {#if section === SECTION_GENERAL}
        <label class="field">
          <span>{LABEL_DEFAULT_RANGE}</span>
          <select bind:value={defaultRange}>
            {#each RANGES as rangeOption (rangeOption)}
              <option value={rangeOption}>{rangeOption}</option>
            {/each}
          </select>
        </label>
        <label class="field">
          <span>{LABEL_REFRESH_SECONDS}</span>
          <input type="text" inputmode="numeric" bind:value={refreshSeconds} />
        </label>
      {:else if section === SECTION_APPEARANCE}
        <label class="field">
          <span>{LABEL_BACKGROUND_COLOR}</span>
          <input type="color" bind:value={backgroundColor} />
        </label>
        <label class="field">
          <span>{LABEL_BACKGROUND_OPACITY} ({opacityPercent}%)</span>
          <input type="range" min="0" max="100" bind:value={opacityPercent} />
        </label>
      {:else}
        <label class="field">
          <span>{LABEL_LOG_LEVEL}</span>
          <select bind:value={logLevel}>
            {#each LOG_LEVELS as level (level)}
              <option value={level}>{level}</option>
            {/each}
          </select>
        </label>
        <label class="field">
          <span>{LABEL_LOG_FILE}</span>
          <input type="text" bind:value={logFile} placeholder={defaultLogPath} />
        </label>
        <label class="field">
          <span>{LABEL_LOG_MAX_SIZE}</span>
          <input type="text" inputmode="numeric" bind:value={logMaxSize} />
        </label>
        <label class="field">
          <span>{LABEL_LOG_ARCHIVES}</span>
          <input type="text" inputmode="numeric" bind:value={logArchives} />
        </label>
      {/if}
    </div>
  </div>
  {#if errorText}
    <p class="error">{errorText}</p>
  {/if}
  <footer class="actions">
    <button class="btn" onclick={onclose}>Cancel</button>
    <button class="btn primary" onclick={save}>Save</button>
  </footer>
</Modal>

<style>
  .body {
    display: flex;
    gap: 0.75rem;
    width: 560px;
    max-width: 88vw;
    min-height: 260px;
  }
  .sections {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    flex-shrink: 0;
  }
  .section-btn {
    background: none;
    color: inherit;
    border: none;
    border-radius: 4px;
    padding: 0.35rem 0.8rem;
    text-align: left;
    cursor: pointer;
  }
  .section-btn:hover {
    background: #2c303c; /* COLOR_HOVER */
  }
  .section-btn.active {
    background: #303a52; /* COLOR_SELECTED */
    font-weight: 600;
  }
  .pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.7rem;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 0.85rem;
  }
  .field span {
    color: #8a8a8a; /* COLOR_NEUTRAL */
  }
  .field input[type='text'],
  .field select {
    background: #1e1e24; /* COLOR_CHART_BG */
    color: inherit;
    border: 1px solid #6e727e; /* COLOR_AXIS */
    border-radius: 4px;
    padding: 0.35rem 0.5rem;
    font-size: 0.9rem;
    outline: none;
  }
  .field input[type='color'] {
    width: 48px; /* SwatchWidth */
    height: 28px; /* SwatchHeight */
    padding: 0;
    border: 1px solid #6e727e;
    border-radius: 4px;
    background: none;
    cursor: pointer;
  }
  .error {
    margin: 0;
    color: #d03a3a; /* COLOR_DOWN */
    font-size: 0.8rem;
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
  .btn:hover {
    background: #303a52; /* COLOR_SELECTED */
  }
  .btn.primary {
    background: #303a52; /* COLOR_SELECTED */
    font-weight: 600;
  }
</style>
