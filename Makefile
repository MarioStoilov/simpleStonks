.PHONY: help build run fmt check vet test test-race test-integration hooks

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
