#!/usr/bin/env bash
# User-trust automated ship bar (replaces interactive Gate 5 click-through).
#
# Usage:
#   ./scripts/ship-bar.sh              # full local bar (live gates need hub/models)
#   ./scripts/ship-bar.sh --package    # artifact / version / updater helpers only
#   ./scripts/ship-bar.sh --quick      # test-all + version consistency only
#
# Exit 0 only when all requested steps PASS.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

MODE="full"
for arg in "$@"; do
  case "${arg}" in
    --package) MODE="package" ;;
    --quick) MODE="quick" ;;
    --full) MODE="full" ;;
    -h|--help)
      sed -n '1,20p' "$0"
      exit 0
      ;;
  esac
done

chmod +x \
  scripts/verify-desktop-version-consistency.sh \
  scripts/verify-updater-manifest.sh \
  scripts/ship-bar-package.sh \
  2>/dev/null || true

fail=0
run_step() {
  local name="$1"
  shift
  echo ""
  echo "=== ship-bar: ${name} ==="
  if "$@"; then
    echo "PASS ${name}"
  else
    echo "FAIL ${name}" >&2
    fail=1
  fi
}

case "${MODE}" in
  package)
    run_step "package-verify" ./scripts/ship-bar-package.sh
    ;;
  quick)
    run_step "test-all" make test-all
    run_step "desktop-version" ./scripts/verify-desktop-version-consistency.sh
    ;;
  full)
    run_step "test-all" make test-all
    run_step "desktop-version" ./scripts/verify-desktop-version-consistency.sh
    run_step "user-flows" make layer-gate LAYER=user-flows
    run_step "layer-climb" make layer-climb CONTINUE=1
    run_step "desktop-e2e" make desktop-e2e
    # Tier B soak (parity last — hub restarts)
    run_step "chat-full" make layer-gate LAYER=chat-full
    run_step "collab" make layer-gate LAYER=collab
    run_step "collab-full" make layer-gate LAYER=collab-full
    run_step "parity" make layer-gate LAYER=parity
    run_step "package-verify" ./scripts/ship-bar-package.sh
    ;;
esac

if [[ "${fail}" -ne 0 ]]; then
  echo ""
  echo "ship-bar: FAILED" >&2
  exit 1
fi
echo ""
echo "ship-bar: PASS (${MODE})"
exit 0
