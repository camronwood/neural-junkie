# Layer gate — collab-core — 2026-07-12-2131 UTC

layer=collab-core
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-core` | FAIL | 1213s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-12-2131.log`

## Failures (tail)

### collab-scenarios-core (exit 2)

```text
leanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-participation-three-agent


=== scenario: collab-human-planning-interject ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab eb7eea30 → collab-eb7eea30-77e2-4df9-9f45-ab98c7418dbd
  ✓ [1] wait_phase: phase=planning
  ✓ [2] send: Please focus only on README.md and core/sample/main.go — kee
  ✓ [3] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [4] wait_phase: phase=reviewing
  ✓ [5] wait_planning_recap: planning_recap_status=complete
  ✓ [6] assert_plan: plan ok (tasks=2)
  ✓ [7] assert_messages: message assertions ok
  ✓ [8] assert_collab: collab snapshot ok
=== PASS: collab-human-planning-interject ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after collab-human-planning-interject)...

>>> Hub restart (after collab-human-planning-interject)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 26 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): PlatformEngineer, RustExpert, DataMLEngineer, Copilot, CodeReviewer, MobileEngineer, Swift-Development-iOSDeveloper, Assistant…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-human-planning-interject] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-human-planning-interject


=== scenario: collab-generation-error-resilience ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab bb6340bc → collab-bb6340bc-757b-417d-b8cc-d585172bf851
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] send: /resume-plan bb6340bc-757b-417d-b8cc-d585172bf851
  ✓ [9] wait_tasks: executing settle 180.0s statuses=['completed', 'pending']
  ✓ [10] assert_messages: message assertions ok
  ✓ [11] assert_collab: collab snapshot ok
=== PASS: collab-generation-error-resilience ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after collab-generation-error-resilience)...

>>> Hub restart (after collab-generation-error-resilience)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 26 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): Gemini, SwitchTarget, CodeReviewer, DatabaseSpecialist, RustExpert, Assistant, Cursor, PlatformEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-generation-error-resilience] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-generation-error-resilience


=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab effc2075 → collab-effc2075-1eaa-42dd-93f6-045cb46adc7d
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
OK: switched all agents → qwen3.5:9b (Switched 26 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): FrontendEngineer, MobileEngineer, Swift-Development-iOSDeveloper, Codex, SwitchTarget, Assistant, Copilot, Cursor…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after planning-two-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after planning-two-agent


=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 017c6c6f → collab-017c6c6f-d4bf-43d8-9544-c24ffa8107f0
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1, 'Claude': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 26 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): Swift-Development-iOSDeveloper, FrontendEngineer, PlatformEngineer, CodeReviewer, RustExpert, ArenaExpert, Copilot, SecurityReviewer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-dependency-prose-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: collab-minimal-completion-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab e354bd57 → collab-e354bd57-0480-443b-8a24-301e1acc5e2b
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] send: /resume-plan e354bd57-0480-443b-8a24-301e1acc5e2b
  ✓ [10] wait_tasks: tasks completed
  ✓ [11] approve_file_changes: deliverable on disk (findings.md)
  ✗ [12] assert_files: file collabs/e354bd57-0480-443b-8a24-301e1acc5e2b/findings.md min_markdown_bullets 0 < 3
  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after collab-minimal-completion-regression)...

>>> Hub restart (after collab-minimal-completion-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 26 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): DataMLEngineer, Copilot, FrontendEngineer, DatabaseSpecialist, MobileEngineer, Assistant, Codex, Cursor…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-minimal-completion-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-minimal-completion-regression


=== scenario: document-findings-execution ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 5cfa1598 → collab-5cfa1598-4702-4714-9243-db1e38e8c4de
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/5cfa1598-4702-4714-9243-db1e38e8c4de
  ✓ [10] send: /resume-plan 5cfa1598-4702-4714-9243-db1e38e8c4de
  ✓ [11] wait_tasks: tasks completed
  ✓ [12] approve_file_changes: deliverable on disk (findings.md)
  ✓ [13] assert_files: judge:pass:ollama/qwen2.5-coder:14b: The deliverable file substantively answers the user's request by documenting the findings in the specified format and content.
  ✓ [14] send: /complete-collab 5cfa1598-4702-4714-9243-db1e38e8c4de --forc
  ✓ [15] wait_phase: phase=completed
  ✓ [16] assert_collab: collab snapshot ok
=== PASS: document-findings-execution ===

  ✓ cleanup: cancelled and removed workspace artifacts
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: @SoftwareArchitect @BackendEngineer I agree with the goal constraints. Here is the minimal plan:  - Task 1: @BackendEngineer - Write collabs/e354bd57-0480-443b-
    [collaboration_discussion] BackendEngineer: @BackendEngineer - Write `collabs/e354bd57-0480-443b-8a24-301e1acc5e2b/findings.md` with three bullets citing: - `README.md` (lines 1–4): repo purpose as a mini
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect @BackendEngineer The plan needs refinement based on the actual workspace structure provided. The current task points to `core/sample/main.go`
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect I have analyzed the workspace structure and referenced files (`core/sample/README.md`, `core/sample/main.go`). The plan is refined below to e
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect @BackendEngineer The workspace confirms this is a mixed Go/React fixture. `core/sample/main.go` defines the backend service boundary, while `
    [collaboration_discussion] BackendEngineer: 
    [chat] 67a1fade-04d8-4460-81c2-281d1d4d26f9: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/e354bd57-0480-443b-8a24-3
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/e354bd57-0480-443b-8a24-301e1acc5e2b/findings.md); verifying workspace…
  --- end ---

=== FAIL: collab-minimal-completion-regression ===

make[1]: *** [collab-scenarios-core] Error 1

RESULT collab-scenarios-core: FAIL (exit 2, 1213s)
```

