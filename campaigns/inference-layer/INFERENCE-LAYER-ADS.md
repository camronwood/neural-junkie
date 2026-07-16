# Inference layer ads

**Canonical download:** https://github.com/camronwood/neural-junkie/releases/latest

**Website article:** https://camronwood.github.io/neural-junkie/articles/inference-layer.html

**Regenerate graphics:**

```bash
chmod +x ./scripts/compose-inference-layer-ads.sh
./scripts/compose-inference-layer-ads.sh all
chmod +x ./scripts/compose-inference-layer-article.sh
./scripts/compose-inference-layer-article.sh
```

| Asset | Variant | Angle |
|-------|---------|-------|
| `campaigns/inference-layer/creatives/neural-junkie-inference-skip-ad-1080.png` | `skip` | Closure intent → zero tokens |
| `campaigns/inference-layer/creatives/neural-junkie-inference-gate-ad-1080.png` | `gate` | Context stack → router → model pipeline |
| `campaigns/inference-layer/creatives/neural-junkie-inference-trust-ad-1080.png` | `trust` | Routing badge on every reply |
| `campaigns/inference-layer/creatives/neural-junkie-inference-layer-1200.png` | article | LinkedIn / website cover |

Related: [INFERENCE-LAYER-LINKEDIN.md](INFERENCE-LAYER-LINKEDIN.md) · [MODEL-LAYERING-LINKEDIN.md](../model-layering/MODEL-LAYERING-LINKEDIN.md) · [MODULAR-AI-COMPOSITION-LINKEDIN.md](../modular-ai/MODULAR-AI-COMPOSITION-LINKEDIN.md)

---

## Ad 1 — Inference avoidance

**Headline on image:** "Thanks!" = zero tokens.

**X / LinkedIn (short):**

> Everyone optimizes inference speed. Neural Junkie optimizes inference *avoidance*. Closure intent → canned reply. No 14B wake-up. No workspace scan. No tool dump on "hey."
>
> https://github.com/camronwood/neural-junkie/releases/latest

**LinkedIn (longer):**

> Before any LLM call, every turn passes through a context stack: mode, intent, memory, grounding, persona, budget. "Thanks!" classifies as closure — zero tokens spent. Casual DMs get minimal prompts. Substantive tasks get the full specialist. The fastest inference is the one you don't run.
>
> Open source (macOS / Windows / Linux): https://github.com/camronwood/neural-junkie/releases/latest

---

## Ad 2 — Decision pipeline

**Headline on image:** Decide before you generate.

**X / LinkedIn (short):**

> Neural Junkie's inference layer answers three questions before tokens get spent: Should we infer? Which model? Which provider and tool loop? Context stack → unified router → chat or tool model → routing badge on the reply.
>
> https://github.com/camronwood/neural-junkie/releases/latest

**LinkedIn (longer):**

> One 14B can't be your security reviewer, biology expert, session summarizer, and typo-fixer. The inference layer classifies each job — collab tasks, delegation consults, implementation sessions — through a 9B router with rules fallback. Security → `nj-security:14b`. Biology tools → Qwen tool loop. Typo fix → cheap tier.
>
> Debug: `GET /api/debug/routing-classify?q=...` when `NEURAL_JUNKIE_DEBUG=1`

---

## Ad 3 — Routing trust

**Headline on image:** Which model ran? Now you can see it.

**X / LinkedIn (short):**

> Every agent reply can show a routing badge: chat model, tool model, reason, and whether the classifier was LLM or rules. Modular AI only feels trustworthy when inference decisions are legible — not hidden behind an agent name.
>
> https://github.com/camronwood/neural-junkie/releases/latest

---

## Article launch — feed post (primary)

**Image:** `campaigns/inference-layer/creatives/neural-junkie-inference-layer-1200.png`

**Post copy:**

> Everyone optimizes inference speed. Almost nobody optimizes inference *avoidance*.
>
> New article: how Neural Junkie's inference layer decides whether to call a model, which brain to use, and which provider runs the job — then shows you the answer on the message itself.
>
> "Thanks!" → zero tokens.
> Security collab task → `nj-security:14b`.
> Biology tools → OpenBio chat + Qwen tool loop.
> Every reply → routing badge.
>
> Read on the site: https://camronwood.github.io/neural-junkie/articles/inference-layer.html
> Download: https://github.com/camronwood/neural-junkie/releases/latest
>
> #AI #LocalAI #MultiAgent #DeveloperTools #OpenSource #Ollama #ModularAI

---

## Article launch — comment / second post (technical)

> Three inference paths in one BiologyExpert DM:
> 1. "Thanks!" → closure intent → canned reply, zero LLM
> 2. "Explain scan summary vs analysis" → OpenBio 8B, no tools
> 3. "Analyze and fold this peptide" → OpenBio chat + qwen3.5:9b tool loop
>
> Same specialist. Different inference decisions per turn. Badge shows which model actually ran.

---

## Article cross-post

Use [INFERENCE-LAYER-LINKEDIN.md](INFERENCE-LAYER-LINKEDIN.md) as the long-form LinkedIn article. Cover: `campaigns/inference-layer/creatives/neural-junkie-inference-layer-1200.png`.

**Feed teaser:**

> Everyone optimizes inference speed. Almost nobody optimizes inference *avoidance*. Neural Junkie's inference layer decides whether to call a model, which brain to use, and which provider runs the job — then shows you the answer on the message itself.
