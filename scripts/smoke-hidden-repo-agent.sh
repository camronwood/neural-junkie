#!/usr/bin/env bash
# Smoke test: hidden consult-only repo agents on workspace add.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HUB="${NEURAL_JUNKIE_HUB_URL:-http://127.0.0.1:18765}"
FIXTURE="${ROOT}/scenarios/fixtures/minimal-repo"
LOG="/tmp/nj-hidden-repo-smoke.log"
BIN="/tmp/nj-server-smoke-$$"
SERVER_PID=""

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

say() { printf '\n==> %s\n' "$*"; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

if [[ ! -d "${FIXTURE}" ]]; then
  fail "fixture not found: ${FIXTURE}"
fi

if lsof -ti :18765 >/dev/null 2>&1; then
  say "Port 18765 in use — using existing hub at ${HUB}"
else
  say "Building server"
  (cd "${ROOT}" && go build -o "${BIN}" ./cmd/server)

  say "Starting hub (log: ${LOG})"
  if [[ -f "${ROOT}/load-env.sh" ]]; then
    set +u
    # shellcheck disable=SC1091
    source "${ROOT}/load-env.sh" || true
    set -u
  fi
  NEURAL_JUNKIE_RELAXED_LOCAL=1 NEURAL_JUNKIE_SLACK_DISABLED=1 \
    "${BIN}" -addr :18765 >"${LOG}" 2>&1 &
  SERVER_PID=$!

  for i in $(seq 1 60); do
    if curl -sf "${HUB}/api/ollama/status" >/dev/null 2>&1 || curl -sf "${HUB}/" >/dev/null 2>&1; then
      break
    fi
    if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
      tail -30 "${LOG}" >&2 || true
      fail "server exited during startup"
    fi
    sleep 0.5
  done
fi

say "List agents (user-facing)"
USER_AGENTS="$(curl -sf "${HUB}/api/agents")"
if echo "${USER_AGENTS}" | grep -q '__index:'; then
  fail "consult-only agent leaked into /api/agents"
fi
echo "OK — no __index agents in user-facing list"

say "Add workspace (minimal-repo fixture)"
WS_JSON="$(curl -sf -X POST "${HUB}/api/workspaces" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"smoke-minimal\",\"path\":\"${FIXTURE}\"}")"
WS_PATH="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["path"])' <<<"${WS_JSON}")"
echo "Workspace path: ${WS_PATH}"

sleep 2

say "List agents (include_consult_only=true)"
ALL_AGENTS="$(curl -sf "${HUB}/api/agents?include_consult_only=true")"
export ALL_AGENTS
python3 <<'PY'
import json, os
agents = json.loads(os.environ["ALL_AGENTS"])
hidden = [a for a in agents if a.get("consult_only") and a.get("type") == "repo"]
if not hidden:
    raise SystemExit("FAIL: no consult_only repo agent found")
for a in hidden:
    print(f"  hidden agent: {a['name']} status={a.get('indexing_status')} progress={a.get('index_progress')} path={a.get('repository_path')}")
PY

say "Wait for repo index (up to 90s)"
READY=0
for i in $(seq 1 90); do
  ALL_AGENTS="$(curl -sf "${HUB}/api/agents?include_consult_only=true")"
  STATUS="$(export ALL_AGENTS; python3 <<'PY'
import json, os
agents = json.loads(os.environ["ALL_AGENTS"])
h = [a for a in agents if a.get("consult_only")]
print(h[0].get("indexing_status", "?") if h else "missing")
PY
)"
  if [[ "${STATUS}" == "ready" ]]; then
    READY=1
    echo "Index ready after ${i}s"
    break
  fi
  sleep 1
done
[[ "${READY}" -eq 1 ]] || fail "index did not reach ready (last status: ${STATUS})"

say "Codeindex status"
curl -sf "${HUB}/api/repo/index/status?repo_path=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "${WS_PATH}")" | python3 -m json.tool

say "Hub log lines (hidden-repo)"
if [[ -f "${LOG}" ]]; then
  grep -E '\[hidden-repo\]|consult-only' "${LOG}" | tail -5 || true
fi

say "Smoke test passed"
