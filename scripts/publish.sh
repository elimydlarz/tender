#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "usage: ./scripts/publish.sh <version>"
  echo "example: ./scripts/publish.sh 0.2.0"
  exit 2
fi

if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "error: version must look like semver (for example 1.2.3 or 1.2.3-rc.1)"
  exit 2
fi

TAG="v$VERSION"
DRY_RUN="${DRY_RUN:-0}"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: working tree is dirty; commit or stash changes first"
  exit 1
fi

if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null 2>&1; then
  echo "error: git tag $TAG already exists"
  exit 1
fi

echo "==> Running release checks"
make check-fast
make npx-pack-smoke

echo "==> Setting package version to $VERSION"
npm version --no-git-tag-version "$VERSION" >/dev/null

echo "==> Committing version bump"
git add package.json
git commit -m "release: $TAG"

echo "==> Creating tag $TAG"
git tag "$TAG"

if [[ "$DRY_RUN" == "1" ]]; then
  echo "==> Dry run complete"
  echo "created local commit and tag only; nothing pushed"
  exit 0
fi

echo "==> Pushing commit and tag"
git push origin HEAD
git push origin "$TAG"

cat <<EOF
Release kicked off.
- GitHub Actions workflow: release
- Builds binaries and publishes a GitHub release
- Publishes npm package @tender/cli (requires NPM_TOKEN secret)
EOF
