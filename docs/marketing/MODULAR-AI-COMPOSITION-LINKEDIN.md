# LinkedIn article — Modular AI composition (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `assets/neural-junkie-modular-ai-composition-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-modular-ai-composition-article.sh`

**Suggested title (pick one):**
- Modular AI, Local Hardware: How Neural Junkie Routes Instead of Guessing
- One Hub, Many Brains: Composition Over a Single Giant Model
- See Which Model Ran: Trustworthy Routing for Local Multi-Agent AI

**Feed post teaser:**
> Stop running one model for everything. Neural Junkie composes specialists at the orchestration layer — a small classifier picks domain and cost tier, packs declare chat/tool/LoRA stacks, and every reply shows which model actually ran.

**Hashtags:** `#AI #LocalAI #MultiAgent #ModularAI #LoRA #DeveloperTools #OpenSource #Ollama`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

---

## PASTE START

The industry is converging on **modular AI**: router + small specialists, LoRA adapters, mixture-of-experts inside foundation models. The goal is the same — don't run a 70B jack-of-all-trades for every token.

**Neural Junkie** takes that idea one layer up: **orchestration-layer composition**. You still talk to one specialist. The hub decides which model runs, which tools fire, and which LoRA tag applies — then shows you the answer on the message itself.

This builds on our earlier [model layering article](https://github.com/camronwood/neural-junkie/blob/main/docs/marketing/MODEL-LAYERING-LINKEDIN.md). That piece covered context, weights, routing, and orchestration. This one ships the next layer: **unified routing**, **pack-declared compose stacks**, and **observability you can trust**.

## The problem with one model

One 14B (or 27B) model cannot be your security reviewer, biology expert, session summarizer, typo-fixer, and tool-runner — not on a 16 GB laptop, and not without quality tradeoffs.

Keyword heuristics help (`looksSecurity`, `looksCheap`) but they drift, miss edge cases, and never tell the user what actually ran. Modular AI only feels real when routing is **legible**.

## Pillar 1: A small classifier router (not a 70B gatekeeper)

Neural Junkie now classifies each collab task and delegation consult through a **unified router** (`internal/routing`):

- **LLM default:** `qwen3.5:9b` returns structured JSON — `domain`, `tool_need`, `cost_tier`, `confidence`, `reason`
- **Rules fallback:** consolidated keyword lists when the LLM fails, times out, or confidence drops below 0.6
- **Table-tested:** 80+ deterministic cases mirror the chat quality router harness

A security collab task routes to `nj-security:14b` when installed. A typo-fix routes to the cheap local tier. Biology tool questions trigger `tool_need` without you @mentioning BiologyExpert.

Debug when `NEURAL_JUNKIE_DEBUG=1`:

```bash
curl 'http://localhost:18765/api/debug/routing-classify?q=review+JWT+auth+flow&agent_type=security'
```

## Pillar 2: Composed specialists in every pack

BiologyExpert was always split-brain: **OpenBio for reasoning**, **Qwen 9B for MCP tools**. That pattern is now a first-class **pack compose template**:

```yaml
compose:
  chat_model: koesn/llama3-openbiollm-8b:latest
  tool_model: qwen3.5:9b
  lora_tag: nj-biology:8b
  consult_triggers: [biology, protein, sequence]
```

Life sciences and software development packs ship compose blocks today. Security gets `nj-security:14b` + tool model. The hub resolves chat/tool/LoRA through one code path for DM, collab, and implementation sessions — no more biology-only `if` branches.

## Pillar 3: See which model ran

Every agent response can carry routing metadata:

- `routing_model` — chat model tag
- `routing_tool_model` — tool loop when fallback fired
- `routing_reason` — e.g. `security_lora_local`, `cheap_local`
- `routing_source` — `llm` or `rules`

The desktop shows a compact badge on each reply (`nj-security:14b · llm`). Tooltip shows the full chain. Toggle in **Settings → Layout → Routing badges on messages**.

Modular AI without observability is just vibes. This makes it auditable.

## Pillar 4: Recommended stacks per RAM tier

Two-tier strategy stays: **Qwen for inference**, **Llama/Mistral bases for LoRA compose**. But the docs and API now spell out full stacks:

| Tier | Inference | LoRA (optional) | Disk |
|------|-----------|-----------------|------|
| minimal | `qwen3.5:9b` + cloud | none | ~10 GB |
| light | `qwen3.5:9b` | 1 base + 1 tag | ~15 GB |
| recommended | `qwen3.5:27b` + `qwen3.5:9b` | 3 bases + 4 bootstrap tags | ~35 GB |
| heavy | above + OpenBio | full specialist-tuning | ~50+ GB |

`GET /api/system/hardware` returns `recommended_stacks[]` — same data the wizard and Pack Store can render.

## Walkthrough: verify it yourself

**1. Security collab task**

Run `/collaborate` with a security review task. After approval, the task message and agent reply should show `nj-security:14b` when the tag is installed.

**2. Biology DM**

DM BiologyExpert: "What's the difference between scan summary and scan analysis?" → OpenBio chat, no tool loop.

Then: "Analyze this peptide: MKTAYIAKQRQISFVKSHFSRQ" → Qwen tool loop, PDB path in reply. Badge shows both models.

**3. Typo task**

Ask BackendEngineer to "fix typo in README" in collab execution. Router picks `cost_tier: cheap` → local utility tier.

## What we learned

**1. Orchestration-layer MoE beats model-layer MoE for local-first products.**

You don't need to train a gating network inside a foundation model. Route at the hub with specialists you control.

**2. Observability is a feature, not debug logging.**

`[collab-routing]` logs are not enough. Users need badges.

**3. Packs should declare composition, not hardcode biology.**

`compose:` in `pack.yaml` is the contract. LoRA adapters and chat/tool splits are pack concerns.

**4. Test the router in code.**

Models change weekly. Deterministic routing tests + live collab scenarios catch regressions without guessing from the UI.

## Try it

Personal open-source project — macOS, Windows, Linux.

```bash
make pull-models
make start-all
```

Download: https://github.com/camronwood/neural-junkie/releases/latest

**Docs:**
- [Context model](https://github.com/camronwood/neural-junkie/blob/main/docs/CONTEXT_MODEL.md)
- [Hardware tiers + stacks](https://github.com/camronwood/neural-junkie/blob/main/docs/HARDWARE.md)
- [LoRA adapters](https://github.com/camronwood/neural-junkie/blob/main/docs/LORA_ADAPTERS.md)
- [Delegation](https://github.com/camronwood/neural-junkie/blob/main/docs/DELEGATION.md)

If routing sends a security task to the wrong tier — or a badge shows the wrong model — GitHub issues welcome. That feedback becomes the next scenario in the harness.

Camron Wood — Neural Junkie (personal project)

## PASTE END
