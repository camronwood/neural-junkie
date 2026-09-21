#!/usr/bin/env bash
# Compute the next vMAJOR.MINOR.PATCH-rc.N tag from existing git tags.
# Usage:
#   ./scripts/auto-rc-next-tag.sh              # print next tag
#   ./scripts/auto-rc-next-tag.sh --check-delta # exit 2 if HEAD has no shippable delta vs last v*
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CHECK_DELTA=0
if [[ "${1:-}" == "--check-delta" ]]; then
  CHECK_DELTA=1
fi

git fetch --tags --quiet 2>/dev/null || true

latest="$(git tag -l 'v*' --sort=-v:refname | head -n 1 || true)"
if [[ -z "${latest}" ]]; then
  echo "v0.1.0-rc.1"
  exit 0
fi

base="${latest#v}"
base="${base%%-beta*}"
base="${base%%-rc*}"

if ! echo "${base}" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "FAIL: could not parse base version from tag ${latest}" >&2
  exit 1
fi

max_n=0
for t in $(git tag -l "v${base}-rc.*"); do
  n="${t##*-rc.}"
  if echo "${n}" | grep -Eq '^[0-9]+$'; then
    if [[ "${n}" -gt "${max_n}" ]]; then
      max_n="${n}"
    fi
  fi
done

next_n=$((max_n + 1))
next="v${base}-rc.${next_n}"

if [[ "${CHECK_DELTA}" -eq 1 ]]; then
  last_commit="$(git rev-list -n 1 "${latest}" 2>/dev/null || true)"
  head_commit="$(git rev-parse HEAD)"
  if [[ -n "${last_commit}" && "${last_commit}" == "${head_commit}" ]]; then
    echo "NO_DELTA last_tag=${latest} head=${head_commit}" >&2
    exit 2
  fi
  authors="$(git log --format='%ae' "${latest}..HEAD" 2>/dev/null || true)"
  if [[ -n "${authors}" ]]; then
    non_bot=0
    while IFS= read -r email; do
      [[ -z "${email}" ]] && continue
      case "${email}" in
        *github-actions*|*noreply.github.com*|*dependabot*)
          ;;
        *)
          non_bot=1
          ;;
      esac
    done <<EOF
${authors}
EOF
    if [[ "${non_bot}" -eq 0 ]]; then
      echo "NO_DELTA bot-only commits since ${latest}" >&2
      exit 2
    fi
  fi
fi

echo "${next}"
