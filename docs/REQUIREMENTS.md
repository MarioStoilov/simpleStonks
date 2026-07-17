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
| Config | File-based, editable from the UI, XDG/snap-aware. |

## MVP + polish scope (v1)

Included in v1:

- Add/remove tracked symbols from the UI.
- Persist tracked symbols and settings to a config file.
- Fetch quote/chart data via the free provider and draw a line chart per symbol.
- Grid layout (mini-chart tile per symbol), normal resizable window.
- Range toggles (1D default with live ticking, plus 5D/1W/1M/YTD/1Y/5Y/ALL).
- Percent-change display with up/down coloring.
- Offline / API-error handling (graceful degradation, no crashes).
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

## Proposed project structure

```
simpleStonks/
├── cmd/simplestonks/main.go        # entrypoint
├── internal/
│   ├── config/                     # load/save JSON config (XDG/snap-aware)
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
