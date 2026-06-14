# Modular AI composition ads

**Canonical download:** https://github.com/camronwood/neural-junkie/releases/latest

**Regenerate graphics:**

```bash
chmod +x ./scripts/compose-modular-ai-ads.sh
./scripts/compose-modular-ai-ads.sh all
chmod +x ./scripts/compose-modular-ai-composition-article.sh
./scripts/compose-modular-ai-composition-article.sh
```

| Asset | Variant | Angle |
|-------|---------|-------|
| `assets/neural-junkie-modular-router-ad-1080.png` | `router` | Small 9B classifier routes domain + cost |
| `assets/neural-junkie-routing-badge-ad-1080.png` | `observe` | See which model ran on every reply |
| `assets/neural-junkie-compose-specialist-ad-1080.png` | `compose` | Packs declare chat + tool + LoRA |
| `assets/neural-junkie-hardware-stacks-ad-1080.png` | `stacks` | Recommended inference + LoRA per RAM tier |
| `assets/neural-junkie-modular-ai-composition-1200.png` | article | LinkedIn article cover |

Related: [MODULAR-AI-COMPOSITION-LINKEDIN.md](MODULAR-AI-COMPOSITION-LINKEDIN.md) · [MODEL-LAYERING-LINKEDIN.md](MODEL-LAYERING-LINKEDIN.md)

---

## Ad 1 — Small classifier router

**Headline on image:** A 9B router. Not a 70B guess.

**X / LinkedIn (short):**

> Neural Junkie routes collab tasks and delegation consults through a **unified small-model classifier** (`qwen3.5:9b` JSON) with **rules fallback**. Security tasks → `nj-security:14b`. Typo fixes → cheap local tier. No more keyword drift across four separate code paths.
>
> https://github.com/camronwood/neural-junkie/releases/latest

**LinkedIn (longer):**

> Modular AI at the orchestration layer: a 9B utility model classifies domain, tool need, and cost tier before the hub picks a provider or LoRA tag. LLM default, rules fallback on failure, 80+ table-tested cases. Debug: `GET /api/debug/routing-classify?q=...` when `NEURAL_JUNKIE_DEBUG=1`.
>
> Open source (macOS / Windows / Linux): https://github.com/camronwood/neural-junkie/releases/latest

---

## Ad 2 — Routing observability

**Headline on image:** Which model ran? Now you can see it.

**X / LinkedIn (short):**

> Every agent reply can show a **routing badge**: chat model, tool model, reason, and whether the classifier was LLM or rules. Toggle in Settings → Layout. Modular AI only feels trustworthy when routing is legible.
>
> https://github.com/camronwood/neural-junkie/releases/latest

---

## Ad 3 — Composed specialists

**Headline on image:** Chat brain. Tool hands. LoRA weights.

**X / LinkedIn (short):**

> BiologyExpert's split-brain pattern is now a **pack compose template**: `chat_model` + `tool_model` + `lora_tag` + `consult_triggers` in `pack.yaml`. Life sciences and software development packs ship it today. One code path for DM, collab, and implementation.
>
> https://github.com/camronwood/neural-junkie/releases/latest

---

## Ad 4 — Hardware stacks

**Headline on image:** Pick a stack for your RAM tier.

**X / LinkedIn (short):**

> **Two-tier model strategy, documented:** Qwen for inference, Llama/Mistral for LoRA compose. `GET /api/system/hardware` returns `recommended_stacks[]` — minimal (~10 GB) through heavy (~50+ GB). No more guessing which `nj-*` tags you need.
>
> https://github.com/camronwood/neural-junkie/releases/latest

---

## Article cross-post

Use [MODULAR-AI-COMPOSITION-LINKEDIN.md](MODULAR-AI-COMPOSITION-LINKEDIN.md) as the long-form LinkedIn article. Cover: `assets/neural-junkie-modular-ai-composition-1200.png`.

**Feed teaser:**

> Stop running one model for everything. Neural Junkie composes specialists at the orchestration layer — small classifier, pack-declared stacks, routing badges on every reply.
