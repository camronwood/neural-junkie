# Layer gate — collab — 2026-07-13-1842 UTC

layer=collab
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenario-regression` | FAIL | 245s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-2026-07-13-1842.log`

## Failures (tail)

### collab-scenario-regression (exit 2)

```text
>>> make collab-scenario-regression


=== Regression boot (collab-scenarios) ===
>>> Stopping Neural Junkie processes...
>>> Restoring scenario fixtures from git...
>>> Removing fixture collab runtime dirs...
OK: Ollama (http://127.0.0.1:11434/api/tags)
>>> Warming Ollama models (suite=release)...
>>> Ollama model readiness
  warm: qwen3.5:9b, qwen2.5-coder:14b, qwen2.5:3b
  pull roster: qwen3.5:9b, qwen2.5-coder:14b, qwen2.5:3b, gemma3:12b
  installed: qwen3.5:9b
  installed: qwen2.5-coder:14b
  installed: qwen2.5:3b
  installed: gemma3:12b
  warming qwen3.5:9b …
  OK warm qwen3.5:9b: loaded in 13s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 7s (keep_alive=24h)
  warming qwen2.5:3b …
  OK warm qwen2.5:3b: loaded in 2s (keep_alive=24h)
  smoke qwen3.5:9b …
  OK smoke qwen3.5:9b: ok
  smoke qwen2.5-coder:14b …
  OK smoke qwen2.5-coder:14b: Ok
  smoke qwen2.5:3b …
  OK smoke qwen2.5:3b: ok
  loaded now: qwen2.5:3b, qwen3.5:9b
OK: Ollama models ready
>>> Starting regression hub (in-process specialists)...
OK: hub healthy at http://127.0.0.1:18765
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): CodeReviewer, RustExpert, Swift-Development-iOSDeveloper, Codex, Copilot, Cursor, Gemini, SwitchTarget…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: collab-minimal-completion-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab e70de67c → collab-e70de67c-238b-4bad-b537-685f344f5d8b
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/e70de67c-238b-4bad-b537-685f344f5d8b
  ✓ [10] send: /resume-plan e70de67c-238b-4bad-b537-685f344f5d8b
  ✓ [11] wait_tasks: tasks completed
  ✓ [12] approve_file_changes: deliverable on disk (findings.md)
  ✗ [13] assert_files: judge:fail:ollama/qwen2.5-coder:14b: The deliverable includes information about server/main.go, which was explicitly instructed not to be mentioned.
  ✓ cleanup: cancelled and removed workspace artifacts
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: @BackendEngineer - Task 1: Write `collabs/e70de67c-238b-4bad-b537-685f344f5d8b/findings.md` summarizing `README.md` and `core/sample/main.go`. Constraints: Thre
    [collaboration_discussion] SoftwareArchitect: Task list refined:  - Task 1: @BackendEngineer - Write `collabs/e70de67c-238b-4bad-b537-685f344f5d8b/findings.md` summarizing `README.md` and `core/sample/main.
    [chat] dbc9aed3-ef91-4e68-b7f3-afc992c6f482: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/e70de67c-238b-4bad-b537-6
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/e70de67c-238b-4bad-b537-685f344f5d8b/findings.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/e70de67c-238b-4bad-b537-685f344f5d8b/findings.md).  Verification skipped
    [chat] dbc9aed3-ef91-4e68-b7f3-afc992c6f482: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/e70de67c-238b-4bad-b537-6
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/e70de67c-238b-4bad-b537-685f344f5d8b/findings.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/e70de67c-238b-4bad-b537-685f344f5d8b/findings.md).  Verification skipped
  --- end ---

=== FAIL: collab-minimal-completion-regression ===

make[1]: *** [collab-scenario-regression] Error 1

RESULT collab-scenario-regression: FAIL (exit 2, 245s)
```

