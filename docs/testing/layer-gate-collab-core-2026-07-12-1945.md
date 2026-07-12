# Layer gate — collab-core — 2026-07-12-1945 UTC

layer=collab-core
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-core` | FAIL | 1352s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-12-1945.log`

## Failures (tail)

### collab-scenarios-core (exit 2)

```text
 clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-participation-three-agent


=== scenario: collab-human-planning-interject ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 9fd1f8c0 → collab-9fd1f8c0-1bcf-4a2a-9413-4e558148bc11
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): SREObservabilityEngineer, Codex, Copilot, Cursor, CodeReviewer, RustExpert, MobileEngineer, DataMLEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-human-planning-interject] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-human-planning-interject


=== scenario: collab-generation-error-resilience ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 19de16b0 → collab-19de16b0-67f4-4965-b828-ba2484370f2a
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] send: /resume-plan 19de16b0-67f4-4965-b828-ba2484370f2a
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): SwitchTarget, PlatformEngineer, SecurityReviewer, SREObservabilityEngineer, MobileEngineer, Copilot, Cursor, FrontendEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-generation-error-resilience] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-generation-error-resilience


=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 401681ab → collab-401681ab-255f-4a4a-9b22-91a6628b6afc
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_messages: message assertions ok
  ✓ [6] assert_plan: plan ok (tasks=0)
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): DatabaseSpecialist, Assistant, Cursor, SecurityReviewer, SREObservabilityEngineer, Swift-Development-iOSDeveloper, Copilot, Gemini…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after planning-two-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after planning-two-agent


=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 31e44f06 → collab-31e44f06-00c8-4cd4-8b36-425e08512ec1
  ✓ [1] wait_phase: phase=planning
  wait_discussion: silent agents ['Claude']; nudging
  nudge: @Claude — please add your planning perspective for this collab.
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
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): Codex, PlatformEngineer, RustExpert, MobileEngineer, ArenaExpert, Swift-Development-iOSDeveloper, DatabaseSpecialist, CodeReviewer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-dependency-prose-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: collab-minimal-completion-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 5976f506 → collab-5976f506-7322-42f0-8136-fc28470cc972
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] send: /resume-plan 5976f506-7322-42f0-8136-fc28470cc972
  ✗ [10] wait_tasks: task wait timeout statuses=['in_progress']
  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after collab-minimal-completion-regression)...

>>> Hub restart (after collab-minimal-completion-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 26 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 17 agent(s): Codex, Cursor, FrontendEngineer, SREObservabilityEngineer, Copilot, PlatformEngineer, DataMLEngineer, Gemini…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-minimal-completion-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-minimal-completion-regression


=== scenario: document-findings-execution ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 9187d673 → collab-9187d673-2678-46e7-8fb2-8c0937f96e73
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/9187d673-2678-46e7-8fb2-8c0937f96e73
  ✓ [10] send: /resume-plan 9187d673-2678-46e7-8fb2-8c0937f96e73
  ✓ [11] wait_tasks: tasks completed
  ✓ [12] approve_file_changes: deliverable on disk (findings.md)
  ✓ [13] assert_files: judge:pass:ollama/qwen2.5-coder:14b: The deliverable file correctly documents the findings as requested, summarizing both README.md and core/sample/main.go.
  ✓ [14] send: /complete-collab 9187d673-2678-46e7-8fb2-8c0937f96e73 --forc
  ✓ [15] wait_phase: phase=completed
  ✓ [16] assert_collab: collab snapshot ok
=== PASS: document-findings-execution ===

  ✓ cleanup: cancelled and removed workspace artifacts
  WARN: not all warm models resident simultaneously (smoke passed; cold: qwen2.5-coder:14b)

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect The plan is solid. I will execute **Task 1** immediately.  I am writing `collabs/5976f506-7322-42f0-8136-fc28470cc972/findings.md`. The file 
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect The initial task to create `findings.md` is complete. Since we are now moving into refinement or a new phase, I have no specific blocker or n
    [collaboration_discussion] BackendEngineer: 
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: core/sample/main.go).  Verification skipped (interactive trust — approve proposa
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: core/sample/main.go).  Verification skipped (interactive trust — approve proposa
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: core/sample/main.go).  Verification skipped (interactive trust — approve proposa
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: core/sample/main.go).  Verification skipped (interactive trust — approve proposa
    [collaboration_discussion] BackendEngineer: Implementation session complete — proposals submitted for approval (changes to: core/sample/main.go).  Verification skipped (interactive trust — approve proposa
  --- end ---

=== FAIL: collab-minimal-completion-regression ===

make[1]: *** [collab-scenarios-core] Error 1

RESULT collab-scenarios-core: FAIL (exit 2, 1352s)
```

