# LoRA v2

LoRA v2 turns one-shot adapter training into a **compound specialist** lifecycle — versioning, incremental refresh, unified routing, and team sharing.

Requires the [Specialist tuning](SPECIALIST_TUNING_PACK.md) pack (`lora-training`, `lora-compose`, `personal-learning`).

## Five pillars

| Pillar | What changed |
|--------|----------------|
| **1. Learning loop** | Adapter registry, incremental refresh, curated rows, rollback |
| **2. Dual-tag profiles** | `model_profile` on agents — Qwen inference + Llama compose hidden in UI |
| **3. Smarter routing** | Unified `SelectLoRATag` in chat, collab, and LLM classifier |
| **4. Training pipeline** | Job queue + persistence, MLX on Apple Silicon, post-train eval, HF publish |
| **5. Team + MCP** | MCP bundles include LoRA metadata; import suggests train/compose |

## Adapter registry

Path: `~/.neural-junkie/lora-adapters.json`

| API | Purpose |
|-----|---------|
| `GET /api/lora/adapters` | List adapters (`?agent_id=`, `?ollama_tag=`) |
| `GET /api/lora/adapters/{id}` | One version |
| `POST /api/lora/adapters/{id}/activate` | Rollback — re-compose from stored artifact |
| `GET /api/lora/adapters/{id}/eval` | Post-train eval score |
| `POST /api/lora/adapters/{id}/publish` | Upload to Hugging Face (`repo_id` in body) |

## Training (v2)

| API | Purpose |
|-----|---------|
| `GET /api/lora/train` | List jobs |
| `GET /api/lora/train/active` | Running job |
| `GET /api/lora/train/dataset-preview` | Rows with `row_id` for curation (query params) |
| `POST /api/lora/train/dataset-preview` | Same, with `extra_rows` in body |
| `POST /api/lora/train/index-bootstrap` | Template rows from repo index when chat history is thin |
| `POST /api/lora/train` | Start (supports `incremental`, `prior_adapter_id`, `row_ids`, `extra_rows`) |
| `DELETE /api/lora/train/{id}` | Cancel |

Readiness: `ready_for_lora = chat_rows + learning_rows >= 10`. `refresh_suggested` when a prior adapter exists and delta rows ≥ 20.

## Model profile

```json
{
  "model_profile": {
    "inference_model": "qwen3.5:9b",
    "lora_compose_base": "llama3.1:8b",
    "composed_tag": "nj-repo-myapp:14b",
    "use_composed_for_chat": true
  }
}
```

Resolved by `internal/lora/profile` — tool loop still uses Qwen when composed tag lacks native tools.

## MLX training (Apple Silicon)

```bash
make deps-lora-mlx
```

Hub prefers MLX on `darwin`/`arm64` when `.venv-lora-mlx` exists. Override with `NJ_LORA_FORCE_UNSLOTH=1`.

## MCP + team

Repo MCP exports may include:

```json
"lora": {
  "composed_tag": "nj-repo-myapp:14b",
  "base_ollama_tag": "llama3.1:8b",
  "training_manifest": { "row_count": 142, "adapter_version": 3 }
}
```

`POST /api/import` returns `lora_train_suggestion` when the bundle has LoRA metadata.

## Related

- [LORA_TRAINING.md](LORA_TRAINING.md) — wizard and prerequisites
- [LORA_ADAPTERS.md](LORA_ADAPTERS.md) — import and compose
- [LEARNING_LORA_TEST_HARNESS.md](LEARNING_LORA_TEST_HARNESS.md) — CI layers
- [marketing/LORA-V2-LINKEDIN.md](marketing/LORA-V2-LINKEDIN.md) — launch article
