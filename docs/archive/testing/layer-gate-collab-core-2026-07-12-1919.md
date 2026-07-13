# Layer gate — collab-core — 2026-07-12-1919 UTC

layer=collab-core
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-core` | FAIL | 1102s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-12-1919.log`

## Failures (tail)

### collab-scenarios-core (exit 2)

```text
ygiene complete
OK: hub restarted for after collab-participation-three-agent


=== scenario: collab-human-planning-interject ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 7576cd46 → collab-7576cd46-8bba-4274-97e4-b1c0582d140f
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): MobileEngineer, ArenaExpert, Swift-Development-iOSDeveloper, RustExpert, Assistant, Copilot, Cursor, Gemini…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-human-planning-interject] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-human-planning-interject


=== scenario: collab-generation-error-resilience ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab df5c7477 → collab-df5c7477-cf8f-4eff-9674-024ef86bb471
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] send: /resume-plan df5c7477-cf8f-4eff-9674-024ef86bb471
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): RustExpert, Swift-Development-iOSDeveloper, Assistant, Copilot, Cursor, PlatformEngineer, MobileEngineer, ArenaExpert…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-generation-error-resilience] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-generation-error-resilience


=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 5a9c8f9e → collab-5a9c8f9e-3975-485e-9685-882b12b6f533
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): RustExpert, SREObservabilityEngineer, MobileEngineer, Copilot, FrontendEngineer, PlatformEngineer, CodeReviewer, ArenaExpert…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after planning-two-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after planning-two-agent


=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 32e887fc → collab-32e887fc-2448-43d3-9424-4b891e6a9c68
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): PlatformEngineer, CodeReviewer, Swift-Development-iOSDeveloper, SwitchTarget, SecurityReviewer, RustExpert, SREObservabilityEngineer, MobileEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-dependency-prose-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: collab-minimal-completion-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 035446ea → collab-035446ea-6200-4251-b6ac-6d7b841252b4
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] send: /resume-plan 035446ea-6200-4251-b6ac-6d7b841252b4
  ✓ [10] wait_tasks: tasks completed
  ✓ [11] approve_file_changes: deliverable on disk (findings.md)
  ✗ [12] assert_files: judge:fail:ollama/qwen2.5-coder:14b: The deliverable does not provide three bullets citing README.md and core/sample/main.go as requested.
  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after collab-minimal-completion-regression)...

>>> Hub restart (after collab-minimal-completion-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 26 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): MobileEngineer, DataMLEngineer, ArenaExpert, Assistant, Gemini, CodeReviewer, DatabaseSpecialist, Codex…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-minimal-completion-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-minimal-completion-regression


=== scenario: document-findings-execution ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 873ead76 → collab-873ead76-f730-416c-8482-582b921107ed
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/873ead76-f730-416c-8482-582b921107ed
  ✓ [10] send: /resume-plan 873ead76-f730-416c-8482-582b921107ed
  ✓ [11] wait_tasks: tasks completed
  ✓ [12] approve_file_changes: deliverable on disk (findings.md)
  ✓ [13] assert_files: judge:pass:ollama/qwen2.5-coder:14b: The deliverable file substantively answers the user's request by summarizing the README.md and core/sample/main.go files as instructed.
  ✓ [14] send: /complete-collab 873ead76-f730-416c-8482-582b921107ed --forc
  ✓ [15] wait_phase: phase=completed
  ✓ [16] assert_collab: collab snapshot ok
=== PASS: document-findings-execution ===

  ✓ cleanup: cancelled and removed workspace artifacts
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: I agree with the goal to produce a single `findings.md` deliverable summarizing the minimal repo's intent and structure. Here is the proposed execution plan:  -
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect I agree with the plan. The minimal repo serves as a fixture for scenario tests, containing a simple Go "Hello World" sample and a basic HTTP 
    [collaboration_discussion] BackendEngineer: I agree with the current plan direction. The workspace confirms a Go-based backend (`core/sample/`, `server/main.go`) and a minimal React frontend (`src/`). Sin
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect The plan is clear. I will execute Task 1 immediately to generate `collabs/035446ea-6200-4251-b6ac-6d7b841252b4/findings.md`. This file will s
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect I agree with the focus on Go service contracts. Based on the workspace tree, I will now execute Task 1 to generate `collabs/035446ea-6200-425
    [collaboration_discussion] BackendEngineer: 
    [chat] fa7379c3-dc66-4356-b47d-398df4b3aff6: Approved and applied your edit change to `/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/035446ea-6200-4251-b6ac-6
    [answer] BackendEngineer: Implementation session — applied changes (changes to: collabs/035446ea-6200-4251-b6ac-6d7b841252b4/findings.md); verifying workspace…
  --- end ---

=== FAIL: collab-minimal-completion-regression ===

make[1]: *** [collab-scenarios-core] Error 1

RESULT collab-scenarios-core: FAIL (exit 2, 1102s)
```

