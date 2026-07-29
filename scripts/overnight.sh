#!/usr/bin/env bash
# Walk-away overnight gate: clean reset → hub → preflight → release-prep (or fix-loop).
#
# Usage:
#   make overnight
#   make overnight NJ_OVERNIGHT_TARGET=release-prep-fix-loop MAX_ITER=2
#   tmux attach -t nj-overnight
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

HUB_URL="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
OLLAMA_HEALTH_URL="${OLLAMA_HEALTH_URL:-http://127.0.0.1:11434/api/tags}"
SESSION="${NJ_OVERNIGHT_SESSION:-nj-overnight}"
STAMP="$(date +%Y%m%d-%H%M)"
LOG_DIR="${NJ_OVERNIGHT_LOG_DIR:-${HOME}}"
LOG_FILE="${NJ_OVERNIGHT_LOG:-${LOG_DIR}/nj-overnight-${STAMP}.log}"

# Defaults tuned for unattended runs (override via env or make VAR=value).
: "${NJ_OVERNIGHT_TARGET:=release-prep}"
: "${BENCHMARK_SUITE:=release}"
: "${NO_PULL:=1}"

FORWARD_VARS=(
  SKIP_LIVE
  SKIP_PARITY
  SKIP_BENCHMARK
  NO_FULL
  SKIP_EVERYTHING
  BENCHMARK_SUITE
  BENCHMARK_MODELS
  NO_PULL
  PULL
  BENCHMARK_ALLOW_LARGE
  NO_RESTART_HUB
  VERBOSE
  STOP_ON_FAIL
  NJ_OVERNIGHT_KEEP_ALIVE
  NEURAL_JUNKIE_HUB_URL
  NJ_OVERNIGHT_TARGET
  MAX_ITER
  REPORT
  SKIP_RELEASE_PREP
  SKIP_AGENT
  SKIP_VERIFY
  DRY_RUN
  MODEL
  PREFER_SDK
  AGENT_TIMEOUT
  LAYER
  SKIP_GATE
  NO_COMMIT
  FIX_BRANCH
  BASE_BRANCH
  NO_WORKTREE
  USE_WORKTREE
  SCENARIO
  ALL
)

ollama_healthy() {
  curl -sf "${OLLAMA_HEALTH_URL}" >/dev/null 2>&1
}

run_regression_boot() {
  # shellcheck disable=SC1091
  source load-env.sh
  export NEURAL_JUNKIE_RATE_LIMIT=0
  export NEURAL_JUNKIE_HUB_URL="${HUB_URL}"
  python3 <<PY
import sys
from pathlib import Path
sys.path.insert(0, "scripts")
from lib.regression_boot import boot_regression_stack
if not boot_regression_stack(
    Path("${ROOT}"),
    "${HUB_URL}",
    label="overnight",
    clean=True,
    ready_smoke=True,
):
    raise SystemExit(1)
PY
  export NJ_BOOT_DONE=1
}

build_make_args() {
  local args=()
  local v
  for v in "${FORWARD_VARS[@]}"; do
    if [[ -n "${!v:-}" ]]; then
      args+=("${v}=${!v}")
    fi
  done
  if ((${#args[@]})); then
    printf '%s\n' "${args[@]}"
  fi
  # Gate runs after boot — skip duplicate boot in child make targets.
  printf '%s\n' "SKIP_BOOT=1"
}

build_tmux_inner_command() {
  local inner="cd '${ROOT}' && export NJ_OVERNIGHT_INNER=1 NJ_OVERNIGHT_LOG='${LOG_FILE}'"
  local v
  for v in "${FORWARD_VARS[@]}"; do
    if [[ -n "${!v:-}" ]]; then
      inner+=" ${v}='${!v}'"
    fi
  done
  inner+=" IN_TMUX=0 && '${ROOT}/scripts/overnight.sh'"
  printf '%s' "${inner}"
}

run_gate() {
  local target="${NJ_OVERNIGHT_TARGET}"
  local -a make_args=()
  local line

  while IFS= read -r line; do
    [[ -n "${line}" ]] && make_args+=("${line}")
  done < <(build_make_args)

  case "${target}" in
    release-prep) ;;
    release-prep-fix-loop)
      : "${MAX_ITER:=2}"
      : "${SKIP_BENCHMARK:=1}"
      : "${NO_FULL:=1}"
      : "${SKIP_PARITY:=1}"
      : "${NO_COMMIT:=0}"
      : "${AGENT_TIMEOUT:=18000}"
      ;;
    layer-fix-loop)
      : "${MAX_ITER:=2}"
      : "${NO_COMMIT:=0}"
      : "${AGENT_TIMEOUT:=18000}"
      if [[ -z "${LAYER:-}" ]]; then
        echo "FAIL: LAYER required for layer-fix-loop (make layer-list)" >&2
        exit 1
      fi
      ;;
    sut-loop)
      : "${MAX_ITER:=2}"
      : "${NO_COMMIT:=1}"
      : "${AGENT_TIMEOUT:=18000}"
      if [[ -z "${SCENARIO:-}" && -z "${ALL:-}" ]]; then
        echo "FAIL: SCENARIO=… or ALL=1 required for sut-loop (make sut-loop-list)" >&2
        exit 1
      fi
      ;;
    test-everything)
      make_args+=(CONTINUE=1)
      ;;
    test-everything-full)
      target="test-everything"
      make_args+=(FULL=1 CONTINUE=1)
      ;;
    *)
      echo "FAIL: unknown NJ_OVERNIGHT_TARGET='${target}'" >&2
      echo "  use: release-prep | release-prep-fix-loop | layer-fix-loop | sut-loop | test-everything | test-everything-full" >&2
      exit 1
      ;;
  esac

  echo ">>> Gate: make ${target}"
  if ((${#make_args[@]})); then
    caffeinate -dimsu make "${target}" "${make_args[@]}"
  else
    caffeinate -dimsu make "${target}"
  fi
}

run_overnight() {
  mkdir -p "$(dirname "${LOG_FILE}")"
  {
    echo "================================================================"
    echo "Neural Junkie overnight — $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo "Target:  ${NJ_OVERNIGHT_TARGET}"
    echo "Hub:     ${HUB_URL}"
    echo "Log:     ${LOG_FILE}"
    echo "Reports: ${ROOT}/docs/testing/"
    echo "================================================================"
    echo ""

    run_regression_boot
    echo ""
    echo ">>> Ensure Model Arena pack (post-boot; advisory)"
    python3 <<PY
import sys
sys.path.insert(0, "scripts")
from lib.arena_pack import ensure_model_arena_pack
ok, detail = ensure_model_arena_pack("${HUB_URL}")
print(detail)
if not ok:
    print("WARN: Model Arena pack not ready — release-prep will skip model-benchmark", file=sys.stderr)
raise SystemExit(0)
PY
    echo ""
    run_gate
  } 2>&1 | tee "${LOG_FILE}"
  local rc="${PIPESTATUS[0]}"
  echo ""
  if [[ "${rc}" -eq 0 ]]; then
    echo "=== Overnight PASS ==="
  else
    echo "=== Overnight FAIL (exit ${rc}) ===" >&2
  fi
  echo "Log: ${LOG_FILE}"
  echo "Review:"
  echo "  ls -lt ${ROOT}/docs/testing/release-prep-*.md 2>/dev/null | head -3"
  echo "  ls -lt ${ROOT}/docs/testing/release-prep-fix-loop-*.md 2>/dev/null | head -3"
  return "${rc}"
}

# Outer launcher: detach into tmux unless already inside or IN_TMUX=0.
if [[ -z "${NJ_OVERNIGHT_INNER:-}" ]] && [[ "${IN_TMUX:-1}" != "0" ]] && [[ -z "${TMUX:-}" ]]; then
  chmod +x "${ROOT}/scripts/overnight.sh"
  # Retire legacy session names if present.
  for legacy in nj-release-prep nj-release-prep-fix-loop; do
    tmux kill-session -t "${legacy}" 2>/dev/null || true
  done
  if tmux has-session -t "${SESSION}" 2>/dev/null; then
    echo "FAIL: tmux session '${SESSION}' already running." >&2
    echo "  attach: tmux attach -t ${SESSION}" >&2
    echo "  stop:   tmux kill-session -t ${SESSION}" >&2
    exit 1
  fi
  inner="$(build_tmux_inner_command)"
  if command -v tmux >/dev/null 2>&1; then
    if tmux new-session -d -s "${SESSION}" "${inner}"; then
      echo "Overnight run started in tmux: ${SESSION}"
      echo "  attach: tmux attach -t ${SESSION}"
      echo "  log:    tail -f ${LOG_FILE}"
      echo "  stop:   tmux kill-session -t ${SESSION}"
      exit 0
    fi
    echo "WARN: tmux launch failed — running in foreground" >&2
  else
    echo "WARN: tmux not found — running in foreground (install tmux for walk-away)" >&2
  fi
fi

run_overnight
