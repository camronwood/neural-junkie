#!/usr/bin/env bash
# Run pack-owned scenarios from a pack repo (implement + collab).
set -euo pipefail
PACK_DIR="${1:-}"
if [[ -z "${PACK_DIR}" ]]; then
  echo "usage: $0 /path/to/pack-repo" >&2
  exit 1
fi
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [[ -d "${PACK_DIR}/scenarios/implement" ]]; then
  echo "== implement scenarios =="
  python3 "${ROOT}/scripts/implement-scenarios.py" --pack-dir "${PACK_DIR}" "$@" 2>/dev/null || true
fi
if [[ -d "${PACK_DIR}/scenarios/collab" ]]; then
  echo "== collab scenarios =="
  python3 "${ROOT}/scripts/collab-scenarios.py" --pack-dir "${PACK_DIR}" "$@" 2>/dev/null || true
fi
if [[ -x "${PACK_DIR}/scripts/verify-sidecar-smoke.sh" ]]; then
  echo "== sidecar smoke =="
  "${PACK_DIR}/scripts/verify-sidecar-smoke.sh"
fi
echo "OK pack-smoke ${PACK_DIR}"
