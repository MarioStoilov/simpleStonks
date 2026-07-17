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
- Match the style, naming, and structure of existing code.

## Notes

- Requirements and architecture are still being finalized; check with the user
  before making large structural decisions.
