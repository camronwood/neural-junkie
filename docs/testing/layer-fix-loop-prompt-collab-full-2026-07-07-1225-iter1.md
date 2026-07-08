You are fixing failures from Neural Junkie **layer gate: collab-full**.
Layer goal: Full collab-scenarios sweep (~15 scenarios, 1–3h)
Do not weaken assertions. Fix product/hub/agent behavior first (docs/TESTING.md).
After edits, run the targeted verification commands in this brief.

---


Rules (mandatory):
- Triage product/hub/agent behavior first, harness second (docs/TESTING.md).
- Do NOT weaken test assertions or scenario contracts to greenwash flakes.
- Prefer minimal, focused fixes in the neural-junkie repo.
- After edits, run the targeted verification commands listed below.
- Summarize what you changed and which commands you ran.

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-07-1225-iter1.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: ## Planning Perspective I agree with the three-file  task approach. Here's my structured plan: - Task  1: @BackendEngineer - Write collabs/a48a3ad7-8203-4a46-a8
  --- end ---

agent discussion: total=1 counts={'SoftwareArchitect': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 2
  generation_error posts in channel: 1
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
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
    [collaboration_discussion] Claude: This is a minimal fixture repo — there's  no `resource-api/json_endpoints/` path; the closest analog is `core/server/main.go`,  so I'll treat that as the refere
  --- end ---

agent discussion: total=1 counts={'Claude': 1} (excluding generation_error)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 13
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/6
```

### collab:plan-phoenix-combined-regression [code]
```text
=== FAIL: plan-phoenix-combined-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 1
  generation_error posts in channel: 1
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/8
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
    [collaboration_discussion] Claude: This repo has no `resource-api/json_endpoints/` or `docs/tim/` directories  — the actual codebase is a minimal Go/React  fixture (`core/sample/main.go`, `core/s
  --- end ---

agent discussion: total=1 counts={'Claude': 1} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 8
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
```

### collab:resource-api-schema-regression [code]
```text
=== FAIL: resource-api-schema-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: Checked the repo directly rather than trusting the  stated focus paths: **`resource-api/json_endpoints/` and `docs/tim/` don't exist  anywhere in this tree.** T
    [collaboration_discussion] Claude: # Collaboration Recap: Resource API Schema Standardization **Goal:**  Investigate standardizing and registering schemas for the "resource  API" JSON endpoints r
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=2 counts={'Claude': 2} (excluding generation_error)
  raw discussion posts (incl. errors): 3
  generation_error posts in channel: 1
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 2 message(s)
  system turn handoffs in channel: 14
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/10
```

### collab:solo-vs-collab-parity [code]
```text
=== FAIL: solo-vs-collab-parity ===

  FAIL [solo]: no BackendEngineer reply (timeout waiting for @BackendEngineer (baseline=1, counts={'BackendEngineer': 1, 'User': 1}))
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: Task 1: @BackendEngineer - Write collabs/903336a6-e5ad-4cdb-9004-919152399722/findings.md summarizing project  structure from README.md and core/sample/main.go 
  --- end ---
```

### collab:collab-generation-error-resilience [code]
```text
=== FAIL: collab-generation-error-resilience ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: - Task 1: @BackendEngineer - Write collabs/27cfba7a-bbd7-49f9-95f0-480f48186f30/findings.md summarizing  README.md and file.md (sample main.go) context from min
  --- end ---
```

### collab:collab-human-planning-interject [code]
```text
=== FAIL: collab-human-planning-interject ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/4
```

### collab:collab-participation-three-agent [code]
```text
=== FAIL: collab-participation-three-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: @BackendEngineer @SoftwareArchitect — from an implementation standpoint, a  health-check service should stay dependency-free: standard `net/http`, a  single `/h
  --- end ---

agent discussion: total=1 counts={'Claude': 1} (excluding generation_error)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 8
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
```

### collab:collab-participation-two-agent-strict [code]
```text
=== FAIL: collab-participation-two-agent-strict ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/4
```

### collab:collaboration-station-website [code]
```text
=== FAIL: collaboration-station-website ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: **FrontendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 1
  generation_error posts in channel: 1
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SecurityReviewer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/18
```

### collab:collaboration-station-website-sa [code]
```text
=== FAIL: collaboration-station-website-sa ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: I'll add my architecture perspective, then move toward  a minimal task list. **Architecture Review:** Before implementing,  we need to establish: (1) page struc
    [collaboration_discussion] FrontendEngineer: **FrontendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] FrontendEngineer: **Plan confirmation:** Workspace at `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/` appears empty (fresh  collaboration). I align with the brief for **Collabora
  --- end ---

agent discussion: total=2 counts={'SoftwareArchitect': 1, 'FrontendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 3
  generation_error posts in channel: 1
  ok: @FrontendEngineer — 1 message(s)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/18
```

### collab:document-findings-execution [code]
```text
=== FAIL: document-findings-execution ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: - Task 1: @BackendEngineer - Write collabs/70603b2d-27e5-4e7b-886d-7d0b3fb67355/findings.md summarizing  README.md and core/sample/main.go. - depends: none - Ta
  --- end ---
```

### collab:execute-deliverable [code]
```text
=== FAIL: execute-deliverable ===


  --- transcript (agent messages) ---
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

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: Flagging a blocker before proposing tasks: the focus  paths named in the goal don't exist in  this repo. **Grounding check (not fabricated):** I searched  the f
    [collaboration_discussion] Claude: # Session Recap: Resource API Schema Registration Investigation  **Goal:** Investigate resource API document schema standardization/registration under  `resourc
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] BackendEngineer: --- **Planning Perspective (BackendEngineer):** My focus is on  backend contract design, validation/error handling, and consistency across  JSON endpoints. I'll
  --- end ---

agent discussion: total=3 counts={'Claude': 2, 'BackendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 4
  generation_error posts in channel: 1
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @BackendEngineer — 1 message(s)
  ok: @Claude — 2 message(s)
  system turn handoffs in channel: 12
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/15
```

### collab:phoenix-resource-api-e2e [code]
```text
=== FAIL: phoenix-resource-api-e2e ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: The focus paths `resource-api/json_endpoints/` and `docs/tim/` don't exist  anywhere in this repo — I searched the  full tree and only found a stale backup  fro
    [collaboration_discussion] Claude: Confirmed independently: this repo has no `resource-api/`, `json_endpoints/`,  or `docs/tim/` paths — only `core/`, `src/`, `sample`,  `server`, `docker-compose
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect — I agree. Here's my Backend perspective  for planning: **Proposed Plan (4 tasks):** - Task  1: @SoftwareArchitect - Review existing schemas 
  --- end ---

agent discussion: total=3 counts={'Claude': 2, 'BackendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 4
  generation_error posts in channel: 1
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @BackendEngineer — 1 message(s)
  ok: @Claude — 2 message(s)
  system turn handoffs in channel: 14
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/15
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-07-1225-iter1.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression
- python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience
- python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject
- python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent
- python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict
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

