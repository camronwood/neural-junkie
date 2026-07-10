You are fixing failures from Neural Junkie **layer gate: collab-full**.
Layer goal: Full collab-scenarios sweep (25 scenarios, ~2–4h)
Do not weaken assertions. Fix product/hub/agent behavior first (docs/TESTING.md).
After edits, run the targeted verification commands in this brief.

---


Rules (mandatory):
- Triage product/hub/agent behavior first, harness second (docs/TESTING.md).
- Do NOT weaken test assertions or scenario contracts to greenwash flakes.
- Prefer minimal, focused fixes in the neural-junkie repo.
- After edits, run the targeted verification commands listed below.
- Summarize what you changed and which commands you ran.

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-10-0103-iter1.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:phoenix-resource-api-e2e [code]
```text
=== FAIL: phoenix-resource-api-e2e ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 2
  generation_error posts in channel: 2
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/15
```

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/4
```

### collab:plan-distinct-deliverables-same-agent [code]
```text
=== FAIL: plan-distinct-deliverables-same-agent ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/5
```

### collab:plan-findings-task-regression [code]
```text
=== FAIL: plan-findings-task-regression ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/6
```

### collab:plan-phoenix-combined-regression [code]
```text
=== FAIL: plan-phoenix-combined-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: I've reviewed the workspace. Here's the minimal task  list: - Task 1: @BackendEngineer - Write collabs/a6022582-55d1-4d52-8f0b-c0d122780f6b/api_schema.md  docum
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
  --- end ---

agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 3
  generation_error posts in channel: 2
  ok: @BackendEngineer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 11
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/8
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

### collab:resource-api-schema-planning [code]
```text
=== FAIL: resource-api-schema-planning ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/5
```

### collab:resource-api-schema-regression [code]
```text
=== FAIL: resource-api-schema-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 2
  generation_error posts in channel: 2
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/10
```

### collab:solo-vs-collab-parity [code]
```text
=== FAIL: solo-vs-collab-parity ===

  FAIL [solo]: no BackendEngineer reply (timeout waiting for @BackendEngineer (baseline=0, counts={}))
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/c88c32cc-2473-426d-9417-a68e466706e7/readme-summary.md from  README.md. - Task 2: @BackendEngineer - Write collabs/
  --- end ---

agent discussion: total=1 counts={'SoftwareArchitect': 1} (excluding generation_error)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 11
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/4
```

### collab:collab-human-planning-interject [code]
```text
=== FAIL: collab-human-planning-interject ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/498b968c-25c4-4092-9b28-1e9b03c43585/readme-summary.md from  README.md - Task 2: @BackendEngineer - Write collabs/4
  --- end ---

agent discussion: total=1 counts={'SoftwareArchitect': 1} (excluding generation_error)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 3
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/4
```

### collab:collab-participation-three-agent [code]
```text
=== FAIL: collab-participation-three-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: This is a simple health-check service—minimal state, single  endpoint, no external deps needed. - Task 1:  @BackendEngineer - Write collabs/<id>/requirements.md
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
  --- end ---

agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 2
  generation_error posts in channel: 1
  ok: @BackendEngineer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 24
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
```

### collab:collaboration-station-website [code]
```text
=== FAIL: collaboration-station-website ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: **Task List:** - Task 1: @FrontendEngineer - Write  collabs/3ee7280d-8e64-4d02-8055-1f4e7f6cfa71/architecture-review.md comparing b222bffe/frontend_architecture
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
  --- end ---

agent discussion: total=1 counts={'FrontendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 3
  generation_error posts in channel: 2
  ok: @FrontendEngineer — 1 message(s)
  FAIL: @SecurityReviewer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 11
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/18
```

### collab:collaboration-station-website-sa [code]
```text
=== FAIL: collaboration-station-website-sa ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: I notice that file reading tools are unavailable  in this environment. Based on the collaboration goal  provided, I'll proceed with the minimal task list  using
  --- end ---

agent discussion: total=1 counts={'FrontendEngineer': 1} (excluding generation_error)
  ok: @FrontendEngineer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 1
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/18
```

### collab:document-findings-execution [code]
```text
=== FAIL: document-findings-execution ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **Task List:** - Task 1: @BackendEngineer - Write  `collabs/9c5d932b-c2cf-4c90-92d5-1b4e60e4c7a0/findings.md` summarizing README.md and core/sample/main.go - de
  --- end ---
```

### collab:execute-deliverable [code]
```text
=== FAIL: execute-deliverable ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **Proposed Task List:** - Task 1: @BackendEngineer -  Write collabs/5df862c5-3fe2-4f1c-8971-b9d9a6ca8d86/findings.md with three bullet findings (paths, file  st
  --- end ---
```

### collab:execution-no-stack-commands [code]
```text
=== FAIL: execution-no-stack-commands ===


  --- transcript (agent messages) ---
  --- end ---
```

### collab:make-me-a-website [code]
```text
=== FAIL: make-me-a-website ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
  --- end ---

agent discussion: total=1 counts={'FrontendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 18
  generation_error posts in channel: 17
  ok: @FrontendEngineer — 1 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 14
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/12
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-10-0103-iter1.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression
- python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject
- python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent
- python3 scripts/collab-scenarios.py --scenario collaboration-station-website
- python3 scripts/collab-scenarios.py --scenario collaboration-station-website-sa
- python3 scripts/collab-scenarios.py --scenario document-findings-execution
- python3 scripts/collab-scenarios.py --scenario execute-deliverable
- python3 scripts/collab-scenarios.py --scenario execution-no-stack-commands
- python3 scripts/collab-scenarios.py --scenario make-me-a-website
- python3 scripts/collab-scenarios.py --scenario phoenix-resource-api-e2e
- python3 scripts/collab-scenarios.py --scenario plan-dependency-prose-regression
- python3 scripts/collab-scenarios.py --scenario plan-distinct-deliverables-same-agent
- python3 scripts/collab-scenarios.py --scenario plan-findings-task-regression
- python3 scripts/collab-scenarios.py --scenario plan-phoenix-combined-regression
- python3 scripts/collab-scenarios.py --scenario planning-two-agent
- python3 scripts/collab-scenarios.py --scenario resource-api-schema-planning
- python3 scripts/collab-scenarios.py --scenario resource-api-schema-regression
- python3 scripts/collab-scenarios.py --scenario solo-vs-collab-parity

