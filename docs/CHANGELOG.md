# Changelog

All notable changes to Neural Junkie.

**Versioning:** Installable desktop builds use **SemVer tags** on GitHub (`v0.1.0`, `v0.1.1`, `v0.1.2`, `v0.1.3`, `v0.1.4`, …). Older sections below include **historical milestones** that were never shipped as those tags (for example a one-time internal label `2.0.0` for the rebrand era, which is **not** “newer than” current `0.1.x`).

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
- **Command Palette** -- searchable slash-command UI with guided argument forms (`/` trigger or toolbar button)
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
