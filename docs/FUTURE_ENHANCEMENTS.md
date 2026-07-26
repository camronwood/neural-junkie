# Future Enhancements

Planned improvements and feature ideas for Neural Junkie.

Last Updated: July 2026

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
- ~~Agent-to-WebSocket migration~~ -- Standalone agents use hub WebSocket by default (`GET /api/agents/ws`); HTTP polling via `--poll`
- ~~Git operations from chat~~ -- Agents emit `[GIT_CHANGE]` blocks; desktop **Pending changes** approval for stage/commit/push
- ~~API keys & roles~~ -- `nj_…` service keys, admin/member/viewer roles, Settings → Security
- ~~Message archive search~~ -- FTS5 + `GET /api/messages/search`, desktop channel find bar, cursor pagination
- ~~Unified knowledge router (MVP)~~ -- Rules-based classifier in `internal/routing/`, wired into agent turn pipeline + routing trace
- ~~Remote sidecar bootstrap~~ -- `scripts/install-nj-remote.sh`, `make nj-remote-install`, systemd + launchd templates

## High Priority

### Authentication & Authorization (follow-ups)

**Shipped:**
- Session tokens (`POST /api/auth/session`) and channel ACLs — [SECURITY.md](SECURITY.md)
- API keys (`POST /api/auth/api-keys`) persisted in `~/.neural-junkie/auth.db`
- Roles: admin, member, viewer
- Settings → **Security** hub status + API key management
- `--api-key` / `NEURAL_JUNKIE_API_KEY` on standalone agents

**Backlog:**
- Agent registration approval
- SSO / JWT for enterprise deployments

### Database Persistence (partially shipped)

**Shipped:**
- SQLite message store (`~/.neural-junkie/messages.db`)
- SQLite conversation memory (`~/.neural-junkie/memory.db`)
- Durable channels + channel history export
- Full-text search + pagination (D2)

**Follow-ups:**
- Optional PostgreSQL for shared deployments

## Medium Priority

### Multi-Repository Agents
Single agent that understands multiple related repositories:
- Cross-repo dependency tracking — **partial** (lightweight `CROSS_REPO_HINTS` from go.mod/package.json/docker-compose)
- Monorepo workspace support — **partial** (active + linked workspace scope; sub-root via repo agent path)
- Unified search across repos — **shipped** (`repo_paths[]` semantic search + multi-repo `@codebase`)

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
- **Scenario archetype demos** — “seven paths” narrative for marketing/onboarding (closure / memory / code / delegation / collab light / collab deep)

### Share Agent (gift bundle for friends / coworkers)

**Today:** Repo agents can MCP export/import ([MCP_EXPORTS.md](MCP_EXPORTS.md)), but import **re-indexes from the absolute repo path** in the JSON — embedded resources are mostly for MCP consumers, not live agent hydration. Learnings, custom rules, and LoRA tags travel on separate paths (or not at all). Specialists/pack agents are not exportable. UX is slash-command oriented.

**Idea:** One **Share Agent** action that packages an agent so a friend or coworker can import it on their hub and **retain its knowledge** without needing the same absolute path.

**Target bundle (`.nj-agent` or extended MCP export):**
- Identity — name, type, expertise, system prompt, custom rules
- Knowledge — self-contained index/resources (hydrate-from-bundle, not only re-index-from-path)
- Learnings — optional agent-scoped confirmed personal learnings
- Behavior pointer — LoRA HF id / compose tag when present (populate the existing unused `lora` export field)

**UX:** Agent Info → **Share** → download file; friend → **Import Agent** (file picker) → remap workspace path *or* knowledge-only mode from the bundle.

**Phased sketch:**
1. Hydrate import from embedded resources + path remap UI (repo agents)
2. Include agent-scoped learnings + custom rules in the bundle
3. Write LoRA metadata on export; optional “pull HF adapter” on import
4. Extend beyond repo agents (specialists / pack experts) where knowledge is portable

Not real-time multi-hub sync — portable file handoff (git, AirDrop, Slack, etc.). See also Context Stack sharing notes in [CONTEXT_MODEL.md](CONTEXT_MODEL.md).

### Rate Limiting & Cost Management
- Per-agent API cost tracking
- Budget alerts and limits
- Response caching for repeated queries
- Token usage monitoring

### MCP Tool Servers
- ~~Re-enable Backend/DevOps/Database MCP tool servers~~ — **Done** (beta.8)
- ~~Frontend, Security, Code Review, Architecture, Rust MCP servers~~ — **Done**
- ~~Repo and Confluence in-process runtime search tools~~ — **Done**
- ~~Tool calling for LM Studio / OpenAI-compat providers~~ — **Done** (native OpenAI Chat Completions tool loop + ReAct fallback when native unsupported)

### MCP Tool Wizard (user-defined tools → agents)

**Today:** Agents get tools from first-party in-process MCP servers and pack sidecars ([MCP_INTEGRATION.md](MCP_INTEGRATION.md)). Custom experts (`/create-expert`) are persona/rules only — no tool loop. Users cannot register an arbitrary MCP server or attach a home-grown tool to an agent. Closest workaround: built-in `fetch_url` / WebBrowserExpert, or author a pack sidecar + core wiring.

**Idea:** A desktop **MCP Tool Wizard** that lets users create (or connect) a tool and grant access to chosen agents — e.g. “read this page/API from my website and return structured data.”

**UX sketch:**
1. **Create / Connect** — wizard steps: name, description, input schema, implementation (HTTP fetch template, script stub, or “connect existing MCP server URL/stdio”)
2. **Test** — run the tool once with sample args; preview output in the wizard
3. **Grant access** — pick agents (custom experts, specialists, “all in this channel”) that may call it
4. **Use** — tool appears in those agents’ MCP catalogs; Context Stack / tool loop already know how to execute approved tools

**Phased sketch:**
1. Hub MCP-client registry — Settings: add remote MCP server (URL / stdio); enable for selected agents
2. HTTP-fetch tool template in the wizard (public URL, headers, JSON path extract) — covers the “read my website” case without writing Go
3. Attach tools to `AgentTypeExpert` (custom experts get a real tool loop when grants exist)
4. Script/sidecar stubs + optional export of user tools into Share Agent / pack bundles

Security stays local-first: user approval for sensitive tools, same private-IP / consent gates as `fetch_url` where applicable. Complements pack MCP sidecars — packs ship domain tools; the wizard is for personal/team one-offs.

See also Plugin System (below) and [MCP_INTEGRATION.md](MCP_INTEGRATION.md#future-enhancements).

## Low Priority

### Mobile Companion App (reference)

Exploratory future-build notes for a separate phone-native NJ companion:

- small on-device model
- offline-first chat and personal workflows
- explicit local sync with desktop NJ as the source of truth

See [MOBILE_COMPANION_NOTES.md](MOBILE_COMPANION_NOTES.md). This is a reference/design note, not a scheduled roadmap commitment.

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
- **MCP Tool Wizard** — user-created/connected tools with per-agent grants (see Medium Priority above)

### Agent Memory & Learning
- Long-term context retention across sessions
- Learning from conversation patterns
- Personalized responses based on user history

### Analytics Dashboard
- Agent performance metrics (response time, quality)
- Usage patterns and popular topics
- Cost breakdown by agent and provider
- Repository insights (most-asked-about files, confusion points)
