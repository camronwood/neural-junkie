#!/usr/bin/env bash
# Security regression checks for hub deployment posture (CI-safe; no mutations).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail=0

warn() { echo "WARN: $*" >&2; }
ok() { echo "OK: $*"; }
die() { echo "FAIL: $*" >&2; fail=1; }

# Defaults match production posture unless overridden in env.
AUTH_REQUIRED="${NEURAL_JUNKIE_AUTH_REQUIRED:-}"
LISTEN_ALL="${NEURAL_JUNKIE_LISTEN_ALL:-}"
HUB_TOKEN="${NEURAL_JUNKIE_HUB_TOKEN:-}"
RELAXED="${NEURAL_JUNKIE_RELAXED_LOCAL:-}"

if [[ "${LISTEN_ALL}" == "1" && -z "${HUB_TOKEN}" ]]; then
  if [[ "${RELAXED}" == "1" || "${NEURAL_JUNKIE_DEBUG:-}" == "1" ]]; then
    warn "NEURAL_JUNKIE_LISTEN_ALL=1 without hub token (allowed only with DEBUG or RELAXED_LOCAL)"
  else
    die "NEURAL_JUNKIE_LISTEN_ALL=1 requires NEURAL_JUNKIE_HUB_TOKEN"
  fi
else
  ok "listen-all posture OK"
fi

if [[ -n "${AUTH_REQUIRED}" && "${AUTH_REQUIRED}" != "1" ]]; then
  warn "unexpected NEURAL_JUNKIE_AUTH_REQUIRED=${AUTH_REQUIRED}"
fi

if [[ -f "${HOME}/.neural-junkie/bootstrap.token" ]]; then
  perms="$(stat -f '%OLp' "${HOME}/.neural-junkie/bootstrap.token" 2>/dev/null || stat -c '%a' "${HOME}/.neural-junkie/bootstrap.token" 2>/dev/null || echo '')"
  if [[ -n "${perms}" && "${perms}" != "600" ]]; then
    warn "bootstrap.token permissions ${perms} (expected 600)"
  else
    ok "bootstrap.token permissions"
  fi
fi

if command -v go >/dev/null 2>&1; then
  if go test ./cmd/server/... ./internal/hub/... -count=1 -run 'Auth|Session|Bootstrap|StrictLoopback|LocalOnly' >/dev/null 2>&1; then
    ok "Go auth regression tests"
  else
    die "Go auth regression tests failed (run: go test ./cmd/server/... ./internal/hub/... -run Auth)"
  fi
fi

if command -v python3 >/dev/null 2>&1; then
  if (cd scripts/lib && PYTHONPATH=.. python3 -m unittest hub_auth_test.py -q); then
    ok "hub_auth.py unit tests"
  else
    die "hub_auth.py unit tests failed"
  fi
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "Security preflight failed." >&2
  exit 1
fi
echo "Security preflight passed."
