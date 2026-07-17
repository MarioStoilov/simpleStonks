# simpleStonks — Requirements & Design

Planning artifact capturing the decisions made before implementation. This reflects
the agreed scope for the first working version (v1) and the direction for later work.

## Overview

simpleStonks is a bare-bones, Yahoo Finance-style stock tracker widget/app written in
Go and distributed via the Snap Store. It shows a user-configurable list of stock
indexes/tickers and charts their movement. The tracked list is editable from the UI,
and the list plus other settings are persisted to a config file.

> **UI status: work in progress, design not final.** The current interface is an
> early implementation. Layouts, controls, and flows (including everything under
> [Symbol management](#symbol-management-edit-mode--live-search)) are **targets,
> not commitments to what is built today**. Expect substantial reworks; where this
> document and the current UI disagree, this document describes the intended
> direction.

## Locked decisions

| Area | Decision |
|------|----------|
| Language / runtime | Go |
| GUI toolkit | Fyne (pure Go, self-contained, snap-friendly) |
| Distribution | Snap Store (snapcraft) |
| Data provider | Free/keyless. First implementation uses the Yahoo Finance chart endpoint (`v8/finance/chart`), behind a swappable provider interface. The swap capability is internal only — not advertised and not a committed feature. |
| Chart ranges | **1D with live ticking** is the default. The full range toggles (**1D, 5D, 1W, 1M, YTD, 1Y, 5Y, ALL**) appear in the **detail view**; home-grid cells are 1D-only. |
| MVP scope | MVP + polish (see below). |
| Screens / navigation | Two screens: a **home grid** of all indexes (1D-only cells) and a **detail view** (expanded chart with range toggles + a left sidebar of indexes). See [Screens & navigation](#screens--navigation). |
| Settings | A top-right **cog-wheel** icon opens a separate **settings window** that edits the config file (design TBD). See [Settings window](#settings-window). |
| Form factor | Both **normal window** and **always-on-top widget** will be supported. v1 builds the **normal window**; widget mode is designed-for but deferred. |
| Symbol management | A multi-step **edit mode** (toggled by an Edit button) gates remove/reorder/add; adding uses a **live search** with name + market details (see below). |
| Config | File-based, editable from the UI, XDG/snap-aware, **live two-way reload** (see below). |
| Logging | Leveled (silent → debug), configured in the config file and live-reloaded, written to a rotating file (see below). |
| Testing | Unit tests run on every commit; integration tests run on every push (see below). |

## Screens & navigation

The app has two screens: a home grid and a detail view. (This unifies the earlier
"grid vs list + detail" framing — the grid is the home screen and the "list +
detail" is the drill-down detail view, not two separately selectable layouts.)

### Home (grid)

- The default screen is a **grid** of all configured indexes, one cell each.
- Each cell shows **only the 1D view** (mini chart + price / % change). There are
  **no range toggles on the home grid**.
- The **Edit** button lives here; edit mode gates add / remove / reorder — see
  [Symbol management](#symbol-management-edit-mode--live-search).
- A **cog-wheel** icon in the **top-right** opens the [settings
  window](#settings-window).
- **Clicking a cell opens the detail view** for that index.

### Detail view

- Reached by clicking a home-grid cell (or a sidebar cell).
- The selected index's chart is **expanded to take up most of the screen**.
- The **range toggles (1D, 5D, 1W, 1M, YTD, 1Y, 5Y, ALL) are visible here** and
  switch the detail chart's range. 1D remains the default and keeps live ticking.
- A **left sidebar** lists all configured indexes as **mini cells in a single
  vertical column**, with the currently-viewed index **highlighted / selected**.
- **Clicking any index in the sidebar** switches the detail view to that index
  (staying in the detail view).
- A way back to the home grid (e.g. a back control) is needed; exact affordance
  is an open detail.

Status: **target design.** The current build is a single grid whose cells carry
the range toggles and has no detail view or sidebar yet; the UI will be reworked
to this two-screen model.

## Settings window

- A **cog-wheel icon** in the **top-right of the app** opens a **settings
  window** — a separate window from the home grid and detail view.
- This window is the UI for **application configuration**: it is where the user
  changes settings, and it reads and writes the **config file**. Because config
  supports live two-way reload, edits made here are persisted and applied to the
  running app immediately — see [Live configuration
  reload](#live-configuration-reload).
- It goes through the same `config.Store` (`Get` / `Update`) as everything else,
  so its edits and external file edits stay consistent.
- **Implemented:** a separate window opened by the cog, with a Save/Cancel form
  exposing **default range**, **refresh interval (s)**, and the **logging** group
  (level, file, max size MB, archives kept). Save validates the numeric fields
  and writes through the store, so changes persist and apply live (the logger is
  reconfigured from the same subscription). Layout/form-factor selectors are not
  exposed yet (those modes aren't built); visual polish is still open.

Note: this settings window is distinct from **edit mode** on the home grid, which
manages only the tracked-index list; the settings window covers the rest of the
app configuration.

## MVP + polish scope (v1)

Included in v1:

- Manage the tracked list from the UI via an edit mode — add (with live search),
  remove, and reorder — see [Symbol management](#symbol-management-edit-mode--live-search).
  (The current build ships a simpler add dialog plus per-tile remove and will be
  reworked to this flow.)
- Persist tracked symbols and settings to a config file.
- Fetch quote/chart data via the free provider and draw a line chart per symbol.
- Home grid of all indexes (1D-only cells) and a detail view (expanded chart with
  range toggles + index sidebar) — see [Screens & navigation](#screens--navigation).
  Normal resizable window.
- Percent-change display with up/down coloring.
- Offline / API-error handling (graceful degradation, no crashes).
- Live two-way config reload (UI edits and external file edits both apply
  without a restart).
- Snap packaging (`snapcraft.yaml`).

Deferred (architected-for, not built in v1):

- Always-on-top widget form factor.
- Alternative data providers.

## Assumptions & defaults

- **Config format:** JSON file located via `os.UserConfigDir()`, which maps correctly
  to `$SNAP_USER_DATA` under snap confinement. Stores tracked symbols plus settings.
- **Symbols:** any Yahoo-accepted symbol — indexes (e.g. `^GSPC`, `^IXIC`) and
  tickers (e.g. `AAPL`).
- **Live-tick interval:** configurable; default ~30–60s. Only meaningful for the 1D
  view during market hours.
- **Platform:** Linux-primary (snap target). Fyne keeps cross-platform open, but no
  extra effort is spent on other platforms in v1.
- **License:** MIT (see `LICENSE`; the copyright notice carries the repo URL so
  attribution travels with forks/copies).

## Live configuration reload

The config file stays in sync with the running app in **both** directions,
without a restart:

- **UI → file:** edits made in the UI are persisted immediately (atomic
  write-tmp-then-rename).
- **File → UI:** external edits to the config file (hand edits, another tool)
  are detected and applied to the running UI.

Implementation notes:

- The config directory is watched with `fsnotify` (watch the directory, not the
  file, so atomic rename-based writes keep being followed). Filesystem events are
  debounced before reloading.
- A `config.Store` owns the live config behind a mutex and exposes `Get`,
  `Update` (UI edits), `Subscribe`, and `Close`. Subscribers are notified on any
  change from either direction.
- The app's own writes do not cause a redundant rebuild or a feedback loop:
  after a reload, the new config is compared to the in-memory one and ignored if
  unchanged.
- A malformed external edit is logged and ignored, keeping the last-good config;
  the UI is never left blank or crashed.
- Reloads arrive on a background goroutine, so UI subscribers marshal their work
  onto the UI thread (Fyne's `fyne.Do`); the `config` package has no UI
  dependency.

## Symbol management (edit mode & live search)

Managing the tracked list is an explicit, multi-step flow rather than
always-visible controls, so the home page stays a clean monitoring view.

- **View mode (default).** The grid/home page is read-only: it displays the
  tracked indexes and their charts. No add/remove/reorder controls are shown.
- **Edit mode.** An **Edit** button on the grid/home page toggles edit mode.
  **Only in edit mode** may the user:
  - **remove** a tracked index,
  - **reorder** the tracked indexes (the order is persisted),
  - **add** a new index.
  Leaving edit mode returns to the clean view.
- **Adding with live search.** Adding opens a text input with **live search**: as
  the user types, the app shows a list of matching instruments, each annotated
  with:
  - the instrument's **full name** (e.g. "Apple Inc."), and
  - its **market / exchange location** (e.g. "NASDAQ" / "US"),

  so the correct instrument is chosen unambiguously rather than by guessing a
  ticker. Suggestion rows highlight on hover; **clicking a suggestion opens a
  small preview** (a compact version of the detail view) with **Add** and
  **Cancel** actions. Add tracks the symbol; Cancel returns to the results.
- **No duplicates.** An index cannot be tracked more than once. In the preview,
  if the index is already tracked, **Add is disabled (grayed out) and shows a
  tooltip** ("already tracking this index") on hover.

Implications:

- The data provider gains a **symbol search** capability alongside `Quote` /
  `History`, behind the same swappable interface. The first implementation can use
  Yahoo's search/autocomplete endpoint (`v1/finance/search`), which returns
  symbol, long name, exchange, and quote type.
- Search must be **debounced** and cancelable (each keystroke supersedes the
  previous in-flight request) and degrade gracefully offline.
- Reordering persists the order in the config's ordered `symbols` list; live
  reload already round-trips that order.

Status: **implemented.** The home grid has an Edit button that toggles edit mode;
in edit mode each cell exposes reorder (up/down) and remove controls, and Add
opens the live-search dialog (Yahoo search, debounced, hover-highlighted rows,
click-to-preview with Add/Cancel). Reorder/remove/add persist via the config
store and apply live (reordering updates the grid immediately). Drag-to-reorder
is not implemented (up/down buttons are used instead).

## Logging

The app has a leveled, rotating file logger. Its configuration lives in the
config file and is applied live on reload.

- **Levels (ordered, silent → verbose):** `silent`, `error`, `warn`, `info`,
  `debug`. `silent` produces no output at all. Built on the standard library
  `log/slog`.
- **Configured in the config file** under a `logging` object, so it is persisted
  and live-reloaded with everything else:
  - `level` — one of the levels above.
  - `file` — log file path; empty means the default.
  - `maxSizeMB` — rotate when the log file exceeds this size (0 disables
    rotation).
  - `maxArchives` — how many rotated archives to retain (0 keeps none).
- **Default log path:** the XDG state directory —
  `~/.local/state/simplestonks/simplestonks.log` — which stays writable under
  snap confinement. Overridable via `file`.
- **Rotation / archiving:** when the live log passes `maxSizeMB`, it is rotated
  to `<file>.1`, older archives shift up (`.1` → `.2` …), and anything past
  `maxArchives` is deleted. Implemented in-tree (no external dependency).
- **Live reconfigure:** the top-level `*slog.Logger` is stable and installed as
  slog's default; an inner handler is swapped atomically on config change, so
  level and destination changes apply without a restart and without invalidating
  held logger references.

## Testing & CI

- **Unit tests on every commit.** Fast, dependency-free tests (config,
  provider mapping/parsing, model helpers) run before each commit. Enforced by
  the `.githooks/pre-commit` hook (gofmt check, `go vet`, `go test ./...`).
- **Integration tests on every push.** Broader tests (e.g. the provider against
  the live endpoint, config live-reload end-to-end, UI smoke where feasible) run
  before each push. Enforced by the `.githooks/pre-push` hook
  (`go test -race -tags=integration ./...`); to be mirrored in CI once a CI
  provider is chosen.
- **Integration tag convention.** Integration tests carry the
  `//go:build integration` build tag, so they are excluded from the default
  (commit) test run and included on push. Network-dependent integration tests
  skip themselves when offline so pushes are not blocked.
- **Hook installation.** The hooks are versioned under `.githooks/` and enabled
  per clone with `make hooks` (sets `core.hooksPath`); see `docs/DEVELOPMENT.md`.
- Code is written to be testable: side effects (filesystem, network, clock) are
  reached through interfaces or injectable clients so units can be exercised in
  isolation. The `QuoteProvider` interface and the injectable `*http.Client` in
  the Yahoo provider are examples of this.
- Tests should pass under the race detector (`go test -race ./...`).

## Proposed project structure

```
simpleStonks/
├── cmd/simplestonks/main.go        # entrypoint
├── internal/
│   ├── config/                     # load/save JSON config + live-reload Store (XDG/snap-aware)
│   ├── logging/                    # leveled slog logger + size-based log rotation
│   ├── provider/                   # QuoteProvider interface + yahoo impl
│   │   ├── provider.go             # interface + types (Quote, Candle, Range)
│   │   └── yahoo.go
│   ├── model/                      # domain types, range definitions
│   └── ui/                         # Fyne UI: grid view, chart widget, settings, symbol mgmt
│       ├── app.go
│       ├── gridview.go
│       ├── chart.go
│       └── settings.go
├── snap/snapcraft.yaml             # snap packaging
├── .githooks/                      # versioned pre-commit / pre-push test hooks
├── Makefile                        # build/test/hook helper targets
├── go.mod
├── README.md
├── .gitignore
└── CLAUDE.md
```

The `provider` package's interface is the seam for swapping data sources. The `ui`
package is split so that grid-vs-list layout and window-vs-widget form factor are
pluggable modes rather than hardcoded choices.

## Implemented so far

The core MVP + polish scope is in place:

- Keyless Yahoo provider: `Quote`, `History`, `Search` (behind the swappable
  `QuoteProvider` interface).
- Two-screen UI: 1D-only home grid → detail view (expanded chart, 1D..ALL range
  toggles, index sidebar, back to grid).
- Edit-mode symbol management: reorder (up/down) + remove, and Add via a debounced
  live search with click-to-preview and duplicate prevention (disabled Add +
  tooltip).
- Live two-way config reload; leveled rotating file logging.
- Settings window (cog): default range, refresh interval, logging — applied live.
- Tests gated by `pre-commit` (unit) and `pre-push` (integration) hooks.

## Open items

Before a shippable MVP:

- **Snap packaging**: finalize `snap/snapcraft.yaml` build + confinement (network,
  desktop integration, writable data dirs) and publish to the store.
  (License chosen: MIT — done.)

Deferred features / polish:

- Always-on-top **widget** form factor (`formFactor: widget` is not built).
- Whether to keep a selectable **layout** toggle: the `layout` config field no
  longer changes behavior now that home-grid → detail is the sole navigation.
- **Drag-to-reorder** in edit mode (currently up/down buttons).
- **Chart polish**: gridlines, hover readout, currency formatting. (Done: axis
  labels — size-adaptive y-axis price ticks and range-aware x-axis time labels,
  hours for 1D, days/dates/months/years for longer ranges — on both grid and
  detail charts; a dashed previous-close reference line per range with its
  value on the y axis; 1D charts drawn over the full trading-session window so
  a live day fills in gradually; friendly instrument names under the symbol on
  tiles and the detail header; live price updates flash a semi-transparent
  green/red background behind the price number.)
- Expose **layout / form-factor** in the settings window once those modes exist.
- **CI** mirroring the pre-push checks.
