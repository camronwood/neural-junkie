# Changelog

All notable changes to Neural Junkie.

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
