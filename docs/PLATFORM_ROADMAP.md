# Platform roadmap (post-v1.2 stable)

Epic backlog for multi-user and ops-grade deployment. **Phase D** — after IDE v4.1 stable cut. See [PHASE_D_BACKLOG.md](PHASE_D_BACKLOG.md) for ticket-style breakdown.

Last updated: June 2026

---

## Phase 3 — Platform / enterprise

### Authentication and authorization

**Today:** Session tokens (`POST /api/auth/session`), channel ACLs, optional `NEURAL_JUNKIE_AUTH_REQUIRED=1`, hub token for non-loopback — [SECURITY.md](SECURITY.md).

**Phase 3:**

- JWT / API-key auth for automation and service accounts
- User roles: admin, member, viewer
- Agent registration approval
- Identity beyond username slug (SSO optional)

### Agent transport

**Today:** In-process hub agents (push); standalone `cmd/agent` uses HTTP polling.

**Phase 3:** WebSocket agent connections — lower latency, closes `standalone-agent-polling` limitation.

### Persistence and search

**Today (partial — shipped):**

- SQLite message store: `~/.neural-junkie/messages.db`
- SQLite conversation memory: `~/.neural-junkie/memory.db`
- Durable channels + channel export

**Phase 3:**

- Full-text search API across archived history
- Optional PostgreSQL backend for shared / large deployments
- Pagination for very large channels

### Distributed hub

**Today:** Single-server, in-memory routing with optional SQLite sidecars.

**Phase 3:**

- Redis Pub/Sub or equivalent for multi-instance message routing
- Load balancing and shared state
- Closes `single-hub` limitation

---

## IDE v4 (shipped v1.2)

Full LSP, remote SSH workspaces, dev containers — [IDE_V4.md](IDE_V4.md).

---

## Related docs

- [FUTURE_ENHANCEMENTS.md](FUTURE_ENHANCEMENTS.md) — full idea backlog
- [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md) — v1.0 cut gates
- [RELEASE_UPDATES.md](RELEASE_UPDATES.md) — beta vs stable updater channels
