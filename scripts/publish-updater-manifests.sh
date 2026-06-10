#!/usr/bin/env bash
#
# Generates and uploads updater manifests for a release.
# Syncs rolling beta manifests to updater/beta/ on main (git-backed channel).
#
# Usage: ./scripts/publish-updater-manifests.sh v1.0.0-beta.25 [repo]

set -euo pipefail

VERSION="${1:?Usage: $0 <version-tag> [repo]}"
REPO="${2:-camronwood/neural-junkie}"
BETA_MANIFEST_DIR="updater/beta"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

chmod +x scripts/generate-update-manifests.sh

./scripts/generate-update-manifests.sh "${VERSION}" "${REPO}"

shopt -s nullglob
manifests=(update-*.json)
required=(update-darwin-aarch64.json update-darwin-x86_64.json update-windows-x86_64.json)
if [[ ${#manifests[@]} -eq 0 ]]; then
  echo "No manifests generated" >&2
  exit 1
fi
for need in "${required[@]}"; do
  if [[ ! -f "${need}" ]]; then
    echo "Missing required manifest: ${need}" >&2
    exit 1
  fi
done
if [[ ! -f update-linux-x86_64.json ]]; then
  echo "WARN: Linux updater manifest skipped (AppImage bundle not published yet)" >&2
fi

echo "Uploading manifests to ${VERSION}..."
gh release upload "${VERSION}" "${manifests[@]}" --repo "${REPO}" --clobber

if [[ "${VERSION}" == *beta* ]]; then
  echo "Syncing rolling beta manifests to ${BETA_MANIFEST_DIR}/..."
  mkdir -p "${BETA_MANIFEST_DIR}"
  for manifest in "${manifests[@]}"; do
    cp "${manifest}" "${BETA_MANIFEST_DIR}/${manifest}"
  done
  if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
    git config user.name "github-actions[bot]"
    git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
    git fetch origin main
    git checkout main
    git pull --ff-only origin main
    mkdir -p "${BETA_MANIFEST_DIR}"
    for manifest in "${manifests[@]}"; do
      cp "${manifest}" "${BETA_MANIFEST_DIR}/${manifest}"
    done
  fi
  if ! git diff --quiet -- "${BETA_MANIFEST_DIR}"; then
    git add "${BETA_MANIFEST_DIR}"
    git commit -m "Sync beta updater manifests for ${VERSION}."
    if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
      git push origin main
    else
      git push origin HEAD
    fi
  else
    echo "No changes under ${BETA_MANIFEST_DIR}/"
  fi
fi

echo "Updater manifests published."
