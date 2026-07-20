# Layer gate — collab-full — 2026-07-17-1849 UTC

layer=collab-full
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-all` | FAIL | 4919s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-17-1849.log`

## Failures (tail)

### collab-scenarios-all (exit 2)

```text
se-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: plan-distinct-deliverables-same-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab f6c5cec3 → collab-f6c5cec3-14c2-4610-aff2-cdc90004f750
  ✓ [1] wait_phase: phase=planning
  wait_discussion: silent agents ['BackendEngineer']; nudging
  nudge: @BackendEngineer — please add your planning perspective with concrete `- Task N: @Agent - Write collabs/<id>/file.ext …` lines.
  ✓ [2] wait_discussion: messages total=4 by_agent={'SoftwareArchitect': 2, 'apikey': 1, 'BackendEngineer': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-distinct-deliverables-same-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-distinct-deliverables-same-agent


=== scenario: plan-findings-task-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab fc79417d → collab-fc79417d-b07d-46a3-850b-dec56824c9cc
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=4 by_agent={'BackendEngineer': 2, 'SoftwareArchitect': 1, 'Claude': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-findings-task-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-findings-task-regression


=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab bf3899f3 → collab-bf3899f3-c110-49a4-a96a-5d69880e5466
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after planning-two-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after planning-two-agent


=== scenario: reject-collabs-subfolder ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@Assistant
  started collab 333aeb21 → collab-333aeb21-7719-478a-89f4-aed44455b574
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
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after reject-collabs-subfolder] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after reject-collabs-subfolder


=== scenario: resource-api-schema-planning ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @PlatformEngineer @Claude
  started collab f398d0b0 → collab-f398d0b0-3130-477c-ae17-6883a5efc148
  ✓ [1] wait_phase: phase=planning
  wait_discussion: silent agents ['Claude']; nudging
  nudge: @Claude — please add your planning perspective with concrete `- Task N: @Agent - Write collabs/<id>/file.ext …` lines.
  ✓ [2] wait_discussion: messages total=9 by_agent={'SoftwareArchitect': 2, 'PlatformEngineer': 5, 'apikey': 1, 'Claude': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after resource-api-schema-planning] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after resource-api-schema-planning


=== scenario: resource-api-schema-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @FrontendEngineer @Claude
  started collab 3bec64b0 → collab-3bec64b0-2c65-494d-8fe7-8bf2d2c07dd7
  ✓ [1] wait_phase: phase=planning
  wait_discussion: silent agents ['FrontendEngineer', 'Claude']; nudging
  nudge: @FrontendEngineer — please add your planning perspective with concrete `- Task N: @Agent - Write collabs/<id>/file.ext …` lines.
  nudge: @Claude — please add your planning perspective with concrete `- Task N: @Agent - Write collabs/<id>/file.ext …` lines.
  wait_discussion attempt 1 timed out; retrying
agent discussion: total=13 counts={'BackendEngineer': 3, 'apikey': 2, 'FrontendEngineer': 8} (excluding generation_error)
  ok: @BackendEngineer — 3 message(s)
  ok: @FrontendEngineer — 8 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 20
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=2/10
  nudge: @FrontendEngineer — please add your planning perspective with concrete `- Task N: @Agent - Write collabs/<id>/file.ext …` lines.
  nudge: @Claude — please add your planning perspective with concrete `- Task N: @Agent - Write collabs/<id>/file.ext …` lines.
  ✓ [2] wait_discussion: messages total=19 by_agent={'BackendEngineer': 3, 'apikey': 4, 'FrontendEngineer': 11, 'Claude': 1}; planning ready (after retry 1)
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] assert_collab: collab snapshot ok
  ✓ [7] approve_plan: approve-plan sent
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_deliverable_stubs: 2 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/3bec64b0-2c65-494d-8fe7-8bf2d2c07dd7
  ✓ [10] send: /resume-plan 3bec64b0-2c65-494d-8fe7-8bf2d2c07dd7
  ✓ [11] wait_tasks: executing settle 90.0s statuses=['completed', 'completed']
  ✓ [12] assert_messages: message assertions ok
  ✓ [13] approve_file_changes: file exists (/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/3bec64b0-2c65-494d-8fe7-8bf2d2c07dd7/scope.md)
  ✓ [14] assert_files: judge:pass:SCORE=0.95:ollama/qwen2.5-coder:14b: Reason: The deliverable file "collabs/3bec64b0-2c65-494d-8fe7-8bf2d2c07dd7/scope.md" comprehensively addresses the user's request by defining the scope, key tasks, and deliverables for the "Resource API Schema Standardization" project. It provides a clear objective, detailed key tasks, and specific deliverables, meeting the criteria for completeness and correctness.
=== PASS: resource-api-schema-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after resource-api-schema-regression)...

>>> Hub restart (after resource-api-schema-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after resource-api-schema-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after resource-api-schema-regression


=== scenario: solo-vs-collab-parity ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  solo leg: channel=collab-scenarios-solo output=collabs/parity-solo/findings.md
  solo leg: BackendEngineer replied (1 new)
  ✓ solo leg: /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/parity-solo/findings.md (ok)
  started collab df72f68b → collab-df72f68b-fc72-4da6-ae3c-7f78d3ec0837
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✓ [7] wait_tasks: tasks completed
  ✓ [8] assert_files: judge:pass:SCORE=1.00:ollama/qwen2.5-coder:14b: The deliverable file "collabs/df72f68b-fc72-4da6-ae3c-7f78d3ec0837/findings.md" correctly summarizes the minimal Go fixture repository using the specified files `README.md` and `core/sample/main.go`. It provides substantive findings without including hallucinated paths or unrelated boilerplate.
=== PASS: solo-vs-collab-parity ===

  ✓ cleanup: cancelled and removed workspace artifacts
make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 4919s)
```

