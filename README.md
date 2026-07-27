```
 ███╗   ██╗███████╗██╗   ██╗██████╗  █████╗ ██╗
 ████╗  ██║██╔════╝██║   ██║██╔══██╗██╔══██╗██║
 ██╔██╗ ██║█████╗  ██║   ██║██████╔╝███████║██║
 ██║╚██╗██║██╔══╝  ██║   ██║██╔══██╗██╔══██║██║
 ██║ ╚████║███████╗╚██████╔╝██║  ██║██║  ██║███████╗
 ╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝
       ╦╦ ╦╔╗╔╦╔═╦╔═╗
       ║║ ║║║║╠╩╗║║╣
      ╚╝╚═╝╝╚╝╩ ╩╩╚═╝
```

# Neural Junkie

> Talk to any AI model. Local or cloud. One workspace.

Open-source **multi-agent desktop hub** — bring your own model (Ollama, Claude, OpenAI-compatible, LM Studio, Cursor/Gemini CLI), run specialist agents from domain packs, and keep humans in the loop with file-change and tool approval. Local-first by default.

| Any model | Specialist agents | Human control |
|-----------|-------------------|---------------|
| Per-agent providers, local or cloud | Packs for software, IDE, AWS, life sciences, and more | Approval gates, scoped context — not a shared hive mind |

Agents, tools, and runbooks are **portable composition units** you can share and hydrate — not hub-locked state. See the [Composition Model](https://camronwood.github.io/neural-junkie/articles/composition-model.html) and [Context Stack](https://camronwood.github.io/neural-junkie/articles/context-stack.html).

**Site:** [camronwood.github.io/neural-junkie](https://camronwood.github.io/neural-junkie/) · **5-minute guide:** [start-here.html](https://camronwood.github.io/neural-junkie/start-here.html)

## Install

**macOS (Homebrew):**

```bash
brew tap camronwood/tap
brew install --cask neural-junkie
```

**Linux (Homebrew):**

```bash
brew tap camronwood/tap
brew install neural-junkie
```

See [docs/HOMEBREW.md](docs/HOMEBREW.md) for upgrades and tap details.

**All platforms — GitHub Releases:** [latest release](https://github.com/camronwood/neural-junkie/releases/latest) or the [downloads page](https://camronwood.github.io/neural-junkie/download.html):

| Platform | File |
|----------|------|
| **macOS (Apple Silicon)** | `.dmg` with `aarch64` in the name |
| **macOS (Intel)** | `.dmg` with `x64` or `x86_64` in the name |
| **Windows** | `.msi` installer (or `.exe` setup) |
| **Linux** | `.deb` (x86_64) |

The Go hub is bundled as a Tauri sidecar — you do **not** need Go installed to run the desktop app. Use the **Tauri desktop app** for the full workspace; `http://localhost:18765` is a lightweight browser chat client only.

**macOS:** Official Release builds are **ad-hoc signed**. If Gatekeeper blocks first launch, right-click → **Open**. Notarization is planned for a later stable cut — see [docs/INSTALL_TRUST.md](docs/INSTALL_TRUST.md).

**Ollama:** macOS installers bundle the Ollama runtime; Windows/Linux use slim installers and the setup wizard can install Ollama on first launch. First run pulls a default model once (internet required that one time). Cloud APIs remain optional.

**Auto-update:** macOS and Windows check for updates in-app (Settings → About → **Check for updates**). Linux `.deb` upgrades are manual. Details: [docs/RELEASE_UPDATES.md](docs/RELEASE_UPDATES.md).

**Quick start after install:** [docs/DOWNLOAD.md](docs/DOWNLOAD.md)

## Five-minute first win

Fresh installs start lean: **Assistant** plus any auto-detected CLI agents (Cursor, Gemini, …). Domain packs are **off** until you enable them.

1. Open **Neural Junkie** and finish the setup wizard (local Ollama and/or a cloud key).
2. **Settings → Domain packs → Pack store** — install and enable **Software development** (specialists) and optionally **IDE** (editor depth).
3. Create a repo expert: `/create-repo-agent /path/to/repo MyRepoExpert`
4. Ask: `@MyRepoExpert summarize the architecture and top risk areas`
5. Collaborate: `/collaborate @BackendEngineer @SecurityReviewer harden auth middleware` — review proposals and approve file changes when prompted.

More: [docs/USER_VALUE_GUIDE.md](docs/USER_VALUE_GUIDE.md) · marketing [start-here](https://camronwood.github.io/neural-junkie/start-here.html)

## What you get

- **BYOM** — Bundled Ollama, Claude, OpenAI-compatible APIs, LM Studio, and CLI agents; switch providers per agent or globally
- **Multi-agent workspace** — Channels, DMs, threads, bounded `/collaborate` sessions, Slack Connect, human approval for file edits and tools
- **Neural Canvas** — Durable agent reports, charts, timelines, diagrams, and workbench artifacts beside chat
- **IDE v4 + NJ Fix Loop** — Monaco LSP, Ask/Agent composer, remote SSH via `nj-remote`, closed-loop boot/build repair ([IDE pack](docs/IDE_PACK.md), [IDE v4](docs/IDE_V4.md))
- **Domain packs (12)** — IDE, software development, life sciences, CAD, specialist tuning, AWS, incident management, web browser, music creation, model arena, maps, room-chat — plus Pack Dev Studio for custom packs ([PACKS.md](docs/PACKS.md))
- **Composition + scoped context** — Share Agent bundles, MCP tool grants by name, runbook export/import; conversation context is stacked and scoped, not a shared brain
- **Repo + Confluence experts** — Index codebases and doc spaces; MCP export/import for sharing agent knowledge

Full guides: [docs/features/](docs/features/) on the marketing site.

## Screenshot

Main workspace — files, editor, multi-agent collaboration, and chat (same as the [marketing site](https://camronwood.github.io/neural-junkie/)).

![Neural Junkie desktop: files panel, code editor, multi-agent collaboration, and chat](assets/screenshots/Screenshot%202026-05-29%20at%202.31.27%20PM.png)

## Agents (after packs)

| On by default | How you get more |
|---------------|------------------|
| **Assistant** — reminders, tasks, notes, chat guidance | Enable **Software development** for BackendEngineer, FrontendEngineer, PlatformEngineer, SecurityReviewer, SoftwareArchitect, CodeReviewer |
| **CLI agents** (Cursor, Gemini, …) when on PATH | Enable **IDE** for editor/LSP depth; create repo/Confluence/expert agents with slash commands |

Type `/` or click **`/`** for the command palette. `/help` lists everything. Details: [docs/SOFTWARE_DEVELOPMENT_PACK.md](docs/SOFTWARE_DEVELOPMENT_PACK.md), [docs/CLI_AGENTS.md](docs/CLI_AGENTS.md), [docs/REPO_AGENTS.md](docs/REPO_AGENTS.md).

## Quick start (from source)

```bash
git clone https://github.com/camronwood/neural-junkie.git
cd neural-junkie

make gui-install   # first time: desktop deps
make start-all     # hub (in-process agents) + desktop app
```

`make start-all` does **not** spawn separate `cmd/agent` processes. Optional splits:

```bash
make server    # hub only
make gui       # desktop only
make chat      # terminal chat
make agents    # six specialists as OS processes (avoid duplicate names vs in-process config)
```

Configure providers in the **Setup Wizard** or **Settings → AI Providers**. For source builds without a bundled runtime:

```bash
make pull-models   # Ollama: qwen3.5:27b + qwen3.5:9b (or install from https://ollama.ai)
```

Cloud keys: Settings UI, or `cp env.example env.local` and set `ANTHROPIC_API_KEY`. Slash switches: `/switch-provider`, `/switch-all-providers`. Full setup: [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md).

## Project structure

```
neural-junkie/
├── cmd/                 # server, agent, chat, cli, nj-remote
├── desktop/             # Tauri + React app (src + src-tauri)
├── internal/            # hub, agents, AI providers, packs, repo, MCP, …
├── packs/               # official catalog (catalog.json)
├── scenarios/           # release / user-flow scenario fixtures
├── assets/              # screenshots, marketing images
├── docs/                # documentation + marketing site HTML
├── scripts/             # build and release
└── .github/workflows/   # CI / release
```

## Documentation

Full index: **[DOCS.md](DOCS.md)** (`make docs` prints the same file).

| Doc | What it covers |
|-----|----------------|
| **[Getting Started](docs/GETTING_STARTED.md)** | Source setup, configuration, make targets |
| **[Domain packs](docs/PACKS.md)** | Official packs and install flow |
| **[Composition Model](docs/COMPOSITION_MODEL.md)** | Portable agents, tools, runbooks |
| **[Context Model](docs/CONTEXT_MODEL.md)** | Scoped conversation context stack |
| **[Implementation sessions](docs/IMPLEMENTATION_SESSION.md)** | IDE Agent mode, verify/repair, NJ Fix Loop |
| **[IDE v4](docs/IDE_V4.md)** | Full LSP, remote SSH, dev containers |
| **[Collaboration](docs/COLLABORATION.md)** | Multi-agent planning, delegation, execution |
| **[Architecture](docs/ARCHITECTURE.md)** | System design, data flow, patterns |
| **[Changelog](docs/CHANGELOG.md)** | Version history |
| **[Known issues](docs/KNOWN_ISSUES.md)** | Beta limitations and workarounds |
| **[Contributing](CONTRIBUTING.md)** | How to contribute and run test gates |

## Make targets

```bash
make start-all        # Hub + desktop
make server / gui / chat / agents / stop / refresh
make gui-install      # Desktop dependencies
make build            # Go binaries
make test-go          # Go tests (-count=1)
make test-all         # vet + Go + desktop tsc + Vitest
make pull-models      # Ollama models
make bundle           # Current-platform distributable
make release VERSION=…  # Bump, commit, tag (then push for CI)
```

See the Makefile and [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) for the full list.

## Built for $0

Neural Junkie is an experiment: free for users **and** built, shipped, and marketed with free open-source tools only. See [docs/BUILT_FOR_ZERO.md](docs/BUILT_FOR_ZERO.md), [docs/THANKS.md](docs/THANKS.md), and [built-for-zero.html](https://camronwood.github.io/neural-junkie/built-for-zero.html). Install trust notes: [INSTALL_TRUST.md](docs/INSTALL_TRUST.md).

## License

MIT

## Contributing · Security · Issues

- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [Thank you — open source](docs/THANKS.md)
- [Known issues](docs/KNOWN_ISSUES.md)
- [Report a bug](https://github.com/camronwood/neural-junkie/issues/new?template=bug_report.yml)
