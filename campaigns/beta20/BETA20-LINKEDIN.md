# LinkedIn article — v1.2.0-beta.20 release (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `campaigns/beta20/creatives/neural-junkie-beta20-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-beta20-article.sh`

**Suggested title (pick one):**

- v1.2.0-beta.20: Install, Update, and Ship Artifacts
- What Ships in Neural Junkie v1.2.0-beta.20
- One-Click Ollama, Auto-Updates, and Neural Canvas

**Feed post teaser:**

> v1.2.0-beta.20 is the beta where the desktop gets out of your way: one-click Ollama on Windows, macOS, and Linux with real password/UAC dialogs, signed background auto-updates, Neural Canvas + Maps artifacts, semantic turn routing, and Share Agent packaging — everything that landed since beta.6, culminating in install-and-go local AI.

**Hashtags:** `#AI #LocalAI #MultiAgent #OpenSource #DeveloperTools #Ollama`

**Link:** [https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.20](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.20)

**Website:** [https://camronwood.github.io/neural-junkie/articles/beta-20.html](https://camronwood.github.io/neural-junkie/articles/beta-20.html)

**Download CTA:** [https://camronwood.github.io/neural-junkie/download.html](https://camronwood.github.io/neural-junkie/download.html)

**Suggested post date:** Same week as the `v1.2.0-beta.20` GitHub release tag

**Related articles:** [beta.6](../beta26/BETA26-LINKEDIN.md) · [Semantic turn routing](../semantic-turn-routing/SEMANTIC-TURN-ROUTING-LINKEDIN.md) · [Composition model](../composition-model/COMPOSITION-MODEL-LINKEDIN.md) · [Hardware](../hardware/HARDWARE-LINKEDIN.md)

---



## PASTE START

**Neural Junkie v1.2.0-beta.20** is the open-beta release where the desktop finally gets out of your way — from first install to day-two updates to shipping artifacts agents can actually revise.

If beta.6 gave the workstation a **memory of its own code**, the betas since then closed the other half of the loop: **install local AI without a terminal fight**, **update without hunting GitHub assets**, **route turns by meaning**, and **land Mermaid, maps, and canvas work** that stays grounded in your workspace.

Download: macOS, Windows, Linux — same local-first desktop app, same MIT-licensed hub. Prefer Homebrew? `brew tap camronwood/tap` then install the cask (macOS) or formula (Linux).

## The headline in one paragraph

You get **one-click Ollama install** that works on Windows, macOS, and Linux with real password/UAC dialogs (not a silent GUI failure), **signed Tauri v2 auto-updates** on macOS and Windows, **Neural Canvas** with trusted Mermaid/Markdown/code/table/chart renderers plus **Maps** (`nj.map`), **semantic turn routing** and durable SQLite orchestration, **Share Agent / tool-grant / runbook export**, a **setup wizard that no longer skips itself**, and the chat/IDE polish that makes packaged soaks survivable.

## 1. Install-and-go Ollama — password dialogs that actually appear

Beta users on Linux and Windows kept hitting the same wall: “Install Ollama” from the app looked like it did nothing, because a GUI process cannot own a sudo TTY.

Beta.19–20 fix that end-to-end:

- **Linux** — official installer via `pkexec` (Polkit password dialog)
- **macOS** — non-bundled installs use an administrator prompt (`osascript`)
- **Windows** — winget or silent `OllamaSetup.exe`, with UAC elevation when needed
- **Post-install** — waits for the binary (including `/usr/bin/ollama`) before reporting success
- **OLL chip + model library** — install without walking the wizard again

The setup wizard also stops skipping itself when default config seeds an Ollama provider — first launch finally means first launch.

## 2. Automatic updates that respect your work

Beta.10 migrated the desktop to **Tauri v2**. macOS and Windows check and download eligible updates in the background, then install on a **clean restart** — with draft protection for unsaved editor, runbook, pack, and chat state, plus active streams, collabs, and terminals.

Release policy controls land in the manifests: beta/stable channels, staged rollout, critical deadlines, minimum versions. Linux `.deb` stays manual while installed-package upgrade behavior is validated.

You should not need a release notes scavenger hunt to stay current.

## 3. Neural Canvas and Maps — artifacts agents can revise

Beta.17–18 ship **Neural Canvas**: app-managed, revisioned agent artifacts with trusted Markdown, Mermaid, code, table, chart, timeline, image, and graph renderers — chat cards, workspace tabs, provenance, history, and approved export.

Mermaid creates stay **workspace-grounded** (tree-first, no meta bleed, hard deny of `npx mermaid` on canvas turns). Zoom resizes SVG display size instead of CSS `scale()`, so diagrams stay sharp. **Maps** land as `nj.map` with markers and route polylines.

Packs can declare declarative artifact renderers without executing pack UI code. Canvas asks no longer fall through into FILE_CHANGE / implement sessions.

## 4. Meaning over phrases — semantic routing and durable collab

Beta.11 cut over **semantic turn routing**: one server-authoritative decision per turn — local structured classification for meaning, deterministic policy for writes, recipients, retrieval, and Ask/Plan safety. Deep dive: [semantic turn routing article](https://camronwood.github.io/neural-junkie/articles/semantic-turn-routing.html).

Durable **SQLite orchestration** makes collab claims and HITL gates restart-safe. Agent questions coalesce, pause peers, and continue the original turn after one answer instead of looping the same prompt.

## 5. Composition you can take with you

Beta.18 packages the composition story: **Share Agent** bundles, MCP tool grants scoped to one agent by name, and runbook definition export/import with provenance. See the [Composition Model article](https://camronwood.github.io/neural-junkie/articles/composition-model.html).

Plus the reliability path in between: hub login hotfix for large SQLite archives (beta.12), Knowledge Graph / explorer freeze caps (beta.16), Assistant meeting-notes chat routing, and Gemini CLI cwd safety.

## What we are still honest about

Open beta means we ship progress and name the gaps. Linux auto-update for `.deb` is still manual. Multi-turn user-flow journeys exist under `scenarios/user-flows/` but stay opted out of the default suite until they stabilize. Known issues stays the source of truth for what is still wrong on purpose.

## Get it

- Releases: [v1.2.0-beta.20](https://github.com/camronwood/neural-junkie/releases/tag/v1.2.0-beta.20)
- Download page: [camronwood.github.io/neural-junkie/download.html](https://camronwood.github.io/neural-junkie/download.html)
- Homebrew: `brew tap camronwood/tap` then install cask or formula
- Changelog: [docs/CHANGELOG.md](https://github.com/camronwood/neural-junkie/blob/main/docs/CHANGELOG.md)
- Prior release article: [beta.6](https://camronwood.github.io/neural-junkie/articles/beta-6.html)

Star the repo if this is useful. File issues if something breaks. Beta.20 is for people who want a local multi-agent workstation that **installs cleanly**, **updates itself**, and **ships artifacts you can keep revising** — not just another chat window with a model picker.

## PASTE END
