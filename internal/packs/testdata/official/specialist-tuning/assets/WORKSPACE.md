# Specialist tuning workspace

Experts get sharper over time — without thinking about LoRA.

## Closed learning loop

1. **Enable** Specialist tuning + personal learning (Settings → Memory & learning).
2. **Work** in a repo expert or domain channel. Use `/learn` to save approved preferences.
3. **Nudge** — after 10+ turns the hub suggests **Sharpen expert** (agent info or toast).
4. **Train** — Model library (⇧⌘M) → Train LoRA, or **Sharpen expert** in agent info. Personal learnings are included by default.
5. **Eval** — post-train keyword benchmark runs automatically. Score must meet the pack threshold (default 0.35) before assign.
6. **Assign** — apply the composed `nj-*` tag back to the expert. Incremental refresh when 20+ new rows since last adapter.

## Two-tier models (until Qwen LoRA compose)

- **Inference:** domain packs default to `qwen3.5:9b` (or `qwen3.5:27b`).
- **LoRA train/compose:** `llama3.1:8b` and bootstrap bases (`llama3.2:3b`, `llama3:8b`, `mistral:7b`).

When your Ollama build supports Qwen `ADAPTER` compose, set `NJ_LORA_QWEN_COMPOSE=1` and re-install this pack — training defaults move to `qwen3.5:9b`.

## Bootstrap adapters

Install from Pack store → **Install LoRAs**. Cross-pack tags:

| Tag | Domain | Notes |
|-----|--------|-------|
| `nj-security:14b` | Security | Legacy Llama tier |
| `nj-code-review:14b` | Code review | Legacy |
| `nj-backend:14b` | Backend / SQL | Legacy Mistral |
| `nj-biology:8b` | Life sciences | Legacy |
| `nj-cad:14b` | CAD | Interim code bootstrap |
| `nj-aws:14b` | AWS | Interim infra bootstrap |
| `nj-incident:14b` | Incident | Interim triage bootstrap |
| `nj-browser:14b` | Web browser | Interim frontend bootstrap |
| `nj-music:14b` | Music | Train-first — no community lyrics LoRA yet |

## Training sidecar

GPU-heavy training runs in the pack sidecar (when enabled). One-time setup:

```bash
./scripts/setup-lora-deps.sh
```

Or from the Neural Junkie repo: `make deps-lora` (uses core venv until sidecar venv is configured).

## Repo expert hero path

```
/create-repo-agent /path/to/repo MyAppExpert
→ work in channel (10+ Q&A turns)
→ Sharpen expert
→ review eval score → Apply to expert
```

## Export learnings

Settings → Memory & learning → Export bundle. Training also pulls confirmed learnings via **Include learnings** (on by default in v2).
