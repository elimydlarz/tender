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
PACKAGE_NAME="$(node -pe "require('./package.json').name")"

if [[ "$DRY_RUN" != "1" && -z "${NPM_TOKEN:-}" ]]; then
  echo "error: NPM_TOKEN must be set for npm publish"
  exit 2
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: working tree is dirty; commit or stash changes first"
  exit 1
fi

if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null 2>&1; then
  echo "error: git tag $TAG already exists"
  exit 1
fi

NPMRC=""
cleanup() {
  [[ -n "$NPMRC" ]] && rm -f "$NPMRC"
}
trap cleanup EXIT

if [[ "$DRY_RUN" != "1" ]]; then
  NPMRC="$(mktemp)"
  cat >"$NPMRC" <<EOF
registry=https://registry.npmjs.org/
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
EOF

  echo "==> Verifying npm token"
  if ! NPM_USER="$(NPM_CONFIG_USERCONFIG="$NPMRC" npm whoami 2>/dev/null)"; then
    echo "error: npm auth failed for NPM_TOKEN (token expired/revoked or missing publish permission)"
    echo "hint: create a fresh npm token and ensure it can publish packages"
    exit 2
  fi

  if ! NPM_CONFIG_USERCONFIG="$NPMRC" npm view "$PACKAGE_NAME" version >/dev/null 2>&1; then
    if [[ "$PACKAGE_NAME" == @*/* ]]; then
      SCOPE="${PACKAGE_NAME#@}"
      SCOPE="${SCOPE%%/*}"
      if [[ "$SCOPE" != "$NPM_USER" ]] && ! NPM_CONFIG_USERCONFIG="$NPMRC" npm org ls "$SCOPE" "$NPM_USER" >/dev/null 2>&1; then
        echo "error: package $PACKAGE_NAME is not published yet, and npm user '$NPM_USER' is not a member of org '$SCOPE'"
        echo "hint: add '$NPM_USER' to npm org '$SCOPE' with publish rights, or publish under your own scope"
        exit 2
      fi
    fi
  fi
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
  echo "==> Running npm publish dry run"
  npm publish --access public --dry-run
  echo "==> Dry run complete"
  echo "created local commit and tag only; package was not published"
  exit 0
fi

echo "==> Building and uploading GitHub release binaries"
./scripts/release-assets.sh "$VERSION"

echo "==> Publishing npm package $PACKAGE_NAME"
NPM_CONFIG_USERCONFIG="$NPMRC" npm publish --access public

cat <<EOF
Release complete.
- Uploaded release binaries for $TAG
- Published npm package $PACKAGE_NAME@$VERSION
- Created local commit and tag $TAG
EOF
