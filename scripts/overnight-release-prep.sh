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
  NO_PULL
  NJ_OVERNIGHT_TARGET
  NJ_OVERNIGHT_KEEP_ALIVE
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
  if ((${#args[@]})); then
    printf '%s\n' "${args[@]}"
  fi
}

build_tmux_inner_command() {
  local inner="cd '${ROOT}' && export NJ_OVERNIGHT_INNER=1 NJ_OVERNIGHT_LOG='${LOG_FILE}'"
  local v
  for v in "${FORWARD_VARS[@]}"; do
    if [[ -n "${!v:-}" ]]; then
      inner+=" ${v}='${!v}'"
    fi
  done
  inner+=" IN_TMUX=0 && '${ROOT}/scripts/overnight-release-prep.sh'"
  printf '%s' "${inner}"
}

ensure_models_ready() {
  echo ">>> Ensuring Ollama models are installed, loaded, and smoke-tested..."
  local -a ready_args=(
    --warm
    --smoke
    --keep-alive "${NJ_OVERNIGHT_KEEP_ALIVE:-24h}"
    --suite "${BENCHMARK_SUITE:-quick}"
  )
  if [[ "${NO_PULL:-}" != "1" ]]; then
    ready_args=(--pull-missing "${ready_args[@]}")
  fi
  if [[ -n "${BENCHMARK_MODELS:-}" ]]; then
    ready_args+=(--benchmark-models "${BENCHMARK_MODELS}")
  fi
  if [[ "${SKIP_BENCHMARK:-}" == "1" ]]; then
    ready_args+=(--skip-benchmark)
  fi
  if [[ -n "${BENCHMARK_ALLOW_LARGE:-}" ]]; then
    ready_args+=(--allow-large-models)
  fi
  chmod +x "${ROOT}/scripts/ensure-ollama-models-ready.py"
  python3 "${ROOT}/scripts/ensure-ollama-models-ready.py" "${ready_args[@]}"
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
  {
    echo ">>> Overnight run started $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo ">>> Target: make ${target}"
    echo ">>> Log: ${LOG_FILE}"
    echo ">>> Reports: ${ROOT}/docs/testing/"
    echo ""
    require_ollama
    ensure_models_ready
    echo ""
    # -d display sleep, -i idle sleep, -m disk sleep, -s system sleep, -u user active
    if ((${#make_args[@]})); then
      caffeinate -dimsu make "${target}" "${make_args[@]}"
    else
      caffeinate -dimsu make "${target}"
    fi
  } 2>&1 | tee "${LOG_FILE}"
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
  if ! command -v tmux >/dev/null 2>&1; then
    echo "WARN: tmux not found — running in foreground (set IN_TMUX=0 to silence)" >&2
  elif tmux has-session -t "${SESSION}" 2>/dev/null; then
    echo "FAIL: tmux session '${SESSION}' already exists." >&2
    echo "  attach: tmux attach -t ${SESSION}" >&2
    echo "  or:     tmux kill-session -t ${SESSION}" >&2
    exit 1
  else
    if tmux new-session -d -s "${SESSION}" "${inner}" 2>/dev/null; then
      echo "Started overnight release prep in tmux session: ${SESSION}"
      echo "  attach: tmux attach -t ${SESSION}"
      echo "  log:    tail -f ${LOG_FILE}"
      echo "  kill:   tmux kill-session -t ${SESSION}"
      exit 0
    fi
    echo "WARN: tmux launch failed — running in foreground (set IN_TMUX=0 to silence)" >&2
  fi
fi

run_overnight
