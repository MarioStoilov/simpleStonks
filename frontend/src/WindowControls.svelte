<script lang="ts">
  import {Window} from '@wailsio/runtime';

  // WindowControls replaces the OS decorations on the frameless window with
  // minimise/close buttons, shown at the right edge of the draggable bars.
</script>

<div class="controls no-drag">
  <button class="win-btn" onclick={() => Window.Minimise()} aria-label="Minimise">—</button>
  <button class="win-btn close" onclick={() => Window.Close()} aria-label="Close">✕</button>
</div>

<style>
  .controls {
    display: flex;
    gap: 0.2rem;
    margin-left: 0.4rem;
  }
  .win-btn {
    background: none;
    color: #8a8a8a; /* COLOR_NEUTRAL */
    border: none;
    border-radius: 4px;
    padding: 0.25rem 0.55rem;
    cursor: pointer;
    font-size: 0.8rem;
    line-height: 1;
  }
  .win-btn:hover {
    background: #2c303c; /* COLOR_HOVER */
    color: #e6e6e6;
  }
  .win-btn.close:hover {
    background: #d03a3a; /* COLOR_DOWN */
    color: #fff;
  }
</style>
