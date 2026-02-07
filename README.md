# tender

`tender` is an interactive CLI for scheduling autonomous OpenCode runs in GitHub Actions.

Tender keeps state in workflow files only. No sidecar metadata files.

## Install (pnpm)

One-off run:

```bash
pnpm dlx @tender/cli@latest
```

Install in your repo:

```bash
pnpm add -D @tender/cli
pnpm exec tender
```

## Quick Start

1. Run `pnpm exec tender`.
2. Choose `Add tender`.
3. Enter a name.
4. Pick agent and schedule from the numeric menus.

Tender writes/updates a managed workflow under `.github/workflows/`.

## Commands

- `tender` launches the interactive TUI.
- `tender ls` lists managed tenders.
- `tender run <name>` triggers a tender immediately via `workflow_dispatch` (requires `gh` auth).
- `tender rm <name>` removes a managed tender.

## How It Works

- Uses GitHub Actions workflows as the source of truth.
- Detects OpenCode primary agents from local OpenCode config.
- Generates workflows that run `opencode run --agent ...`.
- Supports on-demand and scheduled runs.
- Uses plain-English trigger display in the CLI.
- Pushes changes directly to `main` from workflow runs.

## Requirements

- GitHub repository with Actions enabled.
- OpenCode config in the repo (or defaults available to OpenCode).
- Secrets expected by your OpenCode setup (for example API keys).

## Contributing

Contributor/developer docs are in `CONTRIBUTING.md`.
