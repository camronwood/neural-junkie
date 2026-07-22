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
├── assets/                    # Icons, desktop screenshots (README)
├── campaigns/                 # Marketing copy + creatives by campaign
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

Agents can propose file and Git changes via typed `change_proposal` message metadata. The proposal managers remain authoritative for lifecycle state, while the original message renders as a durable inline chat card. Users can accept or reject directly in chat; file review opens the editor diff/hunk workflow. Approved file changes are applied by `FileChangeExecutor`, which creates backups before modifying files.

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

Key test files: `hub_test.go`, `commands_test.go`, `assistant_test.go`, `moderator_test.go`, `repo_agent_test.go`, `deduplication_test.go`, `integration_test.go`, `agent_review_test.go`

Chat conversation quality: CI router table in `internal/agent/chat_quality_router_test.go`; live JSON scenarios — [CHAT_SCENARIOS.md](CHAT_SCENARIOS.md) (`make chat-scenario SCENARIO=…`).

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

## Dev builds vs release installers

`make gui` / Tauri dev builds **do not** check the release updater manifest. Update checks are disabled in dev (`import.meta.env.DEV`) so you will not see a “Could not check for updates” banner. Test in-app auto-update with an installed release build from [DOWNLOAD.md](DOWNLOAD.md).

## macOS notarization (release CI)

Release CI signs and notarizes macOS builds when GitHub Actions secrets are configured:

| Secret | Purpose |
|--------|---------|
| `APPLE_CERTIFICATE_BASE64` | Developer ID Application `.p12` (base64) |
| `APPLE_CERTIFICATE_PASSWORD` | `.p12` export password |
| `APPLE_SIGNING_IDENTITY` | e.g. `Developer ID Application: Your Org (TEAMID)` |
| `APPLE_ID` | Apple ID for notarytool |
| `APPLE_APP_SPECIFIC_PASSWORD` | App-specific password for notarytool |
| `APPLE_TEAM_ID` | Team ID |

Scripts: [scripts/ci-import-apple-certificate.sh](../scripts/ci-import-apple-certificate.sh), [scripts/ci-set-macos-signing-identity.sh](../scripts/ci-set-macos-signing-identity.sh), [scripts/notarize-macos-artifacts.sh](../scripts/notarize-macos-artifacts.sh).

Without these secrets, CI falls back to ad-hoc signing (`signingIdentity: "-"`). Local dev builds always use ad-hoc signing.

**Verify on a beta tag before v1.0.0 stable:** install the `.dmg` on a clean Mac without Right-click → Open.

## Marketing assets and static site

Canonical campaign copy + creatives live under `campaigns/<slug>/`. Published site copies are generated — do not edit PNGs under `docs/media/` by hand.

```text
campaigns/<slug>/                 # campaign brief + paste copy (*.md)
campaigns/<slug>/creatives/       # source ads + article covers (*.png)
assets/screenshots/               # product screenshots (README + gallery)
scripts/compose-*.sh              # regenerate creatives into campaigns/
make gallery-sync                 # campaigns/*/creatives → docs/media/gallery/ads/ + manifest
make articles-sync                # campaign LinkedIn MD → docs/articles/*.html + manifest.json
```

Article covers also sync to `docs/media/articles/covers/` during `articles-sync`. After changing source PNGs or LinkedIn markdown, run both sync targets before committing.

## Maintainer scripts (manual / optional)

These are not wired into `Makefile` or CI but are kept for local dogfood:

| Script | Purpose |
|--------|---------|
| `scripts/smoke-hidden-repo-agent.sh` | Quick repo-agent visibility smoke |
| `scripts/smoke-secondary-analysis.sh` | Secondary analysis pipeline smoke |
| `scripts/run-phoenix-resource-api-collab.sh` | Phoenix resource API collab regression |
| `scripts/run-resource-api-schema-regression.sh` | Resource API schema checks |

Thin shell wrappers (`collab-scenarios.sh`, `collab-smoke.sh`, `analyze-last-session.sh`) delegate to the `.py` entry points — prefer calling Python directly or via `make` targets.

Trim old regression logs with `python3 scripts/archive-testing-reports.py --keep 5` (moves extras to `docs/archive/testing/`).

## macOS notarization (local / deferred notes)

Release CI currently produces **ad-hoc signed** `.dmg` files (`tauri.conf.json` `signingIdentity: "-"`). Gatekeeper may require **Right-click → Open** on first launch.

**Follow-up when Apple Developer credentials are available:**

1. Enroll in Apple Developer Program; create **Developer ID Application** certificate.
2. Add GitHub Actions secrets: cert base64 + password, `APPLE_SIGNING_IDENTITY`, and notarization credentials (Apple ID app-specific password or App Store Connect API key).
3. Import cert into CI keychain; set real signing identity in Tauri config.
4. Post-build: `notarytool submit` + `stapler staple` on `.app` / `.dmg`.
5. Remove `macos-notarized` from [KNOWN_ISSUES.md](KNOWN_ISSUES.md) and update [DOWNLOAD.md](DOWNLOAD.md).
