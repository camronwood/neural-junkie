#!/usr/bin/env bash
#
# Validates updater manifest JSON from GitHub release assets.
#
# Usage: ./scripts/verify-updater-manifest.sh v1.0.0-beta.25 [channel]
#   channel: beta (default) or stable

set -euo pipefail

VERSION="${1:?Usage: $0 <version-tag> [beta|stable]}"
CHANNEL="${2:-beta}"
REPO="${VERIFY_REPO:-camronwood/neural-junkie}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck source=scripts/release-bundle-version.sh
source "${ROOT}/scripts/release-bundle-version.sh"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

manifests=(
  "update-darwin-aarch64.json"
  "update-darwin-x86_64.json"
  "update-linux-x86_64.json"
  "update-windows-x86_64.json"
)

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

BETA_MANIFEST_BASE="${BETA_MANIFEST_BASE:-https://raw.githubusercontent.com/${REPO}/main/updater/beta}"

for manifest in "${manifests[@]}"; do
  if [[ "${CHANNEL}" == "beta" ]]; then
    url="${BETA_MANIFEST_BASE}/${manifest}"
  else
    url="https://github.com/${REPO}/releases/latest/download/${manifest}"
  fi

  echo "==> ${manifest}"
  if ! curl -fsSL "${url}" -o "${tmpdir}/${manifest}"; then
    if [[ "${manifest}" == "update-linux-x86_64.json" ]]; then
      echo "WARN: Linux automatic updater manifest is intentionally not required yet; skipping"
      continue
    fi
    echo "FAIL: could not fetch ${url}" >&2
    exit 1
  fi

  version="$(jq -r '.version // empty' "${tmpdir}/${manifest}")"
  if [[ -z "${version}" ]]; then
    echo "FAIL: missing version in ${manifest}" >&2
    exit 1
  fi

  platform_key="${manifest#update-}"
  platform_key="${platform_key%.json}"
  manifest_platform=""
  [[ "${platform_key}" == windows-* ]] && manifest_platform="windows"
  expected_version="$(bundle_version_from_tag "${VERSION}" "${manifest_platform}")"
  if [[ "${version}" != "${expected_version}" ]]; then
    echo "FAIL: ${manifest} version=${version}, expected=${expected_version}" >&2
    exit 1
  fi

  sig="$(jq -r --arg key "${platform_key}" '.platforms[$key].signature // empty' "${tmpdir}/${manifest}")"
  bundle_url="$(jq -r --arg key "${platform_key}" '.platforms[$key].url // empty' "${tmpdir}/${manifest}")"

  if [[ -z "${sig}" ]]; then
    echo "FAIL: empty signature for ${platform_key}" >&2
    exit 1
  fi

  if [[ -z "${bundle_url}" ]]; then
    echo "FAIL: empty bundle url for ${platform_key}" >&2
    exit 1
  fi
  if [[ "${bundle_url}" != *"/releases/download/${VERSION}/"* ]]; then
    echo "FAIL: bundle URL does not reference immutable tag ${VERSION}: ${bundle_url}" >&2
    exit 1
  fi

  policy_channel="$(jq -r '.policy.channel // empty' "${tmpdir}/${manifest}")"
  policy_schema="$(jq -r '.policy.schema_version // empty' "${tmpdir}/${manifest}")"
  rollout="$(jq -r '.policy.rollout.percentage // empty' "${tmpdir}/${manifest}")"
  if [[ "${policy_channel}" != "${CHANNEL}" || "${policy_schema}" != "1" ]]; then
    echo "FAIL: invalid policy channel/schema in ${manifest}" >&2
    exit 1
  fi
  if ! node -e "const n=Number(process.argv[1]); process.exit(Number.isFinite(n)&&n>=0&&n<=100?0:1)" "${rollout}"; then
    echo "FAIL: invalid rollout percentage in ${manifest}: ${rollout}" >&2
    exit 1
  fi

  code="$(curl -s -o /dev/null -w '%{http_code}' -I "${bundle_url}")"
  if [[ "${code}" != "200" && "${code}" != "302" ]]; then
    echo "FAIL: bundle URL not reachable (${code}): ${bundle_url}" >&2
    exit 1
  fi

  echo "OK: version=${version} platform=${platform_key}"
done

echo "All manifests verified for channel=${CHANNEL}"
