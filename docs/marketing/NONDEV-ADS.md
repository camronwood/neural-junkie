# Non-developer audience — ad concepts & copy

**Canonical download:** https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.9

**Regenerate graphics:**

```bash
./scripts/compose-nondev-ads.sh all
```

| Asset | Layout style | Angle |
|-------|----------------|--------|
| `assets/neural-junkie-nondev-providers-ad-1080.png` | Hero typography + tab chaos + icon row (like `local-ai-tokens`) | Multiple AI backends, one desktop |
| `assets/neural-junkie-nondev-experts-ad-1080.png` | 2×2 persona grid + Assistant strip (like `context-scope`) | Custom experts / DMs |
| `assets/neural-junkie-nondev-collaborate-ad-1080.png` | Split column + hex cluster + quote card (like `oss-contributors` / timeline) | Multi-agent planning |

**Honest constraint (use in replies, not on the image):** Neural Junkie uses **API keys and/or local models** (Ollama, LM Studio, etc.), not consumer ChatGPT Plus / Claude Pro logins. Position as “unify your AI stack,” not “import your subscriptions.”

---

## Ad 1 — Providers (`nondev-providers`)

**Headline on image:** Five AI tabs. One desktop.

**X / LinkedIn:**

> Paying for more than one AI?
>
> Neural Junkie is one desktop for **Claude + local Ollama + any OpenAI-compatible API** — different agents, different models, one Slack-style workspace. Draft on local, polish in the cloud, track usage in Settings.
>
> Open source beta (macOS / Windows / Linux): https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.9

**LinkedIn (longer):**

> If you already juggle Claude, ChatGPT, Gemini, and a local model, you’re not alone — you’re just tired of **context living in five browser tabs**.
>
> Neural Junkie routes **per agent**: writing coach on Claude, brainstorming on Ollama, research on your other API. Channels, DMs, threads, plus Assistant for tasks and reminders.
>
> Beta download: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.9

---

## Ad 2 — Custom experts (`nondev-experts`)

**Headline on image:** Not a dev? Still a power user.

**X / LinkedIn:**

> You don’t need `@RustExpert` to use Neural Junkie.
>
> Create a **writing coach**, **trip planner**, or **budget buddy** in a private DM — your API keys or local Ollama, plus Assistant for tasks, notes, and `/summarize`.
>
> https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.9

**LinkedIn (longer):**

> The same app developers use for repo agents also works as a **personal AI control room**: spin up custom experts (`/create-expert` or New DM), keep reminders and notes in one place, and stop re-explaining yourself in every new chat tab.
>
> Site: https://camronwood.github.io/neural-junkie/

---

## Ad 3 — Collaborate (`nondev-collaborate`)

**Headline on image:** Big decisions need a team.

**X / LinkedIn:**

> Launch plan? Trip budget? Investor deck?
>
> `/collaborate` — specialists **talk to each other**, produce **one plan**, you **approve**, then they execute (with limits so it can’t spiral). Not copy-paste between five chats.
>
> https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.9

**LinkedIn (incident / business angle):**

> Product launch, vendor review, or annual planning — one chatbot isn’t enough perspectives.
>
> Neural Junkie: bounded agent discussion → shared plan → your approval → parallel tasks. No codebase required for planning-only workflows.
>
> Beta: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.9

---

## Posting order (optional test)

1. **Experts** — broadest hook, lowest technical bar; pair with site video `docs/media/general-experts-guitar.mp4` if desired.
2. **Providers** — targets multi-AI spenders; expect “how is this different from Poe?” in replies — answer: local-first, per-agent routing, open source, approval gates.
3. **Collaborate** — strongest differentiation; save for when you have a non-dev screen recording (launch planning channel, no terminal visible).

---

## Reply templates

| Objection | Response |
|-----------|----------|
| “I only have ChatGPT Plus” | NJ needs an API key or local Ollama — consumer web logins aren’t wired today. Ollama is free locally; cloud keys are pay-as-you-go. |
| “Too technical” | Start with New DM + one provider in the setup wizard; skip repo/worktree features. Assistant + custom expert is enough for day one. |
| “Why not just use Slack + bots?” | Specialists share one plan and approval flow; collab turn limits; optional local models to control cost. |
