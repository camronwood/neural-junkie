# Changelog

All notable changes to Neural Junkie.

**Versioning:** Installable desktop builds use **SemVer tags** on GitHub (`v1.0.0-beta.1`, `v0.1.x`, …). Sections **0.1.2–0.1.4** below are development milestones bundled into **v1.0.0-beta.1** (first public downloadable beta). Older sections include milestones never tagged (for example internal `2.0.0`, which is **not** semver above `0.1.x`).

## [Unreleased]

## [1.2.0-beta.9] - 2026-07-21

Automatic desktop updates, safer restarts, and more reliable agent questions.

### Added
- **Automatic signed updates** — migrated the desktop app to Tauri v2; macOS and Windows now check and download eligible updates in the background and install them on a clean restart.
- **Release policy controls** — update manifests support beta/stable channels, staged rollout percentages, critical enforcement deadlines, minimum supported versions, and explicit platform availability.
- **Restart safety** — update installation protects unsaved editor, runbook, pack, and chat drafts plus active streams, collaborations, training/analysis jobs, and terminal foreground work.
- **Updater release automation** — immutable signed artifacts, manifest validation, atomic beta-channel promotion, desktop version consistency checks, and Tauri Rust CI gates.

### Changed
- **Agent questions** — equivalent pending questions are coalesced, recent answers are reused, peer agents pause while input is required, and agents continue the original turn after one answer instead of repeatedly asking.
- **Offline update behavior** — accepted update metadata survives restarts, but mandatory updates only block normal use while a verified bundle is immediately installable.
- **Linux updates** — `.deb` distribution remains manual while installed-package upgrade behavior is validated.

### Fixed
- **Question cards** — pending, answered, and expired question states merge reliably in channel history.
- **Update restart lifecycle** — managed PTYs, the Hub sidecar, and bundled Ollama shut down cleanly before installation.

## [1.2.0-beta.7] - 2026-07-20

Hotfix release after beta.6 packaged-app soak: chat reliability, workspace grounding for Knowledge Graph asks, and IDE layout panel independence.

### Fixed
- **WebSocket reconnect race** — stale `onclose` after channel-scoped reconnect no longer flips the hub to a fake disconnect while typing.
- **DM / channel create 429** — rate limiter now reconfigures after config load (`rate_limit_enabled: false` actually applies); dedicated `POST /api/channels/open-dm` is rate-limit exempt; client open-DM path is serialized.
- **Pack toolbar chip icons** — hub icon URLs load via fetch→blob so Tauri CSP does not blank chips; CSP `img-src` allows hub + blob.
- **Hub connection chip** — toolbar status chip mirrors Ollama-style connectivity.
- **`nj_retrieve_context` placeholders** — schema examples no longer look like real refs; placeholder ids are rejected.
- **Knowledge Graph “Ask agents”** — relate-to-codebase prefills keep `workspace_path` so `semantic_search` / code_graph tools are not called with an empty root.
- **IDE layout + hide chat** — hiding the main chat no longer unmounts the editor workspace slot.
- **Chat header panel toggles** — hide-main-chat control sits with left/right sidebar toggles.

### Changed
- **`make local-build` / `make local-install`** — packaged local soak builds (sidecar + Tauri) with optional `/Applications` install.

## [1.2.0-beta.6] - 2026-07-17

The release where the workstation gets a memory of its own code — a native repository knowledge graph, Model Arena, event-driven stream subscriptions, and a wave of collab, packaging, and site hardening.

### Added
- **Repository knowledge graph** — native code graph index (`internal/codeindex/graph`), `/api/repo/graph*` APIs, a `code_graph` knowledge route that grounds agent turns, and a React Flow **Knowledge Graph workbench** in the desktop IDE ([KNOWLEDGE_GRAPH.md](KNOWLEDGE_GRAPH.md) · [feature guide](features/knowledge-graph.html)).
- **Model Arena pack** — head-to-head model matches with reliable move handling and observable match status.
- **Inference usage telemetry** — per-turn inference accounting surfaced across the stack.
- **Stream subscriptions** — MQTT/Kafka subscriptions that trigger runbooks, channels, or webhooks.
- **PrismML Bonsai 27B** — added to the model library with `mmproj` support, a prompted Ollama updater, and min-version gates.
- **Room Chat pack** — ephemeral LAN rooms hosted on a host hub.
- **Homebrew distribution** — macOS and Linux formulae with an automated cask bump on release.
- **Domain pack v2 plumbing** — CAD, AWS, and software-development pack sidecar/platform plumbing.
- **Platform foundations** — turn pipeline, span tracing, and durable collab state.
- **Site** — living development timeline, standard model benchmark run, provider logo strip, Hub architecture article, and SEO meta/sitemap/canonical work.

### Changed
- **Homepage messaging** — leads with any-model, local-or-cloud; marketing site refresh with new demo videos and article cover images in OG/Twitter previews.
- **Collab deliverable scoping** — tightened scoping and banned out-of-scope path disclaimers in minimal-repo findings deliverables.

### Fixed
- **Workspace image previews** — editor images load through the hub as data URLs (local and remote), with a widened Tauri `assetScope` for home-directory assets.
- **Pack toolbar chips** — chips appear immediately on install/enable; the client mutation parser now preserves `capability_registry` and `short_id_collisions`.
- **Collaboration regression gates** — chat layer gate fixes (closure, `@codebase`, public theme), collab-core/full gate hardening, and planning generation-error turn advance.
- **Beta.6 release article** — [articles/beta-6.html](articles/beta-6.html) with cover image.

### Known issues
- **Live parity gate** — `test-parity-stable-restart` (live-model collab parity) is currently failing and is tracked for a follow-up beta; it is not part of the release CI test gate.

## [1.2.0-beta.5] - 2026-07-06

The release where the loops close — ReAct tools, routing trace, Runbooks v2, multi-repo workspace scope, collab hardening, LoRA v2 specialists, personal learning, Slack diagnostics, and release-engineering automation.

### Added
- **ReAct tool wrapper** — MCP tool loops on non-native models (e.g. `gemma3:12b`) via tagged `<tool_call>` parsing, with Qwen swap fallback on iteration cap; config `ollama.react_tools_enabled` / `react_tool_models`; article [react-tools](articles/react-tools.html).
- **Per-turn routing trace (MVP)** — live telemetry drawer, message routing badges, and post-hoc trace panel show model tier, retrieval mode, and governance (composer mode, context scope, impl session) for chat, collab, and implementation turns.
- **Runbooks v2** — persisted `RunbookDefinition` library, `RunExecution` history, connector profiles, pack-owned templates, and desktop library UI ([RUNBOOKS_V2.md](RUNBOOKS_V2.md)).
- **Multi-repo workspace scope** — project sets group workspace roots; desktop **workspace scope chip**; cross-repo hints in agent context; ambient-scope scenario coverage.
- **Connection settings** — hub URL, server/network, automation, and connectors tabs for runtime config without hand-editing JSON.
- **LoRA v2 compound specialists** — import rows, review training JSONL, bootstrap repo indexes, compose Ollama tags from one base + adapters across chat, collab, IDE, and pack surfaces.
- **Personal learning** — explicit user-confirmed memory scoped per expert, globally, or per collaboration; optional export into LoRA training rows.
- **Slack setup diagnostics** — settings checklist, smoke/diagnose endpoints, clearer handler errors for OAuth and channel routing.
- **Release engineering loops** — `make layer-gate`, `make layer-fix-loop`, and `make test-growth-loop` with artifacts in `docs/testing/`.
- **Beta.5 release article** — [articles/beta-5.html](articles/beta-5.html) with cover image.

### Changed
- **Collaboration hardening** — plan parser uses newest agent turn's task block; planning recap timeout 240s → 90s; discussion watchdog advances timed-out discussions; turn handoff retries; approve-plan harness timeout 60s → 180s.
- **Runtime config** — settings restart baseline avoids copying `config.Config` with embedded mutex (go vet clean).

### Fixed
- **Approve-plan test flake** — hub tracks approve-plan async work; tests drain before TempDir cleanup.
- **Pack sidecar IDE typing** — moved sidecar env types to `internal/packs/sidecar_env.go` so gopls no longer reports invalid types in config.

## [1.2.0-beta.4] - 2026-06-28

Agent Runtime reliability, local image generation, music pack v1.0.2, and turn telemetry.

### Added
- **Agent Runtime Reliability (3-pass program)** — Fix Loop command policy wired into LLM `run_command` tool loop; boot-fix `read_file` grounding; Tauri/Vite port playbook; implement-scenario preflight and required outcome assertions; reliable-tier routing (`reliable_tool_model`, opt-in `reliable_provider_id` on repair round 2+); `implementation_session_outcome` on wrong-route redirects; desktop **Implementation session outcome** card and failure toasts.
- **Ollama image generation** — hub-side generated image store, lookup API, agent tools, and **Image generation** settings panel for local image gen via Ollama.
- **Music creation v1.0.2** — ACE-Step model variants (sft/turbo/xl), inference tuning (seed, steps, guidance, ODE/SDE), in-app setup/status, and MusicExpert response cards.
- **Turn telemetry drawer** — per-turn routing trace, tool activity, and slash-command history in the desktop chat UI.
- **Editor agent trust** — expanded trust modes and boot-fix routing in the IDE composer.
- **Thinking activity labels** — hub and desktop show structured in-progress labels during agent tool loops.

### Changed
- **Capability profiles** — refreshed model benchmarks and reliable-tier routing metadata.
- **Implement scenarios** — stronger preflight checks and required outcome assertions across boot-fix and verify-failure fixtures.

## [1.2.0-beta.3] - 2026-06-24

Site navigation polish and CI reliability for the beta line.

### Added
- **Unified site nav** — canonical header + footer explore strip across all `docs/**/*.html` via [`scripts/site_nav.py`](../scripts/site_nav.py).

### Fixed
- **CI release gate** — Node 24 in workflows; PendingChangesPanel tests use mock git change store.

## [1.2.0-beta.2] - 2026-06-24

Marketing site refresh, pack expansion, and IDE v4.1 follow-ups.

### Added
- **Marketing site refresh** — hive-mind narrative, [`start-here.html`](start-here.html), [`security.html`](security.html), and [`features/life-sciences.html`](features/life-sciences.html).
- **AWS, incident-management, and web-browser domain packs** — seven official packs in the store ([`PACKS.md`](PACKS.md)).
- **Pack capability sidecars** — per-pack hub routes when packs are enabled.
- **IDE v4.1** — remote LSP relay and remote collab worktrees ([`IDE_V4.md`](IDE_V4.md)).

### Changed
- **Moderator merged into Assistant** — chat guidance and safety-net behavior live on Assistant; Moderator removed as a separate auto-started agent.
- **Hub auth hardening** — mutations require real sessions or API keys.

### Fixed
- **Cursor parity infrastructure** — release-prep reliability, collab turn routing, implement scenario regressions.
- **Domain pack catalog** — stale hub catalog no longer hides new packs (web-browser, AWS, incident-management).

## [1.2.0-beta.1] - 2026-06-17

**IDE v4** — full Monaco LSP, remote SSH workspaces, and `nj-remote` sidecar. See [IDE_V4.md](IDE_V4.md) and [REMOTE_WORKSPACES.md](REMOTE_WORKSPACES.md).

### Added
- **Full Monaco LSP (local)** — persistent `gopls` / `rust-analyzer` / `pyright-langserver`; WebSocket document sync; diagnostics, hover, completion, go-to-definition, references, rename.
- **WorkspaceBackend** — pluggable local + remote FS, git, search, symbols, file changes, and `@codebase`.
- **`nj-remote` sidecar** — HTTP FS, exec, PTY WebSocket (`cmd/nj-remote`).
- **Remote SSH workspaces** — desktop wizard, token persist, sidecar health checks.
- **Remote terminal** — hub PTY proxy; remote exec routing.
- **Dev containers** — attach plan API for `.devcontainer/devcontainer.json`.
- **tree-sitter symbols** — optional symbol index when CLI is on PATH.
- **IDE v3.5 polish** — LSP-lite squiggles, Problems panel, `yolo` editor trust auto-approve.

### Known limitations (v4.1)
- Remote LSP relay deferred; dev container desktop wizard tab deferred; collab worktrees on remote workspaces not yet supported.

## [1.0.0-beta.35] - 2026-06-10

Working **beta in-app auto-update**: git-backed updater manifests on `main`, dual endpoint fallback, and release CI fixes bundled with beta.34 Slack OAuth relay.

### Added
- **Git-backed beta updater channel** — rolling manifests at `updater/beta/` on `main`; CI syncs after each beta release ([RELEASE_UPDATES.md](RELEASE_UPDATES.md)).
- **Dual updater endpoints** — beta builds try legacy `updater-beta` GitHub release, then raw `main` URLs so Tauri update checks succeed despite immutable release tags.

### Fixed
- **Beta in-app auto-update** — GitHub immutable releases blocked rolling `updater-beta` tag recreation; git-backed channel + dual endpoints restore update checks on fresh installs.
- **Release CI** — learning test globals lock/defer, `SLACK_VENDOR_OAUTH_RELAY_BASE` in build jobs, macOS ad-hoc signing default, manifest publish commits to `main`.

### Note
- **v1.0.0-beta.34 installers** use the old single `updater-beta` endpoint only — upgrade to **beta.35+** for working in-app updates.

## [1.0.0-beta.34] - 2026-06-10

Public **Connect Slack** for any workspace: HTTPS OAuth relay on Cloudflare Workers, loopback redirect auto-upgrade, and desktop layout polish.

### Added
- **Cloudflare Workers Slack OAuth relay** — `workers/slack-oauth-relay/` + `make slack-oauth-relay-deploy-cf` for free `*.workers.dev` HTTPS redirects.
- **`SLACK_VENDOR_OAUTH_RELAY_BASE`** — required CI secret; embeds relay URL in `slackvendor` builds.
- **Relay docs** — [SLACK_OAUTH_RELAY_SETUP.md](SLACK_OAUTH_RELAY_SETUP.md), updated [SLACK_INTEGRATION.md](SLACK_INTEGRATION.md).
- **Desktop chat max-width helper** — `mainChatMaxWidth` respects visible panel chrome when resizing the main chat column.

### Changed
- **Slack OAuth redirect** — loopback `redirect_url` values in user/config files upgrade to the public HTTPS relay by default (`NEURAL_JUNKIE_SLACK_USE_OAUTH_RELAY=0` to opt out for single-workspace dev).
- **Pending changes / file explorer** — layout tweaks alongside chat panel resize behavior.

### Fixed
- **`redirect_uri did not match`** — Connect Slack no longer sends `http://localhost:18765/...` when Slack app only registers HTTPS relay URLs (public distribution).

## [1.0.0] - TBD (stable channel)

First **stable** release on the stable updater channel. Scope: [STABLE_SCOPE.md](STABLE_SCOPE.md). Cut procedure: [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md) and `./scripts/cut-stable-release.sh --execute`.

Includes everything in **1.0.0-beta.33** plus fixes and test coverage on `main` since beta.33 (DM subscription race, macOS tao patch, conversation scenario expansion). **macOS:** ad-hoc signed CI builds at v1.0.0 (Right-click → Open if Gatekeeper warns). **Notarization:** planned **v1.0.1** when Apple Developer credentials are available.

### Added (since beta.33, on main)
- **Stable release infrastructure** — macOS notarization CI (optional until Apple creds), [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md), [STABLE_SCOPE.md](STABLE_SCOPE.md), [PLATFORM_ROADMAP.md](PLATFORM_ROADMAP.md), `scripts/cut-stable-release.sh`.
- **Settings → Security** — hub security status (`GET /api/system/security`).
- **`make test-conversation-contract`** — CI-safe conversation + collab wiring gate.
- **Conversation scenario expansion** — 4 chat scenarios (continuation, topic switch, closure-then-continue, interject) + 3 collab participation scenarios; 23/23 `conversation-scenarios-regression` PASS.
- **Platform smoke runbook** — [testing/stable-platform-smoke.md](testing/stable-platform-smoke.md).

### Changed (since beta.33)
- **IDE v3 GA** — removed beta limitation from known issues; [IDE_V3.md](IDE_V3.md) updated.
- **Stable scope** — v1.0.0 commits to ad-hoc macOS; notarization deferred to v1.0.1.
- **Updater publish** — `publish-updater-manifests.sh` upload-only for existing `updater-beta` (immutable release rule fix).

### Fixed (since beta.33)
- **DM subscription race** — eager agent subscribe on `CreateDMChannel` with delayed history replay.
- **macOS tao crash** — vendored patch for Cmd+KeyUp / keyWindow SIGABRT.

### Known at stable cut
- **`macos-adhoc-sign`** — documented limitation; not a release blocker for v1.0.0.

### Added (beta line through v1.0.0-beta.33)
- **Cursor-like composer** — Ask / Agent / Export modes on all channels with unified send pipeline and turn-intent metadata routing.
- **Durable conversation memory** — SQLite-backed channel history bootstrap; prior-reference export survives hub restart.
- **Tool-first file delivery** — `propose_file_edit` primary write path; legacy `[FILE_CHANGE]` parsing gated by env.
- **Inline markdown edit/preview** — toggle in the code editor for `.md` files.
- **Layout owner picker** — Settings → Domain packs → choose which enabled pack controls IDE vs team layout.
- **Collaboration planning provider** — optional `planning_provider_id` routes planning discussion turns through a chosen provider.
- **Channel history export** — `GET /api/channel-export` and **Export history** in channel info (markdown).
- **Durable channels** — per-channel toggle skips 24h age prune; persisted in `config.json`.
- **Customer pack capability gating** — sideload packs gate `scan-summary-api`, `scan-analysis-viewer`, and secondary-analysis capabilities (requires life-sciences base pack).

### Changed
- **Prior reference + implementation continuation** — stronger grounding on prior turns; improved go-ahead / continuation detection in desktop payload prep.
- **Workspace gate UX** — panel/chat banners, toasts, and **Confirm workspace** primary action when execution waits for ack.
- **`/pause-agent`** — aborts in-flight LLM generations for the paused agent.
- **Pack docs** — life-sciences vs customer-pack capability split in `BIOLOGY_PACK.md` and `PACKS_CUSTOM.md`.
- **Dev update checks** — suppressed in `import.meta.env.DEV` builds (no error banner).

### Fixed
- **Collab scenario harness** — deliverable stubs no longer satisfy file assertions; discussion fallback writes substantive content and strips `TASK_STATUS:` leakage.
- **Scan summary API tests** — enable customer-lab-pack fixture for `scan-summary-api` capability gate.
- **Conversation regression harness** — collab race and session-issue analyzer hardening.

## [1.0.0-beta.32] - 2026-06-07

### Added
- **Wizard Ollama auto-install** — setup wizard installs Ollama on first launch (Linux, macOS without bundle, Windows) with live progress via `POST /api/ollama/install`.

### Changed
- **Windows installers** — slim build without bundled Ollama (was ~1.4 GiB); wizard runs winget or silent `OllamaSetup.exe`.
- **Linux installers** — wizard auto-install replaces manual `install.sh` step documented in beta.31.

## [1.0.0-beta.31] - 2026-06-07

### Changed
- **Linux installers** — omit bundled Ollama (full runtime exceeds GitHub's 2 GiB release asset limit). Install [Ollama](https://ollama.com) separately or use a cloud provider; the hub detects system `ollama` on PATH.

### Fixed
- **Linux `.deb` upload** — slim package publishes successfully; AppImage remains best-effort.

## [1.0.0-beta.30] - 2026-06-07

### Fixed
- **Linux releases** — upload `.deb` immediately after a successful deb build; AppImage is best-effort (`continue-on-error`) so Linux users get an installer even when AppImage tooling fails.
- **Updater manifests** — macOS + Windows publish when Linux AppImage updater bundle is absent.

## [1.0.0-beta.29] - 2026-06-07

### Fixed
- **Linux AppImage CI** — `NO_STRIP=true` for linuxdeploy strip failures; split deb and AppImage build steps; add `universe` repo and `squashfs-tools`.

## [1.0.0-beta.28] - 2026-06-07

### Fixed
- **Linux CI** — install `libfuse2` for AppImage bundling; skip unused RPM target (`--bundles deb,appimage`).
- **macOS updater uploads** — arch-suffixed `.app.tar.gz` names so Apple Silicon and Intel bundles no longer clobber each other on GitHub Releases.
- **Beta updater channel** — recreate `updater-beta` release each publish to avoid immutable-release upload failures.

## [1.0.0-beta.27] - 2026-06-07

### Added
- **In-app auto-updates** — Tauri v1 updater with minisign signatures; separate beta (`updater-beta`) and stable (`latest`) channels.
- **Update UI** — launch banner with **Update Now** / **Later**, 15s auto-dismiss, and Settings → About → **Check for updates**.
- **Hardware guidance** — RAM tiers and model footprint docs; `/api/system/hardware`; setup wizard and AI Providers hints.
- **Downloads page** — `docs/download.html` with direct per-platform installer links; `scripts/update-website-release.sh` refreshes from GitHub Releases.
- **Updater CI** — signed bundles, manifest publish job, and `scripts/verify-updater-manifest.sh`.

### Fixed
- **Windows beta builds** — WiX-safe bundle version mapping (`1.0.0-N` for `v1.0.0-beta.N` tags) restores MSI/EXE CI for beta releases.

## [1.0.0-beta.25] - 2026-06-07

### Added
- **Regression bundle** — `make test-regression-bundle` runs implement (7/7) + chat-regression + conversation-regression; logs under `docs/testing/regression-bundle-*.log`.
- **Parity-stable-restart** — `make test-parity-stable-restart` and `scripts/lib/hub_regression.py` for hub restart between implement sweeps.
- **Collab goal bootstrap** — when planning discussion lacks parseable tasks, hub extracts executable rows from the `/collaborate` goal.

### Changed
- **Collab plan parsing** — plain `Task N @Agent Write …` rows, semicolon-separated goal clauses, and `file.md (@Agent)` goal lists.
- **Collab execution prompts** — findings markdown tasks require substantive bullets grounded in project files.
- **Pack LoRA startup logs** — materialize adapter weights (no symlink escapes); truncated one-line compose warnings.

### Fixed
- **Thanks closure with @mention** — `@Assistant ok thanks` in public channels gets canned closure instead of an LLM re-answer.
- **Collab plan-zero-tasks** — goal bootstrap + parser improvements for `plan-dependency-prose-regression`, `plan-distinct-deliverables-same-agent`, and `plan-combined-resource-api-regression`.
- **Collab deliverable size** — execution prompt hardening for `execution-no-stack-commands` findings files.

## [1.0.0-beta.24] - 2026-06-06

### Added
- **Phase 1 parity bundle** — server-side `@codebase` attachments, implementation session hardening, IDE Agent `auto_apply_edits` default, inline completion in IDE layout.
- **Multi-file implementation loop** — theme tasks continue in one user turn (`tailwind.config.js` → `src/App.tsx`) up to 5 files; scenario `react-theme-multi-file`.
- **Stability harness** — `make test-parity-stable` and `scripts/implement-scenarios-stable.py` (3× implement sweeps, logs under `docs/testing/`).
- **Keyboard shortcuts registry** — centralized desktop shortcut dispatcher; see [KEYBOARD_SHORTCUTS.md](KEYBOARD_SHORTCUTS.md).
- **Two-tier LoRA compose** — `internal/hfhub/lora_compose.go`, bootstrap verify (`cmd/verify-bootstrap-lora/`, `scripts/verify-bootstrap-loras.sh`).
- **Hub file-change guards** — safer apply pipeline for IDE auto-approve paths.
- **Harness expansion** — specialist workspace chat scenarios, collab conversation-quality regressions, `scripts/conversation-scenarios-regression.py`.

### Changed
- **Local-first routing docs** — `fallback_provider_ids` is infrastructure-only when Ollama is absent; no automatic cloud escalation on model failure. Use `@Cursor` explicitly for cloud-grade work.
- **Implement scenarios** — 7/7 gate including `dm-backend-codebase-semantic` and multi-file theme; `assert_file_exists` supports `any_match`.
- **Specialist-tuning pack** — updated LoRA metadata and library entries for composed tags.

### Fixed
- **Continuation turns** — affirmation guidance and deterministic fallback on go-ahead; exclude already-applied paths from re-proposals.

## [1.0.0-beta.23] - 2026-06-06

### Added
- **CLI agents manager** — desktop UI + hub handlers for Codex, Cursor, Gemini, and Copilot CLI agents.
- **Pending tool approvals bar** — hub-wide approval queue synced to the desktop composer.
- **Implementation fallback** — deterministic tailwind/theme repairs when models emit prose-only implementation replies.
- **Code review intent** — dedicated routing so review asks do not open the file-edit implementation loop.
- **Shared TypeScript/ESLint MCP helpers** — `tsc` / `eslint` verification for frontend implementation sessions.
- **Test artifact cleanup** — `scripts/cleanup-test-artifacts.py` and `make cleanup-test-artifacts`; auto-run after `make test-go`.
- **Marketing assets** — non-dev provider/logo ads and edge-IDE campaign creatives under `assets/`.

### Changed
- **Implementation continuation** — skip re-proposing already-applied files; block vague “pick up where you left off” without prior thread context.
- **File-change approvals** — agent-visible approval messages with metadata; auto-continue prompts after UI apply.
- **Ollama dev/runtime** — bundled-runtime chip, `ensure-ollama.sh`, and sidecar PATH resolution for Tauri dev.
- **Test isolation** — `internal/agent/test_main.go` and expanded `test/` home isolation so repo-agent tests no longer pollute `~/.neural-junkie/repos`.

### Fixed
- **Compact assistant prompt test** — user-rules lookup uses a user-like sender type.
- **ChatWindow Vitest** — mock PhoenixBrowserModal, pending approvals, and related stores in collaboration/interject tests.
- **Hub history clear** — persistent store cleanup for `clear-history` API.

## [1.0.0-beta.22] - 2026-06-05

### Added
- **Bundled Ollama** — macOS, Windows, and Linux installers ship the Ollama runtime; production app auto-starts `ollama serve` with models stored in app data. First run still pulls a default model once (internet required).
- **CSV table editor** — editable grid view for exported `.csv` files in the code editor (Table/Text toggle); scan summary CSVs still open in the analysis viewer.
- **Pack Dev Studio fixes** — live YAML validation prefers editor text over disk; dev-link preserves enabled state; dev-reload refreshes overlays; multi-pack test selector.

### Changed
- **Setup wizard** — Ollama step reflects bundled runtime on production installers; Windows no longer requires a separate Ollama install.
- **Release CI** — fetches Ollama v0.30.5 per platform before Tauri bundle.

## [1.0.0-beta.21] - 2026-06-02

### Added
- **Slack personal inbox** — DM the NJ bot from Slack mobile; messages route to a private hub channel and your chosen agent. OAuth captures the installing user as owner.
- **Slack selective forwarding** — Optional rules: `@mention of me` in watched channels, `nj:` prefix, or reaction emoji. Forwarded messages get agent replies in the **original Slack thread**; direct DMs reply in the bot DM thread.
- **Slack human DM away mode (opt-in)** — Separate user OAuth token reads your 1:1 Slack DMs while away (manual toggle or outside work hours). Agent replies in the same DM thread with a labeled prefix (`Assistant (for you): …`) via the user token. Distinct from bot personal inbox and note-to-self.
- **User rules API** — `GET`/`PUT` `/api/user-rules` for per-user markdown rules injected into prompts.
- **Chat find bar** — in-channel search across messages (desktop).
- **Live chat scenarios** — DM and public workspace/echo/closure regressions under `scenarios/chat/`; `make chat-scenarios`, `make chat-scenarios-regression`.
- **Collab routing matrix** — `make collab-routing-matrix`, `solo-vs-collab-parity` scenario.
- **Test isolation** — `internal/testutil` temp-home pattern for hub/server/integration tests; see [TESTING.md](TESTING.md).

### Context model v2

- **Conversation Context Stack** — documented in [CONTEXT_MODEL.md](CONTEXT_MODEL.md): mode → intent → memory → grounding → persona → budget pipeline.
- **Composer conversation mode** — Auto / Chat / Code chip sends `conversation_mode` metadata; Chat mode skips workspace attachment and tooling-heavy prompts.
- **Turn intent v2** — `casual` (was `low_signal`), new `task` intent for code verbs; chat mode biases toward casual replies.
- **DM persona** — direct 1:1 framing in DMs; MCP and `[FILE_CHANGE]` docs suppressed on casual chat turns.
- **Broader session summaries** — rolling summaries on public channels and `dm-*` specialist slugs, not only typed DM/custom channels.
- **Thread-scoped history** — agents in threads use thread messages for LLM history instead of full channel noise.
- **Context budget** — ~32KB prompt cap with section-aware truncation before LLM calls.
- **Workspace visibility** — deterministic replies for “can you see my workspace?” with streaming/tool-path fallback when models ignore the question.
- **Baseline fixes** — `@mention` overrides IDE route; session persist slimming for `workspace_context`.

### Changed
- **Collaboration agent order** — preserve `@mention` order when creating collaborations (fixes nondeterministic round-robin assignees).
- **Scenario runners** — chat scenarios inject `workspace_context` when `context_scope` is set; learning query URLs are properly encoded.

### Fixed
- **Echo follow-ups** — “What?” after a long reply no longer quotes the first user message.
- **Closure** — “I know you said that already” returns canned won't-repeat responses.
- **Slack OAuth tests** — hermetic temp home dir for env-vs-user-file resolution tests.

## [1.0.0-beta.20] - 2026-05-30

### Added
- **Personal learning v2** — multi-scope memory (`agent`, `global`, `collaboration`) with Ollama embedding retrieval (keyword fallback when offline), edit/export/import APIs, and per-user session isolation. See [PERSONAL_LEARNING_V2.md](PERSONAL_LEARNING_V2.md).
- **Learning proposal UX** — scope toggle in approval modal; grouped lists in Settings; edit + scope badges in agent info; optional agent-suggested learnings (`personal_learning_suggest_enabled`).
- **LoRA + learnings bridge** — include up to 50 confirmed learnings in train preview/export (`include_learnings=1`).
- **Test harness** — v2 JSON scenarios (`learning-global-scope`, `learning-collab-scope`, `learning-export-import`, `learning-retrieval-debug`) and extended CI smoke.

### Changed
- **Prompt injection** — top-k retrieval per scope instead of dump-all learnings; debug metadata includes `injected_learning_ids` when `NEURAL_JUNKIE_DEBUG=1`.
- **Specialist pack sync** — match agents by name before type to avoid clobbering custom specialist entries.

### Fixed
- **Collab @mention during planning** — agents ignore @mentions embedded in another agent's plan prose; humans and system turn prompts still wake assignees.
- **Collab turn prompt** — match `You're up first` without requiring a trailing period.
- **Embedding scheduler** — remove recursive `SetOnEntryChanged` hook that could stack-overflow on save.

## [1.0.0-beta.19] - 2026-05-29

### Added
- **Domain pack store** — Settings → Domain packs → Pack store lists official packs from GitHub; hub installs zip bundles with embedded fallback when offline. See [PACKS.md](PACKS.md).
- **Loose `[FILE_CHANGE]` parsing** — collaboration task replies that use inline `[FILE_CHANGE] path` or `path:` lines (without `[/FILE_CHANGE]`) now register hub file proposals.
- **Desktop collaboration UX** — planning wait banner (0 messages), approved/dispatch banner, file-deliverable hints on in-progress tasks, workspace gate copy for Pending changes.
- **Horizontal panel resize** — drag handles for resizable desktop layout panels.

### Changed
- **Domain packs** — multiple packs can be enabled together; first enabled pack owns UI layout (replaces single-pack exclusivity).
- **Scenario harness** — `last_system_error` matches `⚠️` workspace warnings; `send_message` retries on HTTP 429; `free_scenario_capacity` no longer cancels multi-collab isolation blockers; `resource-api-schema-planning` uses a single combined `any_match` regex.
- **Planning without bound repo** — agents no longer scan the open editor tree or emit `Grounding: I loaded N files` during planning/review when no source workspace is bound.
- **Task dispatch note** — file-shaped tasks include a canonical `[FILE_CHANGE]` example in the prompt.
- **CLI collab role label** — `SuggestRole(cli)` is “Implementation & Code (from approved plan)”.

### Fixed
- **Multi-collab isolation** — executing blocker collabs are not auto-cancelled when other scenarios start on `collab-scenarios`.
- **Rate limit during scenario sweeps** — document `NEURAL_JUNKIE_RATE_LIMIT=0` for local hub when running `make collab-scenario-matrix`.

## [1.0.0-beta.18] - 2026-05-27

### Added
- **Collaboration reliability** — `make collab-smoke` / `LIVE=1` exercises planning → review → execute with real hub agents; `debug-collab.py` for live session inspection.
- **Plan path warnings** — `/approve-plan` surfaces missing task context paths under the bound project repo.

### Changed
- **Execution task prompts** — assignees are told to ship `[FILE_CHANGE]` deliverables and end with `TASK_STATUS: completed` or `blocked` (chat-only replies do not complete tasks).
- **`/collaborate --workspace`** — workspace outline metadata is attached only when `--workspace` is passed (not on every outbound desktop message).
- **Collaboration panel** — phase and snapshot stay in sync while the panel is open (polling + merged hub state).

### Fixed
- **Multi-collab phase routing** — agents use the message’s collaboration id for phase gating, so a planning collab is not blocked when another collab is executing.
- **Source workspace binding** — rejects Neural Junkie sandboxes and project `collabs/<uuid>/` deliverable folders; open the git repo root (e.g. Phoenix), not a prior collab output directory.
- **Task extraction** — parses markdown `## Tasks` numbered bold lines (`1. **Title** (@Agent)`), skips milestone/metadata bullets, and drops weak fragment bullets (`task is to…`, `should perform…`).
- **Run command detection** — bare paths in ` ```bash ` blocks (e.g. `findings.md`, `collabs/<id>/file.md`) are not offered as shell commands.

## [1.0.0-beta.17] - 2026-05-26

### Added
- **Workspace quick switcher** — workspace tabs and searchable switcher make multi-repo work easier; filter by name, path, or branch and jump with keyboard controls.
- **Assistant workspace grounding** — user messages carry scoped workspace metadata, open-file focus, active selections, and scan-summary context when sharing is enabled.
- **Google Meet notes in release builds** — Connect Google can use bundled vendor OAuth credentials from CI (`-tags googlevendor`); custom Google Cloud clients moved to Advanced.

### Changed
- **Assistant meeting context** — meeting-note ingestion is quieter on startup and better grounded for follow-up questions. See [docs/GOOGLE_MEET_NOTES.md](GOOGLE_MEET_NOTES.md).

### Fixed
- **macOS installers** — Tauri bundles are ad-hoc signed in CI and verified before upload so beta.16's broken macOS artifacts are replaced by a fresh release.

## [1.0.0-beta.16] - 2026-05-22

### Added
- **Domain pack exclusivity** — only Software development or Life sciences can be enabled at a time (hub + Settings); migration clears both-on configs (dev pack wins).
- **IDE v1 (Software development pack)** — Git modal (status, commit, pull, push), workspace file search API, quick open (⌘P), Monaco selection in agent workspace context.
- **IDE v2 (Software development pack)** — Git stage/unstage + Monaco diff viewer; go to symbol (⌘⇧O); TS/JS diagnostics + Problems panel; optional `gopls check` for Go; inline file-change hunks; fast edit (⌘K). See [IDE_V2.md](IDE_V2.md).
- **Marketing** — home page Domain packs section; release notes for beta.16.

### Changed
- Enabling Software development pack applies a one-time layout nudge (file explorer + editor visible).

## [1.0.0-beta.15] - 2026-05-22

### Fixed
- **Release installers: Connect Slack** — GitHub Actions writes `vendor/oauth.json` from `SLACK_VENDOR_*` secrets and builds `nj-server` with `-tags slackvendor` so public downloads get one-click Slack Connect (beta.14 CI builds lacked bundled OAuth).

## [1.0.0-beta.14] - 2026-05-22

### Added
- **Slack Connect (local-first)** — bundled NJ Slack app credentials (`-tags slackvendor`), loopback OAuth, `GET /api/slack/connection`, Simple vs Advanced Settings UI, channel picker, install metadata. See [docs/SLACK_INTEGRATION.md](SLACK_INTEGRATION.md).
- **Slack bridge polish** — inbound user display names, policy routing without fake @mentions, NJ→Slack threading via channel parent map, human outbound to Slack, `GET /api/slack/channels`, diagnose endpoint.
- **Marketing gallery** — [docs/gallery/](gallery/) with ads and screenshots; `./scripts/sync-gallery.sh` and `make gallery-sync`.
- **Slack marketing ads** — dev, non-dev, and models variants (`assets/neural-junkie-slack-ad-*.png`); updated `compose-beta13-ads.sh` variants.

### Changed
- **Makefile** — `start-all` / `server` use `-tags slackvendor` when `vendor/oauth.json` exists; `slack-vendor-json` and `gallery-sync` targets.
- **Slack disconnect** — clears bot token and install metadata; keeps bundled app token.

### Fixed
- **Slack OAuth** — redirect URL derived from hub listen port; auto-restart bridge after OAuth.

## [1.0.0-beta.13] - 2026-05-21

### Added
- **CLI agent auto-detection (12 types)** — Cursor, Gemini, Claude, Copilot (modern `copilot` + legacy `github-copilot-cli`), Codex, Aider, OpenCode, Amazon Q (`q`), Crush, Amp, Factory Droid, and Kiro; PATH resolution with per-binary invoke args. See [docs/CLI_AGENTS.md](CLI_AGENTS.md).
- **Local-first security** — default hub bind `127.0.0.1`, restricted CORS, optional hub token, session tokens (`POST /api/auth/session`), channel ACL hooks, HTTP rate limits, encrypted config secrets and Tauri credential blobs. See [docs/SECURITY.md](SECURITY.md).
- **Scan summary viewer** — desktop viewer for plate/scan summary JSON with well navigation and TIFF preview (`ScanSummaryViewer`, hub scan-summary API).
- **Slack integration** — OAuth, channel bindings, config API, and hub handlers. See [docs/SLACK_INTEGRATION.md](SLACK_INTEGRATION.md).
- **Agent delegation** — consult/delegate flow between agents. See [docs/DELEGATION.md](DELEGATION.md).
- **Marketing** — beta.13 ads (`campaigns/beta13/creatives/`); `./scripts/compose-beta13-ads.sh`; [campaigns/beta13/BETA13-ADS.md](../campaigns/beta13/BETA13-ADS.md).

### Fixed
- **Desktop hub API** — `hubFetch` no longer recursed into itself (blocked auto-login and all authenticated hub calls after security pass).
- **Hub startup logs** — WebSocket/Web UI URLs display `localhost:port` correctly when bound to loopback.
- **Tests & UX** — Vitest coverage target, scan-summary and editor store tests, collab reconcile log dedupe, dev-only `devLog`, ChatWindow test mocks.

### Changed
- **Collaboration manager** — reconcile warnings logged once per missing participant (less startup noise).

## [1.0.0-beta.12] - 2026-05-20

### Added
- **Domain packs** — optional **Software development** and **Life sciences** packs in Settings → Domain packs; wizard focus tracks; fresh installs stay lean (Moderator, Assistant, CLI agents only until you enable a pack).
- **Life sciences pack** — BiologyExpert, OpenBioLLM 8B, bio MCP tools (`analyze_sequence`, `fold_protein`), sequence review runbook template. See [docs/BIOLOGY_PACK.md](BIOLOGY_PACK.md).
- **Context model** — turn intent router (closure / low-signal / meta / substantive) and rolling **session summary** for DMs (`qwen2.5:7b`). See [docs/CONTEXT_MODEL.md](CONTEXT_MODEL.md).
- **Runbook action tasks** — deterministic `http_get`, `http_post`, `webhook`, and conditional edges; bundled **runbook templates** (`GET /api/runbook-templates`). See [docs/RUNBOOK_ACTIONS.md](RUNBOOK_ACTIONS.md).
- **Collab recap** — end-of-collaboration summary messages in collab channels.
- **Google Meet notes** — Assistant integration for meeting note ingestion (Settings).
- **Marketing** — beta.12 feature ads (`campaigns/beta12/creatives/`); `./scripts/compose-beta12-ads.sh`; [campaigns/beta12/BETA12-ADS.md](../campaigns/beta12/BETA12-ADS.md).

### Changed
- **Specialist agents** — engineering specialists and MCP tool servers follow pack toggles (migration enables software-development pack when legacy config had dev agents enabled).
- **Sidebar** — auto-unhide DM/collab/agent shortcuts when opened; stable hide keys survive agent restarts.
- **Ollama** — native tool calling path and capability detection for supported models.

## [1.0.0-beta.11] - 2026-05-19

### Added
- **Runbook builder** — desktop **RB** button and `/runbook`: define tasks, dependencies, and agent assignments; **Graph** view (xyflow) with drag-connect edges, inspector, auto-layout; import markdown runbooks.
- **Runbook collaborations** — `POST /api/runbooks` creates `source: runbook` collaborations; DAG validation, suggest-assign for Auto tiles, hub dispatch and lifecycle aligned with slash-command collabs.
- **Collab completion UX** — channel banner when a collaboration completes (`Collaboration complete — N/M tasks done`); read-only closed channel; desktop panel sync.
- **Non-developer marketing assets** — `campaigns/nondev/creatives/` and `campaigns/nondev/NONDEV-ADS.md`; `./scripts/compose-nondev-ads.sh`.

### Changed
- **Collaboration manager** — runbook task orchestration, artifact handling, and hub limits refined for runbook + discussion flows.

## [1.0.0-beta.10] - 2026-05-19

### Fixed
- **Release builds** — all six platform installers (macOS arm/x64, Windows msi/exe, Linux AppImage/deb); `tauri.conf.json` package version stays `1.0.0` for WiX/MSI; `make release` no longer overwrites it; Vitest files excluded from `tsc`.

## [1.0.0-beta.9] - 2026-05-18

### Added
- **Collaboration git worktree execution** — `/collaborate --worktree` runs approved plans in a real repo copy on branch `nj/collab-<id>` under `<assets-root>/worktrees/`; combine with `--workspace` to bind the source repo at start, or pick the active git workspace at the desktop **Continue** gate (`source_repo_path` on workspace ack).
- **Worktree desktop gate** — collaboration channel shows source repo, branch, and worktree path; blocks Continue until a git workspace is selected when the repo was not bound at start.

### Fixed
- **Release builds** — desktop `npm run build` no longer fails TypeScript check on `workspaceFileDrag.test.ts` (CI had produced source-only beta.8).

## [1.0.0-beta.8] - 2026-05-18

### Added
- **App-store model library** — toolbar **Model library** (⇧⌘M, `/nj-open-model-library`): browse grid + detail for Ollama and Hugging Face catalogs; one primary action per tile; full install/use/download actions on the detail screen. Curated `icon_key` / `publisher` metadata on catalog entries.
- **Hugging Face model integration** — `huggingface` provider type (HF Inference Router), curated catalog (`GET /api/hf/catalog`), HF tab in the model library with hosted (cloud) and download (local GGUF) modes, SSE download progress, `POST /api/hf/import-ollama`, and DM expert creation via `provider_id` or `huggingface` + Hub repo id. Config: `hf.token`, `hf.cache_dir`, `HF_TOKEN` env.
- **MCP tool servers re-enabled** — Backend, DevOps, and Database agents start MCP HTTP servers when `ENABLE_MCP=true` (ports 8081–8083).
- **In-app MCP tool execution** — Claude-backed specialist agents run a tool-use loop via in-process MCP handlers (`internal/agent/mcp_tools.go`, `internal/ai/claude_tools.go`).
- **MCP resource server** — `ENABLE_MCP_RESOURCES=true` starts export knowledge server on port 8086; CLI `--serve-mcp` runs standalone.
- **CLI/API export** — `GET /api/exports`, `POST /api/export`; `MCP_EXPORTS_DIR` honored for export storage.
- **Tests & smoke** — `modelVisuals` unit tests, `internal/mcp/server_test.go`, `mcp_tools_test.go`, `claude_tools_test.go`, `scripts/mcp-smoke.sh`.

### Changed
- **Model installs** — Ollama pull/delete and HF download/import live in the toolbar model library; Settings → AI Providers keeps the provider registry and Ollama endpoint only.
- Consolidated `internal/mcp` package (removed `internal/mcp_disabled` duplicates); dynamic tool list in agent prompts from `ListTools()`.

## [1.0.0-beta.7] - 2026-05-18

### Added
- **Collaboration completion closure** — `/complete-collab` (`--force` when tasks are open), `/collab-task-done`, agent `TASK_STATUS:` lines, plan handoff → task sync, desktop **Mark collaboration done** / read-only closed panel, and completion toast + channel banner.
- **Chat file attachments** — drag-and-drop, paste, and 📎 picker on the message composer attach text files as `prompt_attachments` context (Tauri reads absolute paths from Finder; images still use 📷 / vision). Send with files only (no message body required).
- **Collaboration assets root** — configurable parent directory for execution sandboxes (`collaboration.assets_root` in `~/.neural-junkie/config.json`, Settings → AI Providers, or `NEURAL_JUNKIE_COLLAB_ASSETS_DIR`). Default remains `~/.neural-junkie/collaborations/<collaboration-id>/`.
- **Unified GFM markdown** — chat message bodies use the same `marked` pipeline as file preview and collaboration plans (`RichMarkdownView`, `renderChatMarkdown`); headings, lists, and tables render in the timeline. Fenced code still uses Prism; streaming keeps a lightweight path.
- **RichMarkdownView** — shared component for plans, MD preview, and chat prose segments.
- **Assistant workspace review** — when users ask to review editor content, prompt guidance nudges agents to use open-file workspace context.
- **Tests** — assistant history filter, workspace review heuristics, hub session collab sync and channel dedupe, outbound chat metadata.

### Fixed
- **Chat scroll** — reliable pin-to-bottom during streaming (Virtuoso scroller ref, footer spacer, `streamContentBytes` follow).
- **Session restore** — collaboration discussion transcripts sync into collab channel timelines for scroll/search; channel message dedupe on persist.
- **Assistant** — meeting-note startup uses batch notification (no `#general` flood); filtered history reduces echo; meeting notes dir resolved on load.
- **Collaboration** — max-concurrent error lists active collaborations (phase, channel, task count).

### Changed
- **MarkdownPreview** and **CollaborationPanel** plans use `RichMarkdownView` instead of inline marked wiring.

## [1.0.0-beta.6] - 2026-05-17

### Fixed
- **Collaboration task dispatch** — stop re-sending all `collaboration_task` prompts on every channel message during execution (`TasksDispatched` guard; removed dispatch from `attachCollaborationData`).
- **Collaboration seed noise** — seed “Collaboration Started” messages are internal (`collab_internal_event`); agents no longer burn a turn replying to the seed. Collab agent replies use `collaboration_discussion` for correct UI routing.
- **Hub broadcast** — subscriber channel buffer 512 (was 100) to reduce dropped messages under collab load.
- **Session bloat** — slim `collaboration_data` on WebSocket messages (no nested `discussion.messages`); strip metadata from `last-session.json`; stricter **disk** caps (500 channel / 200 thread messages vs 5000 in-memory); drop history on terminal collab channels; cap persisted collaborations.
- **Session restore** — oversized or corrupt `last-session.json` files are **auto-archived** on startup (no manual cleanup); load limit 64MB.
- **Hub data consent** — first-time modal before agents read workspace metadata; assistant agent respects the same gate.
- **Join dedupe** — desktop suppresses duplicate join/system lines when reconnecting or switching channels.
- **Editor / chat images** — image preview in the editor and inline chat attachments render reliably.

### Added
- **Execution limits** — max 100 agent chat messages per collaboration during executing phase; 3s rate limit on `collaboration_task` replies for hub agents.
- **`scripts/analyze-last-session.sh`** — streaming session file stats.
- **Selective workspace context** — `context_scope` metadata (`none` | `hint` | `outline` | `focus` | `full`); desktop **Auto** mode (default) infers scope per message; composer shows resolved scope; `/collaborate --workspace` opt-in for outline-only project tree during planning (no implicit editor leak).

## [1.0.0-beta.5] - 2026-05-17

### Fixed
- **Collaboration** — DM-spawned agents now subscribe to the collab channel (`EnsureAgentSubscribedToChannel` after `AddAgentToChannel`); join/subscribe and seed/turn failures fail closed with a system message instead of a silent no-op.
- **Cancel / collab UI** — cancel targets the active collaboration channel; clears `activeCollab` and task drawer so the sidebar and composer do not stay locked.
- **Thread streaming** — `stream_delta` / `stream_end` route to thread subscribers (hub `broadcastToThread` + desktop `ThreadPanel`); main chat no longer shows thread-only streams.
- **Chat send** — composer always clears typing on error; send failures surface in the UI.
- **Hub broadcast** — log when a subscriber buffer is full instead of dropping silently.
- **Agent retries** — clear `respondedMessages` when generation fails so a failed turn can be retried.
- **Loading** — single-flight `onReady` on the loading screen avoids duplicate hub connects.
- **Integrations** — GitHub/Confluence “test connection” reports format-only checks honestly (no fake success).
- **`/revise-plan`** — posts to the bound collaboration channel, not the caller’s current channel.
- **Ollama / DeepSeek** — correct chat roles for history; `think` API for reasoning models; collapsible **Reasoning** blocks in the desktop; fewer duplicate DM replies and history echo in prompts.
- **Mermaid** — shared `MermaidCanvas` (sharp SVG fit, macOS zoom smear fix) ported from Dickory Docs.

### Added
- **Editor** — image preview for supported files; file-kind helpers and tests.
- **Tests** — Ollama thinking/roles, chat stream reasoning, hub collab subscribe, file-change fallback.
- **Marketing** — collaboration ad asset and compose script.

### Changed
- **DM agents** — channel discovery disabled only where appropriate; collab rebind ensures subscription on channel switch.

## [1.0.0-beta.4] - 2026-05-16

### Fixed
- **Release CI** — draft-then-publish workflow for immutable GitHub releases; per-platform `gh release upload`; app bundle version `1.0.0` for Windows MSI (tag remains `v1.0.0-beta.4`).

## [1.0.0-beta.1] - 2026-05-16

### Added
- **First public beta installers** — downloadable desktop builds for **macOS** (Apple Silicon + Intel `.dmg`), **Windows** (`.msi`), and **Linux** (`.AppImage` / `.deb`) via GitHub Releases; Go hub ships as a Tauri sidecar (no separate Go install).
- **Download quickstart** — [docs/DOWNLOAD.md](DOWNLOAD.md) for install → wizard → first chat in under five minutes.

### Changed
- **Release CI** — Windows matrix, Go 1.23, rich release notes, prerelease flag for `*-beta*` tags; updater manifest job skipped on beta tags.
- **Marketing site** — landing and README prioritize **Download beta** over clone-only CTAs.

### Includes (since v0.1.1)
- Everything in **0.1.2**, **0.1.3**, and **0.1.4** below: marketing site, port **18765**, collaboration sandbox and smart routing, Ollama model library, slash-command parity, desktop UX polish, and more.

### Notes
- **macOS/Linux:** setup wizard can install/start Ollama. **Windows:** install [Ollama](https://ollama.com) manually or use a cloud API key (in-app Ollama install is not supported on Windows).
- Builds are **unsigned**; macOS may require right-click → **Open** the first time.

## [0.1.4] - 2026-05-14

### Added
- **Ollama model library** — curated catalog (`GET /api/ollama/catalog`, embedded with the hub), browse/search in **Settings → AI Providers**, install with streaming progress, remove installed models (`POST /api/ollama/delete`), and **Use for agents** to set the Ollama provider model plus agent wiring from the desktop.
- **Collaboration smart routing** (optional) — `collaboration.smart_routing_enabled` in config; when on, **collaboration execution tasks** (`collaboration_task` with task metadata) can be answered using a **different configured provider** than the agent’s default, chosen by a **static capability/cost heuristic** (for example vision requirements, simple local-friendly prompts, security-like keywords). Normal channel chat still uses each agent’s configured provider. Applies to **in-process hub agents** only (not standalone `cmd/agent` subprocess specialists unless extended later).
- **Shared AI provider construction and cache** — provider instances built from config are reused and invalidated when providers or the AI block in settings change.

### Changed
- **Hub build and dev commands** — `Makefile` targets run `go build` / `go run` on the `./cmd/server` package so additional server source files (not only `main.go`) compile together.

### Documentation
- **Collaboration and user value guides** — document smart routing behavior and the model library in-repo (`docs/COLLABORATION.md`, `docs/USER_VALUE_GUIDE.md`).

## [0.1.3] - 2026-05-14

### Added
- **Collaboration execution sandbox** — on `/approve-plan`, the hub creates `~/.neural-junkie/collaborations/<id>/` and exposes `working_directory` on collaboration snapshots.
- **Workspace confirmation gate** — `WorkspaceAcknowledged` must be set before `collaboration_task` messages are sent: desktop **Continue** dialog on the collaboration channel, **`POST /api/collaboration-workspace-ack`**, or **`/ack-collab-workspace`**.
- **Command suggestion `cwd`** — detected bash blocks can run with the collaboration sandbox as working directory when executing from the desktop.

### Changed
- **Collaboration execution** — task prompts, workspace context on tasks, and `resume-plan` redispatch respect the workspace gate; `attachCollaborationData` and snapshot heal paths avoid racing task delivery ahead of user confirmation.
- **Agent prompts** — execution phase documents `[FILE_CHANGE]`, workspace fallback, collaboration sandbox path, and shell blocks for **Run**; `CollaborationClient` gains `GetCollaborationWorkingDirectory`.

### Fixed
- Collaboration agents could reply without machine-readable file proposals; executing-phase guidance shares the canonical `[FILE_CHANGE]` block with normal chat.

## [0.1.2] - 2026-05-13

### Added
- **Marketing site** — GitHub Pages content under `docs/`: expanded landing, feature deep-dives, release notes page, early-access banner.
- **Per-channel typing indicators** in the desktop channel sidebar.

### Changed
- **Default hub port** is **18765** (previously 8080); `make start-all` health checks and process management align with `SERVER_PORT`.
- **Slash commands** — real execution with parity enforcement against the hub; command palette metadata refreshes on demand.
- **Collaboration** — workflow hardening across server, desktop UI, and tests; runtime reliability updates; collaboration round counter clamps at configured maximum.

### Fixed
- Drop empty messages from ingestion paths including history reload.
- Hub channel ordering stability; Ollama version surface; auto-register CLI providers when applicable.
- CLI and agent chat rendering when markdown code fences are malformed.
- **Desktop** — migrate saved hub URLs from legacy `localhost:8080` to **18765**.

### Improved
- Hub HTTP/WebSocket surface: security and robustness hardening.
- Desktop UX — dark-theme toasts, accessible toolbar controls, loading and login polish.
- Developer settings — remove non-functional test mode control.

### Removed
- In-hub `/app` screenshot gallery and live-gallery docs (replaced by the static `docs/` site and README assets).

## [0.1.1] - 2026-02-23

### Added -- Multi-Agent Collaboration
- **Collaboration manager** (`internal/collaboration`) for structured multi-agent orchestration
- **Bounded discussion sessions** with round limits, turn budgets, total message caps, and timeout enforcement
- **Collaboration phases**: planning -> reviewing -> approved -> executing -> completed/cancelled
- **Shared plan artifacts** with version history and edit tracking
- **Task delegation model** with per-agent assignment and status tracking
- **Consensus detection** (signal + heuristic) with disagreement escalation
- **New slash commands**: `/collaborate`, `/approve-plan`, `/revise-plan`, `/cancel-plan`, `/collab-status`
- **New message types**: `collaboration_plan`, `collaboration_task`, `collaboration_status`, `collaboration_discussion`

### Added -- Desktop Collaboration UX
- **CollaborationPanel** for phase, participants, tasks, plan artifact, and control actions
- **Collaboration message rendering** in chat with collaboration-specific visual cues
- **TypeScript protocol updates** for collaboration entities and metadata helpers

### Added -- Test Coverage
- Added `test/collaboration_test.go` covering lifecycle, bounded discussion logic, consensus, task tracking, artifact versioning, and extraction parsing

## [0.1.0] - 2026-02-20

First packaged release -- Neural Junkie ships as a single distributable desktop app.

### Added -- Desktop Packaging
- **Tauri sidecar architecture** -- Go server bundled inside the Tauri app, launched and managed automatically
- **First-run Setup Wizard** -- guided onboarding to choose AI backend (Ollama or cloud), configure providers, and enable agents
- **Auto-update system** -- in-app update banner with download progress and one-click restart via Tauri updater
- **Loading screen** -- server health polling with status feedback during startup

### Added -- AI Provider Registry
- **Dynamic provider management** -- add, edit, remove, and test AI providers from Settings UI
- **OpenAI-compatible provider** -- generic adapter for any OpenAI-compatible API (Amazon Q, Azure OpenAI, Together AI, Groq, etc.)
- **Provider Manager UI** -- full CRUD interface with connection testing
- **Multi-provider support** -- use multiple cloud and local providers simultaneously, assign per-agent

### Added -- Ollama Lifecycle Management
- **Automatic detection** -- detect Ollama installation on macOS and Linux
- **Install from app** -- install Ollama directly from the Setup Wizard or Settings
- **Server management** -- start/stop Ollama server from the UI
- **Model pulling** -- pull models with real-time progress streaming (SSE)
- **Ollama Manager UI** -- dedicated panel in Settings for full Ollama control

### Added -- Configuration System
- **JSON config file** -- persistent configuration at `~/.neural-junkie/config.json`
- **Environment variable migration** -- auto-migrates from `env.local` to config file on first load
- **API key redaction** -- API keys masked in GET responses, preserved on PUT if masked
- **Per-agent provider assignment** -- each agent type can use a different provider

### Added -- CI/CD & Release
- **GitHub Actions release workflow** -- triggered on `v*` tags, builds macOS (arm64 + x86_64) and Linux (x86_64)
- **Cross-compilation matrix** -- Go server compiled for each target, bundled as Tauri sidecar
- **Update manifest generation** -- auto-generates platform-specific JSON manifests for Tauri auto-updater
- **`make release` target** -- bumps versions, commits, and tags in one command

### Added -- CLI Agent Infrastructure
- **CLI agent registry** -- persistent storage for CLI agent configurations
- **CLI agent storage** -- JSON-based persistence for registered CLI agents

### Improved -- UI
- **Terminal panel** -- refactored with XTerminal component
- **Markdown rendering** -- improved code block handling and mermaid diagram support
- **Suggestion banner** -- contextual suggestions in the chat UI
- **Chat window** -- enhanced layout and interaction patterns

---

## Pre-0.1 development — February 2026

> **Not a GitHub semver tag.** This block records the **Neural Junkie** rename and Tauri + React workspace before the first packaged release (**v0.1.0**). It was previously titled `[2.0.0]` as an informal “second generation” note; **do not** read that as a release line above **0.1.x**.

### Renamed
- Project renamed from "AI Chat Room" to **Neural Junkie**
- Go module: `github.com/camronwood/neural-junkie`
- Data directory: `~/.neural-junkie/`
- Tauri bundle: `com.camronwood.neuraljunkie`

### Added -- Desktop App
- **Tauri + React desktop app** replacing the old Fyne GUI
- Slack-inspired UI with dark theme and modern styling
- **Command Palette** -- searchable command UI with guided argument forms (toolbar button, with slash-form command transport compatibility)
- **File Explorer Panel** -- browse and open workspace files
- **Code Editor Panel** -- view and edit code from the app
- **Terminal Panel** -- embedded terminal output
- **Thread Panel** -- threaded conversation view
- **Pending Changes Panel** -- review file change proposals with diff preview
- **Settings Modal** -- Appearance, Layout, Integrations (Anthropic, GitHub, Confluence), AI Providers (Ollama, LM Studio, Claude), Developer, About
- **@Mention Autocomplete** -- agent picker with fuzzy matching
- **Mermaid Diagram Rendering** -- inline diagram support in messages
- **Layout Persistence** -- panel visibility saved across sessions

### Added -- Agents
- **Moderator Agent** -- auto-starts with server, guides users through commands and features, 20s safety-net for unanswered questions
- **Assistant Agent** -- reminders (one-time/recurring), tasks (priority, due dates), notes (tags, search), meeting summaries, scheduling; persistent storage
- **Confluence Agent** -- index Confluence Cloud spaces, search documentation, answer knowledge-base questions
- **Helper Agents** -- template-based custom experts (day-one onboarding, testing, docs)
- **Cursor CLI Agent** -- Cursor CLI subprocess for code analysis and generation
- **Agent Review** -- get second opinions by @mentioning another agent in a thread reply

### Added -- AI Providers
- **Ollama** -- local inference with model listing, connection testing, configurable endpoint
- **LM Studio** -- local OpenAI-compatible server with model listing and connection testing
- **Per-agent provider switching** -- change provider/model for individual agents at runtime
- **Global provider switching** -- switch all agents to a provider with one command
- Two-tier model config: code tier (qwen2.5-coder:14b) and utility tier (qwen2.5:7b)

### Added -- Features
- **50+ slash commands** with metadata for command palette
- **File Change System** -- agents propose file edits, users approve/reject with diff preview and backup
- **MCP Export/Import** -- export agent knowledge to MCP format, import from MCP, MCP resource server
- **Workspace management** -- add, list, remove workspaces
- **Thread support** -- create threads, reply in threads, thread metadata and subscriptions
- **Session persistence** -- periodic save and recovery
- **Connection testing** -- test Anthropic, Ollama, LM Studio, GitHub, Confluence connections from UI
- **Design analysis** -- `/analyze-design` command for UI review

### Improved
- Three-layer message deduplication (polling, handler, agent-type filtering)
- Repository agent caching with staleness detection and incremental reindex
- File watching for auto-reindex on codebase changes
- Agent pause/unpause and remove/recall lifecycle management

## [1.0.0] - 2025-10-14 — AI Chat Room (legacy name)

### Added
- Core hub server with WebSocket real-time communication
- Multi-channel conversation support
- 5 specialized agent types: Frontend, Backend, DevOps, Database, Security
- Repository Expert Agents with codebase indexing and search
- Claude AI integration (Anthropic API and AI Hub)
- Mock AI provider for testing
- Fyne-based desktop GUI (since replaced by Tauri + React)
- Interactive terminal chat client
- Built-in web UI
- CLI tool for automation
- @mention system for targeting agents
- Message history and context

### Fixed
- Message deduplication (agents responding multiple times)
- GUI threading issues (Fyne thread safety)
- Username display (was showing "Human User")

---

For current status, see [STATUS.md](STATUS.md).
