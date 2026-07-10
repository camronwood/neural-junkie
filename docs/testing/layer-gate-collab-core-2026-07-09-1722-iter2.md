# Layer gate — collab-core — 2026-07-09-1722-iter2 UTC

layer=collab-core
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `collab-scenarios-core` | FAIL | 2933s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-09-1722-iter2.log`

## Failures (tail)

### collab-scenarios-core (exit 2)

```text
ween scenarios (after collab-generation-error-resilience)...

>>> Hub restart (after collab-generation-error-resilience)...
>>> Waiting for agent roster...
OK: required agents online
>>> Pinning regression agent models (≤14B)...
OK: switched all agents → qwen3.5:9b (Switched 25 agents to ollama (qwen3.5:9b))
>>> Hub hygiene (pending file changes + scenario channels)...
  rejected 0 pending file change(s)

>>> [after collab-generation-error-resilience] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)
OK: hub hygiene complete
OK: hub restarted for after collab-generation-error-resilience


=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab 7fb7cb14 → collab-7fb7cb14-cc53-4a8b-bad1-f405ebdb4043
  ✓ [1] wait_phase: phase=planning
  ✗ [2] wait_discussion: discussion timeout (need total>=2, each of ['SoftwareArchitect', 'BackendEngineer'] >= 1): counts={'SoftwareArchitect': 2}
agent discussion: total=2 counts={'SoftwareArchitect': 2} (excluding generation_error)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 20
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/4
  ✓ cleanup: cancelled and removed workspace artifacts

>>> flake retry 2/1 for planning-two-agent: discussion timeout (need total>=2, each of ['SoftwareArchitect', 'BackendEngineer'] >= 1): counts={'SoftwareArchitect': 2}
agent discussion: total=2 counts={'So

=== scenario: planning-two-agent ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@SoftwareArchitect @BackendEngineer
  started collab edd5d825 → collab-edd5d825-5426-4e3e-8c33-808d7e4243bf
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
  started collab 8c58e1a0 → collab-8c58e1a0-bfa4-418b-9da5-2f96e34b4db8
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_discussion: messages total=5 by_agent={'BackendEngineer': 2, 'SoftwareArchitect': 2, 'Claude': 1}; planning ready (after retry 1)
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
  started collab 0e9c14ea → collab-0e9c14ea-f8e6-4016-b52b-d9ea7028403c
  ✓ [1] wait_phase: phase=planning
  ✓ [2] wait_phase: phase=reviewing
  ✓ [3] wait_planning_recap: planning_recap_status=complete
  ✓ [4] assert_plan: plan ok (tasks=1)
  ✓ [5] approve_plan: approve-plan sent
  ✓ [6] wait_phase: phase=executing
  ✓ [7] assert_deliverable_stubs: 1 stub(s) in /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/0e9c14ea-f8e6-4016-b52b-d9ea7028403c
  ✓ [8] send: /resume-plan 0e9c14ea-f8e6-4016-b52b-d9ea7028403c
  ✓ [9] wait_tasks: executing settle 120.0s statuses=['pending']
  ✓ [10] approve_file_changes: discussion fallback wrote /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/minimal-repo/collabs/0e9c14ea-f8e6-4016-b52b-d9ea7028403c/findings.md
  ✗ [11] assert_files: judge:fail:ollama/qwen2.5-coder:14b: The deliverable file does not provide the requested content. It only lists tasks without any actual findings or analysis.
  ✓ cleanup: cancelled and removed workspace artifacts

  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/4
=== FAIL: collab-participation-two-agent-strict ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/<id>/design.md with  filter API schema (levels/patterns/flags). - Task 2: @BackendEngineer  - Write collabs/<id>/fi
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/<id>/design.md with  filter API schema (levels/patterns/flags). - Task 2: @BackendEngineer  - Write collabs/<id>/fi
  --- end ---

agent discussion: total=2 counts={'SoftwareArchitect': 2} (excluding generation_error)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 7
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=2/4
=== FAIL: collab-participation-two-agent-strict ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: For a minimal health-check HTTP service, here's my  plan: - Task 1: @BackendEngineer - Write collabs/<id>/requirements.md  defining the /health endpoint contrac
  --- end ---

agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  ok: @BackendEngineer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 17
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
=== FAIL: collab-participation-three-agent ===


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
    [collaboration_discussion] SoftwareArchitect: I need to clarify a few key design  decisions before drafting the minimal task list: 1.  **Key management approach**: Should we use system secrets  (e.g., OS KM
    [collaboration_discussion] SoftwareArchitect: I need to clarify a few key design  decisions before drafting the minimal task list: 1.  **Key management**: Should we use system secrets (OS  KMS, file-based k
  --- end ---

agent discussion: total=2 counts={'SoftwareArchitect': 2} (excluding generation_error)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 20
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/4
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

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: Task 1: @BackendEngineer - Write collabs/0e9c14ea-f8e6-4016-b52b-d9ea7028403c/findings.md summarizing README.md  and core/sample/main.go (explicit goal) - depen
    [collaboration_discussion] BackendEngineer: I agree with the plan structure. Before executing  Task 1, I'll write the findings.md directly since  this is a self-contained summarization task. Here's my  as
    [collaboration_discussion] SoftwareArchitect: - Task 1: @BackendEngineer - Document findings in  collabs/<id>/findings.md summarizing README.md and core/sample/main.go. - Task 2:  @BackendEngineer - Audit A
    [collaboration_discussion] BackendEngineer: Based on the file structure, this is a  **React + TypeScript** application that uses **Workbox** for  Service Worker/PWA capabilities. **Key Technologies:** - *
  --- end ---

=== FAIL: document-findings-execution ===

make[1]: *** [collab-scenarios-core] Error 1

RESULT collab-scenarios-core: FAIL (exit 2, 2933s)
```

