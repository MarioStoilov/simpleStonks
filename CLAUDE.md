# CLAUDE.md

Guidance for Claude Code (and other AI assistants) working in this repository.

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
- Match the style, naming, and structure of existing code.

## Testing

- Write testable code: reach side effects (filesystem, network, clock) through
  interfaces or injectable clients so units can be tested in isolation.
- **Unit tests run on every commit**; **integration tests run on every push**
  (enforced via `pre-commit` / `pre-push` git hooks and, later, CI).
- Tests should pass under the race detector: `go test -race ./...`.

## Notes

- Requirements and architecture are still being finalized; check with the user
  before making large structural decisions.
