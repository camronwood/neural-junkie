# Layer gate — collab — 2026-07-13-1608 UTC

layer=collab
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenario-regression` | FAIL | 1263s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-2026-07-13-1608.log`

## Failures (tail)

### collab-scenario-regression (exit 2)

```text
 cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): FrontendEngineer, PlatformEngineer, SecurityReviewer, Codex, Copilot, Cursor, SREObservabilityEngineer, MobileEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 865ba165 → collab-865ba165-2840-4dc2-b8ac-aaeba416b17c
  ✓ [1] wait_phase: phase=planning
  wait_discussion: silent agents ['Claude']; nudging
  nudge: @Claude — please add your planning perspective for this collab.
  ✓ [2] wait_discussion: messages total=3 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=4)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_plan: plan ok (tasks=4)
=== PASS: plan-dependency-prose-regression ===

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
  OK warm qwen3.5:9b: loaded in 15s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 9s (keep_alive=24h)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): Swift-Development-iOSDeveloper, Codex, Gemini, SREObservabilityEngineer, Assistant, Cursor, FrontendEngineer, CodeReviewer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: plan-findings-task-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab b4df515c → collab-b4df515c-cedb-44cf-bc5c-286ff3529bc1
  ✓ [1] wait_phase: phase=planning
  wait_discussion: silent agents ['Claude']; nudging
  nudge: @Claude — please add your planning perspective for this collab.
  ✓ [2] wait_discussion: messages total=3 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=4)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_plan: plan ok (tasks=4)
=== PASS: plan-findings-task-regression ===

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
  OK warm qwen3.5:9b: loaded in 12s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 8s (keep_alive=24h)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): Codex, CodeReviewer, PlatformEngineer, DataMLEngineer, Swift-Development-iOSDeveloper, SwitchTarget, SecurityReviewer, DatabaseSpecialist…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: plan-distinct-deliverables-same-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 63730e25 → collab-63730e25-f50f-49b5-8ec1-d4784fa6078f
  ✓ [1] wait_phase: phase=planning
  wait_discussion: silent agents ['BackendEngineer']; nudging
  nudge: @BackendEngineer — please add your planning perspective for this collab.
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=3)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] assert_plan: plan ok (tasks=3)
=== PASS: plan-distinct-deliverables-same-agent ===

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
  OK warm qwen3.5:9b: loaded in 7s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 9s (keep_alive=24h)
  warming qwen2.5:3b …
  OK warm qwen2.5:3b: loaded in 2s (keep_alive=24h)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): DataMLEngineer, Swift-Development-iOSDeveloper, Gemini, PlatformEngineer, SREObservabilityEngineer, Codex, Copilot, FrontendEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: execute-deliverable ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab b355f58b → collab-b355f58b-46f6-46fa-9a88-cfe3d50bdf4d
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✓ [7] workspace_ack: workspace ack
  ✓ [8] send: /resume-plan b355f58b-46f6-46fa-9a88-cfe3d50bdf4d
  ✓ [9] wait_tasks: tasks completed
  ✓ [10] assert_messages: message assertions ok
  ✓ [11] approve_file_changes: deliverable on disk (findings.md)
  ✗ [12] assert_files: judge:fail:ollama/qwen2.5-coder:14b: The deliverable includes findings about a file (src/App.tsx) that was not requested in the user's task. The task specifically asked for findings about README.md and core/sample/main.go only.
  ✓ cleanup: cancelled and removed workspace artifacts
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: I agree with the plan direction, but I need to refine the task scope based on the workspace context. The project appears to be a Go backend (`core/sample/main.g
    [collaboration_discussion] BackendEngineer: I agree with Task 1 as the primary deliverable. Before writing the findings, I need to inspect the referenced files to extract accurate technical facts:  - Read
    [chat] 13cf62b7-62c7-4c8c-ac83-ef15c38786c5: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/b355f58b-46f6-46fa-9a88-c
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/b355f58b-46f6-46fa-9a88-cfe3d50bdf4d/findings.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/b355f58b-46f6-46fa-9a88-cfe3d50bdf4d/findings.md).  Verification skipped
    [chat] 13cf62b7-62c7-4c8c-ac83-ef15c38786c5: Approved and applied your create change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/b355f58b-46f6-46fa-9a88
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/b355f58b-46f6-46fa-9a88-cfe3d50bdf4d/draft.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/b355f58b-46f6-46fa-9a88-cfe3d50bdf4d/findings.md).  Verification skipped
  --- end ---

=== FAIL: execute-deliverable ===

make[1]: *** [collab-scenario-regression] Error 1

RESULT collab-scenario-regression: FAIL (exit 2, 1263s)
```

