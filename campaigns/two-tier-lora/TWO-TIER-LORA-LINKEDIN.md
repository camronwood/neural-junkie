# LinkedIn article — Two-tier LoRA strategy (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines. Use LinkedIn's title field for the headline. One blank line between paragraphs is enough.

**Cover image:** `campaigns/two-tier-lora/creatives/neural-junkie-two-tier-lora-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-two-tier-lora-article.sh`

**Suggested title (pick one):**
- Why We Split Inference and LoRA Into Two Tiers
- Qwen for Chat, Llama for LoRA: What We Learned Building Local Multi-Agent AI
- The Ollama LoRA Gotcha Nobody Warned Us About (And How We Fixed It)

**Feed post teaser:**
> We wanted one Qwen base + tiny LoRA adapters for every specialist. Ollama said no — safetensors LoRA only works on Llama/Mistral/Gemma. Here's how Neural Junkie split inference (Qwen) from train/compose (Llama) without breaking the hub.

**Hashtags:** `#AI #LoRA #LocalAI #Ollama #DeveloperTools #OpenSource`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

---

## PASTE START

Neural Junkie runs multiple AI specialists locally — security reviewer, backend engineer, repo expert — on your hardware. LoRA adapters promised the dream: one shared base model, tens-of-megabytes deltas per role, composed Ollama tags like `nj-security:14b`.

We shipped bootstrap presets on Hugging Face. We documented `qwen2.5-coder:14b` as the base. Then `ollama create` failed with **unsupported architecture**.

## The problem

The Specialist tuning pack shipped community LoRAs trained on **Qwen 2.5 Coder 14B** — SecureCode for security, code-feedback for review, text-to-SQL for backend. Our docs said: pull Qwen, download the adapter, compose, assign.

Ollama's safetensors `ADAPTER` path supports **Llama, Mistral, and Gemma** architectures. **Not Qwen.** Same adapter files that work in Hugging Face PEFT fail at compose time in Ollama.

Symptoms we saw in the wild:

- Pack LoRA install retries on every startup (`unsupported architecture`)
- Training UI suggesting Qwen as a base — fine for chat, useless for compose-after-train
- Users conflating "best coding model" (Qwen) with "LoRA-compatible base" (Llama/Mistral)

We were optimizing for inference quality and LoRA composability with a single model tag. Those are different constraints.

## The insight

**Inference tier:** `qwen2.5-coder:14b` stays the default for software specialists — strong code, tools, implementation loops. Unchanged.

**LoRA tier:** train and compose on bases Ollama actually supports — `llama3.1:8b` for code training (default), `llama3.2:3b` + SecureCode for security bootstrap, `mistral:7b` for SQL bootstrap, `llama3:8b` for biology and code-review presets.

Composed tags keep familiar names (`nj-security:14b`, `nj-backend:14b`). The `:14b` suffix is a **role label**, not the LoRA base size. Assign a composed tag when you want domain-tuned weights; otherwise agents keep Qwen.

## What we changed

1. **Replaced Qwen bootstrap adapters** in the Specialist tuning pack with verified Llama/Mistral community presets.
2. **Aligned compose defaults** — `DefaultLoRABaseTag` is now `llama3.1:8b`, not Qwen.
3. **Guarded training** — the hub rejects Qwen bases for LoRA jobs with a clear error and suggested alternatives.
4. **Deprecated Qwen adapter catalog entries** — Model library marks them unsupported; compose is blocked in the UI.
5. **Extended `make pull-models`** — pulls LoRA bases (`llama3.1:8b`, `llama3.2:3b`, `mistral:7b`) alongside inference Qwen.
6. **Added `scripts/verify-bootstrap-loras.sh`** — reproducible compose verification for pack presets.

Routing, collaboration, MCP tools, and file approvals are unchanged. LoRA is still orthogonal to prompts and the context stack.

## How to use it

```bash
make pull-models          # Qwen inference + Llama/Mistral LoRA bases
make deps-lora            # once, before training
```

Enable **Specialist tuning** → **Install LoRAs** (or `POST /api/packs/specialist-tuning/install-loras`). Assign composed tags via agent info or `/switch-provider`. Train repo experts on **`llama3.1:8b`**, not Qwen.

## What we did not do (yet)

- **GGUF LoRA conversion** for Qwen (`llama.cpp`) — possible future path if you need Qwen train→compose end-to-end.
- **Rename composed tags** to `:8b` — would break existing configs; role labels stay as-is.

## Try it

Personal open-source project — macOS, Windows, Linux.

Download: https://github.com/camronwood/neural-junkie/releases/latest

Docs: [LORA_ADAPTERS.md](https://github.com/camronwood/neural-junkie/blob/main/docs/LORA_ADAPTERS.md) · [LORA_TRAINING.md](https://github.com/camronwood/neural-junkie/blob/main/docs/LORA_TRAINING.md) · [SPECIALIST_TUNING_PACK.md](https://github.com/camronwood/neural-junkie/blob/main/docs/SPECIALIST_TUNING_PACK.md)

Issues welcome if you hit compose edge cases on a new adapter — that feedback shapes the next bootstrap preset.

Camron Wood — Neural Junkie (personal project)

## PASTE END
