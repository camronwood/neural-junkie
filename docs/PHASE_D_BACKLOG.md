# Phase D backlog (post-stable)

Structured epics after IDE v4.1 and stable cut (Phase C). **Not blockers for `v1.2.0` stable.**

Sequence follows dependency order in [PLATFORM_ROADMAP.md](PLATFORM_ROADMAP.md).

## D1 — Agent WebSocket transport ✅ (shipped)

| Item | Detail |
|------|--------|
| Goal | Replace HTTP polling in `cmd/agent` with persistent hub WebSocket |
| Status | **`GET /api/agents/ws`**, standalone agent WS default; `--poll` fallback |
| Files | `cmd/agent/`, `cmd/server/agent_ws_handlers.go`, `internal/hub/wsclient/` |

## D2 — Persistence and search ✅ (shipped)

| Item | Detail |
|------|--------|
| Goal | Full-text search API over SQLite message archive |
| Status | FTS5 `messages_fts`, `GET /api/messages/search`, cursor history pagination, desktop find bar |
| Optional | PostgreSQL backend for shared deployments (deferred) |
| Files | `internal/store/sqlite/`, `cmd/server/messages_search_handlers.go` |

## D3 — Auth and multi-user ✅ (shipped MVP)

| Item | Detail |
|------|--------|
| Goal | API keys for automation; roles admin/member/viewer |
| Status | `nj_…` keys in `~/.neural-junkie/auth.db`, Settings → Security UI, `--api-key` on `cmd/agent` |
| Optional | SSO, JWT (deferred) |
| Files | `internal/hub/authstore/`, `cmd/server/api_keys_handlers.go`, `cmd/server/auth_handlers.go` |

## D4 — Distributed hub

| Item | Detail |
|------|--------|
| Goal | Multi-instance routing (Redis Pub/Sub or equivalent) |
| Closes | `single-hub` limitation |
| Out of scope | Multi-tenant SaaS, horizontal web UI parity |

## Explicitly deferred

- Marketing rewrites for historical beta ads (`docs/marketing/BETA*.md`)
- Dropping Windows EXE from release artifacts
- macOS notarization (track as v1.2.1 when Apple creds ready)

## Suggested tags after stable

| Tag | Contents |
|-----|----------|
| `v1.2.0` or `v1.3.0` | Phase C stable cut |
| `v1.2.1` | macOS notarization (optional) |
| `v1.3.0-beta.N` | Phase D soak line |
