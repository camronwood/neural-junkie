# LinkedIn article — LoRA v2 upgrade (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `campaigns/lora-v2/creatives/neural-junkie-lora-v2-1200.png` (1200×627)

**Feed ad:** `campaigns/lora-v2/creatives/neural-junkie-lora-v2-ad-1080.png` (1080×1080)

**Regenerate:** `./scripts/compose-lora-v2-article.sh`

**Suggested title (pick one):**
- LoRA v2: When Your Repo Expert Starts Compounding
- Train Once, Compound Forever: LoRA v2 in Neural Junkie
- From One-Shot Adapters to Specialists That Keep Learning

**Feed post teaser:**
> LoRA v1 let you train a specialist from chat history and assign an Ollama tag. It worked — but it felt like a sidecar. LoRA v2 closes the loop: incremental refresh, dual-tag profiles, unified routing, MLX on Apple Silicon, and team sharing via MCP + Hugging Face.

**Hashtags:** `#AI #LoRA #LocalAI #DeveloperTools #OpenSource #FineTuning`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

**Website:** https://camronwood.github.io/neural-junkie/articles/lora-v2.html

**Download CTA:** https://camronwood.github.io/neural-junkie/download.html

**Related articles:** [LoRA v1](../lora/LORA-LINKEDIN.md) · [Two-tier LoRA](../two-tier-lora/TWO-TIER-LORA-LINKEDIN.md) · [MCP + LoRA](../mcp-lora/MCP-LORA-LINKEDIN.md) · [Personal learning](../personal-learning/PERSONAL-LEARNING-LINKEDIN.md)

---

## PASTE START

LoRA v1 let you train once from chat history — import a Hugging Face adapter, install pack presets, or export channel/collab/repo transcripts to Unsloth and compose an Ollama tag. It worked. It also felt like a sidecar: one-shot training, manual base-tag confusion, collab-only routing, no rollback, no team path.

LoRA v2 closes the loop. Personal learning v2 handles prompt-time memory; LoRA v2 handles weight-time compounding — same Specialist tuning pack, same gates, same local Ollama stack.

## v1 vs v2

| Area | v1 | v2 |
|------|----|----|
| Training | Full export every time | Incremental refresh + curated rows |
| Lifecycle | Train when 10+ turns | Ready / refresh badges, version rollback |
| Models | User picks Llama base | Transparent dual-tag profile (Qwen chat, Llama compose) |
| Routing | Keyword rules, collab tasks | Unified classifier, chat + collab + impl |
| Pipeline | One in-memory job | Queue, MLX on Apple Silicon, post-train eval |
| Sharing | Manual HF upload | Publish + MCP manifest + import suggestions |

## Five pillars

**1. Compound learning loop** — Adapter registry at `~/.neural-junkie/lora-adapters.json`. Refresh from delta rows instead of restarting. Roll back to a prior version with one API call.

**2. Transparent dual-tag profiles** — Agents store `model_profile`: inference on Qwen, compose on Llama, one composed tag in the UI. Tool loops still run on Qwen when the composed tag lacks native tools.

**3. Smarter routing** — One `SelectLoRATag` path for chat, collaboration, and planning. LLM classifier can emit `lora_tag`. Repo-aware preference when `nj-repo-*` is installed.

**4. Training you can trust** — FIFO queue with persisted jobs. `make deps-lora-mlx` on Apple Silicon. Post-train eval before assign. Publish adapters to Hugging Face from the hub.

**5. MCP + team** — Repo MCP exports include LoRA metadata. Import suggests train or compose. Teammates pull weights from HF while learnings travel as JSON bundles.

## Hero workflow

```
v1: create-repo-agent → 10+ Q&A → Train LoRA → assign tag
v2: same expert compounds → learnings inject daily → refresh when delta ready
     → eval passes → profile updates → teammate imports MCP + HF adapter
```

## What didn't change

Specialist tuning pack gates. User confirmation for training and learnings. Ollama-local compose. Prompt personas, context stack, collaboration gates unchanged.

## Try it

```bash
make pull-models
make deps-lora
make deps-lora-mlx   # Apple Silicon
```

Enable **Specialist tuning** → train or refresh from agent info → Model library.

Docs: [LORA_V2.md](https://github.com/camronwood/neural-junkie/blob/main/docs/LORA_V2.md)

Download: https://camronwood.github.io/neural-junkie/download.html

---

*Neural Junkie is a personal open-source project. Feedback welcome on GitHub.*

## PASTE END
