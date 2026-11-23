# Development

How to set up a machine to work on simpleStonks.

## Prerequisites

| Tool | Version / notes |
|------|-----------------|
| Go | 1.23+ (module targets `go 1.23.5`) |
| Git | any recent version |
| C toolchain + Qt 6 dev headers | required to build the Qt UI (cgo via miqt) |
| snapcraft | only for building the snap locally; releases are built by CI |

### System build dependencies (Linux)

The Qt UI (miqt bindings) compiles against Qt 6 via cgo, so these OS packages
are required to `go build ./...`. On Debian/Ubuntu:

```sh
sudo apt install pkg-config gcc qt6-base-dev qt6-tools-dev
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
starts fetching data.

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

## Releasing

Releases are automated: a `vX.Y.Z` tag on the `release` branch triggers
`.github/workflows/release.yml`, which runs the test suite, builds the snap in
LXD on an `ubuntu-24.04` runner (matching `base: core24`), keeps the `.snap` as
a workflow artifact, and uploads it to the Snap Store.

### Branches and tags

- `main` is where development happens.
- `release` only ever fast-forwards to `main`; it exists so a tag can be
  checked against it. The workflow refuses tags whose commit is not on
  `release`, since GitHub cannot filter tag events by branch.
- Tags are annotated, `vX.Y.Z`, and must equal both `version` in
  `snap/snapcraft.yaml` and `constants.AppVersion` (the workflow checks).
- A plain tag releases to `stable,edge`; a suffixed tag such as `v1.5.0-rc1`
  releases to `edge` only.

### Cutting a release

1. On `main`: bump the two versions, write the changelog into the snapcraft
   `description`, commit, push.
2. `make release VERSION=X.Y.Z` - verifies the tree is clean, on `main`, pushed,
   and that both versions match; then fast-forwards `release`, creates the
   annotated tag, and pushes branch and tag with `git push --atomic`.
3. Follow the run under the repo's Actions tab. If the `snapstore` environment
   has a required reviewer configured, approve the "Build and publish" job
   there.

### Store credential

The workflow authenticates with `SNAPCRAFT_STORE_CREDENTIALS`, a secret of the
`snapstore` GitHub Environment (Settings, Environments). It is the content of a
file produced by `snapcraft export-login`, scoped to this snap, to the
`stable`/`edge` channels, and to an expiry date. When it expires the upload
step fails with an authentication error; renew it on a machine where
`snapcraft` is logged in:

```sh
snapcraft export-login \
  --snaps=simplestonks \
  --acls=package_access,package_push,package_update,package_release \
  --channels=stable,edge \
  --expires=<ISO 8601 date about a year out> \
  simplestonks-ci-login.txt
SNAPCRAFT_STORE_CREDENTIALS="$(cat simplestonks-ci-login.txt)" snapcraft whoami   # check scope and expiry
```

Paste the file's content into the environment secret, then `rm` the file
(`*-login.txt` is gitignored as a safety net). `package_update` is required
because `snapcraft upload` syncs the store listing links from snapcraft.yaml.

### Building locally

`snapcraft pack` still works for a local smoke test (needs LXD; on hosts
running Docker, allow `lxdbr0` in the `DOCKER-USER` iptables chain or
container networking is blocked). Local builds are not uploaded.
