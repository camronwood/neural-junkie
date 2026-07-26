# Layer gate — collab-full — 2026-07-25-1615 UTC

layer=collab-full
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-all` | FAIL | 6404s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-25-1615.log`

## Failures (tail)

### collab-scenarios-all (exit 2)

```text
 for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 27 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen3.5:9b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-findings-task-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-findings-task-regression


=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab c9898b0d → collab-c9898b0d-1fc7-4569-88b4-5de244c3959c
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
OK: switched all agents → qwen3.5:9b (Switched 27 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen3.5:9b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after planning-two-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after planning-two-agent


=== scenario: reject-collabs-subfolder ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@Assistant
  started collab a1488a9e → collab-a1488a9e-5466-4c48-9d3b-f1ea17f19d33
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
OK: switched all agents → qwen3.5:9b (Switched 27 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen3.5:9b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after reject-collabs-subfolder] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after reject-collabs-subfolder


=== scenario: resource-api-schema-planning ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @PlatformEngineer @Claude
  started collab b24168f9 → collab-b24168f9-2819-4744-a6a7-da345fa1009d
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=10 by_agent={'SoftwareArchitect': 2, 'PlatformEngineer': 5, 'apikey': 1, 'Claude': 2}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 27 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen3.5:9b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after resource-api-schema-planning] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after resource-api-schema-planning


=== scenario: resource-api-schema-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @FrontendEngineer @Claude
  started collab bf2853b4 → collab-bf2853b4-6c2c-45fc-ab31-4ed2a0c4085f
  ✓ [1] wait_phase: phase=planning
  ✗ [2] wait_discussion: discussion timeout (need total>=3, each of ['BackendEngineer', 'FrontendEngineer', 'Claude'] >= 1): counts={'BackendEngineer': 3, 'apikey': 4, 'FrontendEngineer': 13}
agent discussion: total=20 counts={'BackendEngineer': 3, 'apikey': 4, 'FrontendEngineer': 13} (excluding generation_error)
  ok: @BackendEngineer — 3 message(s)
  ok: @FrontendEngineer — 13 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 28
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/10

  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: I agree with the plan to produce exactly two tasks focused on scope and schema review. Given the file tree shows `core/sample/main.go` and `src/` (React), I wil
    [collaboration_discussion] FrontendEngineer: Based on the current workspace structure, I see we are refining a React component that handles dynamic theme switching between light, dark, and system defaults 
    [collaboration_discussion] FrontendEngineer: I agree with the current plan to produce exactly two tasks: scope definition and schema review. Both will be assigned to @BackendEngineer as requested, targetin
    [collaboration_discussion] FrontendEngineer: - Task 1: @BackendEngineer - Define Scope with Deliverable collabs/bf2853b4-6c2c-45fc-ab31-4ed2a0c4085f/scope.md   - Task 2: @BackendEngineer - Review API docs 
    [collaboration_discussion] FrontendEngineer: I agree with the current plan to produce exactly two tasks focused on scope definition and schema review. Given the file tree shows `core/sample/main.go` and `s
    [collaboration_discussion] FrontendEngineer: - Task 1: @BackendEngineer - Define Scope with Deliverable `collabs/bf2853b4-6c2c-45fc-ab31-4ed2a0c4085f/scope.md` outlining standardization goals for the resou
    [collaboration_discussion] FrontendEngineer: I agree with the plan to produce exactly two tasks focused on scope and schema review. Given the file tree shows `core/sample/main.go` and `src/` (React), I wil
    [collaboration_discussion] FrontendEngineer: # Collaboration: Investigate resource api document schema standardization/registration  **Goal:** @BackendEngineer @FrontendEngineer @Claude Investigate resourc
  --- end ---

agent discussion: total=20 counts={'BackendEngineer': 3, 'apikey': 4, 'FrontendEngineer': 13} (excluding generation_error)
  ok: @BackendEngineer — 3 message(s)
  ok: @FrontendEngineer — 13 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 28
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/10
=== FAIL: resource-api-schema-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> flake retry 2/1 for resource-api-schema-regression: discussion timeout (need total>=3, each of ['BackendEngineer', 'FrontendEngineer', 'Claude'] >= 1): counts={'BackendEngineer': 3, 'apikey': 4, 'FrontendEngineer

=== scenario: resource-api-schema-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @FrontendEngineer @Claude
  started collab d22b1971 → collab-d22b1971-1621-498f-a566-71e66589a5f0
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=21 by_agent={'BackendEngineer': 4, 'apikey': 4, 'FrontendEngineer': 12, 'Claude': 1}; planning ready (after retry 1)
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] assert_collab: collab snapshot ok
  ✓ [7] approve_plan: approve-plan sent
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_deliverable_stubs: 2 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/d22b1971-1621-498f-a566-71e66589a5f0
  ✓ [10] send: /resume-plan d22b1971-1621-498f-a566-71e66589a5f0
  ✓ [11] wait_tasks: executing settle 90.0s statuses=['completed', 'completed']
  ✓ [12] assert_messages: message assertions ok
  ✓ [13] approve_file_changes: discussion fallback wrote /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/d22b1971-1621-498f-a566-71e66589a5f0/scope.md
  ✓ [14] assert_files: judge:warn:SCORE=0.00:ollama/qwen2.5-coder:14b: Reason: The deliverable does not contain the required tasks as specified by the user. It only provides stubs and placeholders without fulfilling the task requirements.
=== PASS: resource-api-schema-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after resource-api-schema-regression)...

>>> Hub restart (after resource-api-schema-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 27 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen3.5:9b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
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
  started collab 5969075a → collab-5969075a-4b6b-42f9-b14a-d94c944abf88
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 2, 'BackendEngineer': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✓ [7] wait_tasks: tasks completed
  ✓ [8] assert_files: judge:pass:SCORE=1.00:ollama/qwen2.5-coder:14b: The deliverable file correctly summarizes the minimal-repo fixture using the specified files and does not include any hallucinated paths or unrelated boilerplate.
=== PASS: solo-vs-collab-parity ===

  ✓ cleanup: cancelled and removed workspace artifacts
make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 6404s)
```

