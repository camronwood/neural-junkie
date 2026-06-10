# Neural Junkie — Security

Neural Junkie is a **local-first** developer tool: the hub and desktop app are designed to run on your machine, not as a multi-tenant internet service.

## Threat model (default install)

| Threat | Mitigation |
|--------|------------|
| Random website calls `http://127.0.0.1:18765` | Hub binds **127.0.0.1**; restricted CORS; sensitive routes need loopback or hub token |
| Path traversal (workspace / hub data) | `pathutil.WithinRoot`, `resolveHubDataPath` |
| XSS in chat markdown | DOMPurify; `style` attribute disallowed |
| Agent impersonation via `/api/send` | `from` ignored unless `NEURAL_JUNKIE_HUB_TOKEN` is set |
| Hub on LAN | Opt-in `NEURAL_JUNKIE_LISTEN_ALL=1` + `NEURAL_JUNKIE_HUB_TOKEN` |
| Credential theft from disk | Tauri: AES-GCM in `credentials.dat` (machine-bound key) |
| API keys / Slack tokens in config | AES-GCM in `config.json` via `secrets.key` or `NEURAL_JUNKIE_CONFIG_KEY` |
| API abuse | Per-IP/session rate limits (disable with `NEURAL_JUNKIE_RATE_LIMIT=0`) |

## Authentication and channel ACLs

1. Desktop calls `POST /api/auth/session` with `{ "username": "Camron" }` on login.
2. Hub returns `token` → desktop sends `X-NJ-Session` on **all** API calls (`chatAPI.hubFetch`).
3. **Channel ACL** (when a session header is present, or `NEURAL_JUNKIE_AUTH_REQUIRED=1`):
   - **Public** channels: any authenticated user
   - **DM** `dm-{user}-{agent}`: only matching username slug
   - **Custom** with `created_by`: creator + `human_members` list
   - **Collaboration** rooms: open to authenticated users on this hub

Strict mode (always require session for mutations):

```bash
export NEURAL_JUNKIE_AUTH_REQUIRED=1
```

**Desktop:** Settings → **Security** shows hub token, strict auth, and listen-all status (`GET /api/system/security`).

## Rate limiting

Default: **300 GET** / **120 mutating** requests per minute per client key (IP, session, or hub token).

| Variable | Purpose |
|----------|---------|
| `NEURAL_JUNKIE_RATE_LIMIT=0` | Disable |
| `NEURAL_JUNKIE_RATE_READ` | Override GET limit |
| `NEURAL_JUNKIE_RATE_MUTATE` | Override POST/PUT/DELETE limit |

## Secrets at rest

### Hub `config.json`

On save, these fields are encrypted with prefix `enc:v1:`:

- Slack `app_token`, `bot_token`, `client_secret`
- Hugging Face `hf.token`
- Each AI provider `api_key`

Key material:

- `NEURAL_JUNKIE_CONFIG_KEY` — 64 hex chars (32 bytes), or
- Auto-created `~/.neural-junkie/secrets.key` (mode `0600`)

`config.json` is written with mode **0600**.

### Desktop credentials

In Tauri builds, remembered login is stored as an encrypted blob (`encrypt_credential_blob` / `decrypt_credential_blob`) using a key derived from home directory + OS username. Browser-only Vite dev still uses plaintext store.

## Environment variables

| Variable | Purpose |
|----------|---------|
| `NEURAL_JUNKIE_HUB_TOKEN` | Shared secret for non-loopback access (`X-NJ-Hub-Token`) |
| `VITE_NJ_HUB_TOKEN` | Same for desktop |
| `NEURAL_JUNKIE_CONFIG_KEY` | 32-byte hex key for config encryption |
| `NEURAL_JUNKIE_AUTH_REQUIRED=1` | Require session on all mutations |
| `NEURAL_JUNKIE_LISTEN_ALL=1` | Bind `0.0.0.0` |
| `NEURAL_JUNKIE_CORS_ANY=1` | Wildcard CORS (legacy) |
| `NEURAL_JUNKIE_SESSION_TTL_HOURS` | Session lifetime (default 168h) |
| `NEURAL_JUNKIE_DEBUG=1` | Debug routes + pprof (loopback) |

## Local-only routes (hub access)

Require loopback **or** valid `NEURAL_JUNKIE_HUB_TOKEN` (`X-NJ-Hub-Token`, `Authorization: Bearer`, or WebSocket `?hub_token=`). Many also require session + channel ACL when `NEURAL_JUNKIE_AUTH_REQUIRED=1` or `X-NJ-Session` is sent.

| Route pattern | Methods | Notes |
|---------------|---------|-------|
| `/ws` | GET (upgrade) | Real-time chat; hub token via query when LAN-exposed |
| `/api/send`, `/api/broadcast` | POST | Mutations; session + ACL when enforced |
| `/api/hub-data/read` | POST | Hub data reads |
| `/api/workspaces`, `/api/files`, `/api/file-content` | GET/POST | Workspace file APIs |
| `/api/file-create`, `/api/file-rename`, `/api/file-delete` | POST | Workspace mutations |
| `/api/file-changes/*` (approve/reject/propose) | POST | File change workflow |
| `/api/git-*` | GET/POST | Git operations (also requires software-dev pack) |
| `/api/ollama/install`, `/start`, `/stop`, `/pull`, `/delete` | POST | Ollama lifecycle |
| `/api/hf/download`, `/delete`, `/import-ollama`, `/local` | GET/POST | Hugging Face downloads |
| `/api/agents/restart` | POST | Restart configured agents |
| `/api/lora/train` | POST | LoRA training |
| `/api/channels/delete`, `/clear-history`, `/api/channel-export` | POST/GET | Channel admin |
| `/api/local-image` | GET | Local image proxy |

Routes not listed here (e.g. `/api/messages`, `/api/health`, read-only Ollama status) remain CORS + rate-limit protected only; safe on default loopback bind.

## Production checklist

1. Set `NEURAL_JUNKIE_HUB_TOKEN` and `NEURAL_JUNKIE_CONFIG_KEY` to random secrets.
2. Set `NEURAL_JUNKIE_AUTH_REQUIRED=1` on shared machines.
3. Do not use `NEURAL_JUNKIE_LISTEN_ALL` or `NEURAL_JUNKIE_CORS_ANY` without TLS + proxy auth.
4. Restrict `config.json`, `secrets.key`, and `credentials.dat` file permissions.
5. Review MCP tools (shell, kubectl, docker) — they run as your OS user.

## Reporting

Contact internal security or maintainers directly for vulnerabilities (no public issues for unfixed bugs).
