# Getting Started

Get Neural Junkie running in under 5 minutes.

## Downloaded app (recommended)

No Go, Node, or Rust required.

1. Install from [GitHub Releases — v1.0.0-beta.12](https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.12).
2. Open the app and complete the **setup wizard** — choose **Software development**, **Life sciences**, or **Team chat & productivity** (Ollama local or cloud API key).
3. Follow [DOWNLOAD.md](DOWNLOAD.md) for first chat and slash commands.

The desktop app bundles the Go hub as a sidecar and starts it automatically.

**Platform notes:**

- **macOS / Linux:** wizard can install or start Ollama.
- **Windows:** install [Ollama](https://ollama.com) manually or use a cloud key (in-app Ollama install is not supported on Windows).

---

## From source (developers)

### Prerequisites

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
make start-all     # Hub (in-process specialists) + desktop app
```

This launches:
- The **Hub server** on `http://localhost:18765`
- **Moderator** and **Assistant** (auto-started with the server)
- **Specialist agents** from enabled **domain packs** in `~/.neural-junkie/config.json` (Software development pack adds six engineering experts when on)
- **CLI agents** (Cursor, Gemini, Claude, Copilot) when their binaries are on PATH — not pack-gated
- The **Tauri desktop app**

`make start-all` does **not** run `make agents`; specialists are started inside the hub (`initializeConfiguredAgents`). Use `make agents` only when you intentionally want **separate** `cmd/agent` processes (avoid duplicate agent names versus in-process config).

### Option 2: Manual Setup (Separate Terminals)

**Terminal 1 -- Server:**
```bash
make server
```

**Terminal 2 -- Interface:**
```bash
make gui          # Desktop app (recommended)
# OR
make chat         # Terminal chat
# OR
open http://localhost:18765   # Web UI (browser chat client)
```

**Optional -- standalone specialist processes** (same six roles as separate OS processes; see README *Specialist agents* for duplication caveats):
```bash
make agents
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

### Hugging Face (Hosted or local GGUF)

1. Create a token at [huggingface.co/settings/tokens](https://huggingface.co/settings/tokens) and set `HF_TOKEN` in `env.local` or add a **Hugging Face** row under **Settings → AI Providers → Provider registry** (`type: huggingface`, model = Hub repo id such as `Qwen/Qwen2.5-Coder-7B-Instruct`).
2. **Hosted (cloud):** open the toolbar **Model library** (⇧⌘M), **Hugging Face** tab, **Hosted**, then **Use for agents** (or **Add provider** on the detail screen).
3. **Local download:** use the **Download** tab to pull a curated GGUF, then **Import to Ollama** (requires Ollama running). Agents use the Ollama provider with the imported tag.
4. When creating a DM expert, choose **Hugging Face (hosted)** or **From hub providers** to bind `provider_id` from the registry.

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
| **Web UI** | `http://localhost:18765` | Browser chat client against the hub |
| **CLI** | `go run cmd/cli/main.go` | Scripting, automation, MCP server |

## Using the Desktop App

### Command Palette

Type `/` in the chat input or click the **`/`** button in the toolbar to open the command palette. It provides:
- Searchable list of all 50+ slash commands
- Organized by category (Repo Agents, Provider, etc.)
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

## Make Targets

```bash
# Lifecycle
make start-all        # Hub (in-process specialists) + desktop
make server           # Hub server (loads env.local)
make agents           # Six specialist agents as separate processes (optional)
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

# Dynamic agents
make repo-agent PATH=/path/to/repo NAME="Agent Name"

# Build & test
make build            # Build all Go binaries
make test-go          # Run Go tests (-count=1)
make test-all         # go vet + Go tests + desktop tsc + Vitest
make test             # Alias for test-go
make pull-models      # Pull Ollama models
make deps             # Download Go dependencies
make clean            # Remove build artifacts
```

## Environment Variables

All make targets automatically load from `env.local`. Key variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `USE_AI_HUB` | Use AI Hub endpoint | `true` |
| `AI_HUB_ENDPOINT` | AI Hub URL | (configurable) |
| `ANTHROPIC_API_KEY` | Claude API key | -- |
| `AI_HUB_MODEL` | Claude model | `claude-sonnet` |
| `OLLAMA_MODEL` | Ollama utility model | `qwen2.5:7b` |
| `OLLAMA_CODE_MODEL` | Ollama code model | `qwen2.5-coder:14b` |
| `SERVER_PORT` | Server port | `18765` |
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
curl http://localhost:18765/api/channels
```

### "No agents responding"
Check agents are running:
```bash
curl http://localhost:18765/api/agents
```

### Port 18765 already in use
Edit `env.local` and set `SERVER_PORT` to a free port (e.g. `18766`).

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
| `~/.neural-junkie/collaborations/` | Default parent for collaboration execution sandboxes (`<id>/` per run). Override with `collaboration.assets_root` in config or `NEURAL_JUNKIE_COLLAB_ASSETS_DIR` |
| `~/.neural-junkie/last-session.json` | Local resume cache (channels, recent messages, in-progress collabs). Bounded on save; oversized/corrupt files are **auto-archived** on startup — users never need to delete it manually. Dev: `scripts/analyze-last-session.sh` |

## Next Steps

- [Architecture](ARCHITECTURE.md) -- How the system works under the hood
- [Repo Agents](REPO_AGENTS.md) -- Deep codebase analysis
- [CLI Agents](CLI_AGENTS.md) -- Cursor CLI agent and custom CLI integrations
- [MCP Exports](MCP_EXPORTS.md) -- Sharing agent knowledge
- [examples/](../examples/) -- Real-world usage scenarios
