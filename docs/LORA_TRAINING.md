# LoRA training

Neural Junkie can fine-tune a LoRA adapter locally from your chat history, collaboration task outputs, or repo-agent transcripts, then compose the result into Ollama.

**Requires the [Specialist tuning](SPECIALIST_TUNING_PACK.md) domain pack** (`lora-training` capability).

## Prerequisites

1. **Enable Specialist tuning** — Settings → Domain packs → install and enable **Specialist tuning**.
2. **Ollama models** — `make pull-models` pulls inference Qwen, utility tier, and LoRA bases (`llama3.1:8b`, `llama3:8b`, `llama3.2:3b`, `mistral:7b`). See [LORA_ADAPTERS.md](LORA_ADAPTERS.md) for the two-tier strategy.
3. **Python training stack** — `make deps-lora` creates `.venv-lora` and installs `requirements-lora.txt`. The hub uses that venv for training jobs automatically.
4. **CUDA** — strongly recommended; CPU training is experimental and slow.
5. **Hugging Face access** for gated bases (Llama 3) — `huggingface-cli login` or set `HF_TOKEN`.

## Training wizard (Desktop)

Open **Model library** (⇧⌘M) → **Train LoRA** tab (visible when Specialist tuning is enabled).

**Repo expert shortcut:** open agent info (ℹ️) on a repo expert → **Train LoRA** — fields are prefilled from session history.

1. **Source** — pick one:
   - **Channel** — DM or channel ID (current channel prefilled when opened from chat).
   - **Collaboration** — collaboration UUID; uses completed task description + output pairs.
   - **Repo** — channel ID plus optional agent name filter for repo experts.
2. **Base & tag** — e.g. base `llama3.1:8b` (default; Qwen bases are rejected), output tag `nj-repo-myapp:14b`.
3. **Hyperparameters** — rank (default 16), epochs (default 1), learning rate (default 2e-4).
4. **Start** — exports JSONL to `~/.neural-junkie/lora-training/{job-id}/`, runs `scripts/lora_train.py`, then `ollama create` via the existing compose pipeline.

Minimum **10** training rows required.

## API

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/lora/train/expert-context?agent_id=…` | Prefill for repo/pack expert training |
| `GET` | `/api/lora/train/preview?source=channel&source_id=general` | Row count estimate |
| `POST` | `/api/lora/train` | Start job |
| `GET` | `/api/lora/train/{id}` | Status + log tail |
| `DELETE` | `/api/lora/train/{id}` | Cancel |

Example:

```bash
curl -X POST http://localhost:18765/api/lora/train \
  -H 'Content-Type: application/json' \
  -d '{
    "source": "channel",
    "source_id": "dm-backend",
    "base_ollama_tag": "llama3.1:8b",
    "ollama_tag": "nj-repo-myapp:14b",
    "hyperparams": {"rank": 16, "epochs": 1}
  }'
```

## After training

- Assign the composed tag via agent info (ℹ️), `/switch-provider`, or **Settings → Advanced → specialist model overrides**.
- Publish to Hugging Face manually if you want to share the adapter (no auto-upload in v2).

## Related

- [SPECIALIST_TUNING_PACK.md](SPECIALIST_TUNING_PACK.md) — enable training, compose, and bootstrap adapters
- [LORA_ADAPTERS.md](LORA_ADAPTERS.md) — import, compose, pack presets
- [LORA-LINKEDIN.md](marketing/LORA-LINKEDIN.md) — LinkedIn article publish copy (cover: `assets/neural-junkie-lora-ad-1200.png`)
- [COLLABORATION.md](COLLABORATION.md) — collaboration task outputs as training data
