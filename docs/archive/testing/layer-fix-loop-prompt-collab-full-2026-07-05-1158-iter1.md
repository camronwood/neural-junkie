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

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-1158-iter1.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:collaboration-station-website-sa [code]
```text
=== FAIL: collaboration-station-website-sa ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description
  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description
  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

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
    [collaboration_discussion] Assistant: # Task List: Resource API Schema Standardization
```

### collab:solo-vs-collab-parity [code]
```text
ssion: total=1 counts={'Gemini': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Gemini — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
=== FAIL: resource-api-schema-planning ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: As @Gemini, my role is to draft the  markdown document from an approved outline provided by  the @Assistant. I will focus on converting the  synthesized schema 
  --- end ---

agent discussion: total=1 counts={'Gemini': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Gemini — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
=== FAIL: resource-api-schema-planning ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: # Resource API Document Schema Standardization/Registration Investigation ##  Workspace Limitation Notice The referenced files context contains  only a stub `pa
    [collaboration_discussion] BackendEngineer: ### Planning Perspective (BackendEngineer) Based on examining `resource-api/json_endpoints/`  and `docs/tim/`, here's my proposed task breakdown: -  **Task 1:**
    [collaboration_discussion] FrontendEngineer: ### Frontend Planning Perspective (3 Tasks) **Task 1:**  @Assistant — Write `collabs/b72d2fc1-8394-4a76-9f15-af7c033a2833/schema-consumption-patterns.md` listin
  --- end ---

=== FAIL: resource-api-schema-regression ===

  FAIL [solo]: no Assistant reply (timeout waiting for @Assistant (baseline=0, counts={}))
=== FAIL: solo-vs-collab-parity (solo leg) ===
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: - Task 1: @Assistant - Write `collabs/<id>/findings.md` summarizing  that this repo is a minimal Go fixture  using only `core/sample/main.go` with no external d
  --- end ---
```

### collab:collab-generation-error-resilience [code]
```text
=== FAIL: collab-generation-error-resilience ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: Task 1: @Assistant - Write `collabs/<id>/findings.md` summarizing README.md  contents and fixture purpose for scenario tests Task  2: @SoftwareArchitect - Write
  --- end ---
```

### collab:collab-human-planning-interject [code]
```text
=== FAIL: collab-human-planning-interject ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/ae287fcf-5768-4e95-8d1c-ceaf573405e1/readme-summary.md summarizing  README.md (lines 1-4). - Task 2: @BackendEngine
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/ae287fcf-5768-4e95-8d1c-ceaf573405e1/readme-summary.md from  README.md - Task 2: @BackendEngineer - Write collabs/a
  --- end ---

agent discussion: total=2 counts={'SoftwareArchitect': 2}
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 3
  pending file changes (hub): 1
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=2/4
```

### collab:collab-participation-two-agent-strict [code]
```text
=== FAIL: collab-participation-two-agent-strict ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={}
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
    [collaboration_discussion] Gemini: As the Implementation & Code agent, my planning  perspective is to efficiently translate the design and  architecture documents into functional web pages. Since
    [collaboration_discussion] FrontendEngineer: **Progress:** I'm reviewing the existing work in `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/`  to inform the new plan. The goal is  to design a "Collaboratio
    [collaboration_discussion] Gemini: I agree with the initial three planning tasks  proposed by @SoftwareArchitect and @FrontendEngineer, which provide the  essential blueprints for implementation.
    [collaboration_discussion] SoftwareArchitect: I've reviewed the existing work under `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/`. The  structure looks solid — static HTML/CSS without backend  dependencie
    [collaboration_discussion] FrontendEngineer: --- **Collaboration Station Project Review Complete** ✅ The  collaboration directory `collabs/54316a5c-da9e-4ec2-aa39-3957d43fa197` is empty and ready for  new 
  --- end ---
```

### collab:phoenix-resource-api-e2e [code]
```text
=== FAIL: phoenix-resource-api-e2e ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: **BackendEngineer** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: Let me first explore the workspace to understand  the context before providing my planning perspective. I'll  check: - Resource API schema endpoints and contrac
    [collaboration_discussion] PlatformEngineer: I agree with the three-file structure. From a  Platform perspective, here's my input: **My contribution:** For  `ci_cd_pipeline.md`, I'll outline GitHub Actions
    [collaboration_discussion] BackendEngineer: I agree with the current task allocation. Here's  my planning perspective: **Task 1 (api_schema.md):** I'll define  the OpenAPI contract including resource mode
    [collaboration_discussion] SoftwareArchitect: I'm ready to contribute. Before finalizing the plan:  **Current State**: This minimal-repo focuses on `resource-api/json_endpoints/` for  neural-junkie compress
    [collaboration_discussion] PlatformEngineer: I'll own the CI/CD pipeline deliverable in `ci_cd_pipeline.md`.  My plan: outline GitHub Actions workflows for build/test/deploy  stages with security scanning 
  --- end ---
```

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={}
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

agent discussion: total=0 counts={}
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
    [collaboration_discussion] Assistant: - Task 1: @BackendEngineer - Write collabs/api_schema.md with  API definitions - Task 2: @SoftwareArchitect - Write  collabs/standards.md for standards/guidelin
    [collaboration_discussion] SoftwareArchitect: @SoftwareArchitect — Planning Perspective Given this is an  API schema documentation effort under `collabs/b9cfe359-b4de-4b37-8575-7ba53d6854f7/`, my lane  prio
  --- end ---

agent discussion: total=2 counts={'Assistant': 1, 'SoftwareArchitect': 1}
  ok: @Assistant — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @SoftwareArchitect — 1 message(s)
  system turn handoffs in channel: 3
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/6
```

### collab:plan-phoenix-combined-regression [code]
```text
=== FAIL: plan-phoenix-combined-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: This is a collaboration planning request! Let me  draft a minimal task list for designing a  CLI file encryption tool: - Task 1: @SoftwareArchitect  - Write col
    [collaboration_discussion] SoftwareArchitect: @Assistant — Here's my architecture-focused planning perspective: Since  we're designing a small CLI file encryption tool  from scratch (no existing workspace),
  --- end ---
```

### collab:planning-two-agent [code]
```text
=== FAIL: planning-two-agent ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: As @Gemini, my primary role is to draft  the markdown document based on an approved outline.  Therefore, my planning perspective focuses on efficient implementa
  --- end ---

agent discussion: total=1 counts={'Gemini': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Gemini — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
```

### collab:resource-api-schema-planning [code]
```text
=== FAIL: resource-api-schema-planning ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: As @Gemini, my role is to draft the  markdown document from an approved outline provided by  the @Assistant. I will focus on converting the  synthesized schema 
  --- end ---

agent discussion: total=1 counts={'Gemini': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Gemini — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/5
```

### collab:resource-api-schema-regression [code]
```text
=== FAIL: resource-api-schema-regression ===

  FAIL [solo]: no Assistant reply (timeout waiting for @Assistant (baseline=0, counts={}))
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-1158-iter1.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression
- python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience
- python3 scripts/collab-scenarios.py --scenario collab-human-planning-interject
- python3 scripts/collab-scenarios.py --scenario collab-participation-two-agent-strict
- python3 scripts/collab-scenarios.py --scenario collaboration-station-website
- python3 scripts/collab-scenarios.py --scenario collaboration-station-website-sa
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

