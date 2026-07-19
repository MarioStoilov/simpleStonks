// Window background styling from the persisted config. Until the window
// itself is translucent, the opacity only matters against the webview's solid
// backing; once the frameless/translucent window lands, the same rgba value
// produces real see-through.

import type {Background} from '../../bindings/github.com/MarioStoilov/simplestonks/internal/config';
import {DEFAULT_BACKGROUND_COLOR, DEFAULT_BACKGROUND_OPACITY} from './constants';

const HEX_COLOR_PATTERN = /^#[0-9a-fA-F]{6}$/;

// applyBackground paints the page background with the configured color and
// opacity, falling back to the defaults on malformed values.
export function applyBackground(background: Background | null | undefined): void {
  applyBackgroundValues(background?.color ?? '', background?.opacity ?? DEFAULT_BACKGROUND_OPACITY);
}

// applyBackgroundValues is the raw variant used for live previews.
export function applyBackgroundValues(color: string, opacity: number): void {
  const hex = HEX_COLOR_PATTERN.test(color.trim()) ? color.trim() : DEFAULT_BACKGROUND_COLOR;
  const alpha = Math.min(1, Math.max(0, opacity));
  const red = parseInt(hex.slice(1, 3), 16);
  const green = parseInt(hex.slice(3, 5), 16);
  const blue = parseInt(hex.slice(5, 7), 16);
  document.body.style.background = `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}
