# Project Status

**Last Updated:** February 2026

## Current State: Active Development

Neural Junkie is a working multi-agent collaboration system used for daily development workflows.

## Working Features

### Core System
- WebSocket-based real-time communication
- Multi-channel support with message history
- Agent registration, presence tracking, and lifecycle management
- Thread support (create, reply, subscribe)
- 50+ slash commands with command palette UI
- Session persistence and recovery
- File change proposal and approval workflow
- Workspace management

### Agent Types (11)
- **Moderator** -- Auto-started, chat guidance, command help, safety-net timer
- **Assistant** -- Reminders, tasks, notes, meetings, scheduling (persistent storage)
- **Frontend** (ReactExpert) -- React, Vue, Angular, TypeScript, CSS, UI/UX, vision
- **Backend** (GoExpert) -- Go, Node, Python, REST/GraphQL/gRPC, microservices
- **DevOps** (DevOpsPro) -- Docker, K8s, CI/CD, AWS/GCP/Azure, Terraform
- **Database** (SQLMaster) -- PostgreSQL, MySQL, MongoDB, Redis, schema, optimization
- **Security** (SecurityExpert) -- Auth, OAuth/JWT, encryption, OWASP, compliance
- **Repository Expert** -- Codebase indexing, file watching, project-specific Q&A
- **Confluence Agent** -- Confluence Cloud space indexing and documentation search
- **Helper Agent** -- Template-based custom knowledge experts
- **Cursor CLI Agent** -- Cursor CLI integration for code analysis

### AI Providers
- **Ollama** -- Local inference, model listing, connection testing
- **Claude** -- Anthropic API direct or via AI Hub proxy
- **LM Studio** -- Local OpenAI-compatible server
- **Mock** -- Rule-based responses for testing
- Per-agent provider switching, global provider switching

### User Interfaces
- **Desktop App** -- Tauri + React + TypeScript with Tailwind CSS
  - Command palette with search and argument forms
  - File explorer, code editor, terminal panel
  - Thread panel, pending changes panel
  - Settings modal (appearance, layout, integrations, AI providers)
  - @mention autocomplete, Mermaid diagram rendering
- **Web UI** -- Built-in HTML served by hub server
- **Terminal Chat** -- Interactive WebSocket-based CLI
- **CLI Tool** -- Scripting, automation, MCP resource server

### Integrations
- **Dispatch CLI** -- kubectl, aws, docker, sops, workstation with approval workflows
- **GitHub CLI** -- Issues, PRs, repos, workflows via dispatch
- **Confluence Cloud** -- Space indexing, page search, documentation Q&A
- **MCP Export/Import** -- Export agent knowledge to MCP format for sharing

## Performance

- Message latency: < 500ms end-to-end
- Tested with 10+ concurrent agents
- Stable memory with built-in cache cleanup (100 messages per channel)
- Repository index caching with staleness detection

## Test Coverage

- Unit tests across core packages
- Integration tests for message flow, commands, dispatch, deduplication
- Agent-specific tests (repo, helper, assistant, moderator, hub, review)
- Architecture and thread-safety tests

## Known Limitations

- **No persistence** -- Messages lost on server restart (session file preserves some state)
- **Single server** -- No distributed deployment
- **No auth** -- Open access to all endpoints
- **Agent polling** -- Agents use HTTP polling, not WebSocket
- **Git endpoints** -- `git-status`, `git-diff`, `git-commit`, `git-push`, `git-pull` return 501 (not yet implemented)

## Documentation

See the [README](../README.md) for the full documentation index.
