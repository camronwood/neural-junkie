# Specialist tuning pack

Neural Junkie can train and compose **LoRA adapters** locally so your experts (especially **repo experts**) get sharper over time — without pulling separate full models for every role.

## What this pack adds

| Capability | Token | Description |
|------------|-------|-------------|
| **Personal learning** | `personal-learning` | Opt-in semantic memory per expert; user-approved prompt injection (no Python/CUDA) |
| **Train LoRA** | `lora-training` | Model library → Train LoRA; export chat/collab/repo data; Unsloth fine-tune; compose into Ollama |
| **Compose adapters** | `lora-compose` | Hugging Face adapter download + `ollama create`; `/create-repo-agent --adapter-repo` |
| **Bootstrap adapters** | `lora-adapters` | Pack store **Install LoRAs**; community presets (SecureCode, code feedback, text-to-SQL, MedMCQA biology) |

This pack does **not** add new specialist agents. Domain packs (Software development, Life sciences) still own those. Specialist tuning owns the **adapter lifecycle**.

## Enable the pack

**Settings → AI & providers → Domain packs → Pack store** — install and enable **Specialist tuning**.

When enabled:

- `qwen2.5-coder:14b` and `llama3:8b` merge into **models to ensure**
- Bootstrap LoRAs install in the background (or via **Install LoRAs**)
- **Train LoRA** tab appears in Model library (⇧⌘M)
- Repo experts show **Train LoRA** in agent info when enough training rows exist

## Hero workflow: repo expert

1. `/create-repo-agent /path/to/repo MyAppExpert`
2. Work in the repo expert channel (10+ Q&A turns recommended)
3. Open agent info → **Train LoRA** (prefills repo source + `nj-repo-{slug}:14b`)
4. Start training → assign composed tag back to the expert

See [LORA_TRAINING.md](LORA_TRAINING.md) and [REPO_AGENTS.md](REPO_AGENTS.md).

## Bootstrap community adapters

Optional presets (assign via agent info, `/switch-provider`, or Settings → Advanced → specialist model overrides):

| Tag | Use case | Base |
|-----|----------|------|
| `nj-security:14b` | Secure coding / OWASP | `qwen2.5-coder:14b` |
| `nj-code-review:14b` | Code feedback | `qwen2.5-coder:14b` |
| `nj-backend:14b` | Text-to-SQL / data layer | `qwen2.5-coder:14b` |
| `nj-biology:8b` | Medical/biology QA | `llama3:8b` |

## Python deps

First-time training stack:

```bash
make deps-lora
```

Or `make deps` after enabling this pack (optional; not required for hub runtime alone).

## Related

- [PERSONAL_LEARNING.md](PERSONAL_LEARNING.md)
- [LORA_ADAPTERS.md](LORA_ADAPTERS.md)
- [LORA_TRAINING.md](LORA_TRAINING.md)
- [SOFTWARE_DEVELOPMENT_PACK.md](SOFTWARE_DEVELOPMENT_PACK.md)
- [BIOLOGY_PACK.md](BIOLOGY_PACK.md)
