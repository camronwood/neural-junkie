#!/usr/bin/env bash
# Start Ollama for local dev when not already healthy (make start-all / make gui).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HEALTH_URL="${OLLAMA_HEALTH_URL:-http://127.0.0.1:11434/api/tags}"
resolve_models_dir() {
  if [[ -n "${NJ_OLLAMA_MODELS:-}" ]]; then
    echo "${NJ_OLLAMA_MODELS}"
    return
  fi
  if [[ -n "${OLLAMA_MODELS:-}" ]]; then
    echo "${OLLAMA_MODELS}"
    return
  fi
  if command -v ollama >/dev/null 2>&1; then
    echo "${HOME}/.ollama/models"
    return
  fi
  echo "${HOME}/.neural-junkie/ollama-models"
}

MODELS_DIR="$(resolve_models_dir)"
LOG_FILE="${OLLAMA_LOG:-/tmp/neural-junkie-ollama.log}"

if curl -sf "${HEALTH_URL}" >/dev/null 2>&1; then
  echo "✅ Ollama already running (${HEALTH_URL})"
  exit 0
fi

mkdir -p "${MODELS_DIR}"

host_triple() {
  if command -v rustc >/dev/null 2>&1; then
    rustc -vV | awk '/host:/{print $2}'
    return
  fi
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"
  case "${os}-${arch}" in
    Darwin-arm64) echo "aarch64-apple-darwin" ;;
    Darwin-x86_64) echo "x86_64-apple-darwin" ;;
    Linux-x86_64) echo "x86_64-unknown-linux-gnu" ;;
    *) echo "" ;;
  esac
}

bundled_bin() {
  local triple dir
  triple="$(host_triple)"
  [[ -z "${triple}" ]] && return 1
  dir="${ROOT}/desktop/src-tauri/ollama/${triple}"
  if [[ -x "${dir}/bin/ollama" ]]; then
    echo "${dir}/bin/ollama"
    return 0
  fi
  if [[ -x "${dir}/ollama" ]]; then
    echo "${dir}/ollama"
    return 0
  fi
  return 1
}

if command -v ollama >/dev/null 2>&1; then
  echo "🦙 Starting system Ollama (ollama serve)…"
  nohup env OLLAMA_HOST=127.0.0.1:11434 ollama serve >>"${LOG_FILE}" 2>&1 &
elif bin="$(bundled_bin)"; then
  echo "🦙 Starting bundled Ollama from ${bin}…"
  nohup env OLLAMA_HOST=127.0.0.1:11434 OLLAMA_MODELS="${MODELS_DIR}" \
    "${bin}" serve >>"${LOG_FILE}" 2>&1 &
else
  echo "⚠️  Ollama is not running."
  echo "   Install: https://ollama.ai  — or run: make fetch-ollama"
  echo "   Then: ollama serve   (or re-run make start-all)"
  exit 0
fi

for _ in $(seq 1 30); do
  if curl -sf "${HEALTH_URL}" >/dev/null 2>&1; then
    echo "✅ Ollama is up (${HEALTH_URL})"
    exit 0
  fi
  sleep 1
done

echo "⚠️  Ollama did not become healthy within 30s. Log: ${LOG_FILE}"
exit 0
