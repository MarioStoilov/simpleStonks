# CLAUDE.md

Guidance for Claude Code (and other AI assistants) working in this repository.

## Start here

Before doing any work, read [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) — it covers
the environment setup, build/test commands, runtime file locations, and how a
session is continued across machines (the Git remote is the source of truth).
This matters because a session may be resumed on a different machine with no
memory of prior work; the docs carry the context.

For scope, locked decisions, and open items, see
[docs/REQUIREMENTS.md](docs/REQUIREMENTS.md).

## Project

**simpleStonks** — a bare-bones stock tracker widget/app written in Go and
distributed via the Snap Store (snapcraft). It shows a configurable list of stock
indexes and charts their movement in a window/widget. The tracked list is editable
from the UI, and the list plus other settings are persisted to a config file.

## Commit conventions

- **Do NOT add AI co-authoring to commit messages.** Never include
  `Co-Authored-By: Claude` (or any similar AI attribution) or
  `Generated with Claude Code` lines in commit messages.
- Write clear, imperative commit subjects (e.g. "Add config persistence").
- Only commit or push when explicitly asked.

## Conventions

- Keep it simple — this is intentionally a minimal app.
- Persisted config/settings should follow the XDG Base Directory spec and must be
  compatible with the snap confinement model (write to snap-provided data dirs).
- The config file supports live two-way reload — UI edits and external file edits
  both apply without restarting. Don't add code paths that require a restart to
  pick up config changes.
- Logging uses the standard library `log/slog` via the `internal/logging` logger,
  which is installed as slog's default. Log through `slog` (levels: silent →
  debug); logging config (level, file, rotation) lives in the config file and is
  live-reloaded.
- **Naming: no one- or two-character identifiers.** Receivers, parameters,
  locals, loop variables, and struct fields must use descriptive names of at
  least three characters (e.g. `app *App` not `a *App`, `series, err :=` not
  `s, err :=`, `for _, value :=` not `for _, v :=`, `idx` not `i`). Allowed
  exceptions: the comma-ok idiom (`v, ok :=` keeps `ok`) and the standard
  `t *testing.T`.
- **No magic numbers or strings.** All numeric literals, technical strings
  (URLs, file names, time/number formats), and user-facing text (labels,
  titles, messages) are defined in `internal/constants` — grouped by domain —
  and referred to from there. Exceptions that stay at their definition sites:
  typed enum values (`model.Range`, `config.LogLevel`), struct tags, protocol
  mapping tables (e.g. `yahooParams`), diagnostic log/error-wrap messages, and
  test fixtures.
- **Qt stylesheet templates live in `internal/constants/styles.go`.**
  Multi-rule stylesheets (anything with several QSS rules or pseudo-states,
  e.g. a button's hover/disabled variants) are `fmt.Sprintf` templates named
  `Style*` there, filled in with palette colors and metrics at their use
  sites. Small single-rule inline styles (e.g. `"background: transparent;"`
  or one-color concatenations) may stay at their use sites.
- Match the style, naming, and structure of existing code.

## Testing

- Write testable code: reach side effects (filesystem, network, clock) through
  interfaces or injectable clients so units can be tested in isolation.
- **Unit tests run on every commit**; **integration tests run on every push** —
  enforced by the versioned hooks in `.githooks/` (installed per clone with
  `make hooks`).
- Integration tests carry the `//go:build integration` tag so they are excluded
  from the commit run and included on push (`go test -tags=integration ./...`).
  Network-dependent ones skip themselves when offline.
- Tests should pass under the race detector: `go test -race ./...`.
- Common commands are wrapped in the `Makefile` (`make check`, `make
  test-integration`, `make hooks`, …).

## Snap release runbook

When building and publishing a new snap version, follow these steps in order
(machine specifics — LXD, Docker firewall, upload commands — are in
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)):

0. **Preflight:** verify the Docker/LXD firewall fix is applied (container
   networking works) and that snapcraft is logged in.
1. **Clean tree:** all code must be committed and pushed before building.
2. **Version bump:** ask the user whether this is a major, minor, or fix
   release, then bump both the `version` in `snap/snapcraft.yaml` and
   `constants.AppVersion` in `internal/constants` (keep them in sync).
3. **Tag:** create a git tag for the new version (`v<version>`, e.g. `v1.2.1`).
4. **Changelog:** update the `description` in `snap/snapcraft.yaml` to include
   a simple changelog of what has changed / been fixed in this release.
5. **Build:** build the snap (`snapcraft pack`).
6. **Publish:** ask the user which channels to release to (e.g. `edge`,
   `stable`), then upload/release the snap to those channels.

## Notes

- Requirements and architecture are still being finalized; check with the user
  before making large structural decisions.
