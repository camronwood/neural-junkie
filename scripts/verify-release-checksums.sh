#!/usr/bin/env bash
# Download SHA256SUMS from a GitHub Release and verify a local installer file.
set -euo pipefail

TAG="${1:-}"
FILE="${2:-}"
REPO="${NJ_RELEASE_REPO:-camronwood/neural-junkie}"

usage() {
  echo "Usage: $0 <release-tag> <path-to-installer>" >&2
  echo "Example: $0 v1.2.0-beta.5 ~/Downloads/Neural.Junkie_1.2.0-beta.5_aarch64.dmg" >&2
  exit 1
}

[[ -n "$TAG" && -n "$FILE" ]] || usage
[[ -f "$FILE" ]] || { echo "File not found: $FILE" >&2; exit 1; }

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI required (or download SHA256SUMS manually from the release)." >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

gh release download "$TAG" --repo "$REPO" --pattern SHA256SUMS --dir "$tmpdir"
sums_file="${tmpdir}/SHA256SUMS"
[[ -f "$sums_file" ]] || { echo "SHA256SUMS not found on release $TAG" >&2; exit 1; }

basename="$(basename "$FILE")"
line="$(grep -F "  ${basename}" "$sums_file" || grep -F " ${basename}" "$sums_file" || true)"
if [[ -z "$line" ]]; then
  echo "No checksum entry for ${basename} in SHA256SUMS" >&2
  echo "Available entries:" >&2
  cat "$sums_file" >&2
  exit 1
fi

expected="${line%% *}"

if command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$FILE" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$FILE" | awk '{print $1}')"
else
  echo "shasum or sha256sum required" >&2
  exit 1
fi

if [[ "$actual" == "$expected" ]]; then
  echo "OK: ${basename}"
  echo "SHA256: ${actual}"
  exit 0
fi

echo "MISMATCH for ${basename}" >&2
echo " expected: ${expected}" >&2
echo "   actual: ${actual}" >&2
exit 1
