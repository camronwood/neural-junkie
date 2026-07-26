# Layer gate — collab-full — 2026-07-23-0534 UTC

layer=collab-full
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-all` | FAIL | 8567s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-23-0534.log`

## Failures (tail)

### collab-scenarios-all (exit 2)

```text
== PASS: plan-combined-resource-api-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after plan-combined-resource-api-regression)...

>>> Hub restart (after plan-combined-resource-api-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-combined-resource-api-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cancelled 1 active collaboration(s)
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-combined-resource-api-regression


=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 8a830c57 → collab-8a830c57-76ff-43f0-88c6-832f55294b3c
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=8 by_agent={'BackendEngineer': 2, 'SoftwareArchitect': 4, 'apikey': 1, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=3)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_plan: plan ok (tasks=3)
=== PASS: plan-dependency-prose-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after plan-dependency-prose-regression)...

>>> Hub restart (after plan-dependency-prose-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-dependency-prose-regression] fixture collabs + hub channel cleanup
  removed 1 fixture collab dir(s)
  cancelled 1 active collaboration(s)
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: plan-distinct-deliverables-same-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 145cf0cf → collab-145cf0cf-3d32-4fd0-91de-ba4ac0867eaf
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=5 by_agent={'SoftwareArchitect': 3, 'apikey': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=3)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] assert_plan: plan ok (tasks=3)
=== PASS: plan-distinct-deliverables-same-agent ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after plan-distinct-deliverables-same-agent)...

>>> Hub restart (after plan-distinct-deliverables-same-agent)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-distinct-deliverables-same-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cancelled 1 active collaboration(s)
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-distinct-deliverables-same-agent


=== scenario: plan-findings-task-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab ce22e9b6 → collab-ce22e9b6-7a69-4cc3-8c2b-95e4be5b5b4b
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=7 by_agent={'BackendEngineer': 3, 'apikey': 2, 'Claude': 1, 'SoftwareArchitect': 1}; participation ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=4)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_plan: plan ok (tasks=4)
=== PASS: plan-findings-task-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after plan-findings-task-regression)...

>>> Hub restart (after plan-findings-task-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-findings-task-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cancelled 1 active collaboration(s)
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-findings-task-regression


=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 2ae9a615 → collab-2ae9a615-3fe1-4dd6-8242-27a392f33b7c
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_messages: message assertions ok
  ✓ [6] assert_plan: plan ok (tasks=2)
  ✓ [7] assert_collab: collab snapshot ok
=== PASS: planning-two-agent ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after planning-two-agent)...

>>> Hub restart (after planning-two-agent)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after planning-two-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cancelled 1 active collaboration(s)
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after planning-two-agent


=== scenario: reject-collabs-subfolder ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@Assistant
  started collab 91408439 → collab-91408439-bff2-48ed-8f66-0be2cb3383fa
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=0 by_agent={}; participation ready
  ✓ [3] assert_collab: collab snapshot ok
=== PASS: reject-collabs-subfolder ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after reject-collabs-subfolder)...

>>> Hub restart (after reject-collabs-subfolder)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after reject-collabs-subfolder] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cancelled 1 active collaboration(s)
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after reject-collabs-subfolder


=== scenario: resource-api-schema-planning ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @PlatformEngineer @Claude
  started collab 448708e6 → collab-448708e6-1ee5-44da-9b26-d2aed7fdf570
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=5 by_agent={'SoftwareArchitect': 3, 'PlatformEngineer': 1, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_messages: message assertions ok
  ✓ [6] assert_plan: plan ok (tasks=2)
  ✓ [7] assert_collab: collab snapshot ok
=== PASS: resource-api-schema-planning ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after resource-api-schema-planning)...

>>> Hub restart (after resource-api-schema-planning)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after resource-api-schema-planning] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cancelled 1 active collaboration(s)
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after resource-api-schema-planning


=== scenario: resource-api-schema-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @FrontendEngineer @Claude
  started collab 78307204 → collab-78307204-bea7-4120-a98c-7c9002ad5e7f
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=12 by_agent={'BackendEngineer': 3, 'FrontendEngineer': 6, 'apikey': 1, 'Claude': 2}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] assert_collab: collab snapshot ok
  ✓ [7] approve_plan: approve-plan sent
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_deliverable_stubs: 2 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/78307204-bea7-4120-a98c-7c9002ad5e7f
  ✓ [10] send: /resume-plan 78307204-bea7-4120-a98c-7c9002ad5e7f
  ✓ [11] wait_tasks: executing settle 90.0s statuses=['completed', 'in_progress']
  ✓ [12] assert_messages: message assertions ok
  ✓ [13] approve_file_changes: file exists (/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/78307204-bea7-4120-a98c-7c9002ad5e7f/scope.md)
  ✓ [14] assert_files: judge:pass:SCORE=0.95:ollama/qwen2.5-coder:14b: Reason: The deliverable file "collabs/78307204-bea7-4120-a98c-7c9002ad5e7f/scope.md" comprehensively addresses the user's request by defining the scope and deliverables for standardizing resource API document schemas. It includes a clear objective, in-scope and out-of-scope activities, deliverables, and assumptions, making it a substantial and correct response.
=== PASS: resource-api-schema-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after resource-api-schema-regression)...

>>> Hub restart (after resource-api-schema-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after resource-api-schema-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cancelled 1 active collaboration(s)
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after resource-api-schema-regression


=== scenario: solo-vs-collab-parity ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  solo leg: channel=collab-scenarios-solo output=collabs/parity-solo/findings.md
  FAIL [solo]: no BackendEngineer reply (timeout waiting for @BackendEngineer (baseline=0, counts={}))
=== FAIL: solo-vs-collab-parity (solo leg) ===

  ✓ cleanup: cancelled and removed workspace artifacts
make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 8567s)
```

