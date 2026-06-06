# LoRA adapters

Neural Junkie composes **Ollama model tags** from a LoRA-compatible base plus a Hugging Face adapter (`FROM` + `ADAPTER`). Specialists can share small adapter files instead of pulling separate full models for every role.

## Two-tier strategy (inference vs LoRA)

| Tier | Default model | Purpose |
|------|---------------|---------|
| **Inference** | `qwen2.5-coder:14b` | Day-to-day specialist chat, tools, implementation (Software development pack) |
| **LoRA compose / train** | `llama3.1:8b` | Download community adapters, train repo experts, `ollama create` composed tags |

Ollama safetensors `ADAPTER` supports **Llama, Mistral, Gemma** — not Qwen. Qwen remains the recommended inference model; LoRA bootstrap and training use Llama/Mistral bases. Assign composed tags (`nj-security:14b`, etc.) when you want domain-tuned weights.

See also [TWO-TIER-LORA-LINKEDIN.md](marketing/TWO-TIER-LORA-LINKEDIN.md) for the full problem/solution story.

## Concepts

| Piece | Example | Role |
|-------|---------|------|
| **Inference base** | `qwen2.5-coder:14b` | Default specialist model (optional override with composed tag) |
| **LoRA base** | `llama3.1:8b` | Full weights for compose/train; must exist before compose |
| **Adapter** | `adapter_model.safetensors` | Small HF LoRA delta (~ tens of MB) |
| **Composed tag** | `nj-security:14b` | `ollama create` result used at inference time |

Prompt personas and the [context stack](CONTEXT_MODEL.md) are unchanged — LoRA adjusts the model layer underneath.

## Tag conventions

- **Specialists:** `nj-{type}:14b` (e.g. `nj-security:14b`) — suffix is a role label, not base size
- **Repo experts:** `nj-repo-{slug}:14b` (slug from repo directory name)
- **Default LoRA base (code):** `llama3.1:8b`
- **Biology LoRA base:** `llama3:8b`

## Model Library flow

1. Open **Model library** (⇧⌘M) → **Hugging Face** → **Download (local)**.
2. Find an entry with kind **LoRA adapter** (deprecated Qwen entries are marked unsupported).
3. **Download** the adapter file and `adapter_config.json`.
4. Ensure the **LoRA base** is pulled (Ollama tab → e.g. `llama3.1:8b`, `llama3.2:3b`, `mistral:7b`).
5. **Compose & import** — creates the composed Ollama tag.
6. Assign via agent info (ℹ️) → provider/model, `/switch-provider`, or **Settings → Advanced → specialist model overrides**.

API: `POST /api/hf/import-ollama` with `kind: "adapter"`, `base_ollama_tag`, and optional `ollama_tag`.

## Per-agent assignment

`~/.neural-junkie/config.json` agents support an optional `model` field:

```json
{
  "type": "security",
  "name": "SecurityReviewer",
  "enabled": true,
  "provider_id": "ollama-local",
  "model": "nj-security:14b"
}
```

Agents default to `qwen2.5-coder:14b` until you assign a composed tag.

## Domain pack presets

LoRA bootstrap adapters live in the **[Specialist tuning](SPECIALIST_TUNING_PACK.md)** pack:

```yaml
# specialist-tuning/pack.yaml (Llama/Mistral — Ollama-compose compatible)
lora_adapters:
  - agent_type: security
    repo_id: scthornton/llama-3.2-3b-securecode
    ollama_tag: nj-security:14b
    base_ollama_tag: llama3.2:3b
  - agent_type: code-review
    repo_id: juzhengz/LoRI-D_code_llama3_rank_64
    ollama_tag: nj-code-review:14b
    base_ollama_tag: llama3:8b
  - agent_type: backend
    repo_id: visheshgupta/mistral-7b-text2sql-qlora
    ollama_tag: nj-backend:14b
    base_ollama_tag: mistral:7b
  - agent_type: biology
    repo_id: Pk3112/medmcqa-lora-llama3-8b-instruct
    ollama_tag: nj-biology:8b
    base_ollama_tag: llama3:8b
```

Install bootstrap LoRAs (requires Specialist tuning pack enabled):

```bash
curl -X POST http://localhost:18765/api/packs/specialist-tuning/install-loras
```

`make pull-models` pulls inference Qwen plus LoRA bases (`llama3.1:8b`, `llama3:8b`, `llama3.2:3b`, `mistral:7b`).

## Repo agents

```bash
/create-repo-agent /path/to/repo MyAppExpert ollama
/create-repo-agent /path/to/repo --adapter-repo org/myapp-lora
/create-repo-agent /path/to/repo --model nj-repo-myapp:14b
```

Train on `llama3.1:8b` (not Qwen). See [LORA_TRAINING.md](LORA_TRAINING.md).

## Limitations

- **Qwen safetensors LoRA** cannot be composed in Ollama (use Llama/Mistral or GGUF conversion — not bundled)
- Hosted HF inference stays full-model (no PEFT on router)
- Safetensors adapters only (not GGUF LoRA blobs)
- Each compose creates a distinct Ollama tag (no hot-swap per request)
- LoRA training uses `.venv-lora` (`make deps-lora`); requires Specialist tuning pack

## Biology LoRA vs full GGUF

| Tag | Kind | Base |
|-----|------|------|
| `nj-bio:8b` | Full GGUF import (OpenBio) | Local GGUF file |
| `nj-biology:8b` | LoRA compose | `llama3:8b` + MedMCQA adapter |

## Related

- [LORA_TRAINING.md](LORA_TRAINING.md) — in-app training wizard
- [TWO-TIER-LORA-LINKEDIN.md](marketing/TWO-TIER-LORA-LINKEDIN.md) — LinkedIn article (cover: `assets/neural-junkie-two-tier-lora-1200.png`)
- [LORA-LINKEDIN.md](marketing/LORA-LINKEDIN.md) — LoRA feature article
- [SPECIALIST_TUNING_PACK.md](SPECIALIST_TUNING_PACK.md)
- [GETTING_STARTED.md](GETTING_STARTED.md)
