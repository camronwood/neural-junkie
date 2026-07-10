You are fixing failures from Neural Junkie **layer gate: collab-core**.
Layer goal: Collab participation + planning core (~8 scenarios, ~45–90m; hub restart between)
Do not weaken assertions. Fix product/hub/agent behavior first (docs/TESTING.md).
After edits, run the targeted verification commands in this brief.

---


Rules (mandatory):
- Triage product/hub/agent behavior first, harness second (docs/TESTING.md).
- Do NOT weaken test assertions or scenario contracts to greenwash flakes.
- Prefer minimal, focused fixes in the neural-junkie repo.
- After edits, run the targeted verification commands listed below.
- Summarize what you changed and which commands you ran.

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-09-1722-iter2.md
Failed phases: collab-scenarios-core

## Failures to address

### collab:document-findings-execution [code]
```text
{'SoftwareArchitect': 2} (excluding generation_error)
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
```

### collab:collab-participation-two-agent-strict [code]
```text
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
```

### collab:collab-participation-three-agent [code]
```text
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
```

### collab:planning-two-agent [code]
```text
=== FAIL: planning-two-agent ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/4
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-09-1722-iter2.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent
- python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict
- python3 scripts/collab-scenarios.py --scenario document-findings-execution
- python3 scripts/collab-scenarios.py --scenario planning-two-agent

