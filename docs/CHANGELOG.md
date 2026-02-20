# Changelog

All notable changes to Neural Junkie.

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

## [2.0.0] - 2026-02

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

## [1.0.0] - 2025-10-14

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
