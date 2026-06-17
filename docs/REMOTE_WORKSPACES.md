# Remote workspaces (IDE v4)

Neural Junkie v4 supports **SSH** and **dev container** workspaces via the **`nj-remote` sidecar** — a small HTTP service that exposes filesystem and exec APIs on the machine where code actually lives.

## Architecture

- **Desktop + local hub** stay on your laptop (agents, chat, approvals).
- **`nj-remote`** runs on the remote host or inside a dev container.
- Hub **`WorkspaceBackend`** routes file/git/LSP/exec to local `os.*` or remote sidecar.

## Authentication

- Sidecar optional **Bearer token** (`-token` flag).
- Hub stores `sidecar_url` on workspace record after connect.
- SSH host key verification is the operator's responsibility when deploying sidecar.

## Path containment

- Sidecar resolves all paths under `-root` using `pathutil.WithinRoot` (same as local hub).
- Exec API runs commands with `cwd` under workspace root only.

## Threat model (summary)

| Risk | Mitigation |
|------|------------|
| Token leak | Rotate token; bind sidecar to localhost or VPC security group |
| Arbitrary exec | Commands scoped to workspace root; no shell interpolation in API |
| Data exfil | Sidecar only serves configured root directory |

## Deploy on EC2 (example)

```bash
# On remote host
go install github.com/camronwood/neural-junkie/cmd/nj-remote@latest
nj-remote -root /home/ec2-user/myproject -addr 127.0.0.1:19876 -token "$NJ_REMOTE_TOKEN"

# SSH tunnel from laptop
ssh -L 19876:127.0.0.1:19876 ec2-user@i-06fbd5c90ad9871f9

# Desktop: connect-remote with sidecar_url http://127.0.0.1:19876
```

## Related

- [IDE_V4.md](IDE_V4.md)
- [SECURITY.md](SECURITY.md)
