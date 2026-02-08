# tender

`tender` is an interactive CLI/TUI for managing autonomous OpenCode runs in GitHub
Actions.

Tender keeps state in workflow files only. No sidecar metadata files.

## Run

No install required:

```bash
pnpm dlx @tender/cli@latest
# or: npx @tender/cli@latest
```

Run subcommands the same way:

```bash
pnpm dlx @tender/cli@latest ls
pnpm dlx @tender/cli@latest run my-tender
# or: npx @tender/cli@latest ls
```

## Quick Start

1. Ensure repo secrets are configured for your OpenCode provider.
2. Run `pnpm dlx @tender/cli@latest` (or `npx @tender/cli@latest`).
3. Choose `Add tender`.
4. Enter a name.
5. Pick agent and schedule.
6. Commit and push the generated workflow under `.github/workflows/`.

## Requirements

- GitHub repository with Actions enabled.
- OpenCode config in the repo (`opencode.json` and/or `.opencode/`) or defaults
  available to OpenCode.
- GitHub CLI (`gh`) authenticated for local dispatches (`tender run`).
- Provider API key secrets for OpenCode (for example `OPENAI_API_KEY`,
  `ANTHROPIC_API_KEY`) configured in your repository.

## Authentication

Tender uses two auth flows.

### 1. Local auth (`tender run`)

- `tender run <name>` shells out to `gh workflow run ...`.
- You must be logged into GitHub CLI for the target repo (`gh auth login`).

### 2. GitHub Actions auth (workflow execution)

Generated workflows do this in the `Run OpenCode` step:

- Install OpenCode on the runner.
- Pass provider credentials from repo secrets:
  - `OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}`
  - `ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}`
- Set config paths when present:
  - `OPENCODE_CONFIG=$GITHUB_WORKSPACE/opencode.json`
  - `OPENCODE_CONFIG_DIR=$GITHUB_WORKSPACE/.opencode`

Generated workflows also set `permissions: contents: write`, which allows the job
token to push commits back to `main`.

## Commands

Use these as:

```bash
pnpm dlx @tender/cli@latest <command>
# or: npx @tender/cli@latest <command>
```

- `tender` launches the interactive TUI.
- `tender init` ensures `.github/workflows` exists.
- `tender add [--name <name>] --agent <agent> [--prompt "..."] [--cron "..."] [--manual true|false] [--push true|false] [<name>]`
  creates a tender non-interactively (for coding agents/automation).
- `tender update <name> [--name <new-name>] [--agent <agent>] [--prompt "..."] [--cron "..."] [--clear-cron] [--manual true|false] [--push true|false]`
  updates an existing tender non-interactively.
- `tender ls` lists managed tenders.
- `tender run [--prompt "..."] <name>` triggers a tender immediately via
  `workflow_dispatch`.
- `tender rm [--yes] <name>` removes a managed tender.
- `tender --help` lists commands.
- `tender help [command]` (or `tender <command> --help`) shows command-specific usage.

Example non-interactive flow:

```bash
pnpm dlx @tender/cli@latest add --name nightly --agent Build --cron "0 9 * * 1"
pnpm dlx @tender/cli@latest update nightly --agent TendTests --push true --manual false --clear-cron
```

## How It Works

- Uses GitHub Actions workflow files as the source of truth.
- Detects OpenCode agents via `opencode agent list`.
- Generates workflows that run `opencode run --agent ...`.
- Supports on-demand and scheduled runs.
- Uses plain-English trigger display in the CLI.
- Pushes changes directly to `main` from workflow runs.
- Uses shared concurrency group `tender-main`.

## Contributing

Contributor/developer docs are in `CONTRIBUTING.md`.

## Maintainer Release

```bash
make publish VERSION=0.2.0
```

What it does:

- Runs `check-fast` and npm pack smoke checks.
- Bumps `package.json` version.
- Commits `release: vX.Y.Z`.
- Tags `vX.Y.Z` and pushes commit + tag.
- Triggers GitHub Actions `release` workflow to build binaries and publish
  `@tender/cli`.

Repository secret required for npm publishing:

- `NPM_TOKEN`
