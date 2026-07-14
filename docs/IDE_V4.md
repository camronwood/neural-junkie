# IDE v4 — full editor depth, remote workspaces

**Status:** Shipped in v1.2 — full Monaco LSP client, remote SSH via `nj-remote` sidecar, dev container attach plan, tree-sitter symbols.

IDE v4 completes the [IDE pack](IDE_PACK.md) editor story started in [IDE_V2.md](IDE_V2.md) and [IDE_V3.md](IDE_V3.md). Remote SSH attach UI is core; full LSP on remote hosts requires the IDE pack.

## What's new in v4

| Feature | Description |
|---------|-------------|
| **Full LSP (local)** | Persistent `gopls` / `rust-analyzer` / `pyright-langserver` via hub WebSocket; `didOpen`/`didChange`/`didClose`; real-time diagnostics; Monaco hover, completion, go-to-definition, find references, rename |
| **LSP-lite fallback** | Workspace-wide Go/Rust/Python squiggles via REST when WebSocket session is unavailable |
| **WorkspaceBackend** | Pluggable local + remote filesystem/exec (`internal/workspacebackend/`) — file IO, git, search, symbols, file changes, `@codebase` |
| **nj-remote sidecar** | `cmd/nj-remote` — HTTP FS, one-shot exec, PTY WebSocket on SSH host or container |
| **Remote SSH workspaces** | Desktop **Remote SSH** wizard + `POST /api/workspaces/connect-remote`; token persist + sidecar health check on connect and hub startup |
| **Remote terminal** | Hub proxies `GET /api/terminal/ws` → sidecar PTY; desktop routes terminal tabs and one-shot exec for remote workspaces |
| **Dev containers** | `GET /api/workspaces/devcontainer-plan` — parse `.devcontainer/devcontainer.json` (run `nj-remote` in container after attach) |
| **tree-sitter symbols** | Optional upgrade to symbol index when `tree-sitter` CLI is on PATH |
| **Editor trust** | `yolo` mode auto-approves file changes like `auto_apply_edits` |

## LSP APIs

| Endpoint | Purpose |
|----------|---------|
| `GET /api/lsp/ws?workspace=&lang=` | WebSocket JSON-RPC relay (notifications + requests; `publishDiagnostics` push) |
| `POST /api/lsp/hover` | Hover with optional `didOpen` bootstrap |
| `POST /api/lsp/request` | Arbitrary LSP request (completion, definition, references, rename) |
| `GET /api/lsp/{go,rust,python}/diagnostics` | LSP-lite fallback (one-shot CLI) |

Desktop: `desktop/src/lsp/lspConnection.ts` manages document sync; `useMonacoLSP.ts` registers Monaco providers.

## Remote workspace flow

```mermaid
sequenceDiagram
  participant Desktop
  participant Hub
  participant Sidecar as nj_remote
  participant FS as Remote_FS

  Desktop->>Hub: POST connect-remote
  Hub->>Sidecar: GET /health
  Hub->>Hub: Register RemoteBackend + persist token
  Desktop->>Hub: GET /api/files
  Hub->>Sidecar: GET /api/fs/list
  Sidecar->>FS: os.ReadDir
  Desktop->>Hub: GET /api/terminal/ws
  Hub->>Sidecar: GET /api/pty/ws
```

1. Start sidecar on remote host: `nj-remote -root /path/to/repo -addr :19876 -token SECRET`
2. Tunnel (if needed): `ssh -L 19876:127.0.0.1:19876 user@host`
3. Desktop **Add workspace → Remote SSH** → hub stores `sidecar_url`, `kind=ssh`
4. File/git/search/symbols/file-changes/`@codebase`/terminal route through `WorkspaceBackend`

## nj-remote

```bash
make build-nj-remote
./nj-remote -root ~/projects/myapp -addr :19876 -token "$NJ_REMOTE_TOKEN"
```

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Sidecar liveness (hub checks on connect/startup) |
| `GET /api/fs/list` | Directory listing |
| `GET /api/fs/read` | Read file |
| `POST /api/fs/write` | Write file |
| `GET /api/fs/stat` | Stat |
| `POST /api/exec` | One-shot command |
| `GET /api/pty/ws` | Interactive shell (WebSocket) |

## Dev containers

Place `.devcontainer/devcontainer.json` in repo root. NJ reads attach plan via:

- `GET /api/workspaces/devcontainer-plan?workspace=<id>` — active or linked workspace (remote-aware)
- `GET /api/workspaces/devcontainer-plan?path=<local-repo>` — wizard before connect

Desktop **Add workspace → Dev container**: browse local repo, load plan, tunnel sidecar, connect with `kind=devcontainer`.

Returns container name, image, workspace folder, sidecar port. Run `nj-remote` inside the container after `devcontainer up`.

## v4.1 (shipped in v1.2.0-beta.2+)

| Feature | Status |
|---------|--------|
| **Remote LSP** | Hub proxies `GET /api/lsp/ws` → sidecar → `gopls`/`rust-analyzer`/`pyright` on remote host |
| **Dev container UI** | Desktop wizard tab + path/workspace plan load |
| **Remote collab worktrees** | `git worktree` via sidecar exec under `collabs/worktrees/<id>/` |
| **MCP on remote** | `ContextWithBackend` on agent tool calls; workspace `read_file`/`list_dir`/`run_command` route through sidecar |

## Editor trust (unchanged from v3)

| Mode | Behavior |
|------|----------|
| `interactive` | Manual file-change approval |
| `auto_apply_edits` | Auto-approve + verify when possible |
| `yolo` | Auto-approve file changes (tool parity) |

## Related

- [REMOTE_WORKSPACES.md](REMOTE_WORKSPACES.md) — security model
- [IMPLEMENTATION_SESSION.md](IMPLEMENTATION_SESSION.md) — agent sessions on local and remote backends
- [SOFTWARE_DEVELOPMENT_PACK.md](SOFTWARE_DEVELOPMENT_PACK.md)
