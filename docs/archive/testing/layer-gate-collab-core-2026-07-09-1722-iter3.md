# Layer gate — collab-core — 2026-07-09-1722-iter3 UTC

layer=collab-core
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-core` | FAIL | 3581s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-09-1722-iter3.log`

## Failures (tail)

### collab-scenarios-core (exit 2)

```text
lanning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/4
  ✓ cleanup: cancelled and removed workspace artifacts

>>> flake retry 2/1 for planning-two-agent: discussion timeout (need total>=2, each of ['SoftwareArchitect', 'BackendEngineer'] >= 1): counts={}
agent discussion: total=0 counts={} (excluding generation_e

=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 7ad319e5 → collab-7ad319e5-0825-4796-b80f-d9736759b25c
  ✓ [1] wait_phase: phase=planning
  ✗ [2] wait_discussion: discussion timeout (need total>=2, each of ['SoftwareArchitect', 'BackendEngineer'] >= 1): counts={}
agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/4
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


=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 5edf7bdf → collab-5edf7bdf-cdef-4bcc-81a3-cae0f061cc89
  ✓ [1] wait_phase: phase=planning
  ✗ [2] wait_discussion: discussion timeout (need total>=3, each of ['BackendEngineer', 'SoftwareArchitect', 'Claude'] >= 1): counts={'BackendEngineer': 1}
agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  ok: @BackendEngineer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 13
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/4
  ✓ cleanup: cancelled and removed workspace artifacts

>>> flake retry 2/1 for plan-dependency-prose-regression: discussion timeout (need total>=3, each of ['BackendEngineer', 'SoftwareArchitect', 'Claude'] >= 1): counts={'BackendEngineer': 1}
agent discussion: total=1 cou

=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect @Claude
  started collab 8fdf6f67 → collab-8fdf6f67-9b69-4a00-ba85-1febfd94a265
  ✓ [1] wait_phase: phase=planning
  ✗ [2] wait_discussion: discussion timeout (need total>=3, each of ['BackendEngineer', 'SoftwareArchitect', 'Claude'] >= 1): counts={'BackendEngineer': 1}
agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  ok: @BackendEngineer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 6
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/4
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


=== scenario: collab-minimal-completion-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer
  ✓ cleanup: cancelled and removed workspace artifacts

>>> Hub restart between scenarios (after collab-minimal-completion-regression)...

>>> Hub restart (after collab-minimal-completion-regression)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-minimal-completion-regression] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-minimal-completion-regression


=== scenario: document-findings-execution ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab 168b4d0c → collab-168b4d0c-4633-443d-b43e-9dee1e75120e
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_phase: phase=reviewing
  ✓ [3] wait_planning_recap: planning_recap_status=complete
  ✓ [4] assert_plan: plan ok (tasks=1)
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✓ [7] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/168b4d0c-4633-443d-b43e-9dee1e75120e
  ✓ [8] send: /resume-plan 168b4d0c-4633-443d-b43e-9dee1e75120e
  ✓ [9] wait_tasks: executing settle 120.0s statuses=['pending']
  ✓ [10] approve_file_changes: discussion fallback wrote /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/168b4d0c-4633-443d-b43e-9dee1e75120e/findings.md
  ✗ [11] assert_files: judge:fail:ollama/qwen2.5-coder:14b: The deliverable does not provide the requested resume plan for the collaboration.
  ✓ cleanup: cancelled and removed workspace artifacts

  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/5
=== FAIL: collab-participation-three-agent ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/4
=== FAIL: collab-human-planning-interject ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: - Task 1: @BackendEngineer - Write collabs/767ad862-d487-429f-bddd-b9c2848682f5/findings.md summarizing  README.md's fixture repo purpose - Task 2: @SoftwareArc
  --- end ---

=== FAIL: collab-generation-error-resilience ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: I agree with the two-task structure. Here's a  minimal plan: - Task 1: @BackendEngineer - Write  collabs/a8a5e239-1a99-4905-ab7e-745b599d3ed9/findings.md summar
  --- end ---

=== FAIL: collab-generation-error-resilience ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/4
=== FAIL: planning-two-agent ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/4
=== FAIL: planning-two-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: The referenced file contains Go code (`package main`),  which conflicts with the Node.js/Tauri environment suggested by  packages like `@tauri-apps` and `webpac
  --- end ---

agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  ok: @BackendEngineer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 13
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/4
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: The workspace context reveals a primary Node.js ecosystem  based on tools like Webpack and Babel, while  the referenced `file.md` contains Go code that contradi
  --- end ---

agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  ok: @BackendEngineer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 6
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/4
=== FAIL: plan-dependency-prose-regression ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: Task list (3 tasks, concrete deliverables): - Task  1: @BackendEngineer - Write collabs/168b4d0c-4633-443d-b43e-9dee1e75120e/findings.md summarizing README.md a
    [collaboration_discussion] BackendEngineer: This is a solid starting point, but given  the minimal nature of this fixture repo, we  should consolidate Task 2 & 3 into single  coherent deliverables to redu
    [collaboration_discussion] SoftwareArchitect: - Task 1: @BackendEngineer - Document findings in  collabs/<id>/findings.md summarizing README.md and core/sample/main.go. - Task 2:  @SoftwareArchitect - Defin
    [collaboration_discussion] BackendEngineer: This is a **React/TypeScript PWA project** equipped with  Workbox service worker packages for offline capabilities, precaching,  and other PWA features. The app
  --- end ---

=== FAIL: document-findings-execution ===

make[1]: *** [collab-scenarios-core] Error 1

RESULT collab-scenarios-core: FAIL (exit 2, 3581s)
```

