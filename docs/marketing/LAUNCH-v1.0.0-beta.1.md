# Launch copy — Neural Junkie v1.0.0-beta.1

**Canonical download link (use in every post):**

https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.11

**Site:** https://camronwood.github.io/neural-junkie/

**Quick start:** https://github.com/camronwood/neural-junkie/blob/main/docs/DOWNLOAD.md

**Image (beta.6):** `assets/neural-junkie-context-scope-ad-1080.png`

**Image (general download):** `assets/neural-junkie-beta-download-ad-1080.png`

---

## X / Mastodon (beta.11)

Neural Junkie v1.0.0-beta.11 — **Runbook builder**: plan multi-step work in a visual DAG (RB button or `/runbook`), assign agents, import markdown, start execution. Plus git worktree collabs, model library, HF, MCP. macOS / Windows / Linux installers.

Download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.11

---

## X / Mastodon (beta.9)

Neural Junkie v1.0.0-beta.9 — **`/collaborate --worktree`**: agents execute in a real git worktree on `nj/collab-…` (full repo, isolated branch). Bind with `--workspace` or your active workspace at Continue. Installers back after beta.8 CI fix.

Download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.9

---

## X / Mastodon (beta.8)

Neural Junkie v1.0.0-beta.8 — **app-store model library** in the toolbar (Ollama + Hugging Face): grid browse, one-tap install/download, detail for full actions. HF hosted inference + local GGUF → Ollama. MCP tool servers back.

Download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.9

---

## X / Mastodon (beta.7) — archived

Neural Junkie v1.0.0-beta.7 — chat finally renders **real markdown** (headings, lists, tables) like plans and file preview. Streaming scroll stays pinned; collab history survives session restore.

Download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

---

## X / Mastodon (beta.6) — archived

Neural Junkie v1.0.0-beta.6 — **Auto** workspace context: agents get the right amount of repo (none → full) per message, not your whole open project every time. Collabs stay clean unless you `/collaborate --workspace`. Session files won’t balloon; hub data needs consent first.

Download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

---

## X / Mastodon (beta.5) — archived

Neural Junkie v1.0.0-beta.5 — collab fixes (DM agents actually join the channel), cancel no longer locks the UI, Ollama DeepSeek **Reasoning** blocks, thread streaming in the right pane.

Download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.5

---

## X / Mastodon (general)

Neural Junkie v1.0.0-beta.5 is out — a downloadable desktop app (macOS / Windows / Linux), not just source.

Multi-agent Slack for devs: repo experts, slash commands, approve file edits. Local Ollama or your API keys.

Beta download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

OSS · feedback welcome

---

## LinkedIn

Open-sourcing wasn't enough — we shipped installers.

**Neural Junkie v1.0.0-beta.5** is a Tauri desktop app + Go hub: specialized agents (backend, frontend, security, DevOps…), repo-aware context, collaboration with human approval on file changes.

Install from GitHub Releases (macOS .dmg, Windows .msi, Linux AppImage). Setup wizard walks you through Ollama (macOS/Linux) or cloud providers. On Windows, install Ollama manually or use an API key.

Looking for early testers — especially "does install + first chat work on your machine?"

Download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

Site: https://camronwood.github.io/neural-junkie/

---

## Hacker News (Show HN)

**Title:** Show HN: Neural Junkie – multi-agent desktop for devs (beta download, local Ollama)

**URL:** https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

**First comment:**

Hi HN — Camron here. Neural Junkie is an open-source desktop app + Go hub for multi-agent dev workflows: specialists (@GoExpert, @SecurityExpert, …), repo agents, slash commands, file-change approvals, and bounded `/collaborate` sessions.

**v1.0.0-beta.1** is the first public download (macOS/Windows/Linux). The Go server ships as a Tauri sidecar — no separate Go install. Setup wizard helps wire Ollama (auto-install on macOS/Linux) or cloud APIs.

- Slack-style desktop: channels, palette, editor, threads
- Repo + Confluence agents, 50+ slash commands
- Local-first friendly (Ollama model library in Settings)

Quick start: https://github.com/camronwood/neural-junkie/blob/main/docs/DOWNLOAD.md

Build from source: `make start-all` in the repo.

Beta/unsigned — macOS may need right-click → Open. Would love reports on whether install + first @Moderator chat works on your OS. Issues: https://github.com/camronwood/neural-junkie/issues

---

## Reddit (r/LocalLLaMA — if allowed)

Title: [Beta] Neural Junkie — multi-agent desktop with Ollama model library + repo agents

Body: Shipped v1.0.0-beta.1 installers (macOS/Win/Linux). Tauri app bundles a Go hub; you can browse/install Ollama models from Settings, run specialists in one Slack-style UI, and index local repos for context. Local-first or mix cloud providers per agent.

Download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

Looking for feedback on install friction and local model workflows.

---

## Collaboration feature ad (agent orchestration)

**Image:** `assets/neural-junkie-collaboration-ad-1080.png` (regenerate: `./scripts/compose-collaboration-ad.sh`)

Graphic mock (faux collab channel + panel + agent mesh). For extra proof, reply with a real `collab-…` channel screenshot or 15–30s screen recording.

**X / LinkedIn (orchestration — primary):**

> One chat isn't a team.
>
> Neural Junkie `/collaborate`: your @RustExpert and @SecurityExpert **discuss with each other**, produce one **shared plan**, you **`/approve-plan`**, then they **execute assigned tasks** in a sandbox — with turn/message caps so it can't spiral.
>
> Not @mentions. Not copy-paste between chats.
>
> Open source beta: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

**X / LinkedIn (incident angle — optional follow-up):**

> Security review without playing telephone between five ChatGPT tabs.
>
> `/collaborate @SecurityExpert @DevOpsPro @GoExpert` on an outage or CVE — bounded agent discussion → plan you approve → parallel tasks.
>
> https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

**LinkedIn (longer):**

Hard refactors and incident response don't fit a single LLM thread. Neural Junkie's `/collaborate` is structured multi-agent orchestration: specialists debate in a dedicated `collab-…` channel (turn-limited), refine one plan artifact, and wait for your approval before execution. Tasks fan out by agent strength; file changes still go through your approve/reject flow.

Try the beta (macOS / Windows / Linux): https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.7

---

## Posting order

1. Tag pushed + CI green + assets on GitHub
2. Run `./scripts/smoke-release-install.sh`
3. X + LinkedIn with beta-download ad
4. HN Show HN next US morning
5. Follow-up in 3–5 days: `assets/neural-junkie-local-ai-tokens-ad-1080.png`

---

## Optional video (15–30s)

1. Launch installed app → setup wizard "Get started"
2. Command palette → `/help` or `/agents`
3. Agent reply in channel

Attach as reply on X or carousel slide 2 on LinkedIn.
