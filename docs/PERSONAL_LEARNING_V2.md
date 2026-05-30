# Personal learning v2

Multi-scope memory with **Ollama embedding retrieval**, edit/lifecycle APIs, collaboration scope, agent-suggested proposals, per-user isolation, and optional LoRA JSONL export.

See [PERSONAL_LEARNING.md](PERSONAL_LEARNING.md) for v1 baseline. v2 keeps the same gates: **Specialist tuning** pack + **`personal-learning`** capability + Settings opt-in. Nothing is persisted without user confirmation.

## Scopes

| Scope | Stored as | Injected when |
|-------|-----------|---------------|
| `agent` (default) | `agent_id` + provenance | That expert responds |
| `global` | Same file; filtered by `scope` | Any expert for the same user |
| `collaboration` | Requires `collaboration_id` | Expert responds in that collab channel |

Legacy rows with empty `user_id` remain visible to all sessions until re-saved.

## Storage

- `~/.neural-junkie/learnings.json` — v2 envelope `{ "version": 2, "entries": [...] }` (v1 arrays migrate on load)
- `~/.neural-junkie/learning-embeddings.json` — sidecar `learning_id → { model, vector, embedded_at }`

Fields added in v2: `scope`, `user_id`, `collaboration_id`, `updated_at`, `use_count`, `last_used_at`, `content_hash`.

## Retrieval (prompt time)

1. Current user message is the query (not full history).
2. Ollama `POST /api/embed` (default model `nomic-embed-text`, override via `features.learning_embed_model`).
3. Cosine top-k per section: global (3), agent (5), collaboration (3).
4. If Ollama is unreachable within ~200ms, **keyword overlap** fallback is used.
5. Vectors are cached on add/update; query is embedded once per turn.

Prompt sections (when present):

```
=== LEARNINGS FOR ALL EXPERTS (user-confirmed) ===
=== LEARNINGS FOR THIS EXPERT (user-confirmed) ===
=== LEARNINGS FOR THIS COLLABORATION (user-confirmed) ===
```

With `NEURAL_JUNKIE_DEBUG=1`, agent metadata may include `injected_learning_ids[]`.

## Settings toggles

| Toggle | Config key | Default |
|--------|------------|---------|
| Enable personal learning | `features.personal_learning_enabled` | off |
| Allow agent suggestions | `features.personal_learning_suggest_enabled` | off (requires main opt-in) |
| Embed model (advanced) | `features.learning_embed_model` | `nomic-embed-text` |

Agent suggestions emit `client_action: learning_proposal` at most once per agent/channel/5 minutes — still requires modal approval.

## REST API (v2 additions)

All routes require pack + opt-in.

| Method | Path | Purpose |
|--------|------|---------|
| `PUT` | `/api/learnings/{id}` | Edit content, category, scope |
| `GET` | `/api/learnings/query?agent_id=&q=&scope=&collaboration_id=&channel=` | Retrieval preview |
| `POST` | `/api/learnings/export` | JSON bundle for current user |
| `POST` | `/api/learnings/import` | Merge bundle (dedup by `content_hash`) |

Session scoping: when logged in, CRUD filters by hub session `user_id`.

`GET /api/learnings/stats` adds `global_count`, `collab_count`, `embedding_index_ready`.

## Desktop UX

- **Learning proposal modal** — scope select: This expert | All experts | This collaboration (collab option when channel has active collaboration).
- **Settings** — grouped lists (global / by expert / by collaboration), export/import, suggest toggle.
- **Agent info** — scope badge, edit in place.
- **LoRA train panel** — checkbox **Include confirmed personal learnings** (preview + train, capped at 50 rows).

## LoRA bridge

`GET /api/lora/train/preview?...&include_learnings=1&agent_id=` adds `learning_rows` to the count.

Training POST accepts `include_learnings` + `agent_id` to prepend Alpaca rows from confirmed learnings.

## Prerequisites for embeddings

Local hub with Ollama:

```bash
ollama pull nomic-embed-text
```

CI uses mocked HTTP / keyword fallback — no live embed model required.

## Related docs

- [LEARNING_LORA_TEST_HARNESS.md](LEARNING_LORA_TEST_HARNESS.md) — v2 scenarios and CI layers
- [LORA_TRAINING.md](LORA_TRAINING.md) — train/compose workflow
