# Layer gate — collab-core — 2026-07-17-0107 UTC

layer=collab-core
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-core` | FAIL | 1247s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-17-0107.log`

## Failures (tail)

### collab-scenarios-core (exit 2)

```text
t, PlatformEngineer, DataMLEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-participation-three-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-participation-three-agent


=== scenario: collab-human-planning-interject ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab adf8bbeb → collab-adf8bbeb-9b3f-4796-8d5b-c7daaa36f439
  ✓ [1] wait_phase: phase=planning
  ✓ [2] send: Please focus only on README.md and core/sample/main.go — kee
  ✓ [3] wait_discussion: messages total=6 by_agent={'SoftwareArchitect': 4, 'apikey': 1, 'BackendEngineer': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): RustExpert, MobileEngineer, DataMLEngineer, FrontendEngineer, SecurityReviewer, SREObservabilityEngineer, Swift-Development-iOSDeveloper, Copilot…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-human-planning-interject] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-human-planning-interject


=== scenario: collab-generation-error-resilience ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 01b9e8e4 → collab-01b9e8e4-bb55-4b15-94d3-ce6c458c8830
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'BackendEngineer': 2, 'SoftwareArchitect': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] send: /resume-plan 01b9e8e4-bb55-4b15-94d3-ce6c458c8830
  ✓ [9] wait_tasks: executing settle 180.0s statuses=['completed', 'completed']
  ✓ [10] assert_messages: message assertions ok
  ✓ [11] assert_collab: collab snapshot ok
=== PASS: collab-generation-error-resilience ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after collab-generation-error-resilience)...

>>> Hub restart (after collab-generation-error-resilience)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): FrontendEngineer, MobileEngineer, Swift-Development-iOSDeveloper, Copilot, SwitchTarget, RustExpert, DatabaseSpecialist, Assistant…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-generation-error-resilience] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-generation-error-resilience


=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab bd94d2da → collab-bd94d2da-88f5-40cc-8f8c-a50840c59ce8
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_messages: message assertions ok
  ✗ [6] assert_plan: tasks=1 plan_task_lines≈1 want >=2

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: Here is the minimal architectural plan for the CLI file encryption tool:  - Task 1: @SoftwareArchitect - Write collabs/cli-tool/encryption-spec.md defining the 
    [collaboration_discussion] SoftwareArchitect: @BackendEngineer - Design the CLI entry point and core encryption service interface (`src/cli/main.py`, `src/core/encryption_service.py`).     - depends: 1 (sch
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect, your `encryption-spec.md` is critical as it defines the immutable contract for our CLI. I suggest adding a section on **key storage semantic
    [collaboration_discussion] BackendEngineer: I agree with the current plan. It correctly separates high-level interface standards from internal service contracts, preventing overlap between @SoftwareArchit
  --- end ---

=== FAIL: planning-two-agent ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after planning-two-agent)...

>>> Hub restart (after planning-two-agent)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): Swift-Development-iOSDeveloper, Codex, Copilot, DatabaseSpecialist, FrontendEngineer, CodeReviewer, RustExpert, SREObservabilityEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after planning-two-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after planning-two-agent


=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 67310a66 → collab-67310a66-4643-4320-8079-5a12f6f2e212
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=12 by_agent={'BackendEngineer': 3, 'SoftwareArchitect': 7, 'apikey': 1, 'Claude': 1}; planning ready
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
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): CodeReviewer, SREObservabilityEngineer, Copilot, Cursor, FrontendEngineer, Swift-Development-iOSDeveloper, Assistant, PlatformEngineer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-dependency-prose-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: collab-minimal-completion-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 035e91b0 → collab-035e91b0-0bd5-4b77-9135-253fafc31f15
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'BackendEngineer': 2, 'SoftwareArchitect': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/findings-scope-repo/collabs/035e91b0-0bd5-4b77-9135-253fafc31f15
  ✓ [10] send: /resume-plan 035e91b0-0bd5-4b77-9135-253fafc31f15
  ✓ [11] wait_tasks: tasks completed
  ✓ [12] approve_file_changes: deliverable on disk (findings.md)
  ✓ [13] assert_files: judge:warn:SCORE=0.00:ollama/qwen2.5-coder:14b: Reason: The deliverable does not provide three substantive bullets grounded only in the README.md and core/sample/main.go files as requested.
  ✓ [14] send: /complete-collab 035e91b0-0bd5-4b77-9135-253fafc31f15 --forc
  ✓ [15] wait_phase: phase=completed
  ✓ [16] assert_collab: collab snapshot ok
=== PASS: collab-minimal-completion-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after collab-minimal-completion-regression)...

>>> Hub restart (after collab-minimal-completion-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (slim roster / cloud Claude)...
OK: Claude → ollama (qwen3.5:9b); core agents already unpaused; paused 16 agent(s): Copilot, Cursor, SwitchTarget, CodeReviewer, RustExpert, DataMLEngineer, Assistant, SecurityReviewer…
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-minimal-completion-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-minimal-completion-regression


=== scenario: document-findings-execution ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 24ba1db4 → collab-24ba1db4-7136-46eb-b991-a993a0c5c420
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] workspace_ack: workspace ack
  ✓ [9] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/findings-scope-repo/collabs/24ba1db4-7136-46eb-b991-a993a0c5c420
  ✓ [10] send: /resume-plan 24ba1db4-7136-46eb-b991-a993a0c5c420
  ✓ [11] wait_tasks: tasks completed
  ✓ [12] approve_file_changes: deliverable on disk (findings.md)
  ✓ [13] assert_files: judge:pass:SCORE=1.00:ollama/qwen2.5-coder:14b: The deliverable file correctly summarizes the content of README.md and core/sample/main.go as requested.
  ✓ [14] send: /complete-collab 24ba1db4-7136-46eb-b991-a993a0c5c420 --forc
  ✓ [15] wait_phase: phase=completed
  ✓ [16] assert_collab: collab snapshot ok
=== PASS: document-findings-execution ===

  ✓ cleanup: cancelled and removed workspace artifacts
make[1]: *** [collab-scenarios-core] Error 1

RESULT collab-scenarios-core: FAIL (exit 2, 1247s)
```

