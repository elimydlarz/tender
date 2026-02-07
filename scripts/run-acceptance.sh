#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p .gocache .tender/test-work
export GOCACHE="$ROOT/.gocache"

go test -tags acceptance ./internal/tender -v
