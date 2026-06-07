# LinkedIn article — Model layering (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `assets/neural-junkie-model-layering-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-model-layering-article.sh`

**Suggested title (pick one):**
- We Don't Use One Model. We Layer Them.
- How Neural Junkie Stacks Models to Make Local Multi-Agent AI Actually Work
- Prompts, Weights, and Routing: The Three Layers Behind Our AI Specialists

**Feed post teaser:**
> One 14B model can't be your security reviewer, biology expert, session summarizer, and cheap typo-fixer. Neural Junkie layers models at four levels — context, weights, routing, and orchestration — so local multi-agent work stays fast, grounded, and under your control.

**Hashtags:** `#AI #LocalAI #MultiAgent #LoRA #DeveloperTools #OpenSource #Ollama`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

---

## PASTE START

Most multi-agent demos look impressive until you ask a simple question: **which model is actually running, and why?**

If the answer is "the same one for everything," you pay for it — in RAM, latency, hallucinated tool calls on "thanks," and security reviews that read like generic chat.

**Neural Junkie** is a personal open-source desktop hub where you run AI specialists on your hardware — security reviewers, architects, repo experts, biology specialists — under your control. The product bet isn't one perfect model. It's **layering different models at the right level** so each turn gets the right brain, the right context, and the right cost.

Here's how we stack them.

## The four layers (at a glance)

Every user turn flows through a stack before you see a reply:

1. **Context** — decide how much model you need (mode, intent, memory, budget)
2. **Weights** — decide which model tag (14B coder, 7B utility, LoRA tags, domain models)
3. **Routing** — decide which provider per job (collab tasks, implementation, delegation)
4. **Orchestration** — multi-model flows that still feel like one product

Same hub. Same UI. Different model decisions at each stage.

## Layer 1: Context — decide *how much* model you need

Before any LLM call, every user turn passes through a **Conversation Context Stack**:

1. **Mode** — chat, code, or collaboration (from the composer or auto-detected signals)
2. **Intent** — closure, casual, substantive, or task
3. **Memory** — rolling session summary + capped history
4. **Grounding** — workspace scan only when you're actually coding
5. **Persona** — DM vs channel vs collab framing
6. **Budget** — byte caps so prompts don't blow context windows

This layer often **avoids a model call entirely**.

"Thanks!" → canned closure reply. No 14B inference. No tool dump. No workspace scan.

A casual DM → minimal prompt with 2 history rows and a lightweight utility summary — not the full MCP + file-change block you'd send for a refactor task.

**The insight:** routing *context* is cheaper and more reliable than asking a big model to "behave appropriately" every turn.

## Layer 2: Weights — decide *which* model tag

Once we know the turn deserves a real inference, we pick the **weight layer**:

| Role | Default model | Why |
|------|---------------|-----|
| **Software specialists** | `qwen2.5-coder:14b` | Strong code, tools, implementation loops |
| **Utility / background** | `qwen2.5:7b` | Session summaries, moderator, assistant |
| **Domain chat (biology)** | `koesn/llama3-openbiollm-8b` | Med/science-tuned weights |
| **LoRA specialists** | `nj-security:14b`, `nj-biology:8b`, … | Composed tags — tens of MB, not tens of GB |

### Two-tier LoRA strategy

We wanted one Qwen base + tiny LoRA adapters for every role. Ollama's safetensors `ADAPTER` path supports **Llama, Mistral, and Gemma — not Qwen**.

So we split:

- **Inference tier:** Qwen stays the default for day-to-day specialist chat and tools
- **LoRA tier:** train and compose on Llama/Mistral bases, assign composed tags when you want domain-tuned weights

Prompt personas and the context stack stay the same. LoRA adjusts the layer underneath.

### MCP vs LoRA (knowledge vs behavior)

Two more mechanisms at different layers:

- **MCP export** — portable *knowledge* (indexed repo context, prompts, architecture notes)
- **LoRA** — portable *behavior* (weight adapters that lean inference toward your domain)

Same repo expert can **export context for teammates** and **train a local adapter from session history**. Complementary, not interchangeable.

## Layer 3: Routing — decide *which provider per job*

Per-agent defaults aren't enough. Some *jobs* need different models than the agent's default chat tag.

### Collaboration smart routing

When agents execute tasks after you approve a plan, the hub can route per task:

- **Security keywords** → local `nj-security:14b` if installed, else premium cloud provider
- **Short wording/typo tasks** → cheapest local tier
- **User images + vision** → lowest-cost provider that supports vision

Normal chat and per-agent defaults are unchanged. Routing kicks in for **collaboration execution tasks** — the work that would otherwise burn your best model on "fix this typo."

### Implementation sessions (local-first)

Implementation mode prefers **local Ollama first** for tool loops, with a smaller tool model (`qwen2.5-coder:7b`) when configured. Cloud providers are fallbacks — not the default path for every file edit.

### Agent delegation (consult, don't hand off)

When delegation is enabled, a responding agent can **silently consult** other specialists:

1. Hub scores relevance across registered experts
2. Up to 2 consultants run in-process (no public handoff)
3. Results inject as `=== DELEGATE_RESULTS ===`
4. The **responding agent's model** synthesizes one reply

Delegation is **skipped for closure and casual intents** — greetings shouldn't trigger a biology consult.

For biology specifically, we split further:

- **Domain reasoning** → OpenBio chat model
- **Domain tools** (MCP sequence/fold tools) → specialist runtime + **`qwen2.5:7b` tool loop** when the chat model lacks native tool support

One user-facing answer. Multiple models under the hood. By design.

## Example: OpenBio + Qwen = one smart biology specialist

The life-sciences pack is the clearest example of **layering models on purpose** instead of picking one "bio model" and hoping it does everything.

### The constraint

**OpenBioLLM** (`koesn/llama3-openbiollm-8b:latest`) is strong at biomedical reasoning — pathways, assay context, sequence interpretation in plain language. But in Ollama it **does not expose native tool calling**.

BiologyExpert still needs real tools:

- `analyze_sequence` — validate DNA/RNA/protein, length, reverse complement
- `fold_protein` — ESMFold via Hugging Face → PDB on disk
- `summarize_scan_summary` / `summarize_scan_analysis` — plate QC from Phoenix-style exports
- `run_12plex_qc` — SOP pass/fail per analyte

You can't drop OpenBio and call it done. You also can't swap in Qwen 14B for everything — you lose domain-tuned weights and burn RAM on chat that doesn't need it.

### The split: domain brain + tool hands

| Job | Model | Why |
|-----|-------|-----|
| **Chat / reasoning** | OpenBio 8B | Med/science-tuned weights; natural explanations |
| **MCP tool loop** | `qwen2.5:7b` | Native Ollama tool calling; small footprint (~4.5 GB) |
| **Empty-reply safety net** | `qwen2.5:7b` | Fallback if OpenBio returns an empty stream |

Same Ollama endpoint. Same BiologyExpert persona. **Different model per layer of the turn.**

From the user's side: one DM with BiologyExpert. Under the hood: **OpenBio for fluency, Qwen for execution.**

### Walkthrough: two messages, two models

**Message 1 — pure reasoning (OpenBio)**

> "What's the difference between a scan summary and a scan analysis export in our Phoenix workflow?"

No MCP tools required. The hub streams through **OpenBio**. You get domain-aware language without spinning up a 14B coder or a tool loop.

**Message 2 — tools required (Qwen orchestrates)**

> "Here's a peptide: MKTAYIAKQRQISFVKSHFSRQ. Analyze it and fold the structure."

BiologyExpert has MCP tools registered. The hub sees the chat model lacks native tool support and **routes the tool loop through `qwen2.5:7b`** on the same Ollama host:

1. Qwen calls `analyze_sequence` → length, composition, validation
2. Qwen calls `fold_protein` (HF token from Settings) → PDB under `~/.neural-junkie/bio/`
3. Qwen synthesizes one reply with paths and interpretation

OpenBio didn't need to pretend it could call tools. Qwen didn't need to pretend it was fine-tuned on MedMCQA. **Each model does what it's good at.**

### Delegation: any agent can borrow the stack

With **agent delegation** enabled, you don't have to DM BiologyExpert directly.

Ask **BackendEngineer** in a mixed channel:

> "We're adding a new analyte to the 12-plex — any QC gotchas from a biology perspective?"

The hub scores consultants, picks BiologyExpert, and classifies intent:

| Intent | What runs |
|--------|-----------|
| `domain_reasoning` | **OpenBio** via model consult — concise internal answer |
| `domain_tools` | BiologyExpert runtime + **Qwen tool loop** if the question needs `run_12plex_qc` or scan summaries |

Results inject as `=== DELEGATE_RESULTS ===`. BackendEngineer's model **merges them into one reply**. You still talk to one agent; the biology stack runs silently underneath.

Config (optional overrides in `~/.neural-junkie/config.json`):

```json
"delegation": {
  "enabled": true,
  "biology_chat_model": "koesn/llama3-openbiollm-8b:latest",
  "biology_tool_model": "qwen2.5:7b"
}
```

### Optional third layer: LoRA weights

Want local biology behavior baked into weights, not just prompts?

- **`nj-biology:8b`** — LoRA compose on `llama3:8b` + MedMCQA adapter (Specialist tuning pack)
- **`nj-bio:8b`** — full OpenBio GGUF import with Llama 3 chat template

Same hub routing: if the assigned tag lacks native tools, **Qwen still runs the tool loop.** LoRA adjusts the chat layer; it doesn't replace the orchestration.

### Why this pattern generalizes

Biology is the template for how we think about layering elsewhere:

- **Don't force one model to be expert + tool-runner + summarizer.**
- **Use domain weights where they matter** (OpenBio for reasoning).
- **Use a small, tool-capable model where reliability matters** (Qwen 7b for MCP).
- **Let the hub pick per turn** — not the user memorizing which tag to `@mention`.

That's what "smart bio model" means in Neural Junkie: not a single merged checkpoint, but a **composed specialist** — the right models, wired together, one coherent answer.

Pull both models:

```bash
ollama pull koesn/llama3-openbiollm-8b:latest
ollama pull qwen2.5:7b
```

Enable **Life sciences** in Settings → Domain packs, DM BiologyExpert, and try the two-message walkthrough above.

## Layer 4: Orchestration — multi-model flows that still feel like one product

The hardest layer isn't picking a tag. It's making **multi-model orchestration** feel coherent:

**Session summaries** run asynchronously on `qwen2.5:7b` every few user turns — keeping long DM threads coherent without re-sending full transcripts to the 14B specialist every time.

**Collaboration** uses bounded phases, task DAGs, workspace gates, and file approvals. Models improvise; the **hub enforces gates**. Chat markdown doesn't write to disk — structured file proposals do, after your approval.

**Cross-provider BYOM** lets you mix Ollama, Anthropic, OpenAI-compatible, and CLI-backed providers. Smart routing picks among *your configured* providers — not a hardcoded cloud default.

The product promise: **you talk to one specialist; the system layers models so that specialist has the right context, the right weights, and the right consults — without you managing the stack manually.**

## What we learned building this

**1. One model is a demo. Layering is a product.**

Specialists need different weights. Background tasks need smaller models. Tool loops sometimes need a different tag than chat. Collab tasks need per-task routing. Pretending one 14B does all of it wastes RAM and produces worse UX.

**2. Context routing beats prompt engineering for the easy cases.**

Don't send MCP tools and workspace scans on "hey." Classify intent first. Save the big model for substantive work.

**3. Orthogonal layers compose.**

LoRA doesn't replace MCP export. Smart routing doesn't replace per-agent defaults. Delegation doesn't replace collaboration orchestration. Each layer solves one constraint.

**4. Local-first, cloud when it earns its keep.**

We default to Ollama for specialists and tool loops. Cloud providers are configured fallbacks and premium paths for high-stakes tasks — not the only way to run the hub.

**5. Test the orchestrator in code; test the conversation in scenarios.**

Models change weekly. Deterministic tests cover phase transitions, DAG readiness, and routing heuristics. Live JSON scenarios cover "did the security task actually prefer the LoRA tag?" without guessing from the UI.

## Try it

Personal open-source project — macOS, Windows, Linux.

```bash
make pull-models   # qwen2.5-coder:14b + qwen2.5:7b + LoRA bases
make start-all
```

Download: https://github.com/camronwood/neural-junkie/releases/latest

**Docs:**
- [Context model](https://github.com/camronwood/neural-junkie/blob/main/docs/CONTEXT_MODEL.md)
- [LoRA adapters](https://github.com/camronwood/neural-junkie/blob/main/docs/LORA_ADAPTERS.md)
- [Biology pack](https://github.com/camronwood/neural-junkie/blob/main/docs/BIOLOGY_PACK.md)
- [Delegation](https://github.com/camronwood/neural-junkie/blob/main/docs/DELEGATION.md)
- [Collaboration](https://github.com/camronwood/neural-junkie/blob/main/docs/COLLABORATION.md)

If you run a multi-model setup and hit a routing edge case — wrong model on a collab task, delegation firing on small talk, LoRA compose failing — GitHub issues welcome. That feedback becomes the next scenario in the harness.

Camron Wood — Neural Junkie (personal project)

## PASTE END
