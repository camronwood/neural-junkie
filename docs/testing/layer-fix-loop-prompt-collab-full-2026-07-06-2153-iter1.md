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

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-06-2153-iter1.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/5
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
    [collaboration_discussion] Claude: Note: the focus path `resource-api/json_endpoints/` doesn't exist in  this repo — the only API-adjacent code is  `core/server/main.go` (a minimal Go handler ser
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] Claude: # Collaboration Recap: API Schema Documentation Plan **Goal:**  Plan 4 tasks to document API schema for  this repo under `collabs/987ea8f1-6d97-4a93-a74e-63631a
  --- end ---

agent discussion: total=2 counts={'Claude': 2} (excluding generation_error)
  raw discussion posts (incl. errors): 3
  generation_error posts in channel: 1
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 2 message(s)
  system turn handoffs in channel: 13
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/6
```

### collab:plan-phoenix-combined-regression [code]
```text
=== FAIL: plan-phoenix-combined-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 2
  generation_error posts in channel: 2
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 4
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/8
```

### collab:planning-two-agent [code]
```text
=== FAIL: planning-two-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 1
  generation_error posts in channel: 1
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
    [collaboration_discussion] Claude: This repo doesn't actually contain the focus paths  named in the brief — there's no `resource-api/json_endpoints/`  or `docs/tim/` anywhere in the tree. What ex
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
    [collaboration_discussion] Claude: @BackendEngineer @FrontendEngineer — flagging a mismatch before we  plan: the focus paths given (`resource-api/json_endpoints/` and `docs/tim/`)  don't exist an
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] FrontendEngineer: **FrontendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=1 counts={'Claude': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 3
  generation_error posts in channel: 2
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 14
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/10
```

### collab:solo-vs-collab-parity [code]
```text
=== FAIL: solo-vs-collab-parity (solo leg) ===

make[1]: *** [collab-scenarios] Error 1

RESULT collab-scenarios-all: FAIL (exit 2, 14863s)
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: Here is a minimal 2-task plan: - Task  1: @BackendEngineer - Write collabs/9a73d132-cbd6-4eae-bafb-80136be7e813/findings.md summarizing README.md fixture  purpo
  --- end ---
```

### collab:collab-generation-error-resilience [code]
```text
=== FAIL: collab-generation-error-resilience ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: Based on the minimal workspace and collaboration goal:  - **Task 1**: @BackendEngineer - Write `collabs/25af1846-f518-4d0b-b56e-9fd85551c8e1/findings.md` summar
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
  pending file changes (hub): 1
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/4
```

### collab:collab-participation-three-agent [code]
```text
=== FAIL: collab-participation-three-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: - No project workspace is bound to this  collaboration yet — before any file lands in  `collabs/<id>/`, we need the user to run `/collaborate`  from an open pro
  --- end ---

agent discussion: total=1 counts={'Claude': 1} (excluding generation_error)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 9
  pending file changes (hub): 1
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
  pending file changes (hub): 1
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/4
```

### collab:collaboration-station-website [code]
```text
=== FAIL: collaboration-station-website ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: **FrontendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SecurityReviewer: I need to analyze the existing work before  planning. Let me start by exploring what's in  the reference collaboration directory. ## Current State Analysis  Let
  --- end ---

agent discussion: total=1 counts={'SecurityReviewer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 2
  generation_error posts in channel: 1
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @SecurityReviewer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 4
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/18
```

### collab:collaboration-station-website-sa [code]
```text
=== FAIL: collaboration-station-website-sa ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: I'll review the existing work under `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/` to  understand the current state before planning the new  deliverables. Let 
    [collaboration_discussion] SoftwareArchitect: Let me first review the existing work under  `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/` to understand the current state. I'll use  list_dir and glob_file_s
    [collaboration_discussion] FrontendEngineer: ### Task List (3 Tasks - Planning Phase)  - Task 1: @SoftwareArchitect - Write collabs/c8c2df68-d4f3-4df3-ba18-ddc4634c8760/site-structure.md defining  navigati
  --- end ---

agent discussion: total=3 counts={'FrontendEngineer': 2, 'SoftwareArchitect': 1} (excluding generation_error)
  ok: @FrontendEngineer — 2 message(s)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 9
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/18
```

### collab:document-findings-execution [code]
```text
=== FAIL: document-findings-execution ===


  --- transcript (agent messages) ---
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
    [collaboration_discussion] Claude: The directories named in this task — `resource-api/json_endpoints/`  and `docs/tim/` — don't exist anywhere in this  repo. The actual structure is `core/server`
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=1 counts={'Claude': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 4
  generation_error posts in channel: 3
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 14
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/15
```

### collab:phoenix-resource-api-e2e [code]
```text
=== FAIL: phoenix-resource-api-e2e ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: The paths this collaboration is scoped to —  `resource-api/json_endpoints/` and `docs/tim/` — don't exist anywhere in  this repo. The actual structure is `core/
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=1 counts={'Claude': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 4
  generation_error posts in channel: 3
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 15
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/15
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-06-2153-iter1.log

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

