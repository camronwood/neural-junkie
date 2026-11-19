#!/usr/bin/env bash
# Automated Gate 5 package checks: version consistency + optional updater verify.
# Does not replace a real OS install, but covers artifact integrity users depend on.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

chmod +x scripts/verify-desktop-version-consistency.sh

echo "=== desktop version consistency ==="
./scripts/verify-desktop-version-consistency.sh

# If a release tag is provided, verify updater manifests for that channel.
TAG="${SHIP_BAR_TAG:-}"
CHANNEL="${SHIP_BAR_CHANNEL:-beta}"
if [[ -n "${TAG}" ]]; then
  chmod +x scripts/verify-updater-manifest.sh
  echo "=== updater manifest ${TAG} (${CHANNEL}) ==="
  ./scripts/verify-updater-manifest.sh "${TAG}" "${CHANNEL}"
else
  # Prefer latest beta tag from git when online; skip softly if none.
  if latest="$(git tag -l 'v*-beta.*' --sort=-v:refname | head -1)"; then
    if [[ -n "${latest}" ]]; then
      echo "=== updater manifest ${latest} (beta) — best-effort ==="
      chmod +x scripts/verify-updater-manifest.sh
      if ./scripts/verify-updater-manifest.sh "${latest}" beta; then
        echo "updater verify PASS for ${latest}"
      else
        echo "WARN: updater verify failed for ${latest} (network/manifest); version consistency still required" >&2
      fi
    fi
  fi
fi

echo "ship-bar-package: PASS"
