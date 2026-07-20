# Layer gate — collab-full — 2026-07-18-0623 UTC

layer=collab-full
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-all` | FAIL | 6050s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-18-0623.log`

## Failures (tail)

### collab-scenarios-all (exit 2)

```text
ting
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
  started collab e74518b1 → collab-e74518b1-67e4-4e46-9478-e1484d119ee4
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
  started collab 89d9c17e → collab-89d9c17e-3c69-4d06-9d4d-a104081cdf08
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
  started collab cfb3bbfb → collab-cfb3bbfb-1913-4401-9049-df8cb20b31c2
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=4 by_agent={'SoftwareArchitect': 2, 'PlatformEngineer': 1, 'Claude': 1}; planning ready
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
  started collab 176f4840 → collab-176f4840-63a8-4ba3-8865-28616974ebd0
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=8 by_agent={'BackendEngineer': 4, 'apikey': 2, 'FrontendEngineer': 1, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] assert_collab: collab snapshot ok
  ✓ [7] approve_plan: approve-plan sent
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_deliverable_stubs: 2 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/176f4840-63a8-4ba3-8865-28616974ebd0
  ✓ [10] send: /resume-plan 176f4840-63a8-4ba3-8865-28616974ebd0
  ✓ [11] wait_tasks: executing settle 90.0s statuses=['completed', 'completed']
  ✓ [12] assert_messages: message assertions ok
  ✓ [13] approve_file_changes: file exists (/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/176f4840-63a8-4ba3-8865-28616974ebd0/scope.md)
  ✓ [14] assert_files: judge:pass:SCORE=0.95:ollama/qwen2.5-coder:14b: Reason: The deliverable file "scope.md" comprehensively addresses the user's request by defining the scope, in-scope activities, out-of-scope items, and deliverables for investigating resource API document schema standardization. It also includes success criteria, making it a well-structured and detailed document.
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
  started collab 2035158c → collab-2035158c-6ee9-47d0-b7c6-a67fef7fd67d
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✗ [7] wait_tasks: task wait timeout statuses=['in_progress']

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: I agree with the current plan direction. Before finalizing Task 1's deliverable, I need to inspect `core/sample/main.go` and `README.md` to identify:   1. API s
    [collaboration_discussion] BackendEngineer: 
    [collaboration_discussion] BackendEngineer: I agree with the plan to generate `findings.md`. Before writing it, I need to inspect the actual code and docs to ensure the findings are grounded in reality ra
    [collaboration_discussion] BackendEngineer: The current workspace shows a clear separation between the legacy dark-mode components and the new light-theme implementation, which complicates the existing ta
    [collaboration_discussion] BackendEngineer: I have read the source files and will  now ship the canonical `[FILE_CHANGE]` block for `collabs/2035158c-6ee9-47d0-b7c6-a67fef7fd67d/findings.md`.  ``` [FILE_C
    [collaboration_discussion] BackendEngineer: I've read the source files: `README.md` (fixture repo  description), `core/sample/main.go` (Go hello world), and `core/sample/README.md` (sample  docs). I'll wr
    [collaboration_discussion] BackendEngineer: Based on the source files provided: - **README.md**  defines this as a minimal fixture repo for  collab scenario tests, specifically used by `execute-deliverabl
    [collaboration_discussion] BackendEngineer: I will read the source files (`README.md` and  `core/sample/main.go`) and generate substantive findings in the deliverable  file. [FILE_CHANGE] operation: creat
  --- end ---

=== FAIL: solo-vs-collab-parity ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> flake retry 2/1 for solo-vs-collab-parity: task wait timeout statuses=['in_progress']

=== scenario: solo-vs-collab-parity ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  solo leg: channel=collab-scenarios-solo output=collabs/parity-solo/findings.md
  ✓ solo leg: /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/parity-solo/findings.md (ok)
  started collab 1cb2e25e → collab-1cb2e25e-15f3-4b12-a712-2236cf4af093
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✗ [7] wait_tasks: task wait timeout statuses=['in_progress']

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: Task 1: @BackendEngineer - Write collabs/1cb2e25e-15f3-4b12-a712-2236cf4af093/findings.md summarizing repository purpose and structural observations   - depends
    [collaboration_discussion] BackendEngineer: I agree with the task list direction. Before writing the findings, I need to analyze `core/sample/main.go` and `README.md` to extract concrete backend observati
    [collaboration_discussion] BackendEngineer: 
    [collaboration_discussion] BackendEngineer: Task List: - Task 1: @BackendEngineer - Write collabs/1cb2e25e-15f3-4b12-a712-2236cf4af093/findings.md summarizing repository purpose and structural observation
    [collaboration_discussion] BackendEngineer: @camronwood I see the workspace contains `README.md`, `core/sample/main.go`,  and an existing stub at `collabs/1cb2e25e-15f3-4b12-a712-2236cf4af093/findings.md`
    [collaboration_discussion] BackendEngineer: I have analyzed the referenced files `README.md` and  `core/sample/main.go`. The repository is a minimal Go fixture  where the `main.go` file defines a `HelloWo
    [collaboration_discussion] BackendEngineer: [FILE_CHANGE] operation: create path: collabs/1cb2e25e-15f3-4b12-a712-2236cf4af093/file.md content: "// Package  main is a minimal sample for collaboration scen
    [collaboration_discussion] BackendEngineer: I will update `collabs/1cb2e25e-15f3-4b12-a712-2236cf4af093/findings.md` with substantive findings grounded  in the referenced source files. [FILE_CHANGE] opera
  --- end ---

=== FAIL: solo-vs-collab-parity ===

  ✓ cleanup: cancelled and removed workspace artifacts
make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 6050s)
```

