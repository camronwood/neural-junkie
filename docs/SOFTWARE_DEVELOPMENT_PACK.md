# Software Development Pack (v1)

Neural Junkie includes an optional **Software development** domain pack for in-process engineering specialists, dev MCP tools, and Qwen Coder models.

## One pack at a time

Install the pack from **Settings → Domain packs → Pack store**, then enable it. You can run **multiple packs** at once; the **first pack you enable** sets the UI layout (IDE vs team). Enabling another pack adds specialists and tools without changing your layout.

## What you get

| Piece | Description |
|-------|-------------|
| **IDE v1** (dev pack only) | Git modal (status, commit, pull, push), quick open (⌘P), editor selection sent to agents with workspace context |
| **IDE v2/v2c** (dev pack only) | Git SCM, symbols, Problems, inline hunks, fast edit (⌘K), IDE layout, LSP-lite (Go/Rust/Python), inline completion. See [IDE_V2.md](IDE_V2.md) |
| **IDE v3** (dev pack only) | Main chat IDE mode (Ask/Agent, @codebase, specialist routing), review bar. See [IDE_V3.md](IDE_V3.md) |
| **Qwen 3.5 27B** | `qwen3.5:27b` — shared base for all specialists |
| **Utility tier** | `qwen3.5:9b` — merged into `models_to_ensure` for background tasks |
| **BackendEngineer** | APIs, services, integrations, business logic |
| **FrontendEngineer** | Web/desktop UI, accessibility, design systems |
| **PlatformEngineer** | Deployment, CI/CD, cloud infrastructure |
| **SecurityReviewer** | Auth, encryption, threat modeling, OWASP |
| **SoftwareArchitect** | System design, service boundaries, migrations |
| **CodeReviewer** | Correctness, maintainability, tests, regressions |
| **DatabaseSpecialist** | SQL, schema design, query optimization |
| **Dev MCP** | MCP tool servers for backend, frontend, platform, database, security, code review, and architecture specialists |

### MCP tool matrix (software development pack)

| Specialist | Port | Example tools |
|------------|------|----------------|
| BackendEngineer | 8081 | `analyze_go_code`, `run_go_tests`, `check_dependencies` |
| PlatformEngineer | 8082 | `kubectl_query`, `validate_yaml`, `check_pod_logs` |
| DatabaseSpecialist | 8083 | `explain_query`, `validate_schema`, `generate_migration` |
| FrontendEngineer | 8084 | `run_typescript_check`, `run_eslint`, `check_package_json` |
| SecurityReviewer | 8085 | `run_gosec`, `run_npm_audit`, `scan_secrets` |
| CodeReviewer | 8089 | `analyze_go_code`, `run_eslint` (read-only review) |
| SoftwareArchitect | 8090 | `validate_yaml`, `validate_schema`, `check_dependencies` |

Enablement follows pack `mcp_agents` in `pack.yaml`; override per specialist in **Settings → Domain packs → MCP specialist tools**.

## Enable the pack

**Settings → AI & providers → Domain packs** — toggle **Software development** on or off.

When enabled:

- Seven engineering specialists are added to configured hub agents (toggle triggers reconcile + restart).
- Preset slugs (`backend`, `frontend`, `devops`, `security`, `architecture`, `code-review`, `database`) appear in **New DM** and `/create-expert`.
- `qwen3.5:27b` and `qwen3.5:9b` are merged into **models to ensure** for Ollama.
- Optional LoRA adapters (security, code-review, backend) are in the separate **[Specialist tuning](SPECIALIST_TUNING_PACK.md)** pack — install and assign manually if desired.
- If **Life sciences** is also enabled, the hub does **not** auto-switch your default Ollama chat model (avoid bio vs coder conflicts); pick the model in Settings.

When disabled, pack-owned specialists are stopped; **Moderator**, **Assistant**, and **auto-detected CLI agents** (Cursor, Gemini, Claude, Copilot, Codex) are unchanged.

You can also enable the pack via the **Software development** setup wizard track (`packs.enabled["software-development"]` in `~/.neural-junkie/config.json`).

## Install models

```bash
ollama pull qwen3.5:27b
ollama pull qwen3.5:9b
```

Or use **Model library** (⇧⌘M) → **Ollama** tab.

## Core runtime (not in this pack)

These are always available when the hub starts (no pack toggle):

- **ChatModerator** — commands and chat help
- **Assistant** — reminders, tasks, notes
- **CLI agents** — auto-joined to `#general` when binaries are on PATH ([CLI_AGENTS.md](CLI_AGENTS.md))

## Team chat & productivity wizard

The **general** wizard track enables Assistant + utility Ollama model with **both** domain packs off. CLI tools still auto-detect.

## Upgrade from older configs

On first load after upgrading, if any legacy specialist (`backend`, `frontend`, etc.) was enabled in `config.json`, the hub auto-enables the software-development pack so existing dev setups keep working.

## See also

- [BIOLOGY_PACK.md](BIOLOGY_PACK.md) — Life sciences pack
- [MCP_INTEGRATION.md](MCP_INTEGRATION.md) — MCP ports and tools
- [MCP_EXTERNAL_CLIENTS.md](MCP_EXTERNAL_CLIENTS.md) — Claude Desktop / external MCP hosts
- [CLI_AGENTS.md](CLI_AGENTS.md) — Cursor / Gemini / Claude / Copilot
