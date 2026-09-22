# Agent code map

Where to edit what. Prefer this over guessing from file size or similar names.

**Desktop shell is Tauri** (`desktop/src-tauri`), not Electron. Do not invent `BrowserWindow` / `main.js` without inspecting the tree.

## Edit here for X

| If you need to change… | Start here | Notes |
|------------------------|------------|-------|
| Hub HTTP client (desktop) | `desktop/src/api/domains/*` then thin facade `desktop/src/api/chatAPI.ts` | Keep public `ChatAPI` method names stable; put new endpoints in a domain module |
| Protocol / shared TS types | `desktop/src/types/protocol.ts` | ChatAPI-local DTOs live under `desktop/src/api/types/` |
| Chat UI (messages, composer, channel toolbar) | `desktop/src/components/ChatWindow.tsx` and `desktop/src/components/chat/` (`ChatMessageList`, `ChatInputArea`, `ChatChannelToolbar`) | Prefer extracting into `chat/` over growing ChatWindow |
| Collab UI | `desktop/src/components/CollaborationPanel.tsx` | Phase machine is hub-side |
| File explorer / IDE chrome | `desktop/src/components/FileExplorerPanel.tsx`, IDE pack panels | |
| Native FS / window / updater | `desktop/src-tauri/src/main.rs` | High risk; security + signing |
| Thin browser hub UI | `public/` | Chat + pending file-change only — **not** the desktop IDE |
| Message / agent dispatch | `internal/hub/hub_dispatch.go` | Lifecycle + routing into agents |
| Slash commands | `internal/hub/commands_*.go` via `commandExecutors()` in `commands_core.go` | See [Hub slash ownership](#hub-slash-ownership) |
| Implementation sessions | `internal/agent/implementation_*.go` | Large graph; prefer surgical fixes |
| Turn pipeline / shortcuts | `internal/agent/turn_pipeline.go`, `artifact_shortcut.go`, `response_echo.go` | Shortcut vs full pipeline — check both |
| File proposals / `[FILE_CHANGE]` | `internal/agent/agent_file_proposal.go` | Chat markdown alone does **not** write disk |
| Turn intent stamps | `internal/intent/` | Semantic stamps for eval — **not** model/knowledge routing |
| Knowledge / model routing | `internal/routing/` | Separate from `internal/intent` |
| Assistant agent | `internal/agent/assistant_agent.go` | Reminders, tasks, notes |
| Live scenarios | `scenarios/` + `make layer-gate` / `make *-scenario` | Do not weaken asserts to go green |
| Away / overnight agent rules | `AGENTS.md`, `docs/AWAY_OPERATIONS.md` | |

## Naming traps

| Trap | Reality |
|------|---------|
| `LAYER=parity` | Test portfolio layer name — **not** the same as `scenarios/parity/` |
| `internal/intent` vs `internal/routing` | Intent = turn stamps; routing = knowledge/model selection |
| Chat message with code fences | Does **not** write files; needs tools / `[FILE_CHANGE]` + Pending changes approval |
| `make gui` / “GUI test” | Tauri desktop; `test/gui_test.go` is hub WebSocket legacy naming |
| Web UI at `/` | Thin chat shell; full IDE is Tauri `desktop/` |
| `implementation_intent.go` “deprecated” stubs | Intentionally false / museum — do not “fix” them into live heuristics without product review |

## Do not touch (without explicit human scope)

- `updater/beta/`, Homebrew bump scripts/workflows, release tags / `make release`
- Live scenario assertion softening (`wait_discussion`, implement `until_*`, chat canaries)
- Auth / ACL / security routes without review
- Secrets, `.env`, API keys in commits
- Drive-by refactors outside the issue/failure brief

## Hub slash ownership

Dispatch: `CommandHandler.commandExecutors()` in [`internal/hub/commands_core.go`](../internal/hub/commands_core.go). Handlers live in the files below.

| File | Slash families (representative) |
|------|----------------------------------|
| `commands_agents.go` | `/create-repo-agent`, `/create-expert`, `/delete-agent`, `/pause-agent`, `/list-agents`, `/remove-agent`, `/recall-agent`, channel CRUD, `/create-cli-agent`, `/open-terminal`, `/tools-list` |
| `commands_confluence.go` | `/create-confluence-agent`, `/reindex-confluence-agent`, `/list-confluence-agents`, `/test-confluence-connection` |
| `commands_exports.go` | `/export-agent-mcp`, `/import-agent-mcp`, `/list-exports`, `/delete-export`, `/export-all-agents`, `/test-anthropic-connection`, `/test-github-connection` |
| `commands_providers.go` | `/switch-provider`, `/switch-all-providers` |
| `commands_files.go` | `/open-file`, `/add-workspace`, `/list-workspaces`, `/approve-file`, `/reject-file`, `/list-file-changes`, `/generate-image`, `/generate-music`, `/analyze-design` |
| `commands_collab.go` | `/collaborate`, `/runbook`, `/runbook-run`, `/approve-plan`, `/submit-plan`, `/resume-plan`, `/revise-plan`, `/cancel-plan`, `/complete-collab`, `/collab-*` |
| `commands_core.go` | `/help`, `/remind*`, `/task-*`, `/note-*`, `/learn*`, `/meeting-*`, `/summarize`, `/help-assistant`, `commandExecutors` map + definitions |

When adding a slash command: register in `commandExecutors()`, add a definition in `buildCommandDefinitions()`, and put the handler in the matching `commands_*.go` file.

## Verify before push

See [TESTING.md](TESTING.md) and [TEST_PORTFOLIO.md](TEST_PORTFOLIO.md).

- Always: `make test-all`
- Go changed: also `make test-race`
- Desktop TS: `cd desktop && npx tsc --noEmit && npm test`
