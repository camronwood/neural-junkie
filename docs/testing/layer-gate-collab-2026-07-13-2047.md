# Layer gate — collab — 2026-07-13-2047 UTC

layer=collab
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenario-regression` | FAIL | 2578s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-2026-07-13-2047.log`

## Failures (tail)

### collab-scenario-regression (exit 2)

```text
 ✓ [9] wait_tasks: tasks completed
  ✓ [10] assert_messages: message assertions ok
  ✓ [11] approve_file_changes: deliverable on disk (findings.md)
  ✓ [12] assert_files: judge:pass:ollama/qwen2.5-coder:14b: The deliverable file provides substantive summaries of the `README.md` and `core/sample/main.go` files as requested.
  ✓ [13] send: /complete-collab 6baa8c61-ca0e-4845-929b-8c38a1249cfb --forc
  ✓ [14] wait_phase: phase=completed
  ✓ [15] assert_collab: collab snapshot ok
=== PASS: execute-deliverable ===

  ✓ cleanup: cancelled and removed workspace artifacts

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
  OK warm qwen3.5:9b: loaded in 11s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 1s (keep_alive=24h)
  warming qwen2.5:3b …
  OK warm qwen2.5:3b: loaded in 3s (keep_alive=24h)
  smoke qwen3.5:9b …
  OK smoke qwen3.5:9b: ok
  smoke qwen2.5-coder:14b …
  OK smoke qwen2.5-coder:14b: ok
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 13 agent(s): Codex, Cursor, DatabaseSpecialist, MobileEngineer, Copilot, SwitchTarget, CodeReviewer, Assistant…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: document-findings-execution ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab d5df6a1c → collab-d5df6a1c-f6e9-46e5-9fbc-0e891e84cf2d
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/findings-scope-repo/collabs/d5df6a1c-f6e9-46e5-9fbc-0e891e84cf2d
  ✓ [10] send: /resume-plan d5df6a1c-f6e9-46e5-9fbc-0e891e84cf2d
  ✓ [11] wait_tasks: tasks completed
  ✓ [12] approve_file_changes: deliverable on disk (findings.md)
  ✓ [13] assert_files: judge:pass:ollama/qwen2.5-coder:14b: The deliverable file substantively answers the user's request by documenting the findings in the specified format and content.
  ✓ [14] send: /complete-collab d5df6a1c-f6e9-46e5-9fbc-0e891e84cf2d --forc
  ✓ [15] wait_phase: phase=completed
  ✓ [16] assert_collab: collab snapshot ok
=== PASS: document-findings-execution ===

  ✓ cleanup: cancelled and removed workspace artifacts

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
  OK warm qwen3.5:9b: loaded in 3s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 1s (keep_alive=24h)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 13 agent(s): SREObservabilityEngineer, MobileEngineer, Codex, Cursor, DataMLEngineer, Swift-Development-iOSDeveloper, Copilot, SwitchTarget…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: execution-no-stack-commands ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @PlatformEngineer
  started collab a65d0770 → collab-a65d0770-4bd7-41b8-b9b4-ba0708bdd2b5
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'BackendEngineer': 1, 'PlatformEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] send: /resume-plan a65d0770-4bd7-41b8-b9b4-ba0708bdd2b5
  ✓ [9] wait_tasks: executing settle 120.0s statuses=['completed']
  ✓ [10] assert_messages: message assertions ok
  ✓ [11] approve_file_changes: deliverable on disk (findings.md)
  ✓ [12] assert_files: judge:pass:ollama/qwen2.5-coder:14b: The deliverable file substantively answers the user's request by providing three bullets grounded in README.md and core/sample/main.go, as specified.
=== PASS: execution-no-stack-commands ===

  ✓ cleanup: cancelled and removed workspace artifacts

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
  OK warm qwen3.5:9b: loaded in 14s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 0s (keep_alive=24h)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 13 agent(s): Copilot, DatabaseSpecialist, Swift-Development-iOSDeveloper, Cursor, SwitchTarget, CodeReviewer, SREObservabilityEngineer, Assistant…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: collab-conversation-quality-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 17d255dc → collab-17d255dc-97a4-4189-bc34-7f34a56f7561
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] assert_messages: message assertions ok
  ✓ [9] send: /resume-plan 17d255dc-97a4-4189-bc34-7f34a56f7561
  ✗ [10] wait_tasks: task wait timeout statuses=['in_progress']
  ✓ cleanup: cancelled and removed workspace artifacts
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: The plan is sound. Before executing Task 1, I need to inspect `core/sample/main.go` and `README.md` to ensure the bullets in `findings.md` accurately reflect th
    [collaboration_discussion] BackendEngineer: I agree with the direction. Before finalizing the task list, I need to inspect `core/sample/main.go` and `README.md` to ensure the bullets in `findings.md` accu
    [collaboration_discussion] BackendEngineer: I see the plan is to have @BackendEngineer write `collabs/17d255dc-97a4-4189-bc34-7f34a56f7561/findings.md` with three bullets about `README.md` and `core/sampl
    [collaboration_discussion] BackendEngineer: I agree with the current plan direction. Before finalizing the task list, I need to inspect `core/sample/main.go` and `README.md` to ensure the findings in `fin
    [collaboration_discussion] BackendEngineer: The plan is solid and adheres to the constraints: focusing strictly on `README.md` (core/sample) and `core/sample/main.go`, while ignoring `src/theme.css` and o
    [collaboration_discussion] BackendEngineer: I agree with the current plan. Before finalizing the task assignment to `@BackendEngineer`, I need to inspect the referenced files to ensure the findings are gr
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: src/theme.css).  Verification skipped (interactive trust — approve proposals to 
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: src/theme.css).  Verification skipped (interactive trust — approve proposals to 
  --- end ---

=== FAIL: collab-conversation-quality-regression ===

make[1]: *** [collab-scenario-regression] Error 1

RESULT collab-scenario-regression: FAIL (exit 2, 2578s)
```

