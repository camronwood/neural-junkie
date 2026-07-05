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

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-1158-iter3.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:phoenix-resource-api-e2e [code]
```text
=== FAIL: phoenix-resource-api-e2e ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: { "response": "The task involves creating three markdown  files in the 'collaboration scenario fixtures' directory: `api.md`,  `markdown.md`, and `file.md`. Eac
    [collaboration_discussion] BackendEngineer: The provided code snippet is written in Go,  as indicated by the package declaration and the  `main` function. This is a minimal Go program  that does not conta
    [collaboration_discussion] SoftwareArchitect: The project is primarily built with **JavaScript** using  the **React** framework. The file tree includes React  components (`.tsx` files) such as `App.tsx`, `A
    [collaboration_discussion] BackendEngineer: The provided code snippet is written in Go,  as indicated by the `package main` declaration and  the use of the `func` keyword for defining  the `main` function
    [collaboration_discussion] BackendEngineer: Sure, here is my planning perspective: - **API  Design**: I will focus on defining a clear  and efficient API schema that aligns with our  business requirements
  --- end ---

agent discussion: total=5 counts={'BackendEngineer': 4, 'SoftwareArchitect': 1}
  ok: @BackendEngineer — 4 message(s)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 8
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=3/4
```

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect:
```

### collab:solo-vs-collab-parity [code]
```text
resource-api-schema-planning ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: ### Proposed Task List (Resource API Schema Standardization  Investigation) ```markdown 1. @Assistant - Write collabs/api-investigation/001-scope.md defining  A
  --- end ---

agent discussion: total=1 counts={'Assistant': 1}
  ok: @Assistant — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/10
=== FAIL: resource-api-schema-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: ### Task 1: Define Scope @Assistant - Write  collabs/58240560-c516-477b-a533-f4d4d465cacf/scope.md summarizing the goals and constraints of the  resource API do
    [collaboration_discussion] FrontendEngineer: I agree with the current plan. Each task  is clearly defined with a deliverable, and the  dependencies are logically structured. Let's move forward with  this p
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=3 counts={'FrontendEngineer': 2, 'Assistant': 1}
  generation_error posts in channel: 1
  ok: @Assistant — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @FrontendEngineer — 2 message(s)
  system turn handoffs in channel: 6
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/10
=== FAIL: resource-api-schema-regression ===

  FAIL [solo]: no Assistant reply (timeout waiting for @Assistant (baseline=0, counts={}))
=== FAIL: solo-vs-collab-parity (solo leg) ===
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: - Task 1: @Assistant - Write `collabs/<id>/findings.md` summarizing  README.md: "Minimal fixture repo for collab scenario tests,  used by execute-deliverable an
    [collaboration_discussion] SoftwareArchitect: This workspace appears to be a React application  with TypeScript support, indicated by the presence of  `.tsx` files (e.g., `AccentButton.tsx`, `Header.tsx`) a
    [collaboration_discussion] SoftwareArchitect: The provided WORKSPACE CONTEXT indicates a project built  with React and TypeScript. The presence of `.tsx`  files like `AccentButton.tsx` and `Header.tsx` conf
    [collaboration_discussion] SoftwareArchitect: The workspace is a React application, as evidenced  by the presence of `.tsx` files (`AccentButton.tsx`, `Header.tsx`,  `App.tsx`), TypeScript declaration files
  --- end ---
```

### collab:collab-generation-error-resilience [code]
```text
=== FAIL: collab-generation-error-resilience ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: Sure, here is my planning perspective: We need  to define the minimal requirements for the health-check  HTTP service, design the API handler, and outline  the 
    [collaboration_discussion] SoftwareArchitect: I agree with the goal of planning a  minimal health-check HTTP service. For the architecture, we  need to define clear API boundaries and document  the expected
  --- end ---

agent discussion: total=2 counts={'BackendEngineer': 1, 'SoftwareArchitect': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @BackendEngineer — 1 message(s)
  ok: @SoftwareArchitect — 1 message(s)
  system turn handoffs in channel: 1
  pending file changes (hub): 1
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/5
```

### collab:collab-participation-three-agent [code]
```text
=== FAIL: collab-participation-three-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: ### Planning Perspective #### @Assistant - Task 1:  @Assistant - Write collabs/<id>/requirements.md. - Define the scope,  objectives, and expected behavior of t
    [collaboration_discussion] Assistant: Here's a minimal task list for the health-check  HTTP service: ```markdown - Task 1: @Assistant -  Write collabs/<id>/requirements.md - Define endpoints, respon
  --- end ---

agent discussion: total=2 counts={'SoftwareArchitect': 1, 'Assistant': 1}
  ok: @Assistant — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @SoftwareArchitect — 1 message(s)
  system turn handoffs in channel: 5
  pending file changes (hub): 1
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/5
```

### collab:collab-participation-two-agent-strict [code]
```text
=== FAIL: collab-participation-two-agent-strict ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: No matching chunks found.
    [collaboration_discussion] Gemini: The directory `collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/` containing the existing `frontend_architecture_plan.md`, HTML,  and CSS is not found. This means t
    [collaboration_discussion] FrontendEngineer: No matching chunks found.
    [collaboration_discussion] SecurityReviewer: Thank you for sharing the project workspace. To  continue the plan discussion and refine task assignments,  I'll first review the available files in the  worksp
    [collaboration_discussion] SecurityReviewer: 368e2270-81f7-4658-9e5f-893ea151c284 (dir)
  --- end ---
```

### collab:collaboration-station-website [code]
```text
=== FAIL: collaboration-station-website ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: **Gemini** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] FrontendEngineer: No matching chunks found.
    [collaboration_discussion] Gemini: **Gemini** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] SoftwareArchitect: ERROR: Error in list_dir: open /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/website-prior-collab/collabs/b222bffe-39e8-4b00-91ca-ee1c
    [collaboration_discussion] FrontendEngineer: ERROR: Error in read_file: open /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/website-prior-collab/collabs/b222bffe-39e8-4b00-91ca-ee1
    [collaboration_discussion] SoftwareArchitect: ### collabs/13b06cd9-23f0-4076-9773-78ce68b25231/plan.md ``` # Plan **Collaboration:** Design a  website named Collaboration Station **Goal:** @FrontendEngineer
    [collaboration_discussion] FrontendEngineer: @FrontendEngineer, @SoftwareArchitect, @DevOpsEngineer: It looks like the project  workspace is currently empty. Let's finalize the task  assignments based on t
    [collaboration_discussion] SoftwareArchitect: ### collabs/13b06cd9-23f0-4076-9773-78ce68b25231/plan.md ``` # Plan **Collaboration:** Design a  website named Collaboration Station **Goal:** @FrontendEngineer
  --- end ---
```

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
    [collaboration_discussion] SoftwareArchitect: Sure, I'll provide a concise plan for the  collaboration to investigate resource API document schema standardization/registration.  ### Task List 1. **Task 1: @
    [collaboration_discussion] BackendEngineer: @Assistant — I will facilitate the goal synthesis,  task structuring, and ensure the collaboration stays on  track. I will also handle any meta-work or  summary
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
    [collaboration_discussion] SoftwareArchitect: @BackendEngineer - I will design the service/API handlers,  data access patterns, and backend contracts to ensure  the API design is efficient, scalable, and ad
  --- end ---
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
    [collaboration_discussion] SoftwareArchitect: Sure, let's start with the planning perspective for  this collaboration: **@Assistant's Perspective:** - Task 1: @BackendEngineer  - Write collabs/5b39c7d9-510a
    [collaboration_discussion] Assistant: - Task 1: @BackendEngineer - Write collabs/<id>/api_schema.md -  Task 2: @SoftwareArchitect - Write collabs/<id>/standards.md - Task  3: @Assistant - Write coll
  --- end ---

agent discussion: total=2 counts={'SoftwareArchitect': 1, 'Assistant': 1}
  ok: @Assistant — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @SoftwareArchitect — 1 message(s)
  system turn handoffs in channel: 0
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/6
```

### collab:plan-phoenix-combined-regression [code]
```text
=== FAIL: plan-phoenix-combined-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=1 counts={'Assistant': 1}
  generation_error posts in channel: 1
  ok: @Assistant — 1 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 0
  pending file changes (hub): 1
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=0/4
```

### collab:planning-two-agent [code]
```text
=== FAIL: planning-two-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: - Task 1: @Assistant - Write collabs/cli-encryption/design.md outlining  the CLI tool's requirements and high-level design. -  Task 2: @SoftwareArchitect - Revi
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---
```

### collab:resource-api-schema-planning [code]
```text
=== FAIL: resource-api-schema-planning ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Gemini: As @Gemini, my role is to draft the  markdown document from an approved outline. My planning  perspective centers on receiving a clear, standardized outline  an
  --- end ---

agent discussion: total=1 counts={'Gemini': 1}
  FAIL: @Assistant — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @Gemini — 1 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 2
  pending file changes (hub): 1
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/5
```

### collab:resource-api-schema-regression [code]
```text
=== FAIL: resource-api-schema-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: ### Task 1: Define Scope @Assistant - Write  collabs/58240560-c516-477b-a533-f4d4d465cacf/scope.md summarizing the goals and constraints of the  resource API do
    [collaboration_discussion] FrontendEngineer: I agree with the current plan. Each task  is clearly defined with a deliverable, and the  dependencies are logically structured. Let's move forward with  this p
    [collaboration_discussion] Assistant: **Assistant** could not complete this turn: Sorry, the response timed out before completion. Please try again.
  --- end ---

agent discussion: total=3 counts={'FrontendEngineer': 2, 'Assistant': 1}
  generation_error posts in channel: 1
  ok: @Assistant — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @FrontendEngineer — 2 message(s)
  system turn handoffs in channel: 6
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/10
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-05-1158-iter3.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression
- python3 scripts/collab-scenarios.py --scenario collab-generation-error-resilience
- python3 scripts/collab-scenarios.py --scenario collab-participation-three-agent
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

