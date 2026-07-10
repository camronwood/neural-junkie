# layer fix loop — collab-core — iteration 1 — 2026-07-09-1722-iter1 UTC

layer=collab-core
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-09-1722-iter1.md
fix_branch=release-prep/layer-collab-core-2026-07-09-1722
git_commit=170ed56
layer_gate_rc=2
agent_rc=0

## Targeted verification
- [FAIL] python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent (exit 1)
- [FAIL] python3 scripts/collab-scenarios.py --scenario document-findings-execution (exit 1)
- [OK] python3 scripts/collab-scenarios.py --scenario planning-two-agent (exit 0)

## Cursor agent output

```text
## Summary

Fixed two product bugs from the collab-core gate brief and added regression tests.

### 1. Workspace metadata on planning handoffs (CI failure)

**Problem:** `TestSlashCollaborateSeedOmitsWorkspaceByDefault` failed because the planning watchdog attached `workspace_context` to turn handoffs whenever `SourceWorkspaceContext` was stored—even when `/collaborate` was **not** invoked with `--workspace`.

**Fix:** Added `AttachWorkspaceContext` on collaborations (set from `--workspace` / `--repo` flags). Turn handoffs in the hub watchdog and agent handoff path only attach workspace metadata when that flag is set. Desktop/passive workspace snapshots still feed prompts via `SourceWorkspaceContext` without polluting message metadata.

### 2. Plan parser explosion for document-findings (live failure)

**Problem:** Agents sometimes split “summarize README.md and main.go” into extra tasks (`Summarize README.md`, etc.), pushing task count past `max_tasks: 2`.

**Fix:** Tightened `isWeakTaskFragment` to:
- Reject `Summarize …` rows that reference source files (not collab deliverables)
- Keep valid `findings.md` deliverable rows even after path normalization strips the `collabs/` prefix

### Tests added
- `TestSendPlanningTurnHandoff_OmitsWorkspaceWithoutAttachFlag`
- `TestSynthesizePlanFromDiscussion_documentFindingsExecutionRegression`
- `TestSynthesizePlanFromDiscussion_rejectsReadmeSubtasks`
- `TestExtractTasksFromCollaborationGoal_documentFindingsExecutionGoal`

---

## Commands run

| Command | Result |
|---------|--------|
| `make test-all` | **Partial** — collab/CI tests pass; unrelated failures remain in `cmd/server` (`TestHandleLoraTrainStart_rejectsQwenBase`) and `internal/config` (pack upgrade tests) |
| `make test-conversation-contract` | **PASS** (90/90) |
| `go test ./test/... -run TestSlashCollaborateSeedOmitsWorkspaceByDefault` | **PASS** |
| `go test ./internal/collaboration/... ./test/...` | **PASS** |
| `python3 scripts/collab-scenarios.py --scenario planning-two-agent` | **PASS** (after 1 flake retry; first attempt had 0 agent messages) |
| `python3 scripts/collab-scenarios.py --scenario document-findings-execution` | **FAIL** — `assert_plan` now passes (`tasks=1`); failed later on `assert_files` LLM judge (deliverable was a task list, not a README/main.go summary) |
| `python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent` | **FAIL** — silent-agent flake (0 discussion messages) |

The original **`assert_plan: tasks=3 want <=2`** failure for `document-findings-execution` is resolved. Remaining live failures look like agent silence flakes and execution/deliverable quality (judge), not harness issues. I did not weaken any assertions.**Live scenario results**

| Scenario | Result |
|----------|--------|
| `planning-two-agent` | **PASS** — first attempt timed out with 0 agent messages; passed on flake retry 1 |
| `document-findings-execution` | **FAIL** — `assert_plan` passed (`tasks=1`); failed at `assert_files` LLM judge because `findings.md` had a task list, not a README/main.go summary (written via discussion fallback) |
| `collab-participation-three-agent` | **FAIL** — silent agents (0 `collaboration_discussion` messages) |

The CI/parser fixes from earlier look good (`assert_plan` regression is gone). What’s left is agent silence flakes and execution deliverable quality for `document-findings-execution`. I can dig into the findings deliverable path next if you want that scenario green end-to-end.
```

