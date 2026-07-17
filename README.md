# simpleStonks

A bare-bones stock tracker widget/app. simpleStonks shows a configurable list of
stock indexes and charts their movement in a small window/widget.

## Features

- Track a user-defined list of stock indexes / tickers
- Chart price movement over time
- Manage the tracked list from the UI
- Persist the tracked list and other settings to a config file

## Status

Early development. Requirements and design are being finalized.

## Tech

- Written in Go
- Distributed via the [Snap Store](https://snapcraft.io/) (snapcraft)

## Building

Requires Go 1.23+ and (for the GUI) some system dev headers. Quick start:

```sh
go run ./cmd/simplestonks
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for prerequisites, the full
build/test commands, runtime file locations, and how to continue a session on
another machine.

## License

_To be decided._
