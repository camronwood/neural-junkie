# Personal learning (all experts)

Pack-gated, **opt-in** semantic memory for every in-process expert (Assistant, repo experts, pack specialists). Nothing is saved without explicit user approval.

## Gates

1. **Specialist tuning** pack installed and enabled
2. Pack capability **`personal-learning`** (works without Python/CUDA — unlike LoRA training)
3. **Settings → AI & providers → Enable personal learning for experts** (default off)

## How capture works

| Trigger | Behavior |
|---------|----------|
| `/learn [draft]` | Opens desktop approval dialog scoped to the channel’s expert |
| Natural phrases | `remember that`, `I prefer`, `always use`, etc. → proposal only (not auto-saved) |
| Agent info → **Add learning** | Manual entry for that agent |
| Settings → **Saved learnings** | Bulk view + forget |

Approved learnings are stored in `~/.neural-junkie/learnings.json`, keyed by **`agent_id`**. Each expert only sees its own learnings in prompts.

## REST API

All routes require pack + opt-in (403 otherwise):

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/learnings?agent_id=` | List (omit param = all, for Settings) |
| `POST` | `/api/learnings` | Confirm/save |
| `DELETE` | `/api/learnings/{id}` | Forget |
| `GET` | `/api/learnings/stats?agent_id=` | Count + `ready_for_lora` |

## Prompt injection

When enabled, confirmed learnings append to the expert system prompt:

```
=== LEARNINGS FOR THIS EXPERT (user-confirmed) ===
```

Budget ~2 KB per agent. Not injected into CLI or moderator agents.

With `NEURAL_JUNKIE_DEBUG=1`, agent turns may include `injected_learnings_count` in message metadata.

## Optional: Train LoRA

When **`lora-training`** is also enabled and the expert has **10+** exportable chat rows:

- Agent info → **Train LoRA** (all expert types, not just repo)
- Stats API returns `ready_for_lora: true`
- Reuses the Specialist tuning train pipeline; tags vary by agent type (`nj-repo-*`, `nj-assistant-*`, `nj-{specialist}:14b`)

Semantic learnings (prompt) and weight adaptation (LoRA) are complementary — same agent info surface, different stores.

## Related docs

- [SPECIALIST_TUNING_PACK.md](SPECIALIST_TUNING_PACK.md) — pack capabilities
- [LORA_TRAINING.md](LORA_TRAINING.md) — train/compose workflow
- [LEARNING_LORA_TEST_HARNESS.md](LEARNING_LORA_TEST_HARNESS.md) — CI + live scenarios
- [ASSISTANT_AGENT.md](ASSISTANT_AGENT.md) — `/learn` in Assistant DM context
