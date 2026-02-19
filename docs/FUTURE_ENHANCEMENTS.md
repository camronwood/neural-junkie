# Future Enhancements

Planned improvements and feature ideas for Neural Junkie.

Last Updated: February 2026

## Implemented (Moved from Previous Roadmap)

These items from the original roadmap have been completed:

- ~~Change detection and incremental reindexing~~ -- Repo agents support file watching and incremental reindex
- ~~MCP integration~~ -- MCP export/import system for agent knowledge sharing
- ~~Threaded conversations~~ -- Full thread support with replies and subscriptions
- ~~Code snippets and syntax highlighting~~ -- Markdown rendering with code blocks in desktop app
- ~~Agent mentions with autocomplete~~ -- @mention system with fuzzy matching and UI autocomplete
- ~~Enhanced agent status display~~ -- Agent list with status indicators in desktop app
- ~~GitHub integration~~ -- GitHub CLI operations (issues, PRs, repos, workflows)

## High Priority

### Agent-to-WebSocket Migration
Move agents from HTTP polling to WebSocket connections for lower latency and reduced server load.

### Git Operations
The server has stub endpoints for `git-status`, `git-diff`, `git-commit`, `git-push`, `git-pull` (currently returning 501). Implement these to enable agents to perform git operations with approval workflows.

### Authentication & Authorization
- JWT/API key auth for all endpoints
- Channel-level access control
- User roles (admin, member, viewer)
- Agent registration approval

### Database Persistence
Replace in-memory message storage with a database backend:
- PostgreSQL or SQLite for message history
- Survive server restarts
- Searchable message archive
- Pagination for large histories

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

### Agent Teams & Delegation
Coordinated groups of agents:
- Coordinator agent that delegates to specialists
- Shared context between team members
- Consensus mechanisms for conflicting advice

### Rate Limiting & Cost Management
- Per-agent API cost tracking
- Budget alerts and limits
- Response caching for repeated queries
- Token usage monitoring

### MCP Tool Servers (Re-enable)
The MCP tool servers for Backend, DevOps, and Database agents are implemented but currently disabled. Re-enable with:
- Go code analysis and testing tools
- Kubernetes query and Docker analysis tools
- SQL explain, index analysis, and schema validation tools

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
