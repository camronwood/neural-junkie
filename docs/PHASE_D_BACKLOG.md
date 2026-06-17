# Phase D backlog (post-stable)

Structured epics after IDE v4.1 and stable cut (Phase C). **Not blockers for `v1.2.0` stable.**

Sequence follows dependency order in [PLATFORM_ROADMAP.md](PLATFORM_ROADMAP.md).

## D1 — Agent WebSocket transport

| Item | Detail |
|------|--------|
| Goal | Replace HTTP polling in `cmd/agent` with persistent hub WebSocket |
| Closes | `standalone-agent-polling` in [KNOWN_ISSUES.md](KNOWN_ISSUES.md) |
| Depends on | Stable hub session/auth surface |
| Files | `cmd/agent/`, `cmd/server/` agent WS handlers, hub dispatch |

## D2 — Persistence and search

| Item | Detail |
|------|--------|
| Goal | Full-text search API over SQLite message archive |
| Optional | PostgreSQL backend for shared deployments |
| Also | Pagination for large channels |
| Files | `internal/hub/session_persist.go`, new search API handlers |

## D3 — Auth and multi-user

| Item | Detail |
|------|--------|
| Goal | JWT / API keys for automation; roles admin/member/viewer |
| Optional | SSO |
| Today | Session tokens + channel ACLs — [SECURITY.md](SECURITY.md) |
| Files | `cmd/server/auth_*`, hub ACL middleware |

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
