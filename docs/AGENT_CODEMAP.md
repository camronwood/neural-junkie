# Agent code map

Where to edit what. Prefer this over guessing from file size or similar names.

**Desktop shell is Tauri** (`desktop/src-tauri`), not Electron. Do not invent `BrowserWindow` / `main.js` without inspecting the tree.

## Edit here for X

| If you need to change… | Start here | Notes |
|------------------------|------------|-------|
| Hub HTTP client (desktop) | `desktop/src/api/domains/*` then thin facade `desktop/src/api/chatAPI.ts` | Keep public `ChatAPI` method names stable; put new endpoints in a domain module — see [Domain convention](#domain-convention) |
| Protocol / shared TS types | `desktop/src/types/protocol.ts` | ChatAPI-local DTOs live under `desktop/src/api/types/` |
| Chat UI (messages, composer, channel toolbar) | `desktop/src/components/ChatWindow.tsx` (re-export) → `desktop/src/components/chat/ChatWindowRoot.tsx` plus `chat/` (`ChatMessageList`, `ChatInputArea`, `ChatChannelToolbar`, `ChatModalHost`) | Prefer extracting into `chat/` over growing ChatWindowRoot |
| Chat send / inbound / command / DM wiring | `desktop/src/hooks/createChat*.ts` and `useChat*.ts` | Orchestration lives in hooks; ChatWindow composes them |
| Chat client state | `desktop/src/stores/chatStore.ts` | Channel, agents, connection — not hub phase machine |
| Collab UI | `desktop/src/components/CollaborationPanel.tsx` | |
| Collab phase machine (hub) | `internal/collaboration/manager.go` | High flake risk; prefer surgical fixes with evidence |
| File explorer / IDE chrome | `desktop/src/components/FileExplorerPanel.tsx` (shell) + `desktop/src/components/fileExplorer/` (`useFileExplorerPanelModel`, `FileExplorerTree`, `FileExplorerContextMenu`, add-workspace / remove confirm, utils) | Prefer extracting into `fileExplorer/` over growing the panel |
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
| Hub WS/HTTP package tests | `test/hub_websocket_test.go` | Formerly misnamed `gui_test.go` — **not** Tauri GUI |
| Live scenarios | `scenarios/` + `make layer-gate` / `make *-scenario` | Do not weaken asserts to go green |
| Away / overnight agent rules | `AGENTS.md`, `docs/AWAY_OPERATIONS.md` | |

## Domain convention

New hub HTTP endpoint for the desktop client:

1. Add or extend a module under `desktop/src/api/domains/*Api.ts` (e.g. `workspaceApi`, `filesApi`, `ideApi`).
2. Add a one-line delegate on `ChatAPI` only if callers still use the facade (keep method names stable).
3. Put shared DTOs in `desktop/src/api/types/` or `desktop/src/types/protocol.ts` when cross-cutting.
4. Do **not** grow real `hubFetch` bodies inside `chatAPI.ts` (transport + delegates only).

## Naming traps

| Trap | Reality |
|------|---------|
| `LAYER=parity` | Test portfolio layer name — **not** the same as `scenarios/parity/` |
| `internal/intent` vs `internal/routing` | Intent = turn stamps; routing = knowledge/model selection |
| Chat message with code fences | Does **not** write files; needs tools / `[FILE_CHANGE]` + Pending changes approval |
| `make gui` / “GUI test” | Tauri desktop app via Makefile; hub WS/HTTP unit tests live in `test/hub_websocket_test.go` (renamed from `gui_test.go`) |
| Web UI at `/` | Thin chat shell; full IDE is Tauri `desktop/` |
| `implementation_intent.go` “deprecated” stubs | Intentionally false / museum — do not “fix” them into live heuristics without product review |

## Do not touch (without explicit human scope)

- `updater/beta/`, Homebrew bump scripts/workflows, release tags / `make release`
- Live scenario assertion softening (`wait_discussion`, implement `until_*`, chat canaries)
- Auth / ACL / security routes without review
- Secrets, `.env`, API keys in commits
- Drive-by refactors outside the issue/failure brief
- Broad rewrites of `hub_dispatch.go`, `implementation_session.go`, `collaboration/manager.go`, or `main.rs`

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
