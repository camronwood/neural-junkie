# LinkedIn article — LoRA adapters & training (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn’s title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `assets/neural-junkie-lora-ad-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-lora-article.sh`

**Suggested title (pick one):**
- One Base Model. Many Specialists. LoRA Inside Neural Junkie.
- Your Chat History Is Training Data. Here’s What We Built With It.
- LoRA Without the MLOps Tax: Import, Compose, Train, Assign

**Feed post teaser:**
> Running five local specialists shouldn’t mean five full 14B downloads. Neural Junkie composes Ollama tags from one base + small LoRA adapters — import from Hugging Face, install pack presets, or train from your own chat and collab history.

**Hashtags:** `#AI #LoRA #LocalAI #DeveloperTools #OpenSource #FineTuning`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

---

## PASTE START

Neural Junkie is a desktop hub where you run multiple AI specialists — security reviewer, architect, biology expert, repo agent — on your hardware, under your control.

If every specialist needs its own full model, local multi-agent work gets expensive fast. A 14B coder here, an 8B biology model there, another 14B for security — disk, RAM, and pull time add up. Prompt personas help agents sound different, but they don’t change what the weights actually know.

LoRA (Low-Rank Adaptation) is what we added: small adapter files on a shared base model, composed into distinct Ollama tags at inference time. Same hub, same UI, same orchestration — different specialist behavior underneath, without five full model pulls.

## Why LoRA, not just prompts

System prompts, lanes, and tooling are necessary — but not enough when you want domain-tuned behavior (security review that reads like security review), repo-specific fluency (patterns from your codebase, not only RAG snippets), and efficient local inference (one qwen2.5-coder:14b base shared across roles).

Adapters are typically tens of megabytes, not tens of gigabytes. Pull the base once, attach per specialist, and Ollama composes tags like nj-security:14b or nj-repo-myapp:14b. Prompt personas and the context stack stay unchanged; LoRA adjusts the model layer underneath. Routing, collaboration, MCP tools, and file approvals work the same.

## Compose, don’t duplicate

Three pieces: a **base model** (full weights in Ollama), an **adapter** (small Hugging Face LoRA delta), and a **composed tag** (what agents call at inference — e.g. nj-security:14b). Base + adapter → ollama create → assign to specialist. The hub handles download, compose, and assignment — no hand-edited Modelfiles per role.

Tag conventions: nj-{type}:14b for specialists, nj-repo-{slug}:14b for repo experts, nj-biology:8b on llama3:8b for the life-sciences pack.

## Three ways to get a composed tag

**Import from Hugging Face.** Model library → Hugging Face → LoRA adapter → Download → Compose & import. Assign via Settings → Specialist model overrides or /switch-provider.

**Domain pack presets.** Software development ships a security adapter on Qwen Coder; life sciences ships a MedMCQA biology adapter on Llama 3. One API call: POST /api/packs/software-development/install-loras on localhost:18765.

**Train from your history.** Model library → Train LoRA. Sources: channel/DM transcripts, collaboration task outputs, or repo-agent history. Set base tag, output tag, rank/epochs, Start. Hub exports JSONL, runs Unsloth (optional Python + CUDA), composes into Ollama on success. Minimum 10 rows; one job at a time. Optional: pip install -r requirements-lora.txt.

## What you can do with it

**Security on a shared base.** Install the dev-pack LoRA — SecurityReviewer uses nj-security:14b on qwen2.5-coder:14b. One pull, security-tuned inference.

**Biology without a second full 8B.** nj-biology:8b from llama3:8b + MedMCQA adapter — lighter than a separate full biology GGUF when LoRA is enough.

**Repo expert from your sessions.** /create-repo-agent on your repo, accumulate Q&A, train nj-repo-myapp:14b from repo source, assign back. Chat and collab become training signal.

**Collab routing.** With smart routing on, security tasks can prefer nj-security:14b locally when installed. Override per task with task_ollama_model when needed.

**Train elsewhere, import here.** Publish to Hugging Face, download in Model library, compose, assign — no lock-in to our trainer.

## How it fits the product

LoRA is orthogonal to what you already use: context stack, collaboration phases, workspace gates, repo agents, and cloud BYOM providers. Fine-tuning is a model-layer concern; the hub still enforces gates and approvals regardless of tag.

## Limitations worth knowing

Safetensors adapters only (not GGUF LoRA blobs). Each compose is a distinct Ollama tag — no hot-swap per request. Training is optional, separate, and not bundled with the hub. No auto-upload to Hugging Face. Tool calling follows the base model. Hosted HF inference stays full-model; PEFT compose is local Ollama. Quality still depends on training data and base choice.

## Try it

Personal open-source project — macOS, Windows, Linux.

Download: https://github.com/camronwood/neural-junkie/releases/latest

ollama pull qwen2.5-coder:14b — then import from Model library, install pack LoRAs, or Train LoRA from channel / collab / repo.

Docs: https://github.com/camronwood/neural-junkie/blob/main/docs/LORA_ADAPTERS.md and https://github.com/camronwood/neural-junkie/blob/main/docs/LORA_TRAINING.md

Issues welcome if you train from collab output or hit compose edge cases — that feedback shapes the next pack preset.

Camron Wood — Neural Junkie (personal project)

## PASTE END
