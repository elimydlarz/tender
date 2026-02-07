SHELL := /bin/bash

ROOT := $(CURDIR)
GOCACHE_DIR := $(ROOT)/.gocache
BIN_DIR := $(ROOT)/bin
BIN := $(BIN_DIR)/tender
NPM_CACHE_DIR := $(ROOT)/.tender/npm-cache

.PHONY: help build run npx-local npx-smoke npx-pack-smoke fmt fmt-check lint test acceptance check-fast check

help:
	@echo "tender project commands"
	@echo ""
	@echo "  make build        Build CLI to ./bin/tender"
	@echo "  make run          Build and run the CLI"
	@echo "  make npx-local    Run via local npx package path (interactive)"
	@echo "  make npx-smoke    Smoke test local npx launcher (--help/--version)"
	@echo "  make npx-pack-smoke  Smoke test packed npm artifact (no publish)"
	@echo "  make fmt          Format Go files"
	@echo "  make fmt-check    Verify Go formatting"
	@echo "  make lint         Run go vet"
	@echo "  make test         Run default tests"
	@echo "  make acceptance   Run acceptance tests (uses act + git)"
	@echo "  make check-fast   Run fmt-check + lint + test + build"
	@echo "  make check        Run full verification (check-fast + acceptance)"

build:
	@mkdir -p "$(BIN_DIR)"
	@mkdir -p "$(GOCACHE_DIR)"
	@GOCACHE="$(GOCACHE_DIR)" go build -o "$(BIN)" ./cmd/tender

run: build
	@"$(BIN)"

npx-local: build
	@mkdir -p "$(NPM_CACHE_DIR)"
	@TENDER_BINARY_PATH="$(BIN)" NPM_CONFIG_CACHE="$(NPM_CACHE_DIR)" npx --yes .

npx-smoke: build
	@mkdir -p "$(NPM_CACHE_DIR)"
	@TENDER_BINARY_PATH="$(BIN)" NPM_CONFIG_CACHE="$(NPM_CACHE_DIR)" npx --yes . --help >/dev/null
	@TENDER_BINARY_PATH="$(BIN)" NPM_CONFIG_CACHE="$(NPM_CACHE_DIR)" npx --yes . --version >/dev/null

npx-pack-smoke: build
	@mkdir -p "$(NPM_CACHE_DIR)"
	@pkg="$$(npm pack --silent)"; \
	TENDER_BINARY_PATH="$(BIN)" NPM_CONFIG_CACHE="$(NPM_CACHE_DIR)" npx --yes "./$$pkg" --help >/dev/null; \
	TENDER_BINARY_PATH="$(BIN)" NPM_CONFIG_CACHE="$(NPM_CACHE_DIR)" npx --yes "./$$pkg" --version >/dev/null; \
	rm -f "$$pkg"

fmt:
	@files="$$(find cmd internal -name '*.go' -type f)"; \
	if [ -n "$$files" ]; then \
		gofmt -w $$files; \
	fi

fmt-check:
	@files="$$(find cmd internal -name '*.go' -type f)"; \
	if [ -z "$$files" ]; then \
		exit 0; \
	fi; \
	unformatted="$$(gofmt -l $$files)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint:
	@mkdir -p "$(GOCACHE_DIR)"
	@GOCACHE="$(GOCACHE_DIR)" go vet ./...

test:
	@mkdir -p "$(GOCACHE_DIR)"
	@GOCACHE="$(GOCACHE_DIR)" go test ./...

acceptance:
	@./scripts/run-acceptance.sh

check-fast: fmt-check lint test build

check: check-fast acceptance
