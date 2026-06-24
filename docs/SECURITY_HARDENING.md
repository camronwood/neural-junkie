# Hub Security Hardening

This document describes the auth hardening rollout (Phases 1–6) and how to migrate existing installs.

See also [SECURITY.md](SECURITY.md) for the full threat model and secrets posture.

## Target model

Three layers:

1. **Network** — default `127.0.0.1`; `NEURAL_JUNKIE_LISTEN_ALL=1` requires `NEURAL_JUNKIE_HUB_TOKEN` (server refuses start otherwise unless `NEURAL_JUNKIE_DEBUG=1` or `NEURAL_JUNKIE_RELAXED_LOCAL=1`).
2. **Identity** — mutations require `X-NJ-Session` or `Authorization: Bearer nj_…`; no anonymous synthetic **admin**.
3. **Authorization** — roles (`admin` / `member` / `viewer`) + channel ACL when a session or API key is present.

## Environment matrix

| Variable | Default (dev) | Release-prep / CI | LAN / shared host |
|----------|---------------|-------------------|-------------------|
| `NEURAL_JUNKIE_AUTH_REQUIRED` | unset | `1` | `1` |
| `NEURAL_JUNKIE_RELAXED_LOCAL` | `1` via `make run-hub` | unset | unset |
| `NEURAL_JUNKIE_HUB_TOKEN` | unset (loopback) | optional | **required** with `LISTEN_ALL` |
| `NEURAL_JUNKIE_BOOTSTRAP_TOKEN` | file at `~/.neural-junkie/bootstrap.token` | env or file | env or file |
| `NEURAL_JUNKIE_API_KEY` | optional | `~/.neural-junkie/automation.key` | per automation user |

### Solo-dev escape hatch

```bash
export NEURAL_JUNKIE_RELAXED_LOCAL=1   # loopback-only synthetic member (not admin)
```

Used by `make run-hub` / `make server` for fast iteration. **Do not** set on shared machines.

### Bootstrap token (admin session minting)

On first hub start, if `~/.neural-junkie/bootstrap.token` is missing, the hub creates one (mode `0600`) and logs the path (not the secret).

Mint an admin session:

```bash
curl -sS -X POST http://127.0.0.1:18765/api/auth/session \
  -H 'Content-Type: application/json' \
  -H "X-NJ-Bootstrap: $(cat ~/.neural-junkie/bootstrap.token)" \
  -d '{"username":"admin","role":"admin"}'
```

Client-supplied `role: admin` is **ignored** unless bootstrap or an existing admin session/API key is presented.

## Breaking changes

| Before | After |
|--------|-------|
| Loopback requests without session got synthetic **admin** | Require session/API key, or `RELAXED_LOCAL=1` synthetic **member** on loopback only |
| `POST /api/auth/session` with `"role":"admin"` always worked | Admin role capped unless bootstrap or admin caller |
| Scripts called hub without auth headers | `scripts/lib/hub_auth.py` + `NEURAL_JUNKIE_API_KEY` or session |
| Debug routes when `DEBUG=1` from any client | Debug routes always `localOnly` |
| WebSocket subscribe any channel | Channel ACL checked before subscribe |
| Thread reply `from` spoofing | Ignored unless hub token configured (matches `/api/send`) |

## Script and automation auth

`scripts/lib/hub_auth.py`:

- `hub_auth_headers()` — API key, hub token, or cached session
- `ensure_hub_session(base)` — `POST /api/auth/session` once per process
- `ensure_automation_api_key(base)` — creates member key via bootstrap (used by `collab-preflight`)

Release-prep sets `NEURAL_JUNKIE_AUTH_REQUIRED=1` and does **not** set `RELAXED_LOCAL`.

Standalone agents:

```bash
export NEURAL_JUNKIE_AUTH_REQUIRED=1
go run ./cmd/agent --api-key "$(cat ~/.neural-junkie/automation.key)" ...
```

## Desktop

- `chatAPI.hubFetch` clears session and emits `nj-hub-unauthorized` on **401** → App returns to login.
- Settings → Security shows bootstrap, relaxed-local, listen-all, and hub-token status.

## WebSocket

- Channel ACL enforced in `/ws` before subscribe.
- Browser clients cannot set `X-NJ-Hub-Token` on WebSocket upgrade; use `?hub_token=` when LAN-exposed (see `desktop/src/api/chatAPI/wsUrl.ts`).

## Verification

```bash
go test ./cmd/server/... ./internal/hub/...
cd scripts/lib && PYTHONPATH=.. python3 -m unittest hub_auth_test.py
./scripts/security-preflight.sh
```

Manual:

```bash
# 401 without session (strict mode)
NEURAL_JUNKIE_AUTH_REQUIRED=1 curl -sS -o /dev/null -w '%{http_code}\n' \
  -X POST http://127.0.0.1:18765/api/send \
  -H 'Content-Type: application/json' \
  -d '{"channel":"general","content":"hi","type":"chat"}'
```

## Follow-up (documented, not blocking)

- Default tool approval for destructive commands
- `NEURAL_JUNKIE_DENY_SHELL=1` to block `run_command` except allowlist
- Pack-gated MCP tools remain OS-user privileged — use `NEURAL_JUNKIE_AUTO_APPROVE_EDITS=0` on shared hosts
