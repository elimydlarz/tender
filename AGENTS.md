# AGENTS.md

## Scope
These instructions apply to the entire repository.

## Audience
- `AGENTS.md` is internal guidance for coding agents working in this repo.
- Public user docs are in `README.md`.
- Contributor-facing project docs are in `CONTRIBUTING.md`.

## Project intent
- `tender` is a Go CLI/TUI for managing autonomous OpenCode runs via GitHub Actions.
- GitHub Actions workflow files are the source of truth for tender state.
- No sidecar metadata files are used for tender definitions.
- Interactive agent selection should prefer locally configured OpenCode primary agents.
- Dogfooding OpenCode config is checked into this repo (`opencode.json`, `.opencode/agents/`).

## Code layout
- CLI entrypoint: `cmd/tender/main.go`
- Core workflow logic: `internal/tender/workflow.go`
- Interactive UX: `internal/tender/ui.go`
- Acceptance tests: `internal/tender/acceptance_test.go`
- Acceptance runner: `scripts/run-acceptance.sh`

## Build and test
- Preferred DX (top-level commands):
  - `make help`: list project commands
  - `make build`: build the CLI
  - `make run`: build and run the CLI
  - `./bin/tender run <name>`: trigger a configured tender immediately (requires `gh`)
  - `make npx-smoke`: verify local npm launcher path without publishing
  - `make npx-pack-smoke`: verify packed npm artifact without publishing
  - `make npx-local`: run interactive CLI through local `npx --yes .`
  - `make fmt`: format Go files
  - `make fmt-check`: fail if formatting is not clean
  - `make lint`: run `go vet`
  - `make test`: run default tests
  - `make acceptance`: run acceptance tests (`act` + `git`)
  - `make check-fast`: run `fmt-check + lint + test + build`
  - `make check`: run full verification (`check-fast + acceptance`)
- Direct commands still supported:
  - Build: `GOCACHE=$PWD/.gocache go build ./...`
  - Unit/default tests: `GOCACHE=$PWD/.gocache go test ./...`
  - Acceptance tests: `./scripts/run-acceptance.sh`

## Runtime/test artifacts
- Keep runtime and dogfooding artifacts under `.tender/`.
- Acceptance fixture repos live under `.tender/test-work/`.
- Local npm cache for launcher testing lives under `.tender/npm-cache/`.
- Keep build/test source code in normal repo paths (`cmd/`, `internal/`, `scripts/`).
- Keep Go cache local to repo via `.gocache/` for reproducible local DX.

## Workflow behavior requirements
Generated workflows must keep these properties:
- Run OpenCode using the configured agent and optional prompt.
- Support `workflow_dispatch` and/or `schedule` triggers.
- Commit and push directly to `main` (buyer beware).
- Use shared concurrency group `tender-main`.

## Distribution contract
- Public install path is package-based (`pnpm dlx @tender/cli@latest` or `pnpm exec tender` when installed).
- Keep CLI interactive-first by default.
- Local launcher verification path is `make npx-smoke`, `make npx-pack-smoke`, and `make npx-local`.
