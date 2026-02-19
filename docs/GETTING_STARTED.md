# Getting Started with Neural Junkie

A multi-agent collaboration system where specialized AI agents work together to solve complex problems.

## Quick Start (5 minutes)

### Option 1: Complete System (Easiest!)

Start everything with one command:

```bash
cd /Users/camron.wood.ext/development/sandbox/neural-junkie
make start-all
```

This will:
- Start the server
- Start all 5 AI agents (Backend, Database, Security, Frontend, DevOps)
- Open the GUI chat application

### Option 2: Automated Test Script

```bash
./scripts/quick-test.sh
make gui
```

### Option 3: Manual Setup

**Terminal 1 - Start Server:**
```bash
make server
# OR: go run cmd/server/main.go
```

**Terminal 2 - Start Agents:**
```bash
make agents
# OR manually:
# go run cmd/agent/main.go --type backend --name "Go Expert"
# go run cmd/agent/main.go --type frontend --name "React Pro"
# go run cmd/agent/main.go --type security --name "Security Expert"
```

**Terminal 3 - Launch Interface:**
```bash
# GUI (Best Experience)
make gui

# OR Terminal Chat
make chat

# OR Web UI
open http://localhost:8080

# OR CLI
go run cmd/cli/main.go --channel general --message "Your question here"
```

## Interface Comparison

| Interface | Command | Best For |
|-----------|---------|----------|
| **GUI** 🖥️ | `make gui` | Visual users, best experience |
| **Terminal** 💬 | `make chat` | Terminal lovers, SSH |
| **Web** 🌐 | `http://localhost:8080` | Remote access, mobile |
| **CLI** ⌨️ | `go run cmd/cli/main.go` | Scripts, automation |

## AI Configuration

### Prerequisites

Ensure `env.local` exists and contains your AI Hub credentials:

```bash
USE_AI_HUB=true
AI_HUB_ENDPOINT=https://aihub.dispatchit.com/v1
ANTHROPIC_API_KEY=your-api-key-here
AI_HUB_MODEL=claude-sonnet
```

If `env.local` doesn't exist:
```bash
make setup-env  # Copies env.example to env.local
```

### Using AI Hub (Recommended)

```bash
# Copy environment template
cp env.example env.local

# Edit env.local with your credentials
# Load environment
source load-env.sh
```

### Using Anthropic API Directly

```bash
export USE_AI_HUB=false
export ANTHROPIC_API_KEY="your-anthropic-key"
```

### Testing Without API Calls

To use mock AI responses (no API calls, no cost):

```bash
go run cmd/agent/main.go --type backend --name "Go Expert" --mock=true
```

## Available Make Commands

All commands automatically load environment variables from `env.local`:

```bash
make start-all      # Start server + agents + GUI
make server         # Start server only
make agents         # Start all 5 agents
make gui            # Launch GUI application
make chat           # Launch terminal chat

# Individual agents:
make agent-backend  # Go Expert
make agent-database # SQL Master
make agent-security # Security Expert
make agent-frontend # React Expert
make agent-devops   # DevOps Pro
```

## Try It Out!

Once connected, ask questions like:

```
How do I prevent SQL injection?
```

```
Our API is slow. What could be wrong?
```

```
How should we implement JWT authentication?
```

Watch as multiple specialized agents collaborate to give you comprehensive answers!

## Using @Mentions

Target specific agents with the `@mention` feature:

### Basic Syntax

```
@agentname your question here
@agenttype your question here
```

### Examples

**Mention by Type:**
```
@frontend How do I center a div in CSS?
@backend What's the best way to handle API rate limiting?
```

**Mention Multiple Agents:**
```
@frontend @backend How should we structure the user authentication flow?
```

**Available Types:**
- `@frontend` - React, Vue, CSS, HTML, UI/UX
- `@backend` - APIs, Server logic, Microservices
- `@database` - SQL, Schema design, Query optimization
- `@devops` - Deployment, CI/CD, Docker, Kubernetes
- `@security` - Authentication, Encryption, Security
- `@repo` - Code structure, Repository analysis

### When to Use @Mention

✅ **Use mentions when:**
- You know which expert you need
- You want a focused answer (not multiple perspectives)
- Following up with a specific agent

❌ **Don't use mentions when:**
- You want multiple perspectives (brainstorming)
- You're not sure which expert is best
- Asking general questions

### List Available Agents

**In CLI Chat:**
```bash
/agents
```

**In GUI:**
Click the "Refresh" button in the Active Agents panel.

## Agent Types

- **Frontend Agent** - React, Vue, UI/UX, accessibility
- **Backend Agent** - APIs, services, business logic, performance
- **DevOps Agent** - Deployment, infrastructure, CI/CD
- **Database Agent** - Schema design, queries, optimization
- **Security Agent** - Vulnerabilities, auth, encryption
- **Repository Expert** - Deep codebase analysis (special agent type)

## Repository Expert Agents

Create agents that become experts on your specific codebases:

```bash
source load-env.sh
go run cmd/agent/main.go --type repo --repo-path /path/to/your/project --name "MyProject Expert"
```

Or from chat:
```
/create-repo-agent /path/to/your/project MyProject Expert
```

**File Watching (Optional):**
```
/enable-watch MyProject Expert   # Auto-update on file changes
/disable-watch MyProject Expert  # Disable watching
```

See [docs/REPO_AGENTS.md](REPO_AGENTS.md) for details.

## Stop Everything

```bash
# Stop all processes
killall -9 main

# Or press Ctrl+C in each terminal
```

## Troubleshooting

### "Connection refused"
```bash
curl http://localhost:8080/api/channels
```
Make sure the server is running.

### "No agents responding"
```bash
curl http://localhost:8080/api/agents
```
Check that agents are active.

### "Using mock AI provider"
Check that `env.local` has the correct credentials and that agents are started without `--mock=true`.

### GUI won't start
```bash
go mod tidy
```

### Port Already in Use
If port 8080 is busy, edit `env.local` and change `SERVER_PORT` to another port (e.g., 8081).

### "env.local not found"
Run `make setup-env` to create it from the example file.

## Project Structure

```
neural-junkie/
├── cmd/              # Executables
│   ├── server/       # Chat hub server
│   ├── agent/        # Agent runner
│   ├── chat/         # Terminal chat client
│   ├── gui/          # GUI application
│   └── cli/          # CLI interface
├── internal/         # Core implementation
│   ├── hub/          # Chat hub & routing
│   ├── agent/        # Agent framework
│   ├── protocol/     # Message protocol
│   ├── ai/           # AI provider integrations
│   └── repo/         # Repository analysis
├── docs/             # Documentation
├── examples/         # Usage scenarios
└── scripts/          # Automation scripts
```

## What's Running?

When you use `make start-all` or `make agents`, you'll get these AI experts:

1. **Go Expert** (Backend) - Backend development and Go expertise
2. **SQL Master** (Database) - Database optimization and SQL queries
3. **Security Expert** (Security) - Security best practices and vulnerabilities
4. **React Expert** (Frontend) - UI/UX and React development
5. **DevOps Pro** (DevOps) - Infrastructure, deployment, and CI/CD

## Environment Variables

All commands automatically load environment variables from `env.local`:

- `USE_AI_HUB` - Use AI Hub (true) or direct Anthropic API (false)
- `AI_HUB_ENDPOINT` - AI Hub endpoint URL
- `ANTHROPIC_API_KEY` - Your API key for Claude
- `AI_HUB_MODEL` - Model to use (claude-sonnet, claude-opus, etc.)
- `SERVER_PORT` - Server port (default: 8080)
- `SERVER_HOST` - Server host (default: localhost)

## Next Steps

1. Read [README.md](../README.md) for full feature overview
2. Read [docs/ARCHITECTURE.md](ARCHITECTURE.md) for technical details
3. Check [examples/](../examples/) for usage scenarios
4. See [docs/FUTURE_ENHANCEMENTS.md](FUTURE_ENHANCEMENTS.md) for roadmap

---

**Ready to collaborate with AI agents? Start now!** 🚀

