.PHONY: help build run fmt check vet test test-race test-integration hooks release

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

build: ## Build all packages (needs the GUI dev headers; see docs/DEVELOPMENT.md)
	go build ./...

run: ## Run the app
	go run ./cmd/simplestonks

fmt: ## Format all Go files
	gofmt -w .

vet: ## Run go vet
	go vet ./...

test: ## Run unit tests
	go test ./...

test-race: ## Run unit tests with the race detector
	go test -race ./...

test-integration: ## Run unit + integration tests (-race)
	go test -race -tags=integration ./...

check: ## Pre-commit gate: fmt check, vet, unit tests
	@test -z "$$(gofmt -l .)" || { echo "not gofmt-clean:"; gofmt -l .; exit 1; }
	go vet ./...
	go test ./...

hooks: ## Install the versioned git hooks (run once per clone)
	git config core.hooksPath .githooks
	@echo "Installed git hooks: core.hooksPath=.githooks"

release: ## Tag a release from main: make release VERSION=1.5.0 (versions already bumped and pushed)
	@test -n "$(VERSION)" || { echo "usage: make release VERSION=X.Y.Z"; exit 1; }
	@test -z "$$(git status --porcelain)" || { echo "working tree not clean"; exit 1; }
	@test "$$(git branch --show-current)" = main || { echo "run from main"; exit 1; }
	@grep -q '^version: "$(VERSION)"$$' snap/snapcraft.yaml || { echo "snap/snapcraft.yaml version is not $(VERSION)"; exit 1; }
	@grep -q 'AppVersion = "$(VERSION)"$$' internal/constants/constants.go || { echo "constants.AppVersion is not $(VERSION)"; exit 1; }
	git fetch origin
	@test "$$(git rev-parse HEAD)" = "$$(git rev-parse origin/main)" || { echo "main is not pushed"; exit 1; }
	git switch release
	git merge --ff-only main
	git tag -a "v$(VERSION)" -m "Release $(VERSION)"
	git push --atomic origin release "v$(VERSION)"
	git switch main
	@echo "Pushed v$(VERSION); the Release snap workflow is now running on GitHub Actions"
