# Future Enhancements

Planned improvements and feature ideas for Neural Junkie.

Last Updated: June 2026

## Implemented (Moved from Previous Roadmap)

These items from the original roadmap have been completed:

- ~~Change detection and incremental reindexing~~ -- Repo agents support file watching and incremental reindex
- ~~MCP integration~~ -- MCP export/import system for agent knowledge sharing
- ~~Threaded conversations~~ -- Full thread support with replies and subscriptions
- ~~Code snippets and syntax highlighting~~ -- Markdown rendering with code blocks in desktop app
- ~~Agent mentions with autocomplete~~ -- @mention system with fuzzy matching and UI autocomplete
- ~~Enhanced agent status display~~ -- Agent list with status indicators in desktop app
- ~~GitHub integration~~ -- GitHub CLI operations (issues, PRs, repos, workflows)
- ~~Agent Teams & Delegation~~ -- Multi-agent collaboration system with bounded discussion, planning/approval/execution phases, shared artifacts, task assignment, and consensus detection

## High Priority

### Agent-to-WebSocket Migration
Move agents from HTTP polling to WebSocket connections for lower latency and reduced server load. See [PLATFORM_ROADMAP.md](PLATFORM_ROADMAP.md) (Phase 3).

### Git Operations (agents)
Hub git REST endpoints are implemented for the desktop Git modal (Software development pack). Remaining work: agent-initiated git operations with approval workflows from chat.

### Authentication & Authorization (Phase 3 — automation / multi-user)

**Shipped for v1.0 (local-first):**
- Session tokens (`POST /api/auth/session`) and channel ACLs — [SECURITY.md](SECURITY.md)
- Optional strict mode: `NEURAL_JUNKIE_AUTH_REQUIRED=1`
- Hub token for non-loopback: `NEURAL_JUNKIE_HUB_TOKEN`
- Settings → **Security** hub status panel

**Phase 3 backlog:**
- JWT/API key auth for automation and service accounts
- User roles (admin, member, viewer)
- Agent registration approval

### Database Persistence (partially shipped)

**Shipped:**
- SQLite message store (`~/.neural-junkie/messages.db`)
- SQLite conversation memory (`~/.neural-junkie/memory.db`)
- Durable channels + channel history export

**Phase 2–3 follow-ups:**
- Full-text in-app search across archived history
- Optional PostgreSQL for shared deployments
- Pagination for very large channels

## Medium Priority

### Multi-Repository Agents
Single agent that understands multiple related repositories:
- Cross-repo dependency tracking
- Monorepo workspace support
- Unified search across repos

### Semantic Code Search
Go beyond text matching for repository agents:
- Understand code intent and functionality
- Find similar patterns across the codebase
- Identify anti-patterns and tech debt

### Advanced Collaboration Orchestration
Collaboration is implemented. Future improvements can focus on:
- Smarter dynamic task rebalancing during execution
- Better cross-collaboration dependency management
- Optional voting strategies beyond current consensus heuristics

### Adaptive orchestration (reference)
See [ADAPTIVE-ORCHESTRATION-NOTES.md](ADAPTIVE-ORCHESTRATION-NOTES.md) — maps external “adaptive intelligence” / per-request routing framing to the Context Stack and collab mesh.

Possible follow-ups (not scheduled):
- **Unified knowledge router** — classify question shape → conversation memory vs codebase vs collab graph vs prior-reference (today split across subsystems)
- **Per-turn routing trace** — dev overlay or debug panel: intent, tier, retrieval mode, governance gates (data largely exists in logs and debug endpoints)
- **Scenario archetype demos** — “seven paths” narrative for marketing/onboarding (closure / memory / code / delegation / collab light / collab deep)

### Rate Limiting & Cost Management
- Per-agent API cost tracking
- Budget alerts and limits
- Response caching for repeated queries
- Token usage monitoring

### MCP Tool Servers
- ~~Re-enable Backend/DevOps/Database MCP tool servers~~ — **Done** (beta.8)
- ~~Frontend, Security, Code Review, Architecture, Rust MCP servers~~ — **Done**
- ~~Repo and Confluence in-process runtime search tools~~ — **Done**
- Tool calling for LM Studio / OpenAI-compat providers (Ollama supported for BiologyExpert + MCP in v1; Claude also supported)

## Low Priority

### Distributed Deployment
- Redis Pub/Sub for message routing across instances
- Load balancing for horizontal scaling
- Shared state via etcd or Consul

### IDE Integration
- VS Code extension for in-editor agent access
- JetBrains plugin
- Neovim integration

### CI/CD Integration
- Agent responses in PR comments
- Automated code review on push
- Build failure analysis

### Plugin System
- Custom analyzers for specific languages/frameworks
- User-defined slash commands
- Integration hooks for external tools

### Agent Memory & Learning
- Long-term context retention across sessions
- Learning from conversation patterns
- Personalized responses based on user history

### Analytics Dashboard
- Agent performance metrics (response time, quality)
- Usage patterns and popular topics
- Cost breakdown by agent and provider
- Repository insights (most-asked-about files, confusion points)
