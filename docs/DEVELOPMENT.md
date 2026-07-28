# Development & Cross-Machine Continuity

How to set up a machine to work on simpleStonks and how to continue a coding
session on a different machine (e.g. moving from work to home).

## The one rule that makes continuity work

**The Git remote is the single source of truth.** Everything needed to resume
work must be committed and pushed. These things are **machine-local and do NOT
travel between machines**, so never rely on them for continuity:

- Claude Code's session history and its file-based memory.
- Your runtime config and logs (see [Runtime file locations](#runtime-file-locations)).
- Local, uncommitted edits and stashes.

So before you switch machines: **commit and push**. After you switch: **pull**.
Anything not pushed does not exist on the other side.

## Prerequisites

| Tool | Version / notes |
|------|-----------------|
| Go | 1.23+ (module targets `go 1.23.5`) |
| Git | any recent version |
| C toolchain + Qt 6 dev headers | required to build the Qt UI (cgo via miqt) |
| snapcraft | only needed to build the snap package |

### System build dependencies (Linux)

The Qt UI (miqt bindings) compiles against Qt 6 via cgo, so these OS packages
are required to `go build ./...`. On Debian/Ubuntu:

```sh
sudo apt install pkg-config gcc qt6-base-dev
```

The runtime also wants `libqt6svg6` (the Qt SVG image plugin) so the About
dialog can render the embedded logo. Without the dev packages, the non-UI
packages (`config`, `logging`, `model`, `provider`, `chartmath`) still build
and test, but the `qtui` package and the `cmd/simplestonks` binary do not.
The first build compiles the miqt wrappers and takes several minutes; strip
release binaries with `-ldflags="-s -w"`.

For the snap build only:

```sh
sudo snap install snapcraft --classic   # plus LXD, per snapcraft's docs
```

## Compile and run locally

From a clean machine to a running app:

1. **Install Go 1.23+** and the **system build dependencies** listed above
   (`pkg-config`, `gcc`, and the GL/X11/Wayland headers). Verify Go:
   ```sh
   go version   # expect go1.23 or newer
   ```
2. **Clone and enter the repo:**
   ```sh
   git clone git@github.com:MarioStoilov/simpleStonks.git
   cd simpleStonks
   ```
3. **Download Go module dependencies:**
   ```sh
   go mod download
   ```
4. **Compile:**
   ```sh
   go build ./...                              # compile all packages (or: make build)
   go build -o bin/simplestonks ./cmd/simplestonks   # produce a runnable binary
   ```
5. **Run:**
   ```sh
   go run ./cmd/simplestonks   # or: make run, or: ./bin/simplestonks
   ```

**JetBrains GoLand:** a shared run configuration named **simpleStonks** is
committed under `.idea/runConfigurations/`, so Run/Debug work out of the box
(the rest of `.idea/` is gitignored).

On first run the app creates its config with a default set of tracked symbols and
starts fetching data. See [Runtime file locations](#runtime-file-locations) for
where the config and logs are written.

If the compile step fails with errors about `GL`, `X11`, `wayland`, or
`pkg-config`, the system build dependencies are missing — install them (step 1)
and retry.

## Getting the code on a new machine

```sh
git clone git@github.com:MarioStoilov/simpleStonks.git
cd simpleStonks
```

### Set the per-repository Git identity

This repo uses a repo-local identity (not the global one). Set it once per clone:

```sh
git config user.name  "Mario Stoilov"
git config user.email "mario.stoilov.93@gmail.com"
```

### Install the git hooks

The test hooks are versioned but not auto-installed on clone; enable them once:

```sh
make hooks
```

### Commit convention (important)

Do **not** add AI co-authoring or "Generated with" trailers to commit messages.
See `CLAUDE.md` for the full conventions.

## Build, run, test

```sh
# Build everything (needs the GUI dev headers above)
go build ./...

# Run the app
go run ./cmd/simplestonks

# Unit tests (fast; run on every commit)
go test ./...

# With the race detector
go test -race ./...

# Unit + integration tests (integration tests hit the network; run on every push)
go test -race -tags=integration ./...

# Non-UI only (works without the GUI dev headers)
go test ./internal/config/... ./internal/logging/... \
        ./internal/model/... ./internal/provider/...
```

A `Makefile` wraps the common commands — run `make help` to list targets
(`make build`, `make run`, `make check`, `make test-integration`, `make hooks`, …).

## Git hooks

Testing is gated by versioned hooks in `.githooks/`:

- **`pre-commit`** — gofmt check, `go vet`, and unit tests (`go test ./...`).
  Integration-tagged tests are excluded so commits stay fast.
- **`pre-push`** — the full suite including integration tests, under the race
  detector (`go test -race -tags=integration ./...`).

Because the hooks live in the repo (not `.git/hooks`), each clone must opt in
once — Git does not do this automatically:

```sh
make hooks          # sets core.hooksPath=.githooks
```

Integration tests carry the `//go:build integration` tag and only run with
`-tags=integration`. Those needing network skip themselves when it is
unavailable, so offline pushes are not blocked. In an emergency a hook can be
bypassed with `git commit --no-verify` / `git push --no-verify`.

## Runtime file locations

These are created at runtime on whichever machine runs the app; they are **not**
synced and are **not** in the repo.

- **Config:** `os.UserConfigDir()/simplestonks/config.json`
  (typically `~/.config/simplestonks/config.json`; `$SNAP_USER_DATA/...` under
  snap). Supports live two-way reload.
- **Logs:** default `~/.local/state/simplestonks/simplestonks.log`
  (XDG state dir; overridable via the `logging.file` config key), with rotated
  archives `simplestonks.log.1`, `.2`, ….

If you want the same tracked symbols/settings on another machine, copy
`config.json` over manually — it is intentionally not part of the repo.

## Continuing a coding session on another machine

1. **Before leaving the current machine**
   - Commit your work (or, for a rough stopping point, commit a WIP on a branch).
   - `git push` so the remote has it. Verify with `git status` (should say your
     branch is up to date with its upstream).
2. **On the other machine**
   - `git clone` (first time) or `git pull` (subsequent times) to get the latest.
   - Ensure prerequisites and the per-repo Git identity are set (above).
   - Install the git hooks once: `make hooks`.
3. **Bring a fresh Claude Code session up to speed.** Because the assistant's
   memory does not cross machines, the repo carries the context. Point it at, or
   read yourself, in this order:
   - `CLAUDE.md` — project summary and working conventions.
   - `docs/REQUIREMENTS.md` — locked decisions, scope, and open items.
   - `docs/DEVELOPMENT.md` — this file.
   - `git log --oneline` — the authoritative record of what has been done.
   - `git status` / `git diff` — any work in progress.

That is enough to reconstruct where the project stands and what comes next
without relying on any machine-local state.

## Current status & next steps

Progress is recorded in `git log`; the short version:

- Repo, docs, and requirements established.
- Go module and package skeleton scaffolded.
- Live two-way config reload implemented and tested.
- Leveled, rotating file logger implemented and tested.
- `pre-commit` / `pre-push` test hooks and the integration-tag harness in place.
- Yahoo provider (`Quote` / `History`) implemented and tested.
- Two-screen UI wired to live data: a home grid of 1D-only cells that drills into
  a detail view (expanded chart with 1D..ALL range toggles plus a left sidebar of
  all symbols), per-tile price/% change with up/down coloring and a real chart
  line, a per-screen polling loop for the live 1D data, and graceful per-tile
  error handling.
- Provider symbol search (Yahoo `v1/finance/search`) and home-grid edit mode:
  Edit toggles reorder (up/down) + remove controls per cell, and Add opens a
  debounced live-search dialog with click-to-preview (Add disabled + tooltip for
  already-tracked indexes).
- Settings window (cog-wheel): a separate window with a section sidebar —
  General (default range, refresh interval), Appearance (window background via
  swatch + color-picker dialog and an opacity slider, live-previewed via a
  theme override), and Logging (level/file/size/archives) — applied live via
  the store.
- Chart styling (Settings → Appearance, all live-previewed and live-reloaded
  via `config.Chart` → the shared `chartStyle` in `internal/qtui/chart.go`):
  configurable plot background color; an optional checkered graph-paper grid
  (off by default; square size in px + line color); and the logo-style
  up/down area fill between the price line and the dashed previous-close
  reference (on by default; green above, red below, split at the crossings
  by the pure `chartmath.FillRegions`, opacity configurable). With the fill
  on, the price line itself also splits at the reference
  (`chartmath.SegmentCrossing`) — green above, red below, logo-style; with
  it off the line keeps its single overall up/down color. In the settings,
  a disabled effect grays out its dependent controls, and the effect
  toggles carry explanatory tooltips.
- Chart axis labels: size-adaptive y-axis price ticks and range-aware x-axis
  time labels (hours for 1D, days/dates/months/years for longer ranges) on both
  the home-grid mini charts and the detail chart.
- Previous-close reference: a dashed line at the prior interval's close
  (Yahoo-style, per range) with its value labeled on the y axis and the y-scale
  widened to keep it in view.
- Session-window 1D axis: intraday charts span the full regular trading session
  (from Yahoo's `currentTradingPeriod`), so a live day fills in gradually as
  the polling loop appends data. While the market is closed, Yahoo pairs the
  previous day's candles with the *upcoming* session; those render evenly
  spaced across the full width, and the chart re-initiates onto the new
  session automatically at the first post-open poll (no restart or range
  change needed).
- Friendly instrument names (Yahoo meta long/short name) shown under the symbol
  on home tiles, sidebar cells, and the detail header.
- Chart hover readout — the detail view's expanded chart only (not grid tiles
  or the search preview): a dot on the nearest data point, a dashed vertical
  guide with the point's time/date boxed on the x axis (clock time for 1D,
  dates for longer ranges), and a tooltip with the price plus its green/red
  % change versus the previous close. Also a button-like hover highlight on
  clickable tiles (home grid + detail sidebar; suppressed in edit mode, where
  tiles don't navigate); the mini chart forwards its hover state to its tile
  so the highlight covers the whole cell.
- Live price flash (`priceText` widget): when a refresh changes a displayed
  price, a semi-transparent green/red background flashes behind the number
  only, fading out; unchanged prices (closed market) never flash.
- About dialog: an info button in the home top bar (next to Edit and the
  settings cog) opens a modal with the logo, version (`constants.AppVersion` in
  `internal/constants` — keep in sync with snapcraft.yaml on release),
  GitHub/license links, and the data disclaimer.
- App logo: a minimalist line-chart SVG (green above the X axis, red below,
  with matching semi-transparent area fills) at `internal/assets/icon.svg` — a
  single source embedded into the binary (`internal/assets`), shown in the
  About dialog, and referenced by the `icon:` key in `snap/snapcraft.yaml`.
  Known limitation: on a native **Wayland** session dev runs show a
  placeholder window icon — Wayland forbids apps setting their own window
  icon. The packaged snap gets its icon via the desktop entry
  (`snap/gui/simplestonks.desktop`).
- Snap packaging verified: `snapcraft pack` builds cleanly in LXD (needs the
  Wayland dev packages in `build-packages`; on hosts running Docker, allow
  `lxdbr0` in the `DOCKER-USER` iptables chain or container networking is
  blocked). The strict snap passed a local smoke test: network fetch, config
  under `$SNAP_USER_DATA/.config`, logs under `$SNAP_USER_DATA/.local/state`,
  desktop entry + taskbar icon.
- **Published on the Snap Store** as `simplestonks`: 0.2.x releases proved out
  the `edge` channel (rev 2 surfaced the stale-log-path refresh bug, fixed in
  0.2.1/rev 3); v1.0.0 (`grade: stable`) is the first `stable` release. On a
  release: bump `version` in snapcraft.yaml **and** `constants.AppVersion` in
  `internal/constants`, `snapcraft pack`, `snapcraft upload
  --release=stable,edge <snap>`.

- Extended-hours (pre-market/after-hours) on the detail view's 1D chart —
  detail view only, controlled by one shared setting (header "Pre/post
  market data" checkbox — visible only on 1D, only for instruments that
  actually trade pre/post (per Yahoo's hasPrePostMarketData chart-meta flag,
  which is false for indexes even though they report pre/post windows), and
  whenever the market is outside its regular session, since that is when the
  setting has an effect — + Settings → General checkbox, default on,
  live-synced both ways):
  during pre-market the chart shows the pre session with a separate
  "Pre-market: price (%)" label next to the regular price; during after-hours
  the regular chart continues with the post candles (dimmed, behind a dashed
  divider at the regular close) plus an "After-hours:" label; while fully
  closed the completed session replays the same way (Yahoo pairs the
  overnight candles with the *upcoming* session's windows, so
  `BuildExtendedDisplay` translates the windows back onto the candles' day
  in whole-day steps); during regular hours everything renders exactly as
  before. Data via `HistoryExtended` (Yahoo `includePrePost`), market state
  derived from `currentTradingPeriod` pre/regular/post windows in pure,
  unit-tested `chartmath` helpers (`StateAt`, `BuildExtendedDisplay`); when
  no closed-market replay can be built (e.g. no post candles) the extended
  response is discarded in favor of a plain `History` fetch.

Immediate next candidates:

- Always-on-top widget form factor.
- Chart polish: currency formatting. (Hover readout and gridlines are done —
  the checkered grid and the up/down area fill live in Settings → Appearance.)
- Drag-to-reorder in edit mode (currently up/down buttons).
- CI mirroring the pre-push checks.
