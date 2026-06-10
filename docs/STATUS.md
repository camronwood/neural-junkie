# Project Status

**Last Updated:** June 2026

## Current State: Open Beta → Stable (v1.0)

Neural Junkie is a working multi-agent workspace — local-first desktop app, Slack integration, domain packs, and bounded collaboration. **Latest tagged build:** [v1.0.0-beta.33](https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.33). **Stable cut:** follow [STABLE_RELEASE_CHECKLIST.md](STABLE_RELEASE_CHECKLIST.md) and [STABLE_SCOPE.md](STABLE_SCOPE.md).

**Marketing site:** [camronwood.github.io/neural-junkie](https://camronwood.github.io/neural-junkie/)

## Working Features

### Core System
- WebSocket-based real-time communication
- Multi-channel support with message history (bounded per channel)
- Agent registration, presence tracking, and lifecycle management
- Thread support (create, reply, subscribe)
- 50+ command actions with command palette UI and slash-form transport compatibility
- Session persistence and recovery
- File change proposal and approval workflow
- Workspace management with quick switcher (beta.17+)
- Context model v2 — Chat/Code composer, turn intent, thread-scoped history (beta.21+)
- In-channel message find bar (beta.21+)
- User rules API (`GET`/`PUT` `/api/user-rules`)

### Agent Types
- **Moderator** — Auto-started, chat guidance, command help, safety-net timer
- **Assistant** — Reminders, tasks, notes, meetings, scheduling, workspace grounding
- **Custom experts** — Any domain via `/create-expert` or DM
- **Engineering specialists** (Software development pack) — Backend, frontend, platform, security, architecture, code review
- **Domain pack experts** — BiologyExpert, CADExpert, etc. per enabled pack
- **Repository Expert** — Codebase indexing, file watching, project-specific Q&A
- **Confluence Agent** — Confluence Cloud space indexing and documentation search
- **CLI agents** — 12 auto-detected types (Cursor, Claude, Gemini, Copilot, Codex, Aider, …)

### AI Providers
- **Bundled Ollama (beta.22)** — Runtime shipped in installers; auto-start on launch
- **Model library** — Curated Ollama + Hugging Face catalog in desktop toolbar
- **Claude** — Anthropic API direct or via AI Hub proxy
- **LM Studio** — Local OpenAI-compatible server
- **OpenAI-compatible** — Groq, Together, Azure, Amazon Q, etc.
- Per-agent provider switching, global provider switching, optional collaboration smart routing (execution tasks only)

### Domain Packs (official)
- **Software development** — IDE v2/v3, Git in app, implementation sessions
- **Life sciences** — OpenBioLLM, sequence tools, scan summary viewer
- **CAD** — OpenSCAD workbench, CADExpert, STL preview
- **Specialist tuning** — LoRA training, personal learning v2
- **Pack store** — Install from GitHub; **Pack Dev Studio** for custom/customer packs

### Integrations
- **Slack Connect** — Channel bind, two-way threads, bundled OAuth
- **Slack inbox & forwarding (beta.21)** — Mobile DM to bot, selective channel forward, away-mode human DMs
- **Confluence Cloud** — Space indexing, page search, documentation Q&A
- **MCP Export/Import** — Export agent knowledge to MCP format for sharing
- **Google Meet notes** — Assistant meeting ingest (bundled OAuth in release builds)

### Collaboration
- Phases: planning → review → approved → executing → completed/cancelled
- Runbook builder with graph view and HTTP action nodes
- Git worktree execution mode (`--worktree`)
- Workspace confirmation gate before task dispatch
- Personal learning scoped to collaboration when enabled

### User Interfaces
- **Desktop App** — Tauri + React; command palette, file explorer, code editor, terminal, threads, pending changes, collaboration panel
- **Web UI** — Lightweight chat client at `/` (not full workspace)
- **Terminal Chat** — Interactive WebSocket-based CLI
- **CLI Tool** — Scripting, automation, MCP resource server

## Performance

- Message latency: < 500ms end-to-end (typical local hub)
- Tested with 10+ concurrent agents
- Stable memory with built-in cache cleanup (100 messages per channel in-memory)
- Repository index caching with staleness detection
- Assistant state refresh: 30s while task panel open; markdown preview poll: 8s

## Test Coverage

- Unit tests across core packages
- Integration tests for message flow, commands, deduplication
- Live scenario harnesses: `make chat-scenarios-regression`, `make collab-smoke`, collab matrix
- Agent-specific tests (repo, expert, assistant, moderator, hub, review)

## Known Limitations

**Public tracker:** [KNOWN_ISSUES.md](KNOWN_ISSUES.md) (repo) and [known-issues.html](known-issues.html) (marketing site). Remove items there when fixed.

- **Hub-local persistence** — Bounded message history; not a full durable archive
- **Single server** — No distributed deployment
- **Session auth** — Optional `NEURAL_JUNKIE_AUTH_REQUIRED=1`; see [SECURITY.md](SECURITY.md)
- **Domain packs** — Multiple packs can be enabled; first enabled pack owns UI layout
- **Collaboration variance** — Local models vary in plan quality; see known issues for active scenario gaps
- **IDE v3** — Cursor-like Ask/Agent routing in IDE layout; see [IDE_V3.md](IDE_V3.md)
- **macOS releases** — Ad-hoc signed CI builds at v1.0.0 (Right-click → Open if Gatekeeper warns); Developer ID + notarized in v1.0.1 when Apple creds available; local dev builds remain ad-hoc

## Documentation

See the [README](../README.md) for the full documentation index and [DOCS.md](../DOCS.md) for a compact map of all guides.
