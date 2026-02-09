#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "usage: ./scripts/release-assets.sh <version>"
  echo "example: ./scripts/release-assets.sh 0.6.0"
  exit 2
fi

if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "error: version must look like semver (for example 1.2.3 or 1.2.3-rc.1)"
  exit 2
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "error: GitHub CLI (gh) is required to publish release assets"
  exit 2
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "error: gh is not authenticated"
  echo "hint: run 'gh auth login' and retry"
  exit 2
fi

GH_REPO="${TENDER_GITHUB_REPO:-elimydlarz/tender}"

TAG="v$VERSION"
TMPDIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMPDIR"
}
trap cleanup EXIT

targets=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

assets=()
for target in "${targets[@]}"; do
  read -r goos goarch <<<"$target"
  ext=""
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
  fi

  output="$TMPDIR/tender_${VERSION}_${goos}_${goarch}${ext}"
  echo "==> Building $goos/$goarch"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -o "$output" ./cmd/tender
  assets+=("$output")
done

if gh release view "$TAG" --repo "$GH_REPO" >/dev/null 2>&1; then
  echo "==> Uploading binaries to existing release $TAG ($GH_REPO)"
  gh release upload "$TAG" --repo "$GH_REPO" --clobber "${assets[@]}"
else
  echo "==> Creating release $TAG in $GH_REPO"
  gh release create "$TAG" --repo "$GH_REPO" --title "$TAG" --notes "Automated release for $TAG" "${assets[@]}"
fi

echo "==> Release assets ready for $TAG"
