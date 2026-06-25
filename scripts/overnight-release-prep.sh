#!/usr/bin/env bash
# Walk-away release gate: ensure Ollama, optional model pre-pull, caffeinate, release-prep.
# Invoked via: make overnight-release-prep
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

OLLAMA_HEALTH_URL="${OLLAMA_HEALTH_URL:-http://127.0.0.1:11434/api/tags}"
SESSION="${NJ_OVERNIGHT_SESSION:-nj-release-prep}"
STAMP="$(date +%Y%m%d-%H%M)"
LOG_DIR="${NJ_OVERNIGHT_LOG_DIR:-${HOME}}"
LOG_FILE="${NJ_OVERNIGHT_LOG:-${LOG_DIR}/nj-overnight-${STAMP}.log}"

# Make variables forwarded from `make overnight-release-prep VAR=1`.
FORWARD_VARS=(
  SKIP_LIVE
  SKIP_PARITY
  SKIP_BENCHMARK
  NO_FULL
  SKIP_EVERYTHING
  BENCHMARK_SUITE
  BENCHMARK_MODELS
  NO_PULL
  BENCHMARK_ALLOW_LARGE
  NO_RESTART_HUB
  VERBOSE
  STOP_ON_FAIL
  PULL
  NJ_OVERNIGHT_TARGET
  NEURAL_JUNKIE_HUB_URL
)

ollama_healthy() {
  curl -sf "${OLLAMA_HEALTH_URL}" >/dev/null 2>&1
}

require_ollama() {
  if ollama_healthy; then
    echo "OK: Ollama already running (${OLLAMA_HEALTH_URL})"
    return 0
  fi
  echo ">>> Ensuring Ollama is running..."
  chmod +x scripts/ensure-ollama.sh
  ./scripts/ensure-ollama.sh || true
  if ollama_healthy; then
    echo "OK: Ollama is up"
    return 0
  fi
  echo "FAIL: Ollama is required for overnight release prep." >&2
  echo "  Install: https://ollama.ai  — or run: make ensure-ollama" >&2
  echo "  Then re-run: make overnight-release-prep" >&2
  exit 1
}

build_make_args() {
  local args=()
  local v
  for v in "${FORWARD_VARS[@]}"; do
    if [[ -n "${!v:-}" ]]; then
      args+=("${v}=${!v}")
    fi
  done
  printf '%s\n' "${args[@]}"
}

build_tmux_inner_command() {
  local target="${NJ_OVERNIGHT_TARGET:-release-prep}"
  local inner="cd '${ROOT}' && export NJ_OVERNIGHT_INNER=1 NJ_OVERNIGHT_LOG='${LOG_FILE}'"
  local v
  for v in "${FORWARD_VARS[@]}" IN_TMUX; do
    if [[ -n "${!v:-}" ]]; then
      inner+=" ${v}='${!v}'"
    fi
  done
  inner+=" IN_TMUX=0 '${ROOT}/scripts/overnight-release-prep.sh'"
  printf '%s' "${inner}"
}

maybe_pre_pull_models() {
  if [[ "${PULL:-}" != "1" ]]; then
    return 0
  fi
  echo ">>> Pre-pulling benchmark models (PULL=1)..."
  local pull_args=()
  if [[ -n "${BENCHMARK_SUITE:-}" ]]; then
    pull_args+=(SUITE="${BENCHMARK_SUITE}")
  fi
  if [[ -n "${BENCHMARK_MODELS:-}" ]]; then
    pull_args+=(MODELS="${BENCHMARK_MODELS}")
  fi
  if [[ -n "${BENCHMARK_ALLOW_LARGE:-}" ]]; then
    pull_args+=(PULL_ALL=1)
  fi
  make pull-benchmark-models "${pull_args[@]}"
}

run_overnight() {
  local target="${NJ_OVERNIGHT_TARGET:-release-prep}"
  local -a make_args=()
  local line

  while IFS= read -r line; do
    [[ -n "${line}" ]] && make_args+=("${line}")
  done < <(build_make_args)

  case "${target}" in
    release-prep) ;;
    test-everything)
      make_args+=(CONTINUE=1)
      ;;
    test-everything-full)
      target="test-everything"
      make_args+=(FULL=1 CONTINUE=1)
      ;;
    *)
      echo "FAIL: unknown NJ_OVERNIGHT_TARGET='${target}' (use release-prep, test-everything, test-everything-full)" >&2
      exit 1
      ;;
  esac

  mkdir -p "$(dirname "${LOG_FILE}")"
  echo ">>> Overnight run started $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
  echo ">>> Target: make ${target}"
  echo ">>> Log: ${LOG_FILE}"
  echo ">>> Reports: ${ROOT}/docs/testing/"
  echo ""

  # -d display sleep, -i idle sleep, -m disk sleep, -s system sleep, -u user active
  caffeinate -dimsu make "${target}" "${make_args[@]}" 2>&1 | tee "${LOG_FILE}"
  local rc="${PIPESTATUS[0]}"
  echo ""
  if [[ "${rc}" -eq 0 ]]; then
    echo "=== Overnight run PASS ==="
  else
    echo "=== Overnight run FAIL (exit ${rc}) ===" >&2
  fi
  echo "Log: ${LOG_FILE}"
  echo "Review latest: ls -lt ${ROOT}/docs/testing/release-prep-*.md ${ROOT}/docs/testing/test-everything-*.md 2>/dev/null | head -3"
  return "${rc}"
}

# Outer launcher: detach into tmux unless already inside or IN_TMUX=0.
if [[ -z "${NJ_OVERNIGHT_INNER:-}" ]] && [[ "${IN_TMUX:-1}" != "0" ]] && [[ -z "${TMUX:-}" ]]; then
  chmod +x "${ROOT}/scripts/overnight-release-prep.sh"
  inner="$(build_tmux_inner_command)"
  if tmux has-session -t "${SESSION}" 2>/dev/null; then
    echo "FAIL: tmux session '${SESSION}' already exists." >&2
    echo "  attach: tmux attach -t ${SESSION}" >&2
    echo "  or:     tmux kill-session -t ${SESSION}" >&2
    exit 1
  fi
  tmux new-session -d -s "${SESSION}" "${inner}"
  echo "Started overnight release prep in tmux session: ${SESSION}"
  echo "  attach: tmux attach -t ${SESSION}"
  echo "  log:    tail -f ${LOG_FILE}"
  echo "  kill:   tmux kill-session -t ${SESSION}"
  exit 0
fi

require_ollama
maybe_pre_pull_models
run_overnight
