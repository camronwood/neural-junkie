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
| **Qwen Coder 14B** | `qwen2.5-coder:14b` — recommended Ollama model for specialists |
| **Utility tier** | `qwen2.5:7b` — merged into `models_to_ensure` for background tasks |
| **BackendEngineer** | APIs, services, integrations, business logic |
| **FrontendEngineer** | Web/desktop UI, accessibility, design systems |
| **PlatformEngineer** | Deployment, CI/CD, cloud infrastructure |
| **SecurityReviewer** | Auth, encryption, threat modeling, OWASP |
| **SoftwareArchitect** | System design, service boundaries, migrations |
| **CodeReviewer** | Correctness, maintainability, tests, regressions |
| **Dev MCP** | Backend and DevOps analysis tools for enabled specialists |

## Enable the pack

**Settings → AI & providers → Domain packs** — toggle **Software development** on or off.

When enabled:

- Six broad engineering specialists are added to configured hub agents (toggle triggers reconcile + restart).
- Preset slugs (`backend`, `frontend`, `devops`, `security`, `architecture`, `code-review`) appear in **New DM** and `/create-expert`.
- `qwen2.5-coder:14b` and `qwen2.5:7b` are merged into **models to ensure** for Ollama.
- If **Life sciences** is also enabled, the hub does **not** auto-switch your default Ollama chat model (avoid bio vs coder conflicts); pick the model in Settings.

When disabled, pack-owned specialists are stopped; **Moderator**, **Assistant**, and **auto-detected CLI agents** (Cursor, Gemini, Claude, Copilot, Codex) are unchanged.

You can also enable the pack via the **Software development** setup wizard track (`packs.enabled["software-development"]` in `~/.neural-junkie/config.json`).

## Install models

```bash
ollama pull qwen2.5-coder:14b
ollama pull qwen2.5:7b
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
- [CLI_AGENTS.md](CLI_AGENTS.md) — Cursor / Gemini / Claude / Copilot
