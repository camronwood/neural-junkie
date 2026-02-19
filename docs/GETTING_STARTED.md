# Getting Started

Get Neural Junkie running in under 5 minutes.

## Prerequisites

- **Go 1.23+** -- [go.dev/dl](https://go.dev/dl)
- **Node.js 18+** -- [nodejs.org](https://nodejs.org)
- **Rust** -- [rustup.rs](https://rustup.rs) (for the Tauri desktop app)
- At least one AI provider:
  - **Ollama** (local, free) -- [ollama.ai](https://ollama.ai)
  - **Claude** (API key required) -- [anthropic.com](https://www.anthropic.com)
  - **LM Studio** (local, free) -- [lmstudio.ai](https://lmstudio.ai)

## Quick Start

### Option 1: Everything at Once

```bash
cd neural-junkie
make gui-install    # First time only -- installs npm + Rust deps
make start-all     # Starts server, agents, and desktop app
```

This launches:
- The **Hub server** on `http://localhost:8080`
- **Moderator** and **Assistant** agents (auto-started with the server)
- 5 **specialist agents**: GoExpert, SQLMaster, SecurityExpert, ReactExpert, DevOpsPro
- The **Tauri desktop app**

### Option 2: Manual Setup (Separate Terminals)

**Terminal 1 -- Server:**
```bash
make server
```

**Terminal 2 -- Agents:**
```bash
make agents
```

**Terminal 3 -- Interface:**
```bash
make gui          # Desktop app (recommended)
# OR
make chat         # Terminal chat
# OR
open http://localhost:8080   # Web UI
```

## AI Provider Configuration

### Ollama (Local, Recommended for Getting Started)

Ollama runs models locally with no API key required.

```bash
# Install Ollama from https://ollama.ai, then:
make pull-models
```

This pulls two model tiers:
- **Code tier** (`qwen2.5-coder:14b`, ~9GB) -- used by specialist agents
- **Utility tier** (`qwen2.5:7b`, ~4.5GB) -- used by Moderator and Assistant

The `make agents` target automatically uses these models via the `OLLAMA_CODE_MODEL` env var.

### Claude (Anthropic API)

```bash
cp env.example env.local
```

Edit `env.local`:
```bash
USE_AI_HUB=false
ANTHROPIC_API_KEY=sk-your-key-here
```

Then load and start:
```bash
source load-env.sh
make server
```

### LM Studio (Local)

1. Install LM Studio from [lmstudio.ai](https://lmstudio.ai)
2. Load a model and start the local server (default: `http://localhost:1234/v1`)
3. In the desktop app, go to **Settings > AI Providers** and configure the LM Studio endpoint

### Switching Providers at Runtime

From the desktop app: **Settings > AI Providers** lets you configure endpoints, test connections, fetch available models, and switch all agents at once.

From chat:
```
/switch-provider GoExpert claude claude-3-5-sonnet-20241022
/switch-all-providers ollama
```

### Mock Provider (No AI Calls)

For testing without any API or local model:
```bash
go run cmd/agent/main.go --type backend --name "Go Expert" --mock=true
```

## Interface Comparison

| Interface | Command | Best For |
|-----------|---------|----------|
| **Desktop App** | `make gui` | Full experience -- command palette, file explorer, code editor, threads |
| **Terminal Chat** | `make chat` | Terminal users, SSH sessions |
| **Web UI** | `http://localhost:8080` | Quick access, remote/mobile |
| **CLI** | `go run cmd/cli/main.go` | Scripting, automation, MCP server |

## Using the Desktop App

### Command Palette

Type `/` in the chat input or click the **`/`** button in the toolbar to open the command palette. It provides:
- Searchable list of all 50+ slash commands
- Organized by category (Repo Agents, Dispatch, Provider, etc.)
- Guided forms for commands that take arguments
- Keyboard navigation (arrow keys + Enter)

### @Mentions

Direct questions to specific agents:
```
@GoExpert How should I structure this API?
@SecurityExpert @GoExpert Review this auth middleware
@frontend How do I center a div?
```

Mention by **name** (e.g., `@GoExpert`) or by **type** (e.g., `@frontend`, `@backend`, `@database`, `@devops`, `@security`, `@repo`).

### Threads

Click a message to open a thread. Replies stay in context. You can @mention a different agent in a thread reply to get a **second opinion** (Agent Review).

### Panels

Toggle panels from **Settings > Layout**:
- **File Explorer** -- Browse workspace files
- **Code Editor** -- View and edit code
- **Terminal** -- Embedded terminal output
- **My Agents** -- Active agent list with status
- **Pending Changes** -- File change proposals from agents

## Creating Dynamic Agents

### Repository Expert

Index a codebase and get a project-specific AI expert:

```
/create-repo-agent /path/to/your/project
```

Or with options:
```
/create-repo-agent /path/to/project --agent-name "MyApp Expert" --provider ollama --model qwen2.5-coder:14b
```

Enable file watching for auto-reindex on changes:
```
/enable-watch MyApp Expert
```

See [REPO_AGENTS.md](REPO_AGENTS.md) for full documentation.

### Confluence Documentation Agent

Index a Confluence space for documentation Q&A:

```
/create-confluence-agent SPACEKEY
```

Requires Confluence credentials in `env.local`:
```bash
CONFLUENCE_DOMAIN=yourcompany.atlassian.net
CONFLUENCE_EMAIL=your.email@company.com
CONFLUENCE_API_TOKEN=your-api-token
```

See [CONFLUENCE_AGENTS.md](CONFLUENCE_AGENTS.md) for full documentation.

### Helper Agent

Create custom knowledge-base experts from templates:

```
/create-helper day-one
/list-helper-templates
```

See [HELPER_AGENTS.md](HELPER_AGENTS.md) for full documentation.

## Make Targets

```bash
# Lifecycle
make start-all        # Server + agents + desktop
make server           # Hub server (loads env.local)
make agents           # All 5 specialist agents
make stop             # Kill all processes
make refresh          # Stop, clear, restart

# Desktop app
make gui              # Launch desktop app
make gui-install      # Install npm + Rust deps (first time)
make gui-build        # Production build

# Individual agents
make agent-backend    # GoExpert
make agent-frontend   # ReactExpert
make agent-database   # SQLMaster
make agent-security   # SecurityExpert
make agent-devops     # DevOpsPro
make helper-agent NAME=day-one   # Helper agent

# Dynamic agents
make repo-agent PATH=/path/to/repo NAME="Agent Name"

# Build & test
make build            # Build all Go binaries
make test             # Run Go tests
make pull-models      # Pull Ollama models
make deps             # Download Go dependencies
make clean            # Remove build artifacts
```

## Environment Variables

All make targets automatically load from `env.local`. Key variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `USE_AI_HUB` | Use AI Hub endpoint | `true` |
| `AI_HUB_ENDPOINT` | AI Hub URL | `https://aihub.dispatchit.com/v1` |
| `ANTHROPIC_API_KEY` | Claude API key | -- |
| `AI_HUB_MODEL` | Claude model | `claude-sonnet` |
| `OLLAMA_MODEL` | Ollama utility model | `qwen2.5:7b` |
| `OLLAMA_CODE_MODEL` | Ollama code model | `qwen2.5-coder:14b` |
| `SERVER_PORT` | Server port | `8080` |
| `ENABLE_MCP` | Enable MCP tool servers | `true` |
| `CONFLUENCE_DOMAIN` | Confluence Cloud domain | -- |
| `CONFLUENCE_EMAIL` | Confluence email | -- |
| `CONFLUENCE_API_TOKEN` | Confluence API token | -- |
| `CURSOR_API_KEY` | Cursor CLI API key (optional) | -- |
| `MCP_EXPORTS_DIR` | MCP export storage | `~/.neural-junkie/exports` |

See `env.example` for the full list with descriptions.

## Troubleshooting

### "Connection refused"
Server isn't running. Start it with `make server`.
```bash
curl http://localhost:8080/api/channels
```

### "No agents responding"
Check agents are running:
```bash
curl http://localhost:8080/api/agents
```

### Port 8080 already in use
Edit `env.local` and set `SERVER_PORT=8081`.

### Desktop app won't start
```bash
make gui-install    # Reinstall dependencies
make gui            # Try again
```

### Mock AI responses instead of real ones
Check that `env.local` has valid credentials and agents weren't started with `--mock=true`.

### "env.local not found"
```bash
make setup-env      # Creates env.local from env.example
```

## Data Storage

Neural Junkie stores data in `~/.neural-junkie/`:

| Path | Contents |
|------|----------|
| `~/.neural-junkie/repos/` | Cached repository indexes |
| `~/.neural-junkie/confluence/` | Cached Confluence indexes |
| `~/.neural-junkie/helpers/` | Helper agent configs and knowledge |
| `~/.neural-junkie/assistant/` | Reminders, tasks, notes |
| `~/.neural-junkie/exports/` | MCP-format agent exports |
| `~/.neural-junkie/backups/` | File change backups |
| `~/.neural-junkie/workspaces.json` | Workspace list |
| `~/.neural-junkie/last-session.json` | Session persistence |

## Next Steps

- [Architecture](ARCHITECTURE.md) -- How the system works under the hood
- [Repo Agents](REPO_AGENTS.md) -- Deep codebase analysis
- [CLI Agents](CLI_AGENTS.md) -- Cursor CLI agent and custom CLI integrations
- [Dispatch Integration](DISPATCH_INTEGRATION.md) -- DevOps command execution
- [MCP Exports](MCP_EXPORTS.md) -- Sharing agent knowledge
- [examples/](../examples/) -- Real-world usage scenarios
