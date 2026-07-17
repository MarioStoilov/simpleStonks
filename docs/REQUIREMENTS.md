# simpleStonks — Requirements & Design

Planning artifact capturing the decisions made before implementation. This reflects
the agreed scope for the first working version (v1) and the direction for later work.

## Overview

simpleStonks is a bare-bones, Yahoo Finance-style stock tracker widget/app written in
Go and distributed via the Snap Store. It shows a user-configurable list of stock
indexes/tickers and charts their movement. The tracked list is editable from the UI,
and the list plus other settings are persisted to a config file.

## Locked decisions

| Area | Decision |
|------|----------|
| Language / runtime | Go |
| GUI toolkit | Fyne (pure Go, self-contained, snap-friendly) |
| Distribution | Snap Store (snapcraft) |
| Data provider | Free/keyless. First implementation uses the Yahoo Finance chart endpoint (`v8/finance/chart`), behind a swappable provider interface. The swap capability is internal only — not advertised and not a committed feature. |
| Chart ranges | Default view is **1D with live ticking**. Range toggles: **1D, 5D, 1W, 1M, YTD, 1Y, 5Y, ALL**. |
| MVP scope | MVP + polish (see below). |
| Layout | Both **grid** and **list + detail** layouts will be supported. v1 builds the **grid** layout; the code is architected so list+detail slots in later without rework. |
| Form factor | Both **normal window** and **always-on-top widget** will be supported. v1 builds the **normal window**; widget mode is designed-for but deferred. |
| Config | File-based, editable from the UI, XDG/snap-aware, **live two-way reload** (see below). |
| Logging | Leveled (silent → debug), configured in the config file and live-reloaded, written to a rotating file (see below). |
| Testing | Unit tests run on every commit; integration tests run on every push (see below). |

## MVP + polish scope (v1)

Included in v1:

- Add/remove tracked symbols from the UI.
- Persist tracked symbols and settings to a config file.
- Fetch quote/chart data via the free provider and draw a line chart per symbol.
- Grid layout (mini-chart tile per symbol), normal resizable window.
- Range toggles (1D default with live ticking, plus 5D/1W/1M/YTD/1Y/5Y/ALL).
- Percent-change display with up/down coloring.
- Offline / API-error handling (graceful degradation, no crashes).
- Live two-way config reload (UI edits and external file edits both apply
  without a restart).
- Snap packaging (`snapcraft.yaml`).

Deferred (architected-for, not built in v1):

- List + detail layout.
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
- **License:** to be decided (placeholder for now).

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

## Open items

- Choose a license before first public release.
- Confirm snap confinement plan (network access, `personal-files`/data dirs) during
  packaging.
