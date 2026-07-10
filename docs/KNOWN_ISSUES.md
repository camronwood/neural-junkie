# Known limitations and issues

**Last updated:** 2026-07-10 · **Current beta:** [v1.2.0-beta.5](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.5) ([latest](https://github.com/camronwood/neural-junkie/releases/latest)) · **Stable path:** [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md)

Living list of what we know is wrong, flaky, or intentionally limited. **Remove an entry when it is fixed** (and note the fix in [CHANGELOG.md](CHANGELOG.md) / [release-notes.html](release-notes.html)).

**Public page:** [known-issues.html](known-issues.html) — keep in sync with this file when you add or close items.

**Report new bugs:** [GitHub Issues](https://github.com/camronwood/neural-junkie/issues) — for collaboration, include the phase (planning, review, workspace gate, execution, files).

---

## Collaboration

| ID | Status | Summary |
|----|--------|---------|
| `collab-chat-not-disk` | **Limitation** | Chat markdown does **not** write to disk. Execution needs `[FILE_CHANGE]` proposals and your approval in **Pending changes**. `TASK_STATUS: completed` alone does not create files. |
| `collab-model-variance` | **Limitation** | Local models (Ollama) vary in discussion quality, silence, and timeouts; hub enforces phase caps and fallbacks but cannot guarantee plan shape. **Mitigation:** Settings → **Collaboration planning provider** (optional cloud/larger model for planning turns). Implementation sessions: use **reliable tool model** / optional **reliable provider** (repair round 2+) in Settings → Implementation sessions; outcome card shows failure reason. |
| `collab-smart-routing-scope` | **Limitation** | Smart routing applies to **execution tasks only**, not normal channel chat. Planning can use optional `planning_provider_id` in Settings (separate from smart routing). |

**Workarounds:** Upgrade hub after each beta; run `make collab-preflight` before scenarios; use `make debug-collab` / discussion diagnosis in [COLLABORATION.md](COLLABORATION.md). For zero-task plans, revise plan or re-run with fewer agents. Inspect per-turn **routing trace** on collab messages when execution picks an unexpected model.

---

## Chat and context

| ID | Status | Summary |
|----|--------|---------|
| `hub-history-bounded` | **Limitation** | Per-channel history is capped (5000 messages) and age-pruned after 24h unless marked **durable**. SQLite sidecar + **Export history** in channel info; not a full in-app search archive. |

---

## Integrations

| ID | Status | Summary |
|----|--------|---------|
| `slack-bridge-local` | **Limitation** | Slack bridge runs **in-process** on the local hub — no public URL required (Socket Mode), but the hub must be running. Use **Settings → Integrations → Slack** diagnostics when OAuth or routing fails. |
| `confluence-setup` | **Limitation** | Confluence agents need Cloud credentials and indexing time; search quality depends on space size and token limits. |
| `room-chat-lan` | **Limitation** | **Room chat** pack requires guests on the **same LAN** as the host hub. Corporate Wi‑Fi client isolation may block joins; host must enable **listen on LAN** and configure a hub token. |

---

## Desktop, IDE, and packs

| ID | Status | Summary |
|----|--------|---------|
| `web-ui-thin` | **Limitation** | Browser hub UI at `/` is a **lightweight chat client** — no full workspace, palette, or file-approval UX. Use the **Tauri desktop** for production work. |
| `git-dev-pack` | **Limitation** | In-app Git operations require the **Software development** pack, `git` on PATH, and a git workspace. |
| `macos-adhoc-sign` | **Limitation** | GitHub Release macOS builds are **ad-hoc signed** until Apple Developer credentials are available. First launch may require **Right-click → Open**. Planned fix: **v1.2.1** notarized builds. |

_Removed in v4.1 (beta.2+): `ide-v4-remote-lsp`, `ide-v4-remote-collab` — see [IDE_V4.md](IDE_V4.md)._

---

## Hub, agents, and providers

| ID | Status | Summary |
|----|--------|---------|
| `single-hub` | **Limitation** | **Single-server** deployment — no horizontal scale or multi-region hub. |
| `react-tools-allowlist` | **Limitation** | **ReAct MCP** on non-native tool models is allowlist-based (e.g. `gemma3:12b` first). Other tags fall back to native tools or Qwen swap. |

---

## How we track quality

- **Deterministic:** `make test-collab-plan`, `make collab-smoke`, `go test ./...`
- **Live collab:** `make collab-scenario SCENARIO=…` — matrix in [testing/collab-matrix.tsv](testing/collab-matrix.tsv) (21/21 PASS)
- **Chat regression:** `make chat-scenarios-regression` — workspace visibility, closure, echo (context v2)
- **Implement parity:** `make implement-scenarios`, `make test-parity-stable` (3× with hub restart)
- **Release engineering:** `make layer-gate`, `make layer-fix-loop`, `make test-growth-loop` — artifacts in [docs/testing/](testing/)

When a row in [collab-matrix.tsv](testing/collab-matrix.tsv) flips to **PASS**, remove the matching **Active** / **Investigating** item above and from [known-issues.html](known-issues.html).
