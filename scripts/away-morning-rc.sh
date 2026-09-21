#!/usr/bin/env bash
# Cut morning RC tag on current HEAD when main moved since last v* tag.
# Usage:
#   ./scripts/away-morning-rc.sh              # tag + push if delta
#   ./scripts/away-morning-rc.sh --dry-run    # print next tag only
#   MORNING_RC_FORCE=1 ./scripts/away-morning-rc.sh  # tag even with no delta
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

DRY_RUN=0
FORCE="${MORNING_RC_FORCE:-0}"
STATUS_DIR="${ROOT}/docs/testing"
STAMP="$(date +%Y-%m-%d)"
STATUS_FILE="${STATUS_DIR}/away-morning-${STAMP}.md"

for arg in "$@"; do
  case "${arg}" in
    --dry-run) DRY_RUN=1 ;;
    --force) FORCE=1 ;;
  esac
done

chmod +x "${ROOT}/scripts/auto-rc-next-tag.sh"

write_status() {
  mkdir -p "${STATUS_DIR}"
  {
    echo "# Away morning status — ${STAMP}"
    echo
    echo "- Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo "- HEAD: \`$(git rev-parse --short HEAD)\`"
    echo "- Branch: \`$(git rev-parse --abbrev-ref HEAD)\`"
    for line in "$@"; do
      echo "- ${line}"
    done
  } >"${STATUS_FILE}"
  echo "Wrote ${STATUS_FILE}"
}

if [[ "${FORCE}" != "1" ]]; then
  if ! next="$("${ROOT}/scripts/auto-rc-next-tag.sh" --check-delta)"; then
    rc=$?
    if [[ "${rc}" -eq 2 ]]; then
      write_status "No morning RC — main has no shippable delta since last release tag." "Next would be: \`$("${ROOT}/scripts/auto-rc-next-tag.sh")\`"
      exit 0
    fi
    exit "${rc}"
  fi
else
  next="$("${ROOT}/scripts/auto-rc-next-tag.sh")"
fi

echo "Next RC tag: ${next}"

if [[ "${DRY_RUN}" -eq 1 ]]; then
  write_status "Dry-run only — would tag \`${next}\`."
  echo "${next}"
  exit 0
fi

git tag -a "${next}" -m "Morning RC ${next}"
git push origin "refs/tags/${next}"
write_status "Tagged and pushed \`${next}\`." "Release workflow should build installers for 07:00 CT smoke."
echo "Pushed ${next}"
