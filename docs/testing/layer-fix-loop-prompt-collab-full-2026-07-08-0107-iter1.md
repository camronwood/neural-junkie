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

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-08-0107-iter1.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:phoenix-resource-api-e2e [code]
```text
=== FAIL: phoenix-resource-api-e2e ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] BackendEngineer: @SoftwareArchitect — Planning Perspective (Backend Architecture & API  Design) As BackendEngineer, I'll focus on service contracts,  validation schemas, and int
    [collaboration_discussion] Claude: I need context on what @SoftwareArchitect and @BackendEngineer  have discovered before providing a complete plan perspective.  Could you share: - What schema st
  --- end ---

agent discussion: total=2 counts={'BackendEngineer': 1, 'Claude': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 3
  generation_error posts in channel: 1
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @BackendEngineer — 1 message(s)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/15
```

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


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
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  raw discussion posts (incl. errors): 1
  generation_error posts in channel: 1
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

  FAIL [solo]: no BackendEngineer reply (timeout waiting for @BackendEngineer (baseline=1, counts={'BackendEngineer': 1, 'User': 1}))
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: Task 1: @BackendEngineer - Write collabs/e70ac84b-e47c-4b1c-bbce-52afc1f496fb/findings.md summarizing the  minimal fixture repo's README purpose and sample entr
  --- end ---
```

### collab:collab-generation-error-resilience [code]
```text
=== FAIL: collab-generation-error-resilience ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **Task List:** - Task 1: @BackendEngineer - Write  collabs/443203d8-3445-42c5-925d-930ba9e5c1b4/findings.md summarizing the minimal fixture repo structure from 
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
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=0/5
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
    [collaboration_discussion] SecurityReviewer: ## Security Review Planning Perspective ### Analysis Context  Before proceeding, I need to review the existing  files under `collabs/b222bffe-39e8-4b00-91ca-ee1
    [collaboration_discussion] FrontendEngineer: **FrontendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=1 counts={'SecurityReviewer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 2
  generation_error posts in channel: 1
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @SecurityReviewer — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 2
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/18
```

### collab:collaboration-station-website-sa [code]
```text
aboration-station-website-sa ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: **Architectural Perspective:** For a 3-page static site, I'd  recommend keeping it simple: single CSS file, semantic  HTML5, and clear URL hierarchy (e.g., `/` 
    [collaboration_discussion] FrontendEngineer: Based on the existing reference work under `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/`,  I can see this is a vanilla HTML/CSS  project with three pages alre
    [collaboration_discussion] FrontendEngineer: I notice the workspace context shows `collabs/` is  empty for this project. The system mentions reviewing  work under `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9
    [collaboration_discussion] Claude: Since I cannot access the file system with  my current tools, I'll proceed based on the  project context provided. Let me refine our task  list: **@Claude (me -
    [collaboration_discussion] FrontendEngineer: ## Task List for Collaboration Station Website -  **Task 1:** @SoftwareArchitect — Write `collabs/04efc7f4-1c48-4e54-b6b5-0724f6b3def7/site-structure.md` (navig
    [collaboration_discussion] FrontendEngineer: **Minimal Task List for Collaboration Station Website:** ```markdown  - Task 1: @SoftwareArchitect - Write collabs/04efc7f4-1c48-4e54-b6b5-0724f6b3def7/site-str
    [collaboration_discussion] FrontendEngineer: - Task 1: @SoftwareArchitect Write collabs/04efc7f4-1c48-4e54-b6b5-0724f6b3def7/site-structure.md (define fixed  navigation header, home/about/contact page hier
  --- end ---

agent discussion: total=7 counts={'SoftwareArchitect': 1, 'FrontendEngineer': 5, 'Claude': 1} (excluding generation_error)
  ok: @FrontendEngineer — 5 message(s)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 7
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/18
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
    [collaboration_discussion] SoftwareArchitect: **SoftwareArchitect** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] Claude: @SoftwareArchitect — I align on architecture-first approach: define  schema standards in `collabs/<id>/resource-api-schema-registration.md` before touching code
    [collaboration_discussion] BackendEngineer: I see @SoftwareArchitect has shared their architectural perspective.  Here's my Backend Engineer planning contribution: **Planning Perspective:**  1. **Schema A
  --- end ---

agent discussion: total=2 counts={'Claude': 1, 'BackendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 3
  generation_error posts in channel: 1
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @BackendEngineer — 1 message(s)
  ok: @Claude — 1 message(s)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/15
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-08-0107-iter1.log

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

