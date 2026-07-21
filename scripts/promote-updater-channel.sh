#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:?Usage: $0 <version-tag> [repo]}"
REPO="${2:-camronwood/neural-junkie}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if [[ "${VERSION}" != *beta* ]]; then
  echo "Stable releases use GitHub latest directly; no channel pointer to advance."
  exit 0
fi

is_draft="$(gh release view "${VERSION}" --repo "${REPO}" --json isDraft -q .isDraft)"
if [[ "${is_draft}" != "false" ]]; then
  echo "Refusing to promote draft release ${VERSION}" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
gh release download "${VERSION}" --repo "${REPO}" --pattern 'update-*.json' --dir "${tmpdir}"

cd "${ROOT}"
if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
  git fetch origin main
  git checkout main
  git pull --ff-only origin main
fi

mkdir -p updater/beta
cp "${tmpdir}"/update-*.json updater/beta/
if git diff --quiet -- updater/beta; then
  echo "Beta channel already points at ${VERSION}."
  exit 0
fi

git add updater/beta
git -c user.name='github-actions[bot]' \
  -c user.email='41898282+github-actions[bot]@users.noreply.github.com' \
  commit -m "Sync beta updater manifests for ${VERSION}."
if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
  git push origin main
else
  echo "Channel pointer committed locally; push it after review."
fi
