# LinkedIn article — Inference layer (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `campaigns/inference-layer/creatives/neural-junkie-inference-layer-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-inference-layer-article.sh`

**Suggested title (pick one):**
- Decide Before You Generate: Neural Junkie's Inference Layer
- Not Every Message Deserves a 14B Model
- The Orchestration Layer Between Your Message and the Model

**Feed post teaser:**
> Everyone optimizes inference speed. Almost nobody optimizes inference *avoidance*. Neural Junkie's inference layer decides whether to call a model, which brain to use, and which provider runs the job — then shows you the answer on the message itself.

**Hashtags:** `#AI #LocalAI #MultiAgent #DeveloperTools #OpenSource #Ollama #ModularAI`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

**Website:** https://camronwood.github.io/neural-junkie/articles/inference-layer.html

**Related articles:** [Model layering](../model-layering/MODEL-LAYERING-LINKEDIN.md) · [Modular AI composition](../modular-ai/MODULAR-AI-COMPOSITION-LINKEDIN.md)

---

## PASTE START

Most AI products optimize **how fast** inference runs. Faster GPUs. Better kernels. Cheaper tokens.

That's necessary. It's not sufficient.

If every user message wakes a 14B model — including "thanks," "hey," and "what model are you?" — you burn RAM, add latency, and get worse answers than a one-line canned reply would have given you.

**Neural Junkie** is my personal open-source desktop hub for running AI specialists on your hardware. The product bet isn't one perfect model. It's an **inference layer** — a decision pipeline that sits between your message and the GPU and answers three questions before tokens get spent:

1. **Should we infer at all?**
2. **Which model tag when we do?**
3. **Which provider and tool loop runs the job?**

This builds on our earlier articles on [model layering](https://github.com/camronwood/neural-junkie/blob/main/campaigns/model-layering/MODEL-LAYERING-LINKEDIN.md) and [modular composition](https://github.com/camronwood/neural-junkie/blob/main/campaigns/modular-ai/MODULAR-AI-COMPOSITION-LINKEDIN.md). Those explain *what stacks together*. This one explains *when and why tokens get spent* — and how you can verify it.

## The inference layer in one diagram

Every user turn flows through a gate before you see a reply:

```
User message
    → Context stack (mode · intent · memory · budget)
    → Skip inference?  → canned reply (zero tokens)
    → Unified router (domain · cost · tool_need)
    → Pick model + provider
    → Tools needed?  → chat model OR tool model loop
    → Routing badge on reply
```

Same hub. Same UI. Different inference decisions at each stage.

## Question 1: Should we infer at all?

Before any LLM call, every turn passes through the **Conversation Context Stack**:

1. **Mode** — chat, code, or collaboration
2. **Intent** — closure, casual, substantive, or task
3. **Memory** — rolling session summary + capped history
4. **Grounding** — workspace scan only when you're actually coding
5. **Persona** — DM vs channel vs collab framing
6. **Budget** — byte caps so prompts don't blow context windows

This layer often **avoids a model call entirely**.

| Intent | What happens | LLM? |
|--------|--------------|------|
| `closure` | "Thanks!" → brief canned ack | **No** |
| `casual` | Minimal prompt, 2 history rows | Small model only |
| `substantive` | Full specialist prompt | Yes |
| `task` | Full prompt + workspace scan | Yes |

**The insight:** routing *context* is cheaper and more reliable than asking a big model to "behave appropriately" every turn.

Session summaries run asynchronously on `qwen3.5:9b` every few user turns — keeping long DM threads coherent without re-sending full transcripts to the 14B specialist every time.

Delegation and collab consults are **skipped for closure and casual intents** — greetings shouldn't trigger a biology consult.

## Question 2: Which model when we do infer?

Once a turn deserves real inference, the hub picks the **weight layer** — but not with one static default per agent.

### Per-agent defaults

Software specialists default to `qwen3.5:9b` (or `qwen3.5:27b` on 16 GB+ RAM). Utility work — session summaries, moderator, assistant — stays on the smaller tier.

### Unified router for jobs

Per-agent defaults aren't enough. Some *jobs* need different models than the agent's default chat tag.

Neural Junkie classifies collab tasks, delegation consults, and implementation sessions through a **unified router** (`internal/routing`):

- **LLM default:** `qwen3.5:9b` returns structured JSON — `domain`, `tool_need`, `cost_tier`, `confidence`, `reason`
- **Rules fallback:** consolidated keyword lists when the LLM fails, times out, or confidence drops below 0.6
- **Table-tested:** 80+ deterministic cases mirror the chat quality router harness

| Job shape | Typical route |
|-----------|---------------|
| Security collab task | `nj-security:14b` if installed, else premium cloud |
| Short typo fix | cheapest local tier |
| Biology tool question | `tool_need: true` → tool model loop |
| User image + vision | lowest-cost provider that supports vision |

Normal chat and per-agent defaults are unchanged. Routing kicks in for **collaboration execution tasks**, **delegation consults**, and **implementation sessions** — the work that would otherwise burn your best model on "fix this typo."

Debug when `NEURAL_JUNKIE_DEBUG=1`:

```bash
curl 'http://localhost:18765/api/debug/routing-classify?q=review+JWT+auth+flow&agent_type=security'
```

## Question 3: Which provider and tool loop?

The inference layer treats **providers as interchangeable backends** — not the product.

| Provider | Role |
|----------|------|
| **Ollama** | Default local inference, tool loops, LoRA composed tags |
| **Anthropic / OpenAI-compatible** | Configured fallbacks and premium paths |
| **Hugging Face Inference** | Hosted models (e.g. ESMFold for protein folding) |
| **Cursor / Gemini CLI** | Subprocess-backed agents when binaries are on PATH |

Implementation mode prefers **local Ollama first** for tool loops, with a smaller tool model (`qwen3.5:9b`) when configured. Cloud providers are fallbacks — not the default path for every file edit.

### Split-brain specialists: one answer, two models

The clearest example of split inference is **BiologyExpert**:

| Job | Model | Why |
|-----|-------|-----|
| **Chat / reasoning** | OpenBio 8B | Med/science-tuned weights |
| **MCP tool loop** | `qwen3.5:9b` | Native Ollama tool calling |
| **Empty-reply safety net** | `qwen3.5:9b` | Fallback if OpenBio returns empty stream |

OpenBio doesn't pretend it can call tools. Qwen doesn't pretend it was fine-tuned on MedMCQA. **Each model does what it's good at.** The hub wires them into one user-facing answer.

Pack compose templates declare this explicitly:

```yaml
compose:
  chat_model: koesn/llama3-openbiollm-8b:latest
  tool_model: qwen3.5:9b
  lora_tag: nj-biology:8b
  consult_triggers: [biology, protein, sequence]
```

Life sciences and software development packs ship compose blocks today. One code path for DM, collab, and implementation — no biology-only `if` branches scattered through the hub.

## Trust: see which model ran

Modular AI without observability is just vibes.

Every agent response can carry routing metadata:

- `routing_model` — chat model tag
- `routing_tool_model` — tool loop when fallback fired
- `routing_reason` — e.g. `security_lora_local`, `cheap_local`
- `routing_source` — `llm` or `rules`

The desktop shows a compact badge on each reply (`nj-security:14b · llm`). Tooltip shows the full chain. Toggle in **Settings → Layout → Routing badges on messages**.

When you're debugging a collab run or arguing with a security review, you need to know **which brain actually ran** — not which agent name appeared in the chat bubble.

## Walkthrough: three messages, three inference paths

**Message 1 — zero inference**

> "Thanks, that helped!"

Intent: `closure`. Canned reply. **Zero tokens.** No workspace scan. No tool dump.

**Message 2 — domain chat, no tools**

> "What's the difference between a scan summary and a scan analysis export in our Phoenix workflow?"

BiologyExpert DM. No MCP tools required. Hub streams through **OpenBio**. Domain-aware language without spinning up a tool loop.

**Message 3 — split inference**

> "Analyze this peptide and fold the structure: MKTAYIAKQRQISFVKSHFSRQ."

MCP tools registered. Chat model lacks native tool support. Hub routes the tool loop through **`qwen3.5:9b`**:

1. Qwen calls `analyze_sequence` → length, composition, validation
2. Qwen calls `fold_protein` (HF token from Settings) → PDB under `~/.neural-junkie/bio/`
3. Qwen synthesizes one reply with paths and interpretation

Reply badge: `routing · chat: koesn/llama3-openbiollm-8b · tool: qwen3.5:9b · source: llm`

Three messages. Three different inference paths. One coherent specialist experience.

## Local-first economics

On a 16 GB laptop, you can't run six 14B models in parallel. Neural Junkie runs **sequential inference** on one Ollama backend when several agents reply in a thread or collaboration.

The inference layer makes that workable:

- **Deflect easy turns** before the GPU wakes up
- **Right-size every call** — utility tier for summaries, LoRA tags for domain tasks, premium cloud only when configured and warranted
- **Document stacks per RAM tier** — `GET /api/system/hardware` returns `recommended_stacks[]` from minimal (~10 GB) through heavy (~50+ GB)

You're not guessing which `nj-*` tags to pull. The wizard and Pack Store render the same data.

## What we learned building this

**1. Inference avoidance is a feature.**

The fastest inference is the one you don't run. Classify intent first. Save the big model for substantive work.

**2. One model is a demo. A decision pipeline is a product.**

Specialists need different weights. Background tasks need smaller models. Tool loops sometimes need a different tag than chat. Collab tasks need per-task routing. Pretending one 14B does all of it wastes RAM and produces worse UX.

**3. Orthogonal layers compose.**

The context stack doesn't replace the unified router. The router doesn't replace per-agent defaults. LoRA doesn't replace MCP export. Each layer solves one constraint.

**4. Local-first, cloud when it earns its keep.**

We default to Ollama for specialists and tool loops. Cloud providers are configured fallbacks and premium paths for high-stakes tasks — not the only way to run the hub.

**5. Test the orchestrator in code; test the conversation in scenarios.**

Models change weekly. Deterministic tests cover phase transitions, routing heuristics, and classifier fallbacks. Live JSON scenarios cover "did the security task actually prefer the LoRA tag?" without guessing from the UI.

## What's still on the roadmap

Honest gaps — not a commitment, but where the inference layer goes next:

- **Semantic answer cache** — deflect repeated questions before compute (lower priority for local desktop)
- **Unified knowledge topology** — vector vs hybrid vs graph retrieval per question shape (partially split across memory, `@codebase`, collab artifacts today)
- **Model/data policy gates** — stronger pre-generation policy beyond tool approval and byte budgets

## Try it

Personal open-source project — macOS, Windows, Linux.

```bash
make pull-models   # qwen3.5:9b + LoRA bases
make start-all
```

Download: https://github.com/camronwood/neural-junkie/releases/latest

**Docs:**
- [Context model](https://github.com/camronwood/neural-junkie/blob/main/docs/CONTEXT_MODEL.md)
- [Delegation](https://github.com/camronwood/neural-junkie/blob/main/docs/DELEGATION.md)
- [LoRA adapters](https://github.com/camronwood/neural-junkie/blob/main/docs/LORA_ADAPTERS.md)
- [Hardware tiers](https://github.com/camronwood/neural-junkie/blob/main/docs/HARDWARE.md)

If you run a multi-model setup and hit a routing edge case — wrong model on a collab task, delegation firing on small talk, closure not skipping inference — GitHub issues welcome. That feedback becomes the next scenario in the harness.

**Earlier in the series:** [Model layering](https://github.com/camronwood/neural-junkie/blob/main/campaigns/model-layering/MODEL-LAYERING-LINKEDIN.md) · [Modular AI composition](https://github.com/camronwood/neural-junkie/blob/main/campaigns/modular-ai/MODULAR-AI-COMPOSITION-LINKEDIN.md)

Camron Wood — Neural Junkie (personal project)

## PASTE END
