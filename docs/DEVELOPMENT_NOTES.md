# Development Notes

Internal notes for developers working on Neural Junkie.

## Code Organization

```
neural-junkie/
├── cmd/                       # Entry points
│   ├── server/                # Hub server (HTTP + WebSocket)
│   ├── agent/                 # Standalone agent runner
│   ├── chat/                  # Interactive terminal chat
│   └── cli/                   # CLI tool + MCP resource server
├── assets/                    # Icons, marketing art, desktop screenshots (README)
├── desktop/                   # Tauri + React desktop app
│   ├── src/                   # React components, stores, hooks, utils
│   │   ├── components/        # 33 React components
│   │   ├── stores/            # Zustand state (chat, settings, editor, files, terminal)
│   │   ├── hooks/             # WebSocket, keyboard shortcuts, editor shortcuts
│   │   ├── api/               # HTTP API clients (chat, terminal)
│   │   ├── types/             # TypeScript protocol types
│   │   └── utils/             # Markdown, secure storage, workspace context
│   └── src-tauri/             # Rust backend (Tauri shell, commands)
├── internal/                  # Core Go packages
│   ├── hub/                   # Hub, commands, workspaces
│   ├── agent/                 # All agent implementations (11 types)
│   ├── protocol/              # Message types, mentions, path/command detection
│   ├── ai/                    # Providers: Ollama, Claude, LM Studio, Mock, CLI
│   ├── repo/                  # Repository indexing, search, watching, compression
│   ├── confluence/            # Confluence client, indexing, search, storage
│   ├── filechange/            # File change proposals, approval, execution
│   └── mcp_export/            # MCP format export/import
├── test/                      # Go tests
├── docs/                      # Documentation + static landing (index.html)
├── examples/                  # Usage scenarios
├── public/                    # Optional static preview (serve repo root for asset paths)
└── scripts/                   # Automation scripts
```

## Hub Web Surfaces

The Go hub (`cmd/server`) serves the browser **chat** UI at **`/`** (WebSocket to the hub).

Marketing / early-access copy for hosting lives under `docs/index.html` + `docs/css/landing.css`. **Feature guides** for the public site live under `docs/features/*.html` (deep dives linked from the landing page).

## Key Design Decisions

### Desktop App (Tauri + React)

The desktop app uses Tauri (Rust) for the native shell and React (TypeScript) with Tailwind CSS for the UI. State is managed with Zustand stores. Settings persist via the Tauri Store plugin (`.neural-junkie-*.dat` files).

The original Fyne-based Go GUI was replaced in late 2025 due to threading limitations. See `docs/archive/TAURI_IMPLEMENTATION.md` for migration details.

### Message Deduplication

Three-layer system prevents agents from responding multiple times:
1. **Polling dedup** (`cmd/agent/main.go`) -- `seenMessages` map filters already-processed messages
2. **Handler-level tracking** (`internal/agent/agent.go`) -- `respondedMessages` prevents re-processing
3. **Agent-type filtering** -- Agents skip messages from other agents to prevent loops

### Command System

Slash commands are handled by `CommandHandler` in `internal/hub/commands.go`. Two key methods:
- `ProcessCommand()` -- routes commands to handlers, returns response
- `GetCommandDefinitions()` -- returns metadata (name, description, category, arguments) for the command palette

The command palette on the frontend (`desktop/src/components/CommandPalette.tsx`) fetches definitions from `GET /api/commands` and renders a searchable, categorized list with dynamic forms for arguments.

### File Change Workflow

Agents can propose file changes via message metadata. The `FileChangeManager` tracks proposals with status (pending, approved, rejected, expired). The desktop app shows proposals in the Pending Changes panel with diff preview. Approved changes are applied by `FileChangeExecutor` which creates backups before modifying files.

### Agent Lifecycle

Agents can be in several states:
- **Active** -- registered and responding to messages
- **Paused** -- registered but not responding (via `/pause-agent`)
- **Removed** -- unregistered but cached for recall (via `/remove-agent`)
- **Deleted** -- permanently removed (via `/delete-agent`)

## Common Development Tasks

### Adding a New Agent Type

1. Add the type constant in `internal/protocol/types.go`
2. Create a constructor in `internal/agent/specialized_agents.go`
3. Register in `AgentFactory` in the same file
4. Add CLI flag handling in `cmd/agent/main.go`
5. Optionally add make target in `Makefile`

### Adding a New Slash Command

1. Add the handler in `CommandHandler.ProcessCommand()` in `internal/hub/commands.go`
2. Add metadata to `GetCommandDefinitions()` for command palette support
3. Argument types: `string`, `path`, `provider`, `model`, `agent-name`

### Adding a New AI Provider

1. Implement `AIProvider` interface in `internal/ai/`
2. Add config loading from environment variables
3. Register in the provider creation logic in `cmd/server/main.go` and `cmd/agent/main.go`
4. Add connection test endpoint if applicable

### Adding a Desktop Component

1. Create component in `desktop/src/components/`
2. Import TypeScript types from `desktop/src/types/protocol.ts`
3. Use API methods from `desktop/src/api/chatAPI.ts`
4. State goes in the appropriate Zustand store
5. Add to `ChatWindow.tsx` or `SettingsModal.tsx` as needed

## Testing

### Go Tests

Located in `test/` directory:
```bash
make test-go       # Go tests only (entire module; -count=1)
make test-all      # go vet + Go tests + desktop tsc + Vitest
make test          # Alias for test-go
go test ./test/... # Run integration test package only
```

Key test files: `hub_test.go`, `commands_test.go`, `assistant_test.go`, `moderator_test.go`, `repo_agent_test.go`, `helper_agent_test.go`, `deduplication_test.go`, `integration_test.go`, `agent_review_test.go`

### Manual Testing

```bash
make server          # Start hub (specialists in-process per config)
# Optional: make agents   # Standalone specialist processes — avoid duplicates vs hub config
make gui             # Open desktop app
# Test commands, mentions, threads, file changes, etc.
```

## Debugging Tips

### Message Flow Issues
1. Check server logs -- hub receives message?
2. Check agent logs -- agent sees message?
3. Verify deduplication -- is message ID already tracked?
4. Check mention parsing -- does `@AgentName` resolve?

### Agent Response Issues
1. Check `shouldRespond()` logic for the agent type
2. Verify expertise keywords match the message
3. Test with mock AI first (`--mock=true`)
4. Check if agent is paused or removed

### Desktop App Issues
1. Check browser console (Tauri dev tools: right-click > Inspect)
2. Verify WebSocket connection to server
3. Check Zustand store state
4. Verify API responses from hub server

## Performance Notes

- Message cache: 100 messages max per channel
- Seen messages: 100 IDs max with cleanup at 50
- Agent history: last 10 messages for AI context
- Hub state: protected by `sync.RWMutex`
- Agent polling: 1-second HTTP poll interval
- Repository indexes: cached with staleness detection
