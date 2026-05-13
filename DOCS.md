# Neural Junkie — documentation index

Start with the [README](README.md) (overview, screenshots, quick start, command index).

## Guides (`docs/`)

| Topic | File |
|--------|------|
| Setup and first run | [GETTING_STARTED.md](docs/GETTING_STARTED.md) |
| System design | [ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| Repo indexing agents | [REPO_AGENTS.md](docs/REPO_AGENTS.md) |
| Confluence agents | [CONFLUENCE_AGENTS.md](docs/CONFLUENCE_AGENTS.md) |
| Assistant (tasks, reminders, meetings) | [ASSISTANT_AGENT.md](docs/ASSISTANT_AGENT.md) |
| Moderator | [MODERATOR_AGENT.md](docs/MODERATOR_AGENT.md) |
| MCP tool servers | [MCP_INTEGRATION.md](docs/MCP_INTEGRATION.md) |
| MCP export/import | [MCP_EXPORTS.md](docs/MCP_EXPORTS.md) |
| Cursor / Gemini CLI agents | [CLI_AGENTS.md](docs/CLI_AGENTS.md) |
| Agent review (@mentions in threads) | [AGENT_REVIEW.md](docs/AGENT_REVIEW.md) |
| Multi-agent collaboration | [COLLABORATION.md](docs/COLLABORATION.md) |
| Product overview | [USER_VALUE_GUIDE.md](docs/USER_VALUE_GUIDE.md) |
| Maintainer / internals | [DEVELOPMENT_NOTES.md](docs/DEVELOPMENT_NOTES.md) |
| Release history | [CHANGELOG.md](docs/CHANGELOG.md) |
| Roadmap / ideas | [FUTURE_ENHANCEMENTS.md](docs/FUTURE_ENHANCEMENTS.md) |
| Current status | [STATUS.md](docs/STATUS.md) |

## Static site

- **Marketing / landing (GitHub Pages style):** [docs/index.html](docs/index.html) + [docs/css/landing.css](docs/css/landing.css)
- **Hub** (when `make server` is running): Web chat at `/`.
- **Optional static preview** of `public/index.html`: serve the **repository root** with a simple HTTP server so `../assets/screenshots/` resolves (footer in that file explains the URL pattern).

## Examples

Scenario write-ups live under [examples/](examples/).
