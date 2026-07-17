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
| C toolchain + GUI dev headers | required to build the Fyne UI (cgo) |
| snapcraft | only needed to build the snap package |

### System build dependencies (Linux)

The Fyne UI links against OpenGL/X11/Wayland via cgo, so these OS packages are
required to `go build ./...`. On Debian/Ubuntu:

```sh
sudo apt install pkg-config gcc \
  libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev
```

Without these, the non-UI packages (`config`, `logging`, `model`, `provider`)
still build and test, but the `ui` package and the `cmd/simplestonks` binary do
not.

For the snap build only:

```sh
sudo snap install snapcraft --classic   # plus LXD, per snapcraft's docs
```

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

# Non-UI only (works without the GUI dev headers)
go test ./internal/config/... ./internal/logging/... \
        ./internal/model/... ./internal/provider/...
```

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
- Go module and package skeleton scaffolded (`QuoteProvider` interface stubbed).
- Live two-way config reload implemented and tested.
- Leveled, rotating file logger implemented and tested.

Immediate next candidates:

- Implement the Yahoo provider (`Quote` / `History`) — currently stubbed with
  `ErrNotImplemented`.
- Wire real data into the grid tiles and the chart widget.
- Set up `pre-commit` (unit) and `pre-push` (integration) test hooks once there
  is enough to test (see `docs/REQUIREMENTS.md` → Testing & CI).
