# Layer gate — collab-full — 2026-07-18-0401 UTC

layer=collab-full
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-all` | FAIL | 7792s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-18-0401.log`

## Failures (tail)

### collab-scenarios-all (exit 2)

```text
rt between scenarios (after plan-findings-task-regression)...

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
  started collab 3b258979 → collab-3b258979-c56a-4bcf-88eb-85cc75860b1b
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
  started collab 1c62d1a8 → collab-1c62d1a8-6331-44b5-9d8a-ea043ff9c429
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
  started collab 21382923 → collab-21382923-0f75-4ecb-a0ea-e886b2b61114
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=8 by_agent={'SoftwareArchitect': 2, 'PlatformEngineer': 5, 'Claude': 1}; planning ready
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
  started collab 35016a6d → collab-35016a6d-9d28-4194-8799-cbdf9c649760
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=14 by_agent={'BackendEngineer': 4, 'apikey': 2, 'FrontendEngineer': 7, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] assert_collab: collab snapshot ok
  ✓ [7] approve_plan: approve-plan sent
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_deliverable_stubs: 2 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/35016a6d-9d28-4194-8799-cbdf9c649760
  ✓ [10] send: /resume-plan 35016a6d-9d28-4194-8799-cbdf9c649760
  ✓ [11] wait_tasks: executing settle 90.0s statuses=['in_progress', 'in_progress']
  ✓ [12] assert_messages: message assertions ok
  ✓ [13] approve_file_changes: discussion fallback wrote /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/35016a6d-9d28-4194-8799-cbdf9c649760/scope.md
  ✓ [14] assert_files: judge:pass:SCORE=0.95:ollama/qwen2.5-coder:14b: Reason: The deliverable file "collabs/35016a6d-9d28-4194-8799-cbdf9c649760/scope.md" comprehensively addresses the user's request by defining the scope and objectives for standardizing the resource API document schema. It includes a detailed in-scope analysis, out-of-scope items, and deliverables, aligning well with the tasks specified.
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
  started collab 15f0a1c6 → collab-15f0a1c6-1209-4648-bbf9-64b430ce8406
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✗ [7] wait_tasks: task wait timeout statuses=['in_progress']

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: Looking at the file tree and the collaboration goal:  The project contains Go code (`core/sample/main.go`, `core/server/main.go`) alongside a React frontend (`s
    [collaboration_discussion] BackendEngineer: Task executed. Here is the content for `collabs/15f0a1c6-1209-4648-bbf9-64b430ce8406/findings.md`:  ```markdown # Architectural Findings: Minimal Fixture Repo  
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect @BackendEngineer - Let's refine the plan to focus on concrete architectural observations from `core/sample/main.go` and `core/sample/README.m
    [collaboration_discussion] BackendEngineer: # Session Recap: Fixture Repo Analysis  ## Summary The team analyzed the provided repository, which serves as a minimal fixture for collaboration scenario tests
    [collaboration_discussion] BackendEngineer: [FILE_CHANGE] path: findings.md content: # Findings\n\n- **Core Backend**:  `core/sample/main.go` exposes a standalone `HelloWorld` function but lacks  explicit
    [collaboration_discussion] BackendEngineer: [FILE_CHANGE] **path**: src/App.tsx ```tsx import React from 'react';  const App: React.FC = () => { const  handleHelloWorld = async () => { try {  // Call Go b
    [collaboration_discussion] BackendEngineer: [FILE_CHANGE] --- a/core/sample/main.go +++ b/core/sample/main.go @@ -1,13 +1,13  @@ package main import "fmt" -// HelloWorld prints  "Hello, World!" to the con
    [collaboration_discussion] BackendEngineer: [FILE_CHANGE] --- a/core/sample/main.go +++ b/core/sample/main.go @@ -1,8 +1,10  @@ package main import "fmt" +// Version constant  for the sample application. 
  --- end ---

=== FAIL: solo-vs-collab-parity ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> flake retry 2/1 for solo-vs-collab-parity: task wait timeout statuses=['in_progress']

=== scenario: solo-vs-collab-parity ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  solo leg: channel=collab-scenarios-solo output=collabs/parity-solo/findings.md
  ✓ solo leg: /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/parity-solo/findings.md (ok)
  started collab 2f34b82c → collab-2f34b82c-d103-464a-b1a2-6bfbc2c767f0
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✗ [7] wait_tasks: task wait timeout statuses=['in_progress']

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: I agree with the core plan direction but need to refine Task 1's scope and dependencies based on the actual workspace structure. The current `core/sample/main.g
    [collaboration_discussion] BackendEngineer: 
    [collaboration_discussion] BackendEngineer: I agree with the plan direction. Before finalizing the task list, I need to inspect `core/sample/main.go` and `README.md` to ground the three findings in actual
    [collaboration_discussion] BackendEngineer: Task 1: @BackendEngineer - Write collabs/2f34b82c-d103-464a-b1a2-6bfbc2c767f0/findings.md with three bullet findings about this repo (README.md and core/sample/
    [collaboration_discussion] BackendEngineer: I have analyzed the source files `README.md` and  `core/sample/main.go` to generate the required findings. The repository  serves as a minimal fixture for scena
    [collaboration_discussion] BackendEngineer: Based on the source files `README.md` and `core/sample/main.go`,  here are the substantive findings: - The repository  is configured as a minimal fixture for co
    [collaboration_discussion] BackendEngineer: I have analyzed the referenced source files (`README.md`  and `core/sample/main.go`) within the project workspace. - The  `README.md` at line 3 explicitly defin
    [collaboration_discussion] BackendEngineer: I have analyzed the referenced source files: `README.md`  (lines 1-4), `core/sample/main.go` (lines 1-13), and `collabs/2f34b82c-d103-464a-b1a2-6bfbc2c767f0/fin
  --- end ---

=== FAIL: solo-vs-collab-parity ===

  ✓ cleanup: cancelled and removed workspace artifacts
make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 7792s)
```

