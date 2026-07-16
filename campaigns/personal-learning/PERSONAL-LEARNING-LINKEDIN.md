# LinkedIn article — Personal learning (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `campaigns/personal-learning/creatives/neural-junkie-personal-learning-1200.png` (1200×627)

**Feed ad image:** `campaigns/personal-learning/creatives/neural-junkie-personal-learning-ad-1080.png` (1080×1080)

**Regenerate images:** `./scripts/compose-personal-learning-article.sh`

**Suggested title (pick one):**
- Nothing Gets Remembered Until You Say So: Personal Learning in Neural Junkie
- Conversation Memory Recalls the Past. Personal Learning Shapes the Future.
- How We Built Opt-In Expert Memory (Without Training on Your Chat)

**Feed post teaser:**
> Your agents shouldn't silently learn from every typo and half-formed thought. Neural Junkie personal learning stores only what you confirm — scoped per expert, globally, or per collaboration — retrieved by local embeddings and optionally exported into LoRA training rows.

**Hashtags:** `#AI #LocalAI #DeveloperTools #OpenSource #FineTuning #MultiAgent`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

**Website:** https://camronwood.github.io/neural-junkie/articles/personal-learning.html

**Related articles:** [Conversation memory](../conversation-memory/CONVERSATION-MEMORY-LINKEDIN.md) · [LoRA](../lora/LORA-LINKEDIN.md) · [LoRA v2](../lora-v2/LORA-V2-LINKEDIN.md) · [Two-tier LoRA](../two-tier-lora/TWO-TIER-LORA-LINKEDIN.md)

---

## Ad / feed copy variants

**Variant A — trust hook:**
> "Remember that we use tabs, not spaces." Most agents either forget that by tomorrow or infer it from noise. Neural Junkie personal learning: you approve every note, choose the scope, and experts pull it back when relevant — local embeddings, no cloud memory service.

**Variant B — collab hook:**
> `/collaborate` teams agree on conventions once. Personal learning scoped to that collaboration injects those conventions into every specialist in the channel — without replaying the whole thread.

**Variant C — LoRA bridge:**
> Confirmed learnings aren't just prompt text. Export them as Alpaca rows and fold them into LoRA training alongside chat history — same Specialist tuning pack, same local Ollama stack.

---

## PASTE START

You tell your backend expert: *"We always use `context.Context` as the first parameter."*

Two weeks later it writes a function with `ctx` buried in the middle.

That isn't always a model failure. Often the preference was never **stored as a durable fact** — it lived in chat, got summarized away, or drowned in tail history.

**Neural Junkie** already has [conversation memory](https://camronwood.github.io/neural-junkie/articles/conversation-memory.html): retrieve relevant *past messages* when you ask a follow-up. Personal learning is the other half of the memory story: **user-confirmed facts** that shape how experts behave going forward.

Nothing is saved without your explicit approval.

## Memory that recalls vs memory that teaches

| | Conversation memory | Personal learning |
|---|---|---|
| **What it stores** | Indexed chat + collab artifacts | Short user-confirmed notes |
| **Default** | On when embed model available | Off until you opt in |
| **Who writes it** | Hub indexes on message write | You approve every entry |
| **Retrieval query** | Your latest question | Your latest question |
| **Typical use** | "What did we decide about auth?" | "Always use pnpm, not npm" |

Both use local Ollama embeddings (`nomic-embed-text`) and stay on your machine. Neither sends your history to a cloud vector DB.

Conversation memory answers **what happened**. Personal learning encodes **how you want work done**.

## Opt-in by design

Personal learning has three gates:

1. **Specialist tuning** pack installed (capability `personal-learning` — no Python/CUDA required)
2. **Settings → AI & providers → Enable personal learning for experts** (default off)
3. **Your confirmation** on every save — modal, `/learn`, or Settings

Natural phrases like "remember that" or "I prefer" may open a **proposal** dialog. They do not auto-persist. Agent-suggested learnings are a separate toggle, rate-limited, and still need your click.

That is intentional. Silent training on chat is convenient until it memorizes a joke, a typo, or something you said once under deadline pressure.

## Three scopes

When you save a learning, you choose where it applies:

**This expert** — injected only when that agent responds (e.g. SecurityReviewer always flags `eval()` in review comments).

**All experts** — global preferences every specialist sees (e.g. "Our API errors use `{code, message}` JSON").

**This collaboration** — scoped to an active `/collaborate` channel so a multi-agent session shares conventions without polluting unrelated DMs.

At prompt time the hub retrieves top matches per section and appends:

```
=== LEARNINGS FOR ALL EXPERTS (user-confirmed) ===
=== LEARNINGS FOR THIS EXPERT (user-confirmed) ===
=== LEARNINGS FOR THIS COLLABORATION (user-confirmed) ===
```

Budget caps keep injection small (~2 KB agent, smaller global/collab slices). If Ollama embed is unreachable, keyword overlap fallback kicks in within ~200ms.

Storage: `~/.neural-junkie/learnings.json` plus a sidecar embedding index. Export/import bundles for portability. Edit or forget any entry in Settings.

## How to use it

**Quick capture:** `/learn we use conventional commits` in a DM or channel — opens the approval dialog with a draft.

**From agent info:** **Add learning** on any expert's profile card.

**Bulk management:** Settings → **Saved learnings** — grouped by global, expert, and collaboration.

**Categories:** preference, fact, workflow, communication — for your own organization, not model routing.

Categories do not change which agent runs. Scopes do.

## Bridge to LoRA

Prompt injection is fast and reversible. Sometimes you want the preference **baked into weights**.

Confirmed learnings can feed LoRA training:

- LoRA train panel → **Include confirmed personal learnings**
- `GET /api/lora/train/preview?include_learnings=1` shows the row count
- Training prepends Alpaca-format rows from approved notes (capped)

Same Specialist tuning pack, same local workflow as training from chat or collab history — but only rows you explicitly confirmed. See our [LoRA article](https://camronwood.github.io/neural-junkie/articles/lora.html) for compose and train details.

## What we deliberately exclude

Personal learnings are **not** injected into CLI agents or the Moderator — those paths stay lightweight and policy-driven.

They complement conversation memory and session summaries; they do not replace repo indexes or MCP tool context.

If you disable the feature, existing entries remain on disk but stop injecting until you turn it back on.

## Try it

```bash
make pull-models   # includes nomic-embed-text
make start-all
```

1. Install **Specialist tuning** from Settings → Domain packs
2. Enable personal learning under Settings → AI & providers
3. DM BackendEngineer: `/learn prefer table-driven tests in internal/`
4. Approve in the modal → ask a new task → preference should surface in retrieved learnings

Debug: `GET /api/learnings/query?q=...` previews retrieval without sending a chat message.

Download: https://github.com/camronwood/neural-junkie/releases/latest

If a scope feels wrong (global note leaking into the wrong collab, embed fallback too noisy) — GitHub issues welcome.

Camron Wood — Neural Junkie (personal project)

## PASTE END
