# LoRA adapters

Neural Junkie can compose **Ollama model tags** from a shared base plus a Hugging Face LoRA adapter (`FROM` + `ADAPTER`), so specialists share one 14B base instead of pulling separate full models.

## Concepts

| Piece | Example | Role |
|-------|---------|------|
| **Base model** | `qwen2.5-coder:14b` | Full weights in Ollama; must exist before compose |
| **Adapter** | `adapter_model.safetensors` | Small HF LoRA delta (~ tens of MB) |
| **Composed tag** | `nj-security:14b` | `ollama create` result used at inference time |

Prompt personas and the [context stack](CONTEXT_MODEL.md) are unchanged — LoRA adjusts the model layer underneath.

## Tag conventions

- **Specialists:** `nj-{type}:14b` (e.g. `nj-security:14b`)
- **Repo experts:** `nj-repo-{slug}:14b` (slug from repo directory name)
- **Default base:** `qwen2.5-coder:14b`

## Model Library flow

1. Open **Model library** (⇧⌘M) → **Hugging Face** → **Download (local)**.
2. Find an entry with kind **LoRA adapter** (or any HF repo with adapter safetensors).
3. **Download** the adapter file.
4. Ensure the **base model** is pulled (Ollama tab → `qwen2.5-coder:14b`).
5. **Compose & import** — creates the composed Ollama tag.
6. If the catalog entry has `agent_type`, the hub assigns that specialist automatically; otherwise use **Settings → AI & providers → Specialist model overrides** or `/switch-provider`.

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

`/switch-provider` and the desktop **Specialist model overrides** panel persist this field. **Switch all providers** clears per-agent overrides.

## Domain pack presets

Software development pack may declare:

```yaml
agents:
  - type: security
    name: SecurityReviewer
    ollama_model: nj-security:14b
  - type: code-review
    name: CodeReviewer
    ollama_model: nj-code-review:14b
  - type: backend
    name: BackendEngineer
    ollama_model: nj-backend:14b

lora_adapters:
  - agent_type: security
    repo_id: scthornton/qwen2.5-coder-14b-securecode
    ollama_tag: nj-security:14b
    base_ollama_tag: qwen2.5-coder:14b
  - agent_type: code-review
    repo_id: JingyaoOng/Qwen2.5-Coder-14B-Instruct-lora-CodeFeedback64k
    ollama_tag: nj-code-review:14b
    base_ollama_tag: qwen2.5-coder:14b
  - agent_type: backend
    repo_id: blank0301/qwen2.5-coder-14b-text2sql-sft-exec_dpo-lora
    ollama_tag: nj-backend:14b
    base_ollama_tag: qwen2.5-coder:14b
```

Install pack LoRAs in one call (also runs automatically when a pack is enabled and Ollama is up):

```bash
curl -X POST http://localhost:18765/api/packs/software-development/install-loras
```

Pack sync copies `ollama_model` into agent config and adds composed tags to `models_to_ensure` (bases are pulled on startup; composed tags are installed from pack manifests in the background).

## Repo agents

```bash
/create-repo-agent /path/to/repo MyAppExpert ollama
/create-repo-agent /path/to/repo --adapter-repo org/myapp-lora
/create-repo-agent /path/to/repo --model nj-repo-myapp:14b
```

With `--adapter-repo`, the hub downloads the adapter, composes `nj-repo-{slug}:14b` (or your `--model` tag), and uses it for the repo expert.

**Manual workflow:** train LoRA externally → publish to Hugging Face → download in Model Library → compose → assign.

## Limitations

- Hosted HF inference stays full-model (no PEFT on router)
- Safetensors adapters only (not GGUF LoRA blobs)
- Each compose creates a distinct Ollama tag (no hot-swap per request)
- Tool calling still follows base model capabilities; biology MCP fallback unchanged
- LoRA training uses the Python stack in `.venv-lora` (`make deps`); no auto-upload to Hugging Face

## Biology LoRA vs full GGUF

| Tag | Kind | Base |
|-----|------|------|
| `nj-bio:8b` | Full GGUF import (OpenBio) | Local GGUF file |
| `nj-biology:8b` | LoRA compose | `llama3:8b` + MedMCQA adapter |

## Related

- [LORA_TRAINING.md](LORA_TRAINING.md) — in-app training wizard
- [LORA-LINKEDIN.md](marketing/LORA-LINKEDIN.md) — LinkedIn article publish copy (cover: `assets/neural-junkie-lora-ad-1200.png`)

- [SOFTWARE_DEVELOPMENT_PACK.md](SOFTWARE_DEVELOPMENT_PACK.md)
- [GETTING_STARTED.md](GETTING_STARTED.md) — model library
- [REPO_AGENTS.md](REPO_AGENTS.md)
