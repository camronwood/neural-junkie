# Known limitations and issues

**Last updated:** 2026-06-06 · **Current beta:** [v1.0.0-beta.24](https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.24)

Living list of what we know is wrong, flaky, or intentionally limited. **Remove an entry when it is fixed** (and note the fix in [CHANGELOG.md](CHANGELOG.md) / [release-notes.html](release-notes.html)).

**Public page:** [known-issues.html](known-issues.html) — keep in sync with this file when you add or close items.

**Report new bugs:** [GitHub Issues](https://github.com/camronwood/neural-junkie/issues) — for collaboration, include the phase (planning, review, workspace gate, execution, files).

---

## Collaboration

| ID | Status | Summary |
|----|--------|---------|
| `collab-plan-zero-tasks` | **Active** | Some live scenarios end planning with **zero parsed tasks** (`assert_plan: tasks=0`) — e.g. `plan-dependency-prose-regression`, `plan-distinct-deliverables-same-agent`, `plan-phoenix-combined-regression`. See [collab-sweep-2026-06-02.md](testing/collab-sweep-2026-06-02.md). |
| `collab-deliverable-size` | **Active** | `execution-no-stack-commands` can fail when `findings.md` exists but is **too small** for harness assertions. |
| `collab-solo-parity` | **Investigating** | `solo-vs-collab-parity` — solo leg did not land file on disk before timeout in a spot check; full serial run pending. |
| `collab-phoenix-planning` | **Investigating** | `resource-api-schema-planning` not completed in last batch sweep (aborted mid-run). |
| `collab-chat-not-disk` | **Limitation** | Chat markdown does **not** write to disk. Execution needs `[FILE_CHANGE]` proposals and your approval in **Pending changes**. |
| `collab-workspace-gate` | **Limitation** | After **Approve plan**, assignees are **not** dispatched until you **Continue** / `/ack-collab-workspace`. Easy to mistake for a hang. |
| `collab-model-variance` | **Limitation** | Local models (Ollama) vary in discussion quality, silence, and timeouts; hub enforces phase caps and fallbacks but cannot guarantee plan shape. |
| `collab-smart-routing-scope` | **Limitation** | Smart routing applies to **execution tasks only**, not planning or normal channel chat. |

**Workarounds:** Upgrade hub after each beta; run `make collab-preflight` before scenarios; use `make debug-collab` / discussion diagnosis in [COLLABORATION.md](COLLABORATION.md). For zero-task plans, revise plan or re-run with fewer agents.

---

## Chat and context

| ID | Status | Summary |
|----|--------|---------|
| `context-v2-edge-cases` | **Active** | Context model v2 (Chat/Code mode, workspace visibility) is new in beta.21 — edge cases may still slip through; report bad “can you see my workspace?” answers. |
| `hub-history-bounded` | **Limitation** | Message history is **bounded and pruned** per channel; not a full durable archive. Session metadata restores from `last-session.json`. |

---

## Integrations

| ID | Status | Summary |
|----|--------|---------|
| `slack-bridge-local` | **Limitation** | Slack bridge runs **in-process** on the local hub — no public URL required (Socket Mode), but the hub must be running. |
| `confluence-setup` | **Limitation** | Confluence agents need Cloud credentials and indexing time; search quality depends on space size and token limits. |

---

## Desktop, IDE, and packs

| ID | Status | Summary |
|----|--------|---------|
| `web-ui-thin` | **Limitation** | Browser hub UI at `/` is a **lightweight chat client** — no full workspace, palette, or file-approval UX. Use the **Tauri desktop** for production work. |
| `git-dev-pack` | **Limitation** | In-app Git operations require the **Software development** pack, `git` on PATH, and a git workspace. |
| `ide-v3-beta` | **Limitation** | IDE v2/v3 layout, diagnostics, and Ask/Agent routing are **beta** — see [IDE_V2.md](IDE_V2.md) / [IDE_V3.md](IDE_V3.md). |
| `implement-deterministic-fallback` | **Active** | Live implement scenarios still rely on deterministic tailwind/App repairs when local 7B/14B emits prose-only replies; tracked via hub logs (`deterministic_impl_fallback`, `app_theme_repair`). |
| `parity-stable-hub-oom` | **Active** | `make test-parity-stable` back-to-back sweeps can **OOM-kill** the regression hub on memory-constrained hosts; restart hub between sweeps (see `docs/testing/parity-stable-*.log`). |
| `pack-layout-first` | **Limitation** | With multiple domain packs enabled, the **first pack turned on** sets IDE vs team layout. |

---

## Hub, agents, and providers

| ID | Status | Summary |
|----|--------|---------|
| `single-hub` | **Limitation** | **Single-server** deployment — no horizontal scale or multi-region hub. |
| `standalone-agent-polling` | **Limitation** | `cmd/agent` processes use **HTTP polling**; in-process hub agents get push delivery (lower latency). |
| `pause-not-abort` | **Limitation** | `/pause-agent` marks an agent paused in the roster; it does **not** abort an in-flight LLM call. |
| `lmstudio-tools` | **Limitation** | MCP tool calling is strongest on **Ollama** (selected flows) and **Claude**; LM Studio / generic OpenAI-compat tool use is limited. |
| `ollama-model-pull` | **Limitation** | Installers bundle the Ollama **runtime** only; models are pulled on first use (one-time download, can be several GB). |
| `macos-notarized` | **Limitation** | macOS builds are **ad-hoc signed**, not notarized — use **Right-click → Open** if Gatekeeper blocks the first launch. |

---

## How we track quality

- **Deterministic:** `make test-collab-plan`, `make collab-smoke`, `go test ./...`
- **Live scenarios:** `make collab-scenario SCENARIO=…` — matrix in [testing/collab-matrix.tsv](testing/collab-matrix.tsv)
- **Chat scenarios:** `make chat-scenarios-regression` (beta.21+)

When a row in [collab-matrix.tsv](testing/collab-matrix.tsv) flips to **PASS**, remove the matching **Active** / **Investigating** item above and from [known-issues.html](known-issues.html).
