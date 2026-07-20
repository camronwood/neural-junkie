# Layer gate — collab-full — 2026-07-18-0209 UTC

layer=collab-full
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-all` | FAIL | 5578s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-18-0209.log`

## Failures (tail)

### collab-scenarios-all (exit 2)

```text
ombined-resource-api-regression)...

>>> Hub restart (after plan-combined-resource-api-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-combined-resource-api-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-combined-resource-api-regression


=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 71d5a18c → collab-71d5a18c-af34-4167-a059-1a181e96f0d7
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=11 by_agent={'BackendEngineer': 3, 'SoftwareArchitect': 6, 'apikey': 1, 'Claude': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-dependency-prose-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: plan-distinct-deliverables-same-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab cb7cddc9 → collab-cb7cddc9-432a-4f43-a7d9-4bfa565d8acf
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
  started collab cbbe311c → collab-cbbe311c-6e9b-4587-b57e-cc1f4b42c9ec
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
  started collab 9945eeb4 → collab-9945eeb4-0566-4fa5-8cff-33bf5440a003
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
  started collab 1b53354e → collab-1b53354e-7e2f-4d46-a04d-5e17a2558c20
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
  started collab e40f240f → collab-e40f240f-009c-46a4-afe6-465e51b5838e
  ✓ [1] wait_phase: phase=planning
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
  started collab e0ca241a → collab-e0ca241a-59f7-45d5-aca1-ddad7c1bb7e3
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=7 by_agent={'BackendEngineer': 4, 'FrontendEngineer': 1, 'apikey': 1, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] assert_collab: collab snapshot ok
  ✓ [7] approve_plan: approve-plan sent
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_deliverable_stubs: 2 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/e0ca241a-59f7-45d5-aca1-ddad7c1bb7e3
  ✓ [10] send: /resume-plan e0ca241a-59f7-45d5-aca1-ddad7c1bb7e3
  ✓ [11] wait_tasks: executing settle 90.0s statuses=['in_progress', 'in_progress']
  ✓ [12] assert_messages: message assertions ok
  ✓ [13] approve_file_changes: discussion fallback wrote /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/e0ca241a-59f7-45d5-aca1-ddad7c1bb7e3/scope.md
  ✓ [14] assert_files: judge:pass:SCORE=0.95:ollama/qwen2.5-coder:14b: Reason: The deliverable file contains two tasks as requested, with clear objectives, tasks, and deliverables. It also includes real markdown deliverables with content relevant to the user's request.
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
  ✓ solo leg: /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/parity-solo/findings.md (ok)
  started collab 4588d670 → collab-4588d670-1d3e-41e7-bfef-601e86d343fc
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✓ [7] wait_tasks: tasks completed
  ✓ [8] assert_files: judge:warn:SCORE=0.00:ollama/qwen2.5-coder:14b: Reason: The deliverable does not contain three bullet findings as requested.
=== PASS: solo-vs-collab-parity ===

  ✓ cleanup: cancelled and removed workspace artifacts
make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 5578s)
```

