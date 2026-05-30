# Learning + LoRA test harness

Three-layer pattern (same as chat/collab): **orchestrator logic in Go; end-to-end behavior in JSON scenarios**.

## Layer 1 — Go unit tests (CI)

| Area | Location |
|------|----------|
| Storage CRUD | `internal/learning/storage_test.go` |
| Prompt scoping | `internal/learning/prompt_test.go` |
| API gates + CRUD | `cmd/server/learnings_handlers_test.go` |
| Expert-context tags | `cmd/server/lora_train_handlers_test.go` |
| Pack capability | `internal/packs/packs_test.go` |

```bash
go test ./internal/learning/... ./cmd/server/... -count=1
```

No Ollama, no `.venv-lora`, no Unsloth.

## Layer 2 — API smoke (CI)

`TestLearningLoRASmoke` in `cmd/server/learnings_handlers_test.go` chains:

1. Pack + opt-in gates
2. POST/GET/DELETE `/api/learnings`
3. GET `/api/learnings/stats`
4. GET `/api/lora/train/expert-context` for Assistant agent

```bash
make learning-lora-smoke
```

## Layer 3 — Live JSON scenarios (local)

**Prerequisites:**

- Hub running (`NEURAL_JUNKIE_RATE_LIMIT=0` recommended)
- Specialist tuning pack enabled
- Personal learning opt-in on (scenarios call `setup_pack` + settings merge)

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
| `post_learning` | `POST /api/learnings` |
| `assert_learnings` | `GET /api/learnings` — `min_count`, `any_match` |
| `assert_learning_proposal` | Last messages include `client_action.type == learning_proposal` |
| `assert_expert_context` | `GET /api/lora/train/expert-context` |
| `forget_learning` | `DELETE /api/learnings/{id}` |
| `send` / `wait_reply` | Chat triggers (reuse collab hub helpers). `wait_reply` supports `learning_proposal: true` to wait for proposal metadata |

### Starter scenarios

| File | Purpose |
|------|---------|
| `learning-save-and-list` | API-only CRUD |
| `learning-proposal-trigger` | Trigger phrase → proposal metadata (not auto-saved) |

## Principles

1. **Assert structure, not model prose** — counts, JSON fields, regex on tags.
2. **Failures become scenarios** — regressions get a JSON file.
3. **CI runs Layers 1–2 only** — Layer 3 needs a live hub (and optional GPU for real train).

See also [CHAT_SCENARIOS.md](CHAT_SCENARIOS.md) and [marketing/CONVERSATIONAL-TEST-HARNESS.md](marketing/CONVERSATIONAL-TEST-HARNESS.md).
