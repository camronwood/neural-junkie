# Layer gate — collab-full — 2026-07-30-0945 UTC

layer=collab-full
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-all` | FAIL | 6362s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-30-0945.log`

## Failures (tail)

### collab-scenarios-all (exit 2)

```text
gents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen3.5:9b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-dependency-prose-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-dependency-prose-regression


=== scenario: plan-distinct-deliverables-same-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 45ee573a → collab-45ee573a-5ece-47b6-92b4-0eb3d821d8d5
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=2 by_agent={'SoftwareArchitect': 1, 'BackendEngineer': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
>>> Collab regression tuning (Claude cloud-preferred / optional slim roster)...
OK: Claude → ollama (qwen3.5:9b) (fallback: ANTHROPIC_API_KEY not sk-ant-… (likely CLI/proxy token; use Ollama))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after plan-distinct-deliverables-same-agent] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after plan-distinct-deliverables-same-agent


=== scenario: plan-findings-task-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab f1737017 → collab-f1737017-6095-454e-97da-9ca4745d3ad9
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=4 by_agent={'BackendEngineer': 1, 'Claude': 2, 'SoftwareArchitect': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
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
  started collab 3ffe4111 → collab-3ffe4111-4cbb-4bef-b051-5c7c3ec83e09
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
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
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
  started collab 0f7c134e → collab-0f7c134e-23fc-4ecf-bd9f-8092cea34e08
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=1 by_agent={'Assistant': 1}; planning ready
  ✓ [3] assert_collab: collab snapshot ok
=== PASS: reject-collabs-subfolder ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after reject-collabs-subfolder)...

>>> Hub restart (after reject-collabs-subfolder)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
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
  started collab 45a930b1 → collab-45a930b1-d8f5-4980-b422-afcbd10fe300
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=3 by_agent={'SoftwareArchitect': 1, 'PlatformEngineer': 1, 'Claude': 1}; planning ready
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
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
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
  started collab fb7d7055 → collab-fb7d7055-7826-4aab-bcf1-4568c0d6bded
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=5 by_agent={'BackendEngineer': 2, 'FrontendEngineer': 1, 'apikey': 1, 'Claude': 1}; planning ready
  ✓ [3] wait_phase: phase=reviewing
  ✓ [4] wait_planning_recap: planning_recap_status=complete
  ✓ [5] assert_plan: plan ok (tasks=2)
  ✓ [6] assert_collab: collab snapshot ok
  ✓ [7] approve_plan: approve-plan sent
  ✓ [8] assert_collab: collab snapshot ok
  ✓ [9] assert_deliverable_stubs: 2 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/fb7d7055-7826-4aab-bcf1-4568c0d6bded
  ✓ [10] send: /resume-plan fb7d7055-7826-4aab-bcf1-4568c0d6bded
  ✓ [11] wait_tasks: executing settle 90.0s statuses=['in_progress', 'completed']
  ✓ [12] assert_messages: message assertions ok
  ✓ [13] approve_file_changes: file exists (/Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/fb7d7055-7826-4aab-bcf1-4568c0d6bded/scope.md)
  ✓ [14] assert_files: judge:pass:SCORE=0.95:ollama/qwen2.5-coder:14b: Reason: The deliverable file "collabs/fb7d7055-7826-4aab-bcf1-4568c0d6bded/scope.md" comprehensively addresses the user's request by defining the scope and approach for investigating resource API document schema standardization and registration. It includes a detailed plan for inventory and analysis of existing artifacts, a registration strategy proposal, and deliverable definitions, all within the specified boundaries.
=== PASS: resource-api-schema-regression ===

  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after resource-api-schema-regression)...

>>> Hub restart (after resource-api-schema-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 28 agents to ollama (qwen3.5:9b))
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
  started collab 9cfe0a4a → collab-9cfe0a4a-a674-4a6e-b75b-d5173271cbce
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
make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 6362s)
```

