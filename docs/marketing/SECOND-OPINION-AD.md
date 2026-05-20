# Second opinion — non-developer ad

**Audience:** People who already use more than one AI (ChatGPT, Claude, Gemini, etc.) and sometimes paste the same question into a second chat to sanity-check the first answer — **not** developers looking for code review.

**Canonical download:** https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.12

**Regenerate graphic:**

```bash
./scripts/compose-second-opinion-ad.sh
```

**Asset:** `assets/neural-junkie-nondev-second-opinion-ad-1080.png`

**Product feature:** [Agent Review](../AGENT_REVIEW.md) — reply in a thread and @mention a different expert. No codebase, repo agents, or `/collaborate` required for the basic story.

**Honest constraint (replies, not on the image):** A second model reduces mistakes; it does not guarantee truth. NJ needs API keys and/or local Ollama — not consumer ChatGPT Plus / Claude Pro web logins.

---

## Headline on image

**TWO AIs · ONE ROOM**

Sub: Ask one expert. Have another check the answer — no new browser tab.

---

## X / LinkedIn (short)

> I don't trust one AI tab for big decisions.
>
> Neural Junkie: ask your **trip planner**, then **@mention your budget buddy** in the same thread for a second opinion — no copy-paste into ChatGPT #2.
>
> Writing, travel, money, research — custom experts in one desktop.
>
> Open source beta: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.12

---

## LinkedIn (longer)

> A friend told me they run important questions through **two LLMs** — not because they're technical, but because one confident answer isn't enough.
>
> If that's you, the painful part isn't the idea. It's **opening another browser tab**, re-pasting context, and hoping both chats remember the same facts.
>
> Neural Junkie keeps **multiple experts in one Slack-style app**: a writing coach, trip planner, budget buddy — whatever you create. Ask one. **Reply in the thread** and @mention another for a quick second look before you book, send, or spend.
>
> Bigger decisions? `/collaborate` adds a structured team discussion and your approval before anything runs — still no codebase required.
>
> Beta (macOS / Windows / Linux): https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.12

---

## Demo video (ideal 15–20s)

1. New DM with a custom expert (e.g. trip planner) — plain language question.
2. Open thread on the answer → @mention a second expert ("does this budget work?").
3. Show both replies in one pane — **no terminal, no file tree**.

Pair with `docs/media/general-experts-guitar.mp4` if you don't have a dedicated clip yet.

---

## Reply templates

| Objection | Response |
|-----------|----------|
| "I already use ChatGPT + Claude in two tabs" | Same habit, one app — shared thread, less re-explaining. Per-expert routing if you use different APIs. |
| "How is this different from Collaborate?" | **Agent review** = one answer, one checker, 30 seconds. **Collaborate** = several experts debate a plan you approve — launch, vendor pick, annual budget. |
| "Will the AIs argue forever?" | Review is one level deep, you @mention who checks whom. Collab has turn limits. |
| "Is this for coding?" | This ad is for **everyday decisions**; devs use the same pattern with @SecurityExpert on @GoExpert. |

---

## Posting order

After **nondev-experts** (broad hook). Before or alongside **nondev-collaborate** — second opinion is the lighter entry; collaborate is the upsell for multi-step planning.
