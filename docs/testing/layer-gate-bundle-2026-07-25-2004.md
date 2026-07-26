# Layer gate — bundle — 2026-07-25-2004 UTC

layer=bundle
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `regression-bundle` | FAIL | 5078s | 1 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-bundle-2026-07-25-2004.log`

## Failures (tail)

### regression-bundle (exit 1)

```text
h ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-topic-switch","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen2.5-coder:14b","actual_model":"qwen2.5-coder:14b","validation_failures":["quality_gate_failure"],"escalation_reasons":["quality_gate_failure"],"repair_attempts":3,"tool_calls":null,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":1,"wall_duration_ms":262277.063}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen2.5-coder:14b","actual_model":"qwen2.5-coder:14b","validation_failures":["quality_gate_failure"],"escalation_count":1,"escalation_reasons":["quality_gate_failure"],"repair_attempts":3,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":262277.063}

>>> python3 scripts/chat-scenarios.py --scenario dm-assistant-continue-after-closure

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

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-assistant-continue-after-closure","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":25233.748}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":25233.748}

>>> python3 scripts/chat-scenarios.py --scenario dm-backend-interject-resume

=== scenario: dm-backend-interject-resume ===
  hub=http://127.0.0.1:18765
  channel=dm-chatscenario-backendengineer agent=BackendEngineer
  ✓ [1] send: What does the main function in @file:core/sample/main.go do?
  ✓ [2] wait_reply: BackendEngineer replied (1 new)
  ✓ [3] channel_interject: channel 'dm-chatscenario-backendengineer' held
  ✓ [4] wait_no_reply: no new replies from @BackendEngineer for 8s (baseline=1)
  ✓ [5] send: What Go package declaration is at the top of that file?
  ✓ [6] wait_reply: BackendEngineer replied (1 new)
  ✓ [7] assert_messages: message assertions ok
  ✓ [8] assert_reply_count: reply count since baseline=1
  ✓ cleanup: cleared channel history
=== PASS: dm-backend-interject-resume ===

EVAL_JSON:{"schema_version":1,"kind":"chat","scenario":"dm-backend-interject-resume","attempts":1,"passed_at_1":true,"eventual_pass":true,"retry_reasons":[],"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"retry_count":0,"nudge_count":0,"escalation_count":0,"wall_duration_ms":43834.916}
METRICS_JSON:{"passed_at_1":true,"eventual_pass":true,"attempts":1,"retry_count":0,"retry_reasons":[],"nudge_count":0,"nudge_reasons":[],"actual_provider":"qwen3.5:9b","actual_model":"qwen3.5:9b","validation_failures":[],"escalation_count":0,"escalation_reasons":[],"repair_attempts":null,"tool_calls":null,"ttft_ms":null,"wall_duration_ms":43834.916}

=== Collab conversation scenarios ===

>>> python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression
SKIP: regression boot (SKIP_BOOT/NJ_BOOT_DONE set)

=== scenario: collab-conversation-quality-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 749763d6 → collab-749763d6-54d3-4ef5-8791-6c7e2ebad2af
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] assert_messages: message assertions ok
  ✓ [9] send: /resume-plan 749763d6-54d3-4ef5-8791-6c7e2ebad2af
  ✓ [10] wait_tasks: tasks completed
  ✓ [11] approve_file_changes: deliverable on disk (findings.md)
  ✓ [12] assert_files: judge:pass:SCORE=1.00:ollama/qwen2.5-coder:14b: The deliverable file correctly addresses the task by providing three relevant bullets about README.md and core/sample/main.go without including any out-of-scope paths.
  ✓ [13] assert_messages: message assertions ok
=== PASS: collab-conversation-quality-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> python3 scripts/collab-scenarios.py --scenario collab-no-edit-after-cancel
SKIP: regression boot (SKIP_BOOT/NJ_BOOT_DONE set)

=== scenario: collab-no-edit-after-cancel ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab c72e3fcf → collab-c72e3fcf-f454-4e63-9136-deb0be5e4c8d
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=5 by_agent={'BackendEngineer': 5}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=1)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] send: /cancel-plan c72e3fcf-f454-4e63-9136-deb0be5e4c8d
  ✓ [9] wait_phase: phase=cancelled
  ✓ [10] assert_collab: collab snapshot ok
  ✓ [11] assert_messages: message assertions ok
=== PASS: collab-no-edit-after-cancel ===


>>> python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience
SKIP: regression boot (SKIP_BOOT/NJ_BOOT_DONE set)

=== scenario: collab-generation-error-resilience ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab eeaa727f → collab-eeaa727f-951b-442c-b5db-56e9b7f697c2
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'BackendEngineer': 2, 'SoftwareArchitect': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=3)
  ✓ [6] approve_plan: approve-plan sent
  ✓ [7] wait_phase: phase=executing
  ✓ [8] send: /resume-plan eeaa727f-951b-442c-b5db-56e9b7f697c2
  ✓ [9] wait_tasks: executing settle 180.0s statuses=['completed', 'completed', 'completed']
  ✓ [10] assert_messages: message assertions ok
  ✓ [11] assert_collab: collab snapshot ok
=== PASS: collab-generation-error-resilience ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict
SKIP: regression boot (SKIP_BOOT/NJ_BOOT_DONE set)

=== scenario: collab-participation-two-agent-strict ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 3c643bcd → collab-3c643bcd-c74f-4548-8a82-0fcb391b4c22
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=4)
  ✓ [6] assert_messages: message assertions ok
  ✓ [7] assert_collab: collab snapshot ok
=== PASS: collab-participation-two-agent-strict ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent
SKIP: regression boot (SKIP_BOOT/NJ_BOOT_DONE set)

=== scenario: collab-participation-three-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 2991fd2c → collab-2991fd2c-5466-4f80-88b8-f6d01db84875
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=8 by_agent={'BackendEngineer': 2, 'SoftwareArchitect': 5, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=6)
  ✓ [6] assert_messages: message assertions ok
  ✓ [7] assert_collab: collab snapshot ok
=== PASS: collab-participation-three-agent ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject
SKIP: regression boot (SKIP_BOOT/NJ_BOOT_DONE set)

=== scenario: collab-human-planning-interject ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab b1464b36 → collab-b1464b36-0cd8-492b-bace-09f7b7ff2450
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

=== Summary ===
PASS 21/23
  --- transcript (last messages) ---
    [question] apikey: @BackendEngineer I want to add theme support to this app
=== FAIL: public-backend-theme-workspace ===

  --- transcript (last messages) ---
    [question] apikey: I want to add theme support to this project
=== FAIL: dm-backend-echo-followup ===

  --- transcript (last messages) ---
    [question] apikey: I want to add theme support to this project
    [file_change] BackendEngineer: 📝 Proposing to edit file: src/index.tsx
    [system_info] System: File change proposal for src/index.tsx was not registered: preflight rejected "src/index.tsx": edit target does not exist: "src/index.tsx"
=== FAIL: dm-backend-echo-followup ===

  --- transcript (last messages) ---
    [question] apikey: I want to add light and dark theme support to this project
    [system_info] BackendEngineer: The model returned an empty reply. Try a shorter question or start a fresh thread; small Ollama models often fail when the system prompt is very long.
=== FAIL: dm-backend-deep-continuation ===

  --- transcript (last messages) ---
    [question] apikey: I want to add light and dark theme support to this project
    [chat] BackendEngineer: The theme toggle is already wired up. I'll  refine the implementation to persist the user's choice  and add a proper dark-mode-first design system so  both themes feel consistent.
    [question] apikey: go deeper on the approach — what would you implement first?
    [chat] BackendEngineer: To go deeper on the approach, I recommend implementing a unified runtime orchestration layer first. This will connect the Go backend, React frontend, and existing sample logic into a cohesive deployme
=== FAIL: dm-backend-deep-continuation ===

FAILED: chat:dm-backend-echo-followup, chat:dm-backend-deep-continuation

=== Summary ===
PASS 1/3
FAILED: chat-regression, conversation-regression
Log archived: /Users/camronwood/development/projects/neural-junkie/docs/testing/regression-bundle-2026-07-25-2004.log

RESULT regression-bundle: FAIL (exit 1, 5078s)
```

