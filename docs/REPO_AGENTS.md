# Repository Expert Agents

Repository Expert Agents are specialized AI agents that become experts on specific code repositories. They analyze the repository structure, dependencies, and code patterns to provide intelligent assistance about your projects.

## Features

- **Deep Repository Analysis**: Scans file structure, key files, dependencies, and git history
- **Context-Aware Responses**: Uses indexed repository data to provide accurate answers
- **Real-Time Status Updates**: Shows indexing progress in the UI
- **Persistent Storage**: Saves indexed data for faster startup
- **Smart Caching**: Loads cached indexes instantly when repository hasn't changed
- **Staleness Detection**: Automatically detects when repository has changed
- **Incremental Reindexing**: Updates only changed files instead of full rescan
- **File Watching**: Optionally monitor repository for changes and auto-reindex
- **Pause/Resume**: Control when agents respond to save API costs

## Quick Start

### 1. Using CLI

Create a repository expert agent from the command line:

```bash
# Load environment variables (includes AI Hub credentials)
source load-env.sh

# Create a repo agent
go run cmd/agent/main.go --type repo --repo-path /path/to/your/repo --name "MyProject Expert"
```

### 2. Using Chat Commands

Create agents directly from the chat interface:

```
/create-repo-agent /Users/myuser/projects/my-app MyApp Expert
```

The agent will:
1. Start indexing the repository
2. Show progress (0-100%)
3. Join the chat when ready

## Chat Commands

### Agent Creation
```
/create-repo-agent <path> [name]
```
Create a new repository expert agent.

**Example:**
```
/create-repo-agent /Users/john/projects/ecommerce-api "Ecommerce API Expert"
```

### Agent Management

#### Reindex Repository
```
/reindex-agent <agent-name>
```
Trigger a reindex of the repository (useful after making changes).

**Example:**
```
/reindex-agent MyProject Expert
```

#### Pause Agent
```
/pause-agent <agent-name>
```
Pause an agent from responding (saves AI API costs).

#### Resume Agent
```
/unpause-agent <agent-name>
```
Resume a paused agent.

#### Delete Agent
```
/delete-agent <agent-name>
```
Remove an agent and delete its indexed data.

#### List Agents
```
/list-agents
```
Show all active agents with their status.

#### Enable File Watching
```
/enable-watch <agent-name>
```
Enable automatic file watching and reindexing when files change.

**Example:**
```
/enable-watch MyProject Expert
```

#### Disable File Watching
```
/disable-watch <agent-name>
```
Disable automatic file watching.

**Example:**
```
/disable-watch MyProject Expert
```

### Get Help
```
/help
```
Display available commands.

## Example Interaction

```
User: /create-repo-agent /Users/sarah/projects/todo-app

System: 🤖 Creating repository expert agent 'todo-app Expert' for todo-app...
        📊 Indexing repository (this may take a minute)...
        The agent will join the chat once indexing is complete.

[After 30 seconds]

todo-app Expert: 👋 todo-app Expert has joined! I've analyzed your repository 
                  and I'm ready to answer questions about the codebase.

User: What's the main entry point of this application?

todo-app Expert: Based on the repository structure, the main entry point is 
                  `cmd/server/main.go`. This file initializes the HTTP server 
                  on port 8080 and sets up the following routes:
                  - GET  /api/todos - List all todos
                  - POST /api/todos - Create a new todo
                  - PUT  /api/todos/:id - Update a todo
                  
                  The application uses the Gin web framework and connects to 
                  a PostgreSQL database.

User: How is authentication handled?

todo-app Expert: Authentication is handled via JWT tokens in the 
                  `internal/auth/middleware.go` file. The system uses:
                  - RS256 algorithm for signing tokens
                  - 15-minute access token expiration
                  - Refresh tokens stored in httpOnly cookies
                  
                  The auth middleware is applied to all /api routes except 
                  /api/auth/login and /api/auth/register.
```

## What Gets Indexed?

The repository agent analyzes:

### 1. File Structure
- Complete directory tree
- File types and languages
- Project organization
- **File modification times** (for change detection)

### 2. Key Files
- README.md
- package.json / go.mod / requirements.txt
- Dockerfile / docker-compose.yml
- Makefile
- Configuration files
- **Tracked with modification timestamps**

### 3. Dependencies
- npm/yarn packages
- Go modules
- Python packages
- Other dependency managers

### 4. Git Information
- Current branch
- Recent commits (last 10)
- Last commit message and date
- **Commit hash for staleness detection**

### 5. Code Patterns
- Detected frameworks (React, Vue, Express, Gin, etc.)
- Containerization (Docker)
- Architecture patterns

## Repository Limits

To ensure reasonable indexing times:

- **Max File Size**: 1MB per file
- **Max Total Size**: 100MB
- **Max Files**: 10,000 files

Ignored directories:
- `node_modules`, `vendor`, `target`
- `dist`, `build`, `__pycache__`
- `.git`, `.svn`, `.hg`
- `venv`, `env`, `.venv`
- IDE directories (`.idea`, `.vscode`)

## Agent Status Indicators

In the GUI sidebar, agents show visual status:

- ✅ **Ready**: Agent is active and responding
- 🔄 **Indexing**: Initial repository analysis (shows progress %)
- 🔄 **Reindexing**: Updating repository knowledge
- ⏸️ **Paused**: Agent is paused (not responding)
- ❌ **Error**: Indexing failed

## Storage Location

Indexed repository data is stored in:
```
~/.neural-junkie/repos/<repo-name>/
  ├── index.json          # Repository index
  └── config.json         # Agent configuration
```

## Environment Configuration

The agents use AI Hub for Claude API access. Configure in `env.local`:

```bash
# AI Hub Configuration
USE_AI_HUB=true
AI_HUB_ENDPOINT=<your-ai-hub-endpoint>
ANTHROPIC_API_KEY=your_ai_hub_key_here

# Model Selection (optional)
AI_HUB_MODEL=claude-sonnet  # or claude-haiku for faster responses
```

## Best Practices

### 1. Naming Agents
Use descriptive names that indicate the repository:
- ✅ "E-commerce API Expert"
- ✅ "Frontend Dashboard Agent"
- ❌ "Agent 1"

### 2. Caching & Performance

**Cache System (Updated October 2025):**
- Caches are keyed by repository PATH (not agent name)
- Multiple agents pointing to same repo SHARE the same cache
- Cache persists across agent deletions and recreations
- User gets visual feedback about cache status

**Performance:**
- **First indexing**: 30-60 seconds (full analysis)
  - Shows: "📊 No cache found, performing full analysis..."
  - Progress: 0-100%
  - Creates cache for future use
- **Cached loading**: <2 seconds (instant)
  - Shows: "✅ Loaded from cache (instant) - repository already indexed!"
  - No progress bar
  - Used when repository hasn't changed
- **Incremental update**: 5-15 seconds (when repository changed)
  - Shows: "🔄 Cache stale (reason), performing incremental update..."
  - Progress shown
  - Only re-analyzes changed files

**Cache Sharing Example:**
```bash
# First agent creates cache (30-60 seconds)
/create-repo-agent /path/to/repo Agent1

# Second agent uses existing cache (instant!)
/create-repo-agent /path/to/repo Agent2

# Even after deletion, cache persists
/delete-agent Agent1
/delete-agent Agent2

# New agent still uses cache (instant!)
/create-repo-agent /path/to/repo Agent3
```

**Cache Location:** `~/.neural-junkie/repos/<sha256-hash>/`
- Enable file watching for active development repositories

### 3. When to Reindex
Reindex after:
- Major code changes (automatic if file watching enabled)
- Dependency updates (automatic if key files changed)
- Architecture refactoring
- New features added

**Note**: Manual reindexing is rarely needed with staleness detection!

### 4. Managing Costs
- Pause agents when not actively needed
- Use `claude-haiku` model for faster, cheaper responses
- Delete agents for repositories no longer in use
- Enable file watching only for actively developed repos

### 5. Question Techniques
Ask specific questions:
- ✅ "What's the authentication flow?"
- ✅ "Where is the user data validated?"
- ✅ "How do I add a new API endpoint?"
- ❌ "Tell me everything about this project"

## Troubleshooting

### Agent Won't Create
**Error**: Repository path does not exist

**Solution**: Verify the path is correct and accessible:
```bash
ls /path/to/repository
```

### Indexing Fails
**Error**: Repository too large

**Solution**: 
- Check repository size: `du -sh /path/to/repository`
- Remove large build artifacts
- Ensure repository is under 100MB

### Agent Not Responding
**Check**:
1. Is agent paused? Use `/unpause-agent`
2. Is agent still indexing? Check progress
3. Is your question relevant to the repository?

### Slow Indexing
**Causes**:
- Large repository (many files)
- Slow disk I/O
- Network latency (if repo is on network drive)

**Solutions**:
- Use local repositories when possible
- Clean up unnecessary files
- Be patient for first-time indexing (cached afterwards)

## Advanced Usage

### Creating Multiple Repo Agents

You can create multiple agents for different repositories:

```bash
# Frontend expert
/create-repo-agent /path/to/frontend "Frontend Expert"

# Backend expert
/create-repo-agent /path/to/backend "Backend Expert"

# Infrastructure expert
/create-repo-agent /path/to/infrastructure "DevOps Expert"
```

They can collaborate in the same channel!

### Using with Monorepos

For monorepos, create separate agents for different packages:

```bash
/create-repo-agent /monorepo/packages/web "Web Package Expert"
/create-repo-agent /monorepo/packages/api "API Package Expert"
```

## Limitations

Current limitations (see FUTURE_ENHANCEMENTS.md for planned improvements):

1. **Single Repository**: Each agent knows about one repository only
2. **Static Analysis**: No code execution or runtime analysis
3. **No File Modification**: Agents provide guidance but don't modify code
4. **English Only**: Primarily designed for English-language code and documentation
5. **No Real-Time Updates**: Requires manual reindexing after changes

## Security Considerations

### Repository Access
- Agents read repository files during indexing
- Files are sent to AI Hub/Claude API for analysis
- Don't use with repositories containing secrets/credentials

### Data Privacy
- Indexed data stored locally in `~/.neural-junkie/repos/`
- Can be deleted anytime using `/delete-agent`
- AI conversations are processed by Claude API

### Best Practices
- Review `key files` before indexing sensitive repos
- Use environment variables, don't commit secrets
- Regularly cleanup unused agent data

## API Integration

For programmatic access (future):

```go
import "github.com/camron/neural-junkie/internal/agent"

// Create repo agent
repoAgent, err := agent.NewRepoAgent("MyApp", "/path/to/repo", aiProvider, hub)

// Start with indexing
err = repoAgent.StartWithIndexing(ctx, "general")

// Reindex
err = repoAgent.Reindex(ctx)

// Cleanup
err = repoAgent.Cleanup()
```

## Contributing

Want to improve repository agents? Check out:
- `internal/repo/analyzer.go` - Repository analysis logic
- `internal/agent/repo_agent.go` - Repo agent implementation
- `internal/hub/commands.go` - Chat command handlers

See FUTURE_ENHANCEMENTS.md for planned features!

## Support

Having issues? 
1. Check this documentation
2. Review FUTURE_ENHANCEMENTS.md
3. Open an issue on GitHub

---

Happy coding with your AI repository experts! 🤖📚


