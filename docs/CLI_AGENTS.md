# CLI Agents

CLI Agents are a special agent type that wraps external AI CLI tools as subprocesses, integrating them into the Neural Junkie chat as first-class participants. Instead of calling an HTTP API, the agent invokes a CLI binary, passes the prompt as an argument, and captures the output.

The system is designed to be generic -- any CLI-based AI tool can be integrated -- but currently ships with built-in support for the **Cursor CLI agent**.

## Cursor CLI Agent

The Cursor CLI agent gives Neural Junkie access to Cursor's agentic capabilities: codebase search, file operations, code generation, refactoring, and shell command execution. It runs Cursor's `agent` binary in headless mode as a subprocess.

### How It Works

```
User sends @Cursor message in chat
    │
    ▼
Cursor agent receives message, builds prompt
(includes system prompt + last 8 messages for context)
    │
    ▼
Invokes: agent -p --output-format text "<prompt>"
(subprocess runs in CURSOR_WORK_DIR)
    │
    ▼
Captures stdout, parses response
    │
    ▼
Sends response back to chat
```

The agent runs with a 120-second timeout per invocation. Complex tasks (multi-file refactors, large codebase analysis) may take 30-120 seconds.

### Setup

#### 1. Install the Cursor CLI

```bash
curl https://cursor.com/install -fsS | bash
```

This installs the `agent` binary to your PATH. Verify it's available:

```bash
which agent
```

#### 2. Configure Environment (Optional)

Add to your `env.local`:

```bash
# Optional -- API key for Cursor. If omitted, the CLI uses stored
# credentials from interactive login (cursor login).
CURSOR_API_KEY=your_cursor_api_key_here

# Optional -- working directory for the Cursor agent. This determines
# which codebase the agent operates on. Defaults to the Neural Junkie
# project directory if not set.
CURSOR_WORK_DIR=/path/to/your/project
```

If you don't set `CURSOR_API_KEY`, the Cursor CLI will use credentials from a previous `cursor login` session.

#### 3. Start the Server

```bash
make server
```

The server automatically detects the Cursor CLI binary on startup. You'll see one of:

```
✅ Cursor CLI binary found, initializing agent...
✅ Cursor CLI agent started (workDir: /your/project)
```

Or if the CLI isn't installed:

```
ℹ️  Cursor CLI ('agent') not found on PATH — skipping.
```

No manual agent startup is needed -- the Cursor agent auto-starts with the server, just like the Moderator and Assistant.

### Usage

The Cursor agent joins the `general` channel as **"Cursor"**. Interact with it like any other agent:

```
@Cursor Analyze the architecture of this project

@Cursor Refactor the error handling in internal/hub/commands.go

@Cursor Write unit tests for the FileChangeManager

@Cursor What are the main dependencies and how are they used?
```

The agent has expertise in: Code Generation, Refactoring, Code Review, Codebase Analysis, Bug Fixing, Testing, Full-Stack Development, Architecture, File Operations, Shell Commands.

### Capabilities

Because the Cursor CLI runs in the context of a real codebase directory, it can:

- **Search and read files** in the working directory
- **Analyze project structure** and dependencies
- **Generate code** that fits existing patterns
- **Propose refactors** across multiple files
- **Run shell commands** for build/test verification

### Limitations

- **120-second timeout** -- very large tasks may time out
- **No vision support** -- cannot process images directly (reference file paths instead)
- **One codebase** -- operates on `CURSOR_WORK_DIR` only; use repo agents for other projects
- **Sequential** -- one invocation at a time per agent instance
- **No streaming** -- the full response is returned after the subprocess completes

## Building Custom CLI Agents

The CLI agent system is generic. You can wrap any CLI tool that accepts a prompt and returns text.

### Provider Configuration

```go
provider := ai.NewCLIAgentProvider(
    "my-cli-tool",           // CLI binary name
    "/path/to/workdir",      // Working directory
    "my-tool",               // Provider display name
    ai.WithTimeout(60 * time.Second),
    ai.WithEnv("MY_API_KEY", "sk-..."),
    ai.WithModel("my-tool-v1"),
    ai.WithBaseArgs([]string{"--prompt"}),  // Args before the prompt text
)
```

### Agent Creation

```go
agent := agent.NewCLIAgent(
    "MyTool",              // Agent display name
    "my-tool",             // Provider name
    provider,              // The CLIAgentProvider
    hub,                   // Hub client
    []string{              // Expertise keywords
        "Code Generation",
        "Analysis",
    },
)
```

### CLIAgentProvider Interface

The provider implements the standard `AIProvider` interface:

| Method | Behavior |
|--------|----------|
| `GenerateResponse` | Builds prompt with conversation history, invokes CLI subprocess, captures and returns stdout |
| `GenerateVisionResponse` | Returns error (not supported; use file path references instead) |
| `GetModel` | Returns the display model name |
| `IsCLIInstalled` | Checks if the CLI binary is on PATH |

### How Prompts Are Built

1. The system prompt and user message are recombined (CLI tools don't use role separation)
2. The last 8 messages from conversation history are prepended as context
3. The full prompt is passed as the final argument to the CLI command

### Output Parsing

The provider tries two strategies:
1. **JSON** -- if the CLI returns `{"result": "..."}`, extracts the `result` field
2. **Plain text** -- otherwise returns the raw stdout

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CURSOR_API_KEY` | No | -- | API key for Cursor CLI auth. Falls back to stored credentials. |
| `CURSOR_WORK_DIR` | No | Server's CWD | Working directory the agent operates in |

## Troubleshooting

### "Cursor CLI ('agent') not found on PATH"

Install the CLI:
```bash
curl https://cursor.com/install -fsS | bash
```

Then restart the server.

### "CLI agent timed out after 2m0s"

The task was too large for a single invocation. Try breaking it into smaller questions, or increase the timeout by modifying the provider configuration.

### "CLI agent returned empty output"

The CLI binary ran but produced no stdout. Check:
- Is the CLI authenticated? Run `cursor login` interactively first.
- Is `CURSOR_WORK_DIR` pointing to a valid directory?
- Check server logs for stderr output from the subprocess.

### Agent doesn't respond to messages

Verify it started successfully in the server logs. The agent only responds when @mentioned or when the message matches its expertise keywords.
