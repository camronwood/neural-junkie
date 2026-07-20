# test-everything — 2026-07-19-1340 UTC

- Hub: `http://127.0.0.1:18765`
- Full collab sweep (`FULL=1`): `True`
- Skip live: `False`
- Overall: **FAIL** (10/12 stages)

## Stage summary

| Stage | Status | Duration |
|-------|--------|----------|
| `test-all` | OK | 97s |
| `test-conversation-contract` | OK | 10s |
| `test-collab-plan` | OK | 2s |
| `test-scenario-assert` | OK | 1s |
| `collab-smoke` | OK | 1s |
| `learning-lora-smoke` | OK | 2s |
| `collab-preflight` | OK | 12s |
| `implement-scenarios` | OK | 212s |
| `chat-scenarios-regression` | FAIL | 707s |
| `conversation-scenarios-regression` | OK | 1514s |
| `collab-scenario-regression` | OK | 3639s |
| `collab-scenarios-all` | FAIL | 6980s |

## Artifacts

- Full log: `/Users/camronwood/development/projects/neural-junkie/docs/testing/test-everything-2026-07-19-1340.log`
- Hub recovery log: `/Users/camronwood/development/projects/neural-junkie/docs/testing/hub-recovery-2026-07-19-1340.log`

## Failures (tail)

### chat-scenarios-regression (exit 1)

```text
=== scenario: already-said-closure ===
  hub=http://127.0.0.1:18765
  channel=chat-scenarios agent=Assistant
  ✓ [1] send: @Assistant What is 2+2?
  ✓ [2] wait_reply: Assistant replied (1 new)
  ✓ [3] send: @Assistant I know you said that already
  ✓ [4] wait_reply: Assistant replied (1 new)
  ✓ [5] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: already-said-closure ===


=== scenario: cross-repo-ambient-scope ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: @codebase What does ComputeSharedSymbol return in cross-repo-primary vs cross-re…
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: cross-repo-ambient-scope ===


=== scenario: dm-architect-workspace ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-softwarearchitect agent=SoftwareArchitect
  ✓ [1] send: can you see my workspace I have open?
  ✓ [2] wait_reply: SoftwareArchitect replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-architect-workspace ===


=== scenario: dm-assistant-continue-after-closure ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-assistant agent=Assistant
  ✓ [1] send: In one short paragraph: how would you add a light/dark theme toggle in a React s…
  ✓ [2] wait_reply: Assistant replied (1 new)
  ✓ [3] send: ok thanks
  ✓ [4] wait_reply: Assistant replied (1 new)
  ✓ [5] send: One more thing — where should the theme toggle live in the settings UI?
  ✓ [6] wait_reply: Assistant replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-assistant-continue-after-closure ===


=== scenario: dm-backend-codebase-semantic ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: @codebase What does ComputeObscureWidget return?
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-backend-codebase-semantic ===


=== scenario: dm-backend-deep-continuation ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: I want to add light and dark theme support to this project
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: go deeper on the approach — what would you implement first?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✗ [5] assert_messages: any_match not found (want one of ['theme|dark|light|toggle|config|CSS|provider']) (agents: ['BackendEngineer'])
  --- transcript (last messages) ---
    [question] apikey: I want to add light and dark theme support to this project
    [chat] BackendEngineer: Grounding: I loaded 6 file(s) from the workspace context for this answer.  I see the React app already has theme state and class toggling, but it’s missing actual Tailwind dark mode directives in CSS.
    [question] apikey: go deeper on the approach — what would you implement first?
    [chat] BackendEngineer: Given the current project state, I would prioritize  implementing a robust authentication middleware using Go's `golang.org/x/crypto`  for secure password hashing with bcrypt before touching  any busi
  ✓ cleanup: cleared channel history
=== FAIL: dm-backend-deep-continuation ===


>>> flake retry 2/1 for dm-backend-deep-continuation: any_match not found (want one of ['theme|dark|light|toggle|config|CSS|provider']) (agents: ['BackendEngineer'])

=== scenario: dm-backend-deep-continuation ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: I want to add light and dark theme support to this project
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: go deeper on the approach — what would you implement first?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✗ [5] assert_messages: any_match not found (want one of ['theme|dark|light|toggle|config|CSS|provider']) (agents: ['BackendEngineer'])
  --- transcript (last messages) ---
    [question] apikey: I want to add light and dark theme support to this project
    [file_change] BackendEngineer: 📝 Proposing to edit file: src/index.css
    [system_info] System: File change proposal for src/index.css was not registered: preflight rejected "src/index.css": edit target does not exist: "src/index.css"
    [chat] BackendEngineer: Grounding: I loaded 6 file(s) from the workspace context for this answer.  I see you have a React app with basic theme state logic, but the CSS lacks the actual `dark` class definitions needed to styl
    [question] apikey: go deeper on the approach — what would you implement first?
    [chat] BackendEngineer: Given the current state of your project, I  would prioritize implementing a robust authentication layer using  JWTs or sessions immediately, as it secures all  subsequent API endpoints and protects us
  ✓ cleanup: cleared channel history
=== FAIL: dm-backend-deep-continuation ===


=== scenario: dm-backend-echo-followup ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: I want to add theme support to this project
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: What?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✓ [5] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-backend-echo-followup ===


=== scenario: dm-backend-interject-resume ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: What does the main function in the open file do?
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] channel_interject: channel 'dm-chatscenario-backendengineer' held
  ✓ [4] wait_no_reply: no new replies from @BackendEngineer for 8s (baseline=1)
  ✓ [5] send: What package is that file in?
  ✓ [6] wait_reply: BackendEngineer replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ [8] assert_reply_count: reply count since baseline=1
  ✓ cleanup: cleared channel history
=== PASS: dm-backend-interject-resume ===


=== scenario: dm-backend-workspace ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: I want to add theme support to this app I am working on now
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: can you see my workspace I have open?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✓ [5] assert_debug_context: debug context ok
  ✓ [6] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-backend-workspace ===


=== scenario: dm-code-reviewer-workspace ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-codereviewer agent=CodeReviewer
  ✓ [1] send: can you see my workspace I have open?
  ✓ [2] wait_reply: CodeReviewer replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-code-reviewer-workspace ===


=== scenario: dm-database-workspace ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-databasespecialist agent=DatabaseSpecialist
  ✓ [1] send: can you see my workspace I have open?
  ✓ [2] wait_reply: DatabaseSpecialist replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-database-workspace ===


=== scenario: dm-frontend-workspace ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-frontendengineer agent=FrontendEngineer
  ✓ [1] send: can you see my workspace I have open?
  ✓ [2] wait_reply: FrontendEngineer replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-frontend-workspace ===


=== scenario: dm-ide-route-backend ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: What does the main function in the open file do?
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-ide-route-backend ===


=== scenario: dm-platform-workspace ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-platformengineer agent=PlatformEngineer
  ✓ [1] send: can you see my workspace I have open?
  ✓ [2] wait_reply: PlatformEngineer replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-platform-workspace ===


=== scenario: dm-safe-readonly-command ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: Please inspect README.md — if you suggest a shell command, use read-only inspect…
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] assert_suggested_commands: skipped (no matching suggested_commands)
  ✓ cleanup: cleared channel history
=== PASS: dm-safe-readonly-command ===


=== scenario: dm-security-workspace ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-securityreviewer agent=SecurityReviewer
  ✓ [1] send: can you see my workspace I have open?
  ✓ [2] wait_reply: SecurityReviewer replied (1 new)
  ✓ [3] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-security-workspace ===


=== scenario: dm-topic-switch ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: review internal/hub/hub.go for error handling gaps
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: what do you think about go vs rust for backend services?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✓ [5] assert_messages: message assertions ok
  ✓ [6] send: now outline the hook changes you'd make in hub.go for better errors
  ✓ [7] wait_reply: BackendEngineer replied (1 new)
  ✓ [8] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: dm-topic-switch ===


=== scenario: public-backend-theme-workspace ===
  hub=http://127.0.0.1:18765
  channel=chat-scenarios agent=BackendEngineer
  ✓ [1] send: @BackendEngineer I want to add theme support to this app
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] send: @BackendEngineer can you see my workspace?
  ✓ [4] wait_reply: BackendEngineer replied (1 new)
  ✓ [5] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: public-backend-theme-workspace ===


=== scenario: public-frontend-theme-continuation ===
  hub=http://127.0.0.1:18765
  channel=chat-scenarios agent=FrontendEngineer
  ✓ [1] send: @FrontendEngineer I want to add UI themes under settings with light and dark mod…
  ✓ [2] wait_reply: FrontendEngineer replied (1 new)
  ✓ [3] send: @FrontendEngineer yes please go ahead
  ✓ [4] wait_reply: FrontendEngineer replied (1 new)
  ✓ [5] assert_messages: message assertions ok
  ✓ cleanup: cleared channel history
=== PASS: public-frontend-theme-continuation ===


=== scenario: thanks-closure ===
  hub=http://127.0.0.1:18765
  channel=chat-scenarios agent=Assistant
  ✓ [1] send: @Assistant How far is Collinsville IL from St Louis MO?
  ✓ [2] wait_reply: Assistant replied (1 new)
  ✓ [3] send: @Assistant ok thanks
  ✓ [4] wait_reply: Assistant replied (1 new)
  ✓ [5] assert_messages: message assertions ok
  ✓ [6] assert_reply_count: reply count since start=2
  ✓ cleanup: cleared channel history
=== PASS: thanks-closure ===
```

### collab-scenarios-all (exit 1)

```text
gents online
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
  started collab 0803306e → collab-0803306e-a9eb-4517-a7f9-3b3afd33b9c9
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
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-dependency-prose-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: plan-distinct-deliverables-same-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab cb0b5962 → collab-cb0b5962-38d4-4d69-8188-e53e6d4011d3
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
  started collab 45584263 → collab-45584263-2eee-4e05-a4c7-c70f3d917055
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'BackendEngineer': 1, 'SoftwareArchitect': 1, 'Claude': 1}; planning ready
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
  started collab 2afb724a → collab-2afb724a-fdcf-48db-b538-6bcf118776b8
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
  started collab 9e53c13f → collab-9e53c13f-e9dd-45d3-bccf-cfd0a2b22a8d
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
  started collab 12ba1266 → collab-12ba1266-f345-4854-a999-1dad863fa259
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
  started collab 473bbe19 → collab-473bbe19-685b-4323-affc-aeb04d05e16a
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=8 by_agent={'BackendEngineer': 4, 'apikey': 2, 'FrontendEngineer': 1, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] assert_collab: collab snapshot ok
  ✓ [7] approve_plan: approve-plan sent
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_deliverable_stubs: 2 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/473bbe19-685b-4323-affc-aeb04d05e16a
  ✓ [10] send: /resume-plan 473bbe19-685b-4323-affc-aeb04d05e16a
  ✓ [11] wait_tasks: executing settle 90.0s statuses=['in_progress', 'in_progress']
  ✓ [12] assert_messages: message assertions ok
  ✓ [13] approve_file_changes: discussion fallback wrote /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/473bbe19-685b-4323-affc-aeb04d05e16a/scope.md
  ✓ [14] assert_files: judge:pass:SCORE=0.95:ollama/qwen2.5-coder:14b: Reason: The deliverable file "collabs/473bbe19-685b-4323-affc-aeb04d05e16a/scope.md" comprehensively addresses the user's request by defining the scope of the investigation for resource API document schema standardization and registration. It includes a clear objective, in-scope and out-of-scope activities, deliverables, and assumptions, aligning well with the tasks specified.
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
  started collab dba67adc → collab-dba67adc-5c93-4b70-bf49-d6c6e3b3ad39
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✓ [7] wait_tasks: tasks completed
  ✓ [8] assert_files: judge:pass:SCORE=1.00:ollama/qwen2.5-coder:14b: Reason: The findings.md file correctly summarizes the minimal-repo fixture using README.md and core/sample/main.go without any hallucinations or unrelated boilerplate.
=== PASS: solo-vs-collab-parity ===

  ✓ cleanup: cancelled and removed workspace artifacts
```

## Hub recovery log

```text
--- hub recovery 2026-06-23-120000 UTC ---
context: test:stage
attempts: 2
recovered: True
detail: hub healthy after restart attempt 2
--- hub recovery 2026-06-23-120000 UTC ---
context: test:stage
attempts: 2
recovered: True
detail: hub healthy after restart attempt 2
```

