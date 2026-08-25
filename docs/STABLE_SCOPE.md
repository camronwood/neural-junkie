# Stable release scope (v1.0)

What Neural Junkie **v1.0 stable** promises vs what remains beta, roadmap, or out of scope.

**Target audience:** Solo developers and small teams running a **local-first** hub on their own machine.

See also: [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md), [KNOWN_ISSUES.md](KNOWN_ISSUES.md), [RELEASE_UPDATES.md](RELEASE_UPDATES.md).

---

## In scope for v1.0

| Area | v1.0 commitment |
|------|-----------------|
| **Desktop app** | Tauri + React — primary UX (chat, workspace, IDE layout, collaboration panel) |
| **Hub** | Single instance on loopback (`localhost:18765`); optional LAN bind with hub token |
| **Auth** | Session tokens + channel ACLs; optional strict mode (`NEURAL_JUNKIE_AUTH_REQUIRED=1`) |
| **History** | Bounded in-memory history; SQLite sidecar for messages + conversation memory; **durable channels** + **Export history** |
| **Agents** | Multi-agent chat, specialists, MCP tools, delegation, bounded collaboration |
| **Domain packs** | Official catalog + customer sideload zips (Pack dev studio) |
| **Integrations** | Slack bridge (local hub), Confluence, GitHub CLI, bundled/wizard Ollama |
| **Updates** | In-app updater — **stable** and **beta** channels |
| **macOS releases** | **Ad-hoc signed** CI builds; first launch may require Right-click → **Open** (Gatekeeper). **v1.0.1+** targets Developer ID + notarization when Apple Developer credentials are available. |

---

## Explicitly out of scope for v1.0

| Area | Status | Notes |
|------|--------|-------|
| Multi-tenant / shared hub SaaS | Out | Single-server local-first |
| Distributed / horizontal scale | Phase 3 | [PLATFORM_ROADMAP.md](PLATFORM_ROADMAP.md) |
| Per-route API keys & user roles | Shipped (v1.2.x) | `nj_…` keys + admin/member/viewer — see [SECURITY.md](SECURITY.md) |
| SSO / JWT for enterprise deployments | Phase 3 | Session + hub token + API keys today |
| Full in-app searchable archive | Phase 2+ | Export + find bar; optional search API later |
| Web UI parity with desktop | Limitation | Browser chat at `/` stays thin |
| IDE v4 (remote SSH, full Monaco LSP) | Shipped v1.2 | [IDE_V4.md](IDE_V4.md) |
| Agent WebSocket transport | Phase 3 | `cmd/agent` still polls |
| Notarized macOS at v1.0.0 | Deferred | Planned **v1.0.1** when Apple account is available — see `macos-adhoc-sign` in [KNOWN_ISSUES.md](KNOWN_ISSUES.md) |

---

## Known limitations (documented, not blockers)

These remain honest limits at v1.0 — see [known-issues.html](known-issues.html):

- Collaboration quality varies by local model; planning provider optional mitigation
- Slack bridge requires local hub running
- Single hub instance
- GitHub Release macOS builds: ad-hoc signed until v1.0.1 notarization
- Local `make gui` / dev builds: ad-hoc macOS sign only

---

## Channel strategy

| Channel | Tags | Users |
|---------|------|-------|
| **Beta** | `v1.0.0-beta.N` | Early adopters, soak testing |
| **Stable** | `v1.0.0`, `v1.1.0`, … | Default download after stable cut |

Switching beta → stable requires a **one-time manual install** of a stable build (updater channels do not cross).
