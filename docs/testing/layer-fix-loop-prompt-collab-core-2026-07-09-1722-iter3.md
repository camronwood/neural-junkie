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

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-09-1722-iter3.md
Failed phases: collab-scenarios-core

## Failures to address

### collab:document-findings-execution [code]
```text
phase='reviewing' discussion.status='timed_out' msgs=1/4
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
```

### collab:collab-participation-three-agent [code]
```text
=== FAIL: collab-participation-three-agent ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/4
```

### collab:collab-human-planning-interject [code]
```text
=== FAIL: collab-human-planning-interject ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: - Task 1: @BackendEngineer - Write collabs/767ad862-d487-429f-bddd-b9c2848682f5/findings.md summarizing  README.md's fixture repo purpose - Task 2: @SoftwareArc
  --- end ---
```

### collab:collab-generation-error-resilience [code]
```text
=== FAIL: collab-generation-error-resilience ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: I agree with the two-task structure. Here's a  minimal plan: - Task 1: @BackendEngineer - Write  collabs/a8a5e239-1a99-4905-ab7e-745b599d3ed9/findings.md summar
  --- end ---
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

### collab:plan-dependency-prose-regression [code]
```text
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
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-core-2026-07-09-1722-iter3.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience
- python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject
- python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent
- python3 scripts/collab-scenarios.py --scenario document-findings-execution
- python3 scripts/collab-scenarios.py --scenario plan-dependency-prose-regression
- python3 scripts/collab-scenarios.py --scenario planning-two-agent

