#!/usr/bin/env bash
# Build SHA256SUMS from all GitHub Release assets and upload (CI + maintainers).
set -euo pipefail

TAG="${1:?Usage: $0 <release-tag> [repo]}"
REPO="${2:-camronwood/neural-junkie}"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI required" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "Downloading release assets for ${TAG}..."
gh release download "$TAG" --repo "$REPO" --dir "$tmpdir"

shopt -s nullglob
files=("$tmpdir"/*)
if [[ ${#files[@]} -eq 0 ]]; then
  echo "No assets downloaded for ${TAG}" >&2
  exit 1
fi

sums_file="${tmpdir}/SHA256SUMS"
(
  cd "$tmpdir"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -- * 2>/dev/null | grep -v ' SHA256SUMS$' > SHA256SUMS || true
  else
    shasum -a 256 -- * 2>/dev/null | grep -v ' SHA256SUMS$' > SHA256SUMS || true
  fi
)

if [[ ! -s "$sums_file" ]]; then
  echo "Failed to generate SHA256SUMS" >&2
  exit 1
fi

echo "Uploading SHA256SUMS to ${TAG}..."
gh release upload "$TAG" "$sums_file" --repo "$REPO" --clobber

echo "Done. Entries:"
cat "$sums_file"
