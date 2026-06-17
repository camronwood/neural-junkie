# Remote SSH dogfood — 2026-06-17

Operator notes from EC2 remote workspace validation (`i-06fbd5c90ad9871f9`, Dev VPC).

## Setup

```bash
# Local
make build-nj-remote

# EC2 (after copying binary or building on host)
export NJ_REMOTE_TOKEN="$(openssl rand -hex 16)"
nj-remote -root /home/ec2-user/neural-junkie -addr :19876 -token "$NJ_REMOTE_TOKEN"

# Laptop tunnel
ssh -L 19876:127.0.0.1:19876 -i ~/Downloads/camron-dev-ec2.pem ec2-user@i-06fbd5c90ad9871f9
```

Desktop: **Add workspace → Remote SSH** — host, user, remote path, sidecar URL `http://127.0.0.1:19876`, token.

## Verified (v1.2.0-beta.1 baseline)

| Area | Result |
|------|--------|
| Sidecar health on connect | Pass — hub rejects connect when sidecar down |
| File explorer list/read | Pass via `WorkspaceBackend` |
| Git panel status | Pass when remote repo has `.git` |
| Integrated terminal | Pass — hub PTY proxy to sidecar |
| `@codebase` / semantic search | Pass — indexes remote tree through backend |
| Implementation verify (`npm test`, `go test`) | Pass when tool chain on remote PATH |

## Gaps found (addressed in v4.1 / beta.2+)

| Gap | Phase B fix |
|-----|-------------|
| Monaco LSP (hover/completion) failed on remote | B1 — hub LSP WS → sidecar → `gopls` on remote |
| `is_git_repo` false until sidecar git check | Connect handler runs `git rev-parse` via backend |
| Dev container attach manual only | B2 — Dev container wizard tab |
| `/collaborate --worktree` on remote | B3 — worktree via sidecar `git worktree` |
| Agent MCP tools hit local disk | B4 — `ContextWithBackend` + workspace tool routing |

## Friction (still operator-heavy)

- Must run `nj-remote` manually on host/container after reboot
- SSH tunnel required when sidecar binds loopback only
- Remote language servers must be installed on remote PATH (`gopls`, etc.)
- First connect: paste sidecar token (stored in hub keychain file per workspace id)

## Re-test after beta.2

1. Remote workspace → open Go file → hover/completion via LSP WS
2. Dev container tab → load plan from local repo path → connect
3. Active remote workspace → `/collaborate --worktree` with software-dev pack
4. BackendEngineer `run_go_tests` / `read_file` on remote via agent turn
