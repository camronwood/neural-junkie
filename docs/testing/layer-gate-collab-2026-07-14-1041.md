# Layer gate — collab — 2026-07-14-1041 UTC

layer=collab
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenario-regression` | FAIL | 2753s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-2026-07-14-1041.log`

## Failures (tail)

### collab-scenario-regression (exit 2)

```text
Ollama (http://127.0.0.1:11434/api/tags)
>>> Warming Ollama models (suite=release)...
>>> Ollama model readiness
  warm: qwen3.5:9b, qwen2.5-coder:14b, qwen2.5:3b
  pull roster: qwen3.5:9b, qwen2.5-coder:14b, qwen2.5:3b, gemma3:12b
  installed: qwen3.5:9b
  installed: qwen2.5-coder:14b
  installed: qwen2.5:3b
  installed: gemma3:12b
  warming qwen3.5:9b …
  OK warm qwen3.5:9b: loaded in 9s (keep_alive=24h)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 13 agent(s): DatabaseSpecialist, MobileEngineer, Swift-Development-iOSDeveloper, SwitchTarget, RustExpert, SREObservabilityEngineer, Codex, Gemini…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: execution-no-stack-commands ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @PlatformEngineer
  started collab 398fdc9c → collab-398fdc9c-56c7-4c67-8192-44ca271a41c3
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'BackendEngineer': 1, 'PlatformEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] send: /resume-plan 398fdc9c-56c7-4c67-8192-44ca271a41c3
  ✓ [9] wait_tasks: executing settle 120.0s statuses=['completed']
  ✓ [10] assert_messages: message assertions ok
  ✓ [11] approve_file_changes: deliverable on disk (findings.md)
  ✓ [12] assert_files: judge:pass:ollama/qwen2.5-coder:14b: The deliverable file substantively answers the user's request by providing three bullets grounded in the README.md and core/sample/main.go files, as requested.
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
  OK warm qwen3.5:9b: loaded in 12s (keep_alive=24h)
  warming qwen2.5-coder:14b …
  OK warm qwen2.5-coder:14b: loaded in 0s (keep_alive=24h)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 13 agent(s): DatabaseSpecialist, SREObservabilityEngineer, Cursor, DataMLEngineer, Swift-Development-iOSDeveloper, Copilot, Gemini, CodeReviewer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: collab-conversation-quality-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 486cf6bd → collab-486cf6bd-8e74-4779-bcf2-3c7027164375
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] assert_messages: message assertions ok
  ✓ [9] send: /resume-plan 486cf6bd-8e74-4779-bcf2-3c7027164375
  ✓ [10] wait_tasks: tasks completed
  ✓ [11] approve_file_changes: deliverable on disk (findings.md)
  ✓ [12] assert_files: judge:pass:ollama/qwen2.5-coder:14b: The deliverable file "collabs/486cf6bd-8e74-4779-bcf2-3c7027164375/findings.md" correctly addresses the user's request by providing three bullets about README.md and core/sample/main.go only, without citing any out-of-scope paths.
  ✓ [13] assert_messages: message assertions ok
=== PASS: collab-conversation-quality-regression ===

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
  OK warm qwen2.5-coder:14b: loaded in 0s (keep_alive=24h)
  warming qwen2.5:3b …
  OK warm qwen2.5:3b: loaded in 3s (keep_alive=24h)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 13 agent(s): SwitchTarget, MobileEngineer, Assistant, Gemini, CodeReviewer, DatabaseSpecialist, RustExpert, SREObservabilityEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [collab-scenarios] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
=== Regression boot complete (collab-scenarios) ===


=== scenario: collab-no-edit-after-cancel ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: The current task is well-scoped and aligns with your lane (architectural summary of scope). However, I notice the deliverable path `collabs/99fec936-fa22-4a55-9
    [chat] 70925295-8459-4c86-b0f3-6a2350359d24: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/findings-scope-repo/collabs/99fec936-fa22-4a55
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/99fec936-fa22-4a55-900b-561a61018cec/findings.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/99fec936-fa22-4a55-900b-561a61018cec/findings.md).  Verification skipped
    [chat] 70925295-8459-4c86-b0f3-6a2350359d24: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/findings-scope-repo/collabs/99fec936-fa22-4a55
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/99fec936-fa22-4a55-900b-561a61018cec/findings.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/99fec936-fa22-4a55-900b-561a61018cec/findings.md).  Verification skipped
    [collaboration_discussion] SoftwareArchitect: 
  --- end ---

=== FAIL: collab-minimal-completion-regression ===

  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: I agree with the task list direction. Let's refine Task 1 to ensure it captures the architectural implications directly from the source files before moving to s
    [chat] b3898b16-f5f1-4232-8677-79a94ad1243d: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/findings-scope-repo/collabs/33fececc-e98b-4425
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/33fececc-e98b-4425-b25b-3db3009fa0fe/findings.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/33fececc-e98b-4425-b25b-3db3009fa0fe/findings.md).  Verification skipped
    [chat] b3898b16-f5f1-4232-8677-79a94ad1243d: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/findings-scope-repo/collabs/33fececc-e98b-4425
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/33fececc-e98b-4425-b25b-3db3009fa0fe/findings.md); verifying workspace…
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: collabs/33fececc-e98b-4425-b25b-3db3009fa0fe/findings.md).  Verification skipped
    [collaboration_discussion] SoftwareArchitect: 
  --- end ---

=== FAIL: execute-deliverable ===

  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)
  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description
make[1]: *** [collab-scenario-regression] Error 1

RESULT collab-scenario-regression: FAIL (exit 2, 2753s)
```

