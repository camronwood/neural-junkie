# Learning + LoRA test harness

Three-layer pattern (same as chat/collab): **orchestrator logic in Go; end-to-end behavior in JSON scenarios**.

## Layer 1 — Go unit tests (CI)

| Area | Location |
|------|----------|
| Storage CRUD + scope/migration | `internal/learning/storage_test.go`, `prompt_test.go` |
| Embeddings + keyword fallback | `internal/learning/embeddings_test.go` |
| Prompt scoping | `internal/learning/prompt_test.go` |
| Registry CRUD + rollback | `internal/lora/registry/registry_test.go` |
| Incremental export filter | `internal/lora/export/export_test.go` |
| Adapter API smoke | `cmd/server/lora_train_handlers_test.go` |
| API gates + CRUD + PUT/query | `cmd/server/learnings_handlers_test.go` |
| Expert-context tags | `cmd/server/lora_train_handlers_test.go` |
| Pack capability | `internal/packs/packs_test.go` |

```bash
go test ./internal/learning/... ./internal/lora/export/... ./cmd/server/... -count=1
```

No Ollama, no `.venv-lora`, no Unsloth. Embedding tests use keyword fallback or mocked HTTP.

## Layer 2 — API smoke (CI)

`TestLearningLoRASmoke` in `cmd/server/learnings_handlers_test.go` chains:

1. Pack + opt-in gates
2. POST/GET/DELETE `/api/learnings`
3. PUT + GET `/api/learnings/query` (v2)
4. GET `/api/learnings/stats`
5. GET `/api/lora/train/expert-context` for Assistant agent

```bash
make learning-lora-smoke
```

## Layer 3 — Live JSON scenarios (local)

**Prerequisites:**

- Hub running (`NEURAL_JUNKIE_RATE_LIMIT=0` recommended)
- Specialist tuning pack enabled
- Personal learning opt-in on (scenarios call `setup_pack` + settings merge)
- Optional for live embedding retrieval: `ollama pull nomic-embed-text` (scenarios use keyword fallback when Ollama is down)

```bash
make learning-scenario SCENARIO=learning-save-and-list
make learning-scenarios
./scripts/learning-scenarios.py --list
```

Scenarios live under `scenarios/learning/`. Runner: `scripts/learning-scenarios.py`.

### Step types

| Step | Purpose |
|------|---------|
| `setup_pack` | Enable specialist-tuning + personal learning opt-in |
| `post_learning` | `POST /api/learnings` — supports `scope`, `collaboration_id` |
| `assert_learnings` | `GET /api/learnings` — `min_count`, `any_match`, optional `scope` |
| `assert_learning_query` | `GET /api/learnings/query` — retrieval preview (v2) |
| `export_import_learnings` | POST export + import round-trip (v2) |
| `assert_learning_proposal` | Last messages include `client_action.type == learning_proposal` |
| `assert_expert_context` | `GET /api/lora/train/expert-context` |
| `forget_learning` | `DELETE /api/learnings/{id}` |
| `send` / `wait_reply` | Chat triggers (reuse collab hub helpers). `wait_reply` supports `learning_proposal: true` to wait for proposal metadata |

### Scenarios

| File | Purpose |
|------|---------|
| `learning-save-and-list` | API-only CRUD |
| `learning-proposal-trigger` | Trigger phrase → proposal metadata (not auto-saved) |
| `learning-global-scope` | Global scope save + query (v2) |
| `learning-collab-scope` | Collaboration scope save + query (v2) |
| `learning-export-import` | Export/import dedup (v2) |
| `learning-retrieval-debug` | Query ranking via keyword fallback (v2) |

## Principles

1. **Assert structure, not model prose** — counts, JSON fields, regex on tags.
2. **Failures become scenarios** — regressions get a JSON file.
3. **CI runs Layers 1–2 only** — Layer 3 needs a live hub (and optional GPU for real train).

See also [PERSONAL_LEARNING_V2.md](PERSONAL_LEARNING_V2.md), [CHAT_SCENARIOS.md](CHAT_SCENARIOS.md), and [marketing/CONVERSATIONAL-TEST-HARNESS.md](marketing/CONVERSATIONAL-TEST-HARNESS.md).
