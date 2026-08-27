# Known limitations and issues

**Last updated:** 2026-08-25 · **Current beta:** [v1.2.0-beta.24](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.24) ([latest](https://github.com/camronwood/neural-junkie/releases/latest)) · **Stable path:** [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md)

Living list of what we know is wrong, flaky, or intentionally limited. **Remove an entry when it is fixed** (and note the fix in [CHANGELOG.md](CHANGELOG.md) / [release-notes.html](release-notes.html)).

**Public page:** [known-issues.html](known-issues.html) — keep in sync with this file when you add or close items.

**GitHub tracker:** [camronwood/neural-junkie issues](https://github.com/camronwood/neural-junkie/issues) — each row below links to a tracked issue. Issue map: [github-issues-map.json](testing/github-issues-map.json).

**Report new bugs:** [GitHub Issues](https://github.com/camronwood/neural-junkie/issues/new/choose) — for collaboration, include the phase (planning, review, workspace gate, execution, files).

---

## Release blockers

| ID | Status | GitHub | Summary |
|----|--------|--------|---------|
| `blocker-platform-smoke` | **Release blocker** | [#18](https://github.com/camronwood/neural-junkie/issues/18) | Gate 5 platform smoke pending operator sign-off (macOS arm64 + Windows or Linux) on **v1.2.0-beta.24** installer. Checklist: [stable-platform-smoke.md](testing/stable-platform-smoke.md), matrix: [platform-smoke-beta24.md](testing/platform-smoke-beta24.md). |

### Cleared for cut (tracker hygiene 2026-07-20)

| ID | Status | GitHub | Summary |
|----|--------|--------|---------|
| `blocker-collab-soak` | **Cleared** | [#16](https://github.com/camronwood/neural-junkie/issues/16) | `LAYER=collab-full` **PASS** on 2026-07-16, [layer-climb-2026-07-19-1809](testing/layer-climb-2026-07-19-1809.md), and `collab-scenarios-all` in [test-everything-2026-07-31-0244.md](testing/test-everything-2026-07-31-0244.md) (Ollama regression roster). Residual edge flakes tracked under #20/#21, not as a cut blocker. |
| `blocker-parity-soak` | **Cleared** | [#17](https://github.com/camronwood/neural-junkie/issues/17) | `test-parity-stable-restart` **PASS** on climb 2026-07-19, [parity-stable-restart-2026-07-20-0255.log](testing/parity-stable-restart-2026-07-20-0255.log), and again in [release-prep-2026-07-31-0244.md](testing/release-prep-2026-07-31-0244.md) (3× restart, min 14/sweep). |
| `blocker-d5-deferred` | **Deferred (post-cut)** | [#19](https://github.com/camronwood/neural-junkie/issues/19) | D5 specialist simplification is Phase D backlog — **not** a v1.2.0 stable cut gate ([PHASE_D_BACKLOG.md](PHASE_D_BACKLOG.md)). |

---

## Active bugs and investigations

| ID | Status | GitHub | Summary |
|----|--------|--------|---------|
| `layer-gate-make-verbose-macos` | **Fixed** | [#22](https://github.com/camronwood/neural-junkie/issues/22) | Harness now passes `VERBOSE=1` as env (not `--verbose` to BSD make). Latest collab-core climb ran ~23m (scenarios executed). Keep closed unless regression returns. |
| `collab-agent-silence` | **Mitigated (closed)** | [#20](https://github.com/camronwood/neural-junkie/issues/20) | Closed 2026-08-25 — P0 handoff + watchdog; core matrix green ×2; `document-findings-execution` PASS. Evidence: [p4-collab-proof-2026-08-25.md](testing/p4-collab-proof-2026-08-25.md). Reopen if regression returns. |
| `collab-generation-error` | **Mitigated (Ollama); cloud deferred** | [#21](https://github.com/camronwood/neural-junkie/issues/21) | Cloud providers (Claude, Gemini) loop `generation_error` during collab planning — P0 turn handoff + CollaborationPanel gen-error banner. **2026-08-25:** Ollama core + `collab-generation-error-resilience` PASS. **2026-08-27 (P5):** cloud repro skipped again (`claude not authenticated` / no `sk-ant-…`). Leave open until operator re-smokes with Claude/Gemini creds. |

---

## Collaboration

| ID | Status | GitHub | Summary |
|----|--------|--------|---------|
| `collab-chat-not-disk` | **Limitation** | [#4](https://github.com/camronwood/neural-junkie/issues/4) | Chat markdown does **not** write to disk. Execution needs `[FILE_CHANGE]` proposals and your approval in **Pending changes**. `TASK_STATUS: completed` alone does not create files. |
| `collab-model-variance` | **Limitation** | [#5](https://github.com/camronwood/neural-junkie/issues/5) | Local models (Ollama) vary in discussion quality, silence, and timeouts; hub enforces phase caps and fallbacks but cannot guarantee plan shape. **Mitigation:** Settings → **Collaboration planning provider** (optional cloud/larger model for planning turns). Implementation sessions: use **reliable tool model** / optional **reliable provider** (repair round 2+) in Settings → Implementation sessions; outcome card shows failure reason. **Benchmark note:** `make model-benchmark SUITE=collab` ranks core collab scenarios; expect higher variance than implement/chat — do not weaken `wait_discussion` asserts to chase green. |
| `collab-smart-routing-scope` | **Limitation** | [#6](https://github.com/camronwood/neural-junkie/issues/6) | Smart routing applies to **execution tasks only**, not normal channel chat. Planning can use optional `planning_provider_id` in Settings (separate from smart routing). |

**Workarounds:** Upgrade hub after each beta; run `make collab-preflight` before scenarios; use `make debug-collab` / discussion diagnosis in [COLLABORATION.md](COLLABORATION.md). For zero-task plans, revise plan or re-run with fewer agents. Inspect per-turn **routing trace** on collab messages when execution picks an unexpected model.

---

## Chat and context

| ID | Status | GitHub | Summary |
|----|--------|--------|---------|
| `hub-history-bounded` | **Limitation** | [#7](https://github.com/camronwood/neural-junkie/issues/7) | Per-channel history is capped (5000 messages) and age-pruned after 24h unless marked **durable**. SQLite sidecar + **Export history** in channel info; not a full in-app search archive. |

---

## Integrations

| ID | Status | GitHub | Summary |
|----|--------|--------|---------|
| `slack-bridge-local` | **Limitation** | [#8](https://github.com/camronwood/neural-junkie/issues/8) | Slack bridge runs **in-process** on the local hub — no public URL required (Socket Mode), but the hub must be running. Use **Settings → Integrations → Slack** diagnostics when OAuth or routing fails. |
| `confluence-setup` | **Limitation** | [#9](https://github.com/camronwood/neural-junkie/issues/9) | Confluence agents need Cloud credentials and indexing time; search quality depends on space size and token limits. |
| `room-chat-lan` | **Limitation** | [#10](https://github.com/camronwood/neural-junkie/issues/10) | **Room chat** pack requires guests on the **same LAN** as the host hub. Corporate Wi‑Fi client isolation may block joins; host must enable **listen on LAN** and configure a hub token. |

---

## Desktop, IDE, and packs

| ID | Status | GitHub | Summary |
|----|--------|--------|---------|
| `web-ui-thin` | **Limitation** | [#11](https://github.com/camronwood/neural-junkie/issues/11) | Browser hub UI at `/` is a **lightweight chat client** — no full workspace, palette, or file-approval UX. Use the **Tauri desktop** for production work. |
| `git-dev-pack` | **Limitation** | [#12](https://github.com/camronwood/neural-junkie/issues/12) | In-app Git operations require the **Software development** pack, `git` on PATH, and a git workspace. |
| `macos-adhoc-sign` | **Limitation** | [#13](https://github.com/camronwood/neural-junkie/issues/13) | GitHub Release macOS builds are **ad-hoc signed** until Apple Developer credentials are available. First launch may require **Right-click → Open**. Planned fix: **v1.2.1** notarized builds. |

_Removed in v4.1 (beta.2+): `ide-v4-remote-lsp`, `ide-v4-remote-collab` — see [IDE_V4.md](IDE_V4.md)._

---

## Hub, agents, and providers

| ID | Status | GitHub | Summary |
|----|--------|--------|---------|
| `single-hub` | **Limitation** | [#14](https://github.com/camronwood/neural-junkie/issues/14) | **Single-server** deployment — no horizontal scale or multi-region hub. |
| `react-tools-allowlist` | **Limitation** | [#15](https://github.com/camronwood/neural-junkie/issues/15) | **ReAct MCP** on non-native tool models is allowlist-based (e.g. `gemma3:12b` first). Other tags fall back to native tools or Qwen swap. |

---

## How we track quality

- **Deterministic:** `make test-collab-plan`, `make collab-smoke`, `go test ./...`
- **Live collab:** `make collab-scenario SCENARIO=…` — matrix in [testing/collab-matrix.tsv](testing/collab-matrix.tsv) (stale 21/21 from 2026-06-09; prefer latest `docs/testing/layer-climb-*.md` / `layer-gate-collab-*.md` for triage)
- **Chat regression:** `make chat-scenarios-regression` — workspace visibility, closure, echo (context v2)
- **Implement parity:** `make implement-scenarios`, `make test-parity-stable` (3× with hub restart)
- **Release engineering:** `make layer-gate`, `make layer-fix-loop`, `make test-growth-loop` — artifacts in [docs/testing/](testing/)

When a row flips to **PASS** or a bug is fixed:
1. Close the matching GitHub issue
2. Remove the row here and from [known-issues.html](known-issues.html)
3. Note the fix in [CHANGELOG.md](CHANGELOG.md) / [release-notes.html](release-notes.html)
