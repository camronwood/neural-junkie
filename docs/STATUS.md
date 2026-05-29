# Project Status

**Last Updated:** May 2026

## Current State: Active Development

Neural Junkie is a working multi-agent collaboration system used for daily development workflows.

## Working Features

### Core System
- WebSocket-based real-time communication
- Multi-channel support with message history
- Agent registration, presence tracking, and lifecycle management
- Thread support (create, reply, subscribe)
- 50+ command actions with command palette UI and slash-form transport compatibility
- Session persistence and recovery
- File change proposal and approval workflow
- Workspace management

### Agent Types (10)
- **Moderator** -- Auto-started, chat guidance, command help, safety-net timer
- **Assistant** -- Reminders, tasks, notes, meetings, scheduling (persistent storage)
- **Frontend** (FrontendEngineer) -- web/desktop UI, accessibility, design systems, visual QA
- **Backend** (BackendEngineer) -- APIs, services, integrations, business logic, performance
- **Platform** (PlatformEngineer) -- deployment, CI/CD, cloud infrastructure, observability
- **Security** (SecurityReviewer) -- auth, encryption, threat modeling, OWASP, compliance
- **Architecture** (SoftwareArchitect) -- system design, service boundaries, migrations
- **Code Review** (CodeReviewer) -- correctness, tests, maintainability, regressions
- **Repository Expert** -- Codebase indexing, file watching, project-specific Q&A
- **Confluence Agent** -- Confluence Cloud space indexing and documentation search
- **Cursor CLI Agent** -- Cursor CLI integration for code analysis

### AI Providers
- **Ollama** -- Local inference, model listing, connection testing
- **Claude** -- Anthropic API direct or via AI Hub proxy
- **LM Studio** -- Local OpenAI-compatible server
- **Mock** -- Rule-based responses for testing
- Per-agent provider switching, global provider switching

### Desktop performance
- Assistant state refresh while the task panel is open: every 30s (reduced from 10s).
- Markdown preview polling for the active file: every 8s (reduced from 2s).
- Parsed markdown parts for large messages are LRU-cached to avoid repeat work when messages re-render.

### User Interfaces
- **Desktop App** -- Tauri + React + TypeScript with Tailwind CSS
  - Command palette with Cmd+Shift+P/Ctrl+Shift+P access, search, and argument forms
  - File explorer, code editor, terminal panel
  - Thread panel, pending changes panel
  - Settings modal (appearance, layout, integrations, AI providers)
  - @mention autocomplete, Mermaid diagram rendering
- **Web UI** -- Built-in HTML chat client served by the hub (`/`)
- **Terminal Chat** -- Interactive WebSocket-based CLI
- **CLI Tool** -- Scripting, automation, MCP resource server

### Integrations
- **Confluence Cloud** -- Space indexing, page search, documentation Q&A
- **MCP Export/Import** -- Export agent knowledge to MCP format for sharing

## Performance

- Message latency: < 500ms end-to-end
- Tested with 10+ concurrent agents
- Stable memory with built-in cache cleanup (100 messages per channel)
- Repository index caching with staleness detection

## Test Coverage

- Unit tests across core packages
- Integration tests for message flow, commands, deduplication
- Agent-specific tests (repo, helper, assistant, moderator, hub, review)
- Architecture and thread-safety tests

## Known Limitations

- **Hub-local persistence** -- Session metadata, channels, and agent registrations restore from `last-session.json`; per-channel message history is bounded and pruned over time (not a full durable message archive).
- **Single server** -- No distributed deployment
- **Session auth** -- Optional `NEURAL_JUNKIE_AUTH_REQUIRED=1` and channel ACLs; see [SECURITY.md](SECURITY.md)
- **Domain packs** -- Installable plugins from Pack store; multiple packs can be enabled; first enabled pack owns UI layout
- **Agent polling** -- Standalone `cmd/agent` processes use HTTP polling; in-process runtime agents use hub push delivery
- **Git endpoints** -- Require Software development pack; need `git` on PATH and a git workspace
- **IDE v2/v2c** -- IDE layout preset, symbol index, Rust/Python diagnostics, inline completion, plus v2a/v2b features. See [IDE_V2.md](IDE_V2.md)
- **IDE v3** -- IDE layout + main chat routing (Ask/Agent, @codebase, auto specialist), review bar, editor trust. See [IDE_V3.md](IDE_V3.md)

## Documentation

See the [README](../README.md) for the full documentation index and [DOCS.md](../DOCS.md) for a compact map of all guides.
