# Contributing

## Scope

This document is for contributors working on `tender` itself.

## Project Layout

- CLI entrypoint: `cmd/tender/main.go`
- Core workflow logic: `internal/tender/workflow.go`
- Interactive TUI: `internal/tender/ui.go`
- OpenCode agent discovery: `internal/tender/opencode_agents.go`
- Acceptance tests: `internal/tender/acceptance_test.go`
- Acceptance runner: `scripts/run-acceptance.sh`

## Local Development

Build:

```bash
make build
```

Run interactive TUI:

```bash
make run
```

Top-level checks:

```bash
make check-fast
make check
```

## Test Strategy

- Unit/default tests run with `go test`.
- Acceptance tests validate generated workflow behavior with `act`.
- TTY regression acceptance coverage includes numeric menu + Enter interaction.

Run acceptance:

```bash
./scripts/run-acceptance.sh
```

## Local Launcher Validation

Use these to verify npm/pnpm-facing launcher behavior without publishing:

```bash
make npx-smoke
make npx-pack-smoke
make npx-local
```

## Releasing

Single command release:

```bash
make publish VERSION=0.2.0
```

Dry run (no publish):

```bash
make release-dry-run VERSION=0.2.0
```

Local publish requirements:
- `NPM_TOKEN` environment variable must be set in your shell.

## Workflow Contract

Managed workflows should keep these properties:

- State is stored in GitHub Actions workflow files only.
- Trigger via `workflow_dispatch` and/or `schedule`.
- Run OpenCode with configured `--agent`.
- Commit/push directly to `main`.
- Use shared concurrency group `tender-main`.

## Runtime Artifacts

- Runtime/test artifacts live under `.tender/`.
- Acceptance fixture repos live under `.tender/test-work/`.
- Local npm cache for launcher tests lives under `.tender/npm-cache/`.
