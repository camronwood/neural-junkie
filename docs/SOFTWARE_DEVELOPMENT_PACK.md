# Software Development Pack (v1)

Neural Junkie includes an optional **Software development** domain pack for in-process engineering specialists, dev MCP tools, and Qwen Coder models.

## One pack at a time

Install the pack from **Settings → Domain packs → Pack store**, then enable it. You can run **multiple packs** at once. For IDE layout and editor depth, enable the separate **[IDE pack](IDE_PACK.md)**. Enabling another pack adds specialists and tools without changing your layout owner unless you choose one in Settings.

## What you get

| Piece | Description |
|-------|-------------|
| **Qwen 3.5 27B** | `qwen3.5:27b` — shared base for all specialists (pack v2) |
| **Utility tier** | `qwen3.5:9b` — merged into `models_to_ensure` for background tasks |
| **BackendEngineer** | APIs, services, integrations, business logic |
| **FrontendEngineer** | Web/desktop UI, accessibility, design systems |
| **PlatformEngineer** | Deployment, CI/CD, cloud infrastructure |
| **SecurityReviewer** | Auth, encryption, threat modeling, OWASP |
| **SoftwareArchitect** | System design, service boundaries, migrations |
| **DatabaseSpecialist** | SQL, schema design, query optimization |
| **RustExpert** | Cargo, async Rust, WASM (pack v2) |
| **SREObservabilityEngineer** | Prometheus, alerts, traces (pack v2) |
| **MobileEngineer** | React Native, iOS/Android (pack v2) |
| **DataMLEngineer** | Notebooks, datasets, ML pipelines (pack v2) |
| **Dev MCP** | Pack-owned `sd-mcp-server` sidecar (ports 8081–8090, 8095–8097) |

### MCP tool matrix (software development pack)

| Specialist | Port | Example tools |
|------------|------|----------------|
| BackendEngineer | 8081 | `analyze_go_code`, `run_go_tests`, `check_dependencies` |
| PlatformEngineer | 8082 | `kubectl_query`, `validate_yaml`, `check_pod_logs` |
| DatabaseSpecialist | 8083 | `explain_query`, `validate_schema`, `generate_migration` |
| FrontendEngineer | 8084 | `run_typescript_check`, `run_eslint`, `check_package_json` |
| SecurityReviewer | 8085 | `run_gosec`, `run_npm_audit`, `scan_secrets` |
| SoftwareArchitect | 8090 | `validate_yaml`, `validate_schema`, `check_dependencies` |

Code review is a **core behavior** of every specialist (read-only when the user asks for a review) — there is no separate CodeReviewer agent.

Enablement follows pack `mcp_agents` in `pack.yaml`; override per specialist in **Settings → Domain packs → MCP specialist tools**.

## Enable the pack

**Settings → AI & providers → Domain packs** — toggle **Software development** on or off.

When enabled:

- Engineering specialists are added to configured hub agents (toggle triggers reconcile + restart).
- Preset slugs (`backend`, `frontend`, `devops`, `security`, `architecture`, `database`, …) appear in **New DM** and `/create-expert`.
- `qwen3.5:27b` and `qwen3.5:9b` are merged into **models to ensure** for Ollama.
- Optional LoRA adapters (security, backend) are in the separate **[Specialist tuning](SPECIALIST_TUNING_PACK.md)** pack — install and assign manually if desired.
- If **Life sciences** is also enabled, the hub does **not** auto-switch your default Ollama chat model (avoid bio vs coder conflicts); pick the model in Settings.

When disabled, pack-owned specialists are stopped; **Moderator**, **Assistant**, and **auto-detected CLI agents** (Cursor, Gemini, Claude, Copilot, Codex) are unchanged.

You can also enable the pack via the **Developer** setup wizard track (`packs.enabled["software-development"]` in `~/.neural-junkie/config.json`). That track also enables the [IDE pack](IDE_PACK.md).

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

On first load after upgrading, if any legacy specialist (`backend`, `frontend`, etc.) was enabled in `config.json`, the hub auto-enables the software-development pack so existing dev setups keep working. If software-development was enabled, the hub also installs and enables the **IDE** pack (IDE capabilities moved out of this pack).

## See also

- [IDE_PACK.md](IDE_PACK.md) — IDE layout, Git, LSP, composer (separate pack)

- Pack workspace guide (installed pack): `assets/WORKSPACE.md` in [neural-junkie-pack-software-development](https://github.com/camronwood/neural-junkie-pack-software-development)
- [BIOLOGY_PACK.md](BIOLOGY_PACK.md) — Life sciences pack
- [MCP_INTEGRATION.md](MCP_INTEGRATION.md) — MCP ports and tools
- [MCP_EXTERNAL_CLIENTS.md](MCP_EXTERNAL_CLIENTS.md) — Claude Desktop / external MCP hosts
- [CLI_AGENTS.md](CLI_AGENTS.md) — Cursor / Gemini / Claude / Copilot
