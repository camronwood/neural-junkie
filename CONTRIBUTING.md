# Contributing to Neural Junkie

Thanks for helping improve Neural Junkie. This guide covers local setup, test gates, and pull request expectations.

## Prerequisites

- **Go 1.23+** — [go.dev/dl](https://go.dev/dl)
- **Node.js 18+** — [nodejs.org](https://nodejs.org)
- **Rust** — [rustup.rs](https://rustup.rs) (Tauri desktop app)
- At least one AI provider (Ollama recommended for local dev)

## Quick start

```bash
git clone https://github.com/camronwood/neural-junkie.git
cd neural-junkie
make gui-install    # first time only
make start-all      # hub + desktop app
```

See [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) for provider configuration and alternate run modes.

## Before you open a PR

Run the relevant gates for your change:

| Change area | Minimum | Recommended |
|-------------|---------|-------------|
| Go hub / agents | `make test-go` | `make test-all` |
| Agent implement / IDE | `make implement-scenarios` | `make test-parity-stable` |
| Collaboration | `make collab-smoke` | `make collab-scenarios-all` |
| Desktop UI | `cd desktop && npm run test` | `make test-all` |

Full parity contract and scenario matrix: [docs/TESTING.md](docs/TESTING.md).

## Pull request expectations

- **Focused diffs** — one logical change per PR when possible.
- **No secrets** — never commit API keys, tokens, or `env.local`. Use `env.example` as reference.
- **Tests** — add or update tests when fixing bugs or changing agent/hub behavior.
- **Docs** — update README, `DOCS.md`, or `docs/CHANGELOG.md` when user-facing behavior changes.
- **Scenarios** — if you change collaboration, implementation sessions, or chat routing, run the matching scenario harness before requesting review.

## Code layout

Internal architecture and design decisions: [docs/DEVELOPMENT_NOTES.md](docs/DEVELOPMENT_NOTES.md).

## Reporting issues

- **Bugs** — use the [bug report template](https://github.com/camronwood/neural-junkie/issues/new?template=bug_report.yml).
- **Features** — use the [feature request template](https://github.com/camronwood/neural-junkie/issues/new?template=feature_request.yml).
- **Known limitations** — check [docs/KNOWN_ISSUES.md](docs/KNOWN_ISSUES.md) before filing duplicates.

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
