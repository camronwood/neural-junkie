You are improving Neural Junkie test coverage — **test-growth loop iteration 1/1**.

Objective: **strengthen the test suite** — add or tighten tests. This is NOT a repair loop.
If your new test exposes a product bug, stop and report it — do NOT patch product code to greenwash.

## Edit allowlist (mandatory)

You may ONLY edit:
- Go test files (`*_test.go`) under `internal/`, `cmd/`, `test/`
- Frontend tests (`*.test.ts`, `*.spec.ts`) under `desktop/src/`
- Python test files under `scripts/lib/`
- Scenario JSON under `scenarios/`
- Scenario helpers: `scripts/lib/scenario_assert.py`, `scenario_contract.py`
- Fixtures under `scenarios/fixtures/` when required for a new edge case

Do NOT edit product/runtime code unless the iteration is explicitly handing off to repair.

## Assertion policy (mandatory)

- ADD tests or TIGHTEN assertions only.
- Do NOT remove `none_match`, `contains_all`, or `expect_deliverables` quality bars.
- Do NOT widen regex patterns to greenwash flakes.
- New live scenarios must follow patterns in docs/CHAT_SCENARIOS.md and docs/TESTING.md.
- Pair new chat scenarios with Layer A cases when the bug was routing-related.

---

## Selected candidate: `failure_repro:chat:dm-backend-codebase-semantic`

**Kind:** failure_repro
**Title:** Convert failure `dm-backend-codebase-semantic` to stronger test
**Score:** 70
**Source:** docs/testing/layer-gate-chat-2026-07-04-1605-iter2.log

Scenario `dm-backend-codebase-semantic` failed in `layer-gate-chat-2026-07-04-1605-iter2.log`. Add or tighten a test that captures this failure class.

**Target paths:**

- `scenarios/chat/dm-backend-codebase-semantic.json`

**Suggested files to create or edit:**

- `scenarios/chat/dm-backend-codebase-semantic.json`

**Metadata:**

- scenario: dm-backend-codebase-semantic
- scenario_kind: chat
- artifact: layer-gate-chat-2026-07-04-1605-iter2.log

## Your task

1. Implement ONE focused test improvement for this candidate.
2. Prefer minimal, high-signal changes over broad rewrites.
3. For new live scenarios: copy a similar JSON from `scenarios/`, set tags/assertions, and add Layer A routing coverage when routing-related.
4. Run the verification commands below before finishing.
5. Summarize: what gap you closed, files changed, commands run.

## Targeted verification (run after your changes)

- make test-scenario-assert
- python3 scripts/chat-scenarios.py --scenario dm-backend-codebase-semantic

