# Neural Junkie — documentation index

Start with the [README](README.md) (overview, screenshots, quick start, command index).

## Guides (`docs/`)

| Topic | File |
|--------|------|
| Download and first run (beta) | [DOWNLOAD.md](docs/DOWNLOAD.md) |
| Setup and first run | [GETTING_STARTED.md](docs/GETTING_STARTED.md) |
| Hardware requirements | [HARDWARE.md](docs/HARDWARE.md) |
| System design | [ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Repo indexing agents | [REPO_AGENTS.md](docs/REPO_AGENTS.md) |
| Confluence agents | [CONFLUENCE_AGENTS.md](docs/CONFLUENCE_AGENTS.md) |
| Assistant (tasks, reminders, meetings) | [ASSISTANT_AGENT.md](docs/ASSISTANT_AGENT.md) |
| Google Meet notes | [GOOGLE_MEET_NOTES.md](docs/GOOGLE_MEET_NOTES.md) |
| Moderator | [MODERATOR_AGENT.md](docs/MODERATOR_AGENT.md) |
| MCP tool servers | [MCP_INTEGRATION.md](docs/MCP_INTEGRATION.md) |
| MCP export/import | [MCP_EXPORTS.md](docs/MCP_EXPORTS.md) |
| Cursor / Gemini CLI agents | [CLI_AGENTS.md](docs/CLI_AGENTS.md) |
| Agent review (@mentions in threads) | [AGENT_REVIEW.md](docs/AGENT_REVIEW.md) |
| Multi-agent collaboration | [COLLABORATION.md](docs/COLLABORATION.md) |
| LoRA adapters (import & compose) | [LORA_ADAPTERS.md](docs/LORA_ADAPTERS.md) |
| LoRA training (in-app wizard) | [LORA_TRAINING.md](docs/LORA_TRAINING.md) |
| Product overview | [USER_VALUE_GUIDE.md](docs/USER_VALUE_GUIDE.md) |
| Maintainer / internals | [DEVELOPMENT_NOTES.md](docs/DEVELOPMENT_NOTES.md) |
| Release history | [CHANGELOG.md](docs/CHANGELOG.md) |
| Roadmap / ideas | [FUTURE_ENHANCEMENTS.md](docs/FUTURE_ENHANCEMENTS.md) |
| Current status | [STATUS.md](docs/STATUS.md) |
| Known issues (beta) | [KNOWN_ISSUES.md](docs/KNOWN_ISSUES.md) · [known-issues.html](docs/known-issues.html) |

## Marketing / publish copy (`docs/marketing/`)

| Topic | File |
|--------|------|
| Multi-agent collaboration (LinkedIn article) | [COLLABORATION-LINKEDIN.md](docs/marketing/COLLABORATION-LINKEDIN.md) |
| Conversational & collab test harness (LinkedIn article) | [CONVERSATIONAL-TEST-HARNESS.md](docs/marketing/CONVERSATIONAL-TEST-HARNESS.md) |
| LoRA adapters & training (LinkedIn article) | [LORA-LINKEDIN.md](docs/marketing/LORA-LINKEDIN.md) |
| Hardware requirements (LinkedIn article) | [HARDWARE-LINKEDIN.md](docs/marketing/HARDWARE-LINKEDIN.md) |
| Collab craft ad | [COLLAB-CRAFT-AD.md](docs/marketing/COLLAB-CRAFT-AD.md) |

## Static site

- **Marketing / landing:** [docs/index.html](docs/index.html) + [docs/css/landing.css](docs/css/landing.css)
- **Known issues (living beta list):** [known-issues.html](docs/known-issues.html) — edit alongside [KNOWN_ISSUES.md](docs/KNOWN_ISSUES.md); remove rows when fixed.
- **Feature deep dives (static HTML):** [docs/features/](docs/features/) — capability pages + [index](docs/features/index.html), linked from the landing “What you get” cards.
- **Hub** (when `make server` is running): Web chat at `/`.
- **Optional static preview** of `public/index.html`: serve the **repository root** with a simple HTTP server so `../assets/screenshots/` resolves (footer in that file explains the URL pattern).

## Examples

Scenario write-ups live under [examples/](examples/).
