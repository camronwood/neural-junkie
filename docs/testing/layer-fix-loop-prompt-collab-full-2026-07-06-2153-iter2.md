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

Release prep summary: /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-06-2153-iter2.md
Failed phases: collab-scenarios-all

## Failures to address

### collab:plan-distinct-deliverables-same-agent [code]
```text
=== FAIL: plan-distinct-deliverables-same-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] BackendEngineer: - Task 1: @BackendEngineer - Write collabs/8c4a1942-8d91-45f7-9184-54fe6bce2e4c/api_schema.md -  Task 2: @SoftwareArchitect - Write collabs/8c4a1942-8d91-45f7-9
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] SoftwareArchitect: I agree. Here's my perspective on the planning  for API schema documentation under `collabs/8c4a1942-8d91-45f7-9184-54fe6bce2e4c/`. - Task  1: @BackendEngineer 
    [collaboration_discussion] BackendEngineer: The provided code is written in Go, as  indicated by the package declaration on line 2  and the use of the `main` function on  line 4. The code defines a minima
  --- end ---

agent discussion: total=3 counts={'BackendEngineer': 2, 'SoftwareArchitect': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 6
  generation_error posts in channel: 3
  ok: @BackendEngineer — 2 message(s)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 23
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='converged' msgs=5/6
```

### collab:plan-findings-task-regression [code]
```text
=== FAIL: plan-findings-task-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] BackendEngineer: - Task 1: @BackendEngineer - Write collabs/27356b1a-519b-4820-8992-6ba189cb226a/api_schema.md -  Task 2: @SoftwareArchitect - Write collabs/27356b1a-519b-4820-8
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] SoftwareArchitect: - Task 1: @BackendEngineer - Write collabs/27356b1a-519b-4820-8992-6ba189cb226a/api_schema.md; -  depends: none - Task 2: @SoftwareArchitect - Write  collabs/27
    [collaboration_discussion] BackendEngineer: The file structure and referenced code suggest a  React application with TypeScript, using React components and  a theme context. The application includes compo
    [collaboration_discussion] SoftwareArchitect: The project uses **TypeScript** and **React** based on  the presence of `.tsx` files (e.g., `AccentButton.tsx`, `Header.tsx`,  `App.tsx`), `react-app-env.d.ts`,
  --- end ---

agent discussion: total=4 counts={'BackendEngineer': 2, 'SoftwareArchitect': 2} (excluding generation_error)
  raw discussion posts (incl. errors): 7
  generation_error posts in channel: 3
  ok: @BackendEngineer — 2 message(s)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 21
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='converged' msgs=5/6
```

### collab:plan-phoenix-combined-regression [code]
```text
=== FAIL: plan-phoenix-combined-regression ===


  --- transcript (agent messages) ---
  --- end ---

agent discussion: total=0 counts={} (excluding generation_error)
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
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] SoftwareArchitect:
```

### collab:solo-vs-collab-parity [code]
```text
s={'BackendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 4
  generation_error posts in channel: 3
  ok: @BackendEngineer — 1 message(s)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 22
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=4/10
=== FAIL: resource-api-schema-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] BackendEngineer: ```markdown # Task Analysis The task requires analyzing  the provided workspace context and creating a task  plan for defining and implementing the API document
  --- end ---

agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 4
  generation_error posts in channel: 3
  ok: @BackendEngineer — 1 message(s)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 24
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=4/10
=== FAIL: resource-api-schema-regression ===

  FAIL [solo]: no Assistant reply (timeout waiting for @Assistant (baseline=0, counts={}))
=== FAIL: solo-vs-collab-parity (solo leg) ===
```

### collab:collab-conversation-quality-regression [code]
```text
=== FAIL: collab-conversation-quality-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/837f040f-12d1-4618-8a6a-0c778b167082/readme-summary.md from  README.md. - Task 2: @BackendEngineer - Write collabs/
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/837f040f-12d1-4618-8a6a-0c778b167082/readme-summary.md from  README.md. - Task 2: @BackendEngineer - Write collabs/
  --- end ---

agent discussion: total=2 counts={'SoftwareArchitect': 2} (excluding generation_error)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 7
  pending file changes (hub): 3
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=2/4
```

### collab:collab-human-planning-interject [code]
```text
=== FAIL: collab-human-planning-interject ===

  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description
  FAIL: no collaboration redirect: {'status': 'ok'}
  hub: ❌ At least 2 agents must be @mentioned.
Usage: /collaborate [--rounds N] [--messages M] [--allow-agent-adds] @Agent1 @Agent2 description

  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] SoftwareArchitect: - Task 1: @BackendEngineer - Write collabs/<id>/requirements.md. -  Task 2: @BackendEngineer - Write collabs/<id>/handler.go. - Task  3: @SoftwareArchitect - Wr
    [collaboration_discussion] SoftwareArchitect: I agree with the current plan. It covers  the necessary steps to define the requirements, implement  the handler, and document the architecture for a  minimal h
  --- end ---

agent discussion: total=2 counts={'SoftwareArchitect': 2} (excluding generation_error)
  raw discussion posts (incl. errors): 4
  generation_error posts in channel: 2
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 20
  pending file changes (hub): 3
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=4/5
```

### collab:collab-participation-three-agent [code]
```text
=== FAIL: collab-participation-three-agent ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: .DS_Store (file) .neural-junkie (dir) .scenario-baseline (dir) README.md (file)  collabs (dir) core (dir) docker-compose.yml (file) file.md (file)  node_modules
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] SoftwareArchitect: I agree. Let's proceed with the plan for  the minimal health-check HTTP service. - Task 1:  @BackendEngineer - Write collabs/<id>/requirements.md. This document
    [collaboration_discussion] BackendEngineer: - Task 1: @BackendEngineer - Write collabs/health-check/requirements.md. -  Task 2: @BackendEngineer - Write collabs/health-check/handler.go. - Task  3: @Softwa
    [collaboration_discussion] SoftwareArchitect: @BackendEngineer, please proceed with writing the requirements document  for the health-check service. Ensure you cover all  functional and non-functional requi
  --- end ---

agent discussion: total=4 counts={'BackendEngineer': 2, 'SoftwareArchitect': 2} (excluding generation_error)
  raw discussion posts (incl. errors): 6
  generation_error posts in channel: 2
  ok: @BackendEngineer — 2 message(s)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 11
  pending file changes (hub): 3
  planning_discussion_ready=True phase='reviewing' discussion.status='converged' msgs=4/5
```

### collab:collab-participation-two-agent-strict [code]
```text
=== FAIL: collab-participation-two-agent-strict ===


  --- transcript (agent messages) ---
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/cli-log-filter/design.md with  API outline. - Task 2: @BackendEngineer - Write  collabs/cli-log-filter/filter.go im
  --- end ---

agent discussion: total=1 counts={'SoftwareArchitect': 1} (excluding generation_error)
  ok: @SoftwareArchitect — 1 message(s)
  FAIL: @BackendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 1
  pending file changes (hub): 2
  planning_discussion_ready=False phase='planning' discussion.status='active' msgs=1/4
```

### collab:collaboration-station-website [code]
```text
=== FAIL: collaboration-station-website ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: {"status":"proposed","path":"frontend_architecture_plan.md"}
    [collaboration_discussion] SecurityReviewer: No matching chunks found.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] FrontendEngineer: No matching chunks found.
    [collaboration_discussion] SecurityReviewer: I agree with the planned tasks and dependencies.  Let's proceed with the outlined plan: - Task  1: @FrontendEngineer - Write collabs/691ff64d-6e5b-4b69-ab17-c6e
    [collaboration_discussion] FrontendEngineer: ### Task 1: @FrontendEngineer Write collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/architecture-review.md comparing b222bffe  frontend_architecture_plan.md to the
  --- end ---

agent discussion: total=5 counts={'FrontendEngineer': 3, 'SecurityReviewer': 2} (excluding generation_error)
  raw discussion posts (incl. errors): 7
  generation_error posts in channel: 2
  ok: @FrontendEngineer — 3 message(s)
  ok: @SecurityReviewer — 2 message(s)
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 8
  pending file changes (hub): 1
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=2/18
```

### collab:collaboration-station-website-sa [code]
```text
=== FAIL: collaboration-station-website-sa ===


  --- transcript (agent messages) ---
    [collaboration_discussion] FrontendEngineer: ERROR: Error in read_file: open /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/website-prior-collab/collabs/b222bffe-39e8-4b00-91ca-ee1
    [collaboration_discussion] SoftwareArchitect: To design the Collaboration Station website, we should  start by defining the site structure and design  system. This will ensure that the frontend development 
    [collaboration_discussion] FrontendEngineer: No matching chunks found.
    [collaboration_discussion] SoftwareArchitect: @PlatformEngineer, in this collaboration, our focus is on  designing a website named Collaboration Station with a  black, white, gray, blue, and red color palet
    [collaboration_discussion] FrontendEngineer: ERROR: Error in list_dir: open /Users/camronwood/development/projects/neural-junkie/scenarios/fixtures/website-prior-collab/collabs/b222bffe-39e8-4b00-91ca-ee1c
  --- end ---

agent discussion: total=5 counts={'FrontendEngineer': 3, 'SoftwareArchitect': 2} (excluding generation_error)
  ok: @FrontendEngineer — 3 message(s)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 10
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


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] FrontendEngineer: {"status":"proposed","path":"contact.html"}
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] FrontendEngineer: - Task 1: @FrontendEngineer - Create a basic  HTML structure for `index.html`, `about.html`, and `contact.html` in  the `collabs/<id>/` directory. - Task 2: @Fr
  --- end ---

agent discussion: total=2 counts={'FrontendEngineer': 2} (excluding generation_error)
  raw discussion posts (incl. errors): 6
  generation_error posts in channel: 4
  ok: @FrontendEngineer — 2 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 15
  pending file changes (hub): 1
  planning_discussion_ready=True phase='reviewing' discussion.status='converged' msgs=3/12
```

### collab:phoenix-resource-api-e2e [code]
```text
act. Let's  break down the analysis for each language and  frame
    [collaboration_discussion] BackendEngineer: The workspace appears to be a React application,  utilizing TypeScript for type safety. The presence of  `tsx` files (like `AccentButton.tsx` and `Header.tsx`) 
    [collaboration_discussion] SoftwareArchitect: ### Analysis of Referenced Files #### core/sample/main.go (Go)  1. **Package Declaration**: - Line 1: `package main`  indicates that this is the main package, w
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] BackendEngineer: This workspace is for a React application, as  evidenced by the presence of `.tsx` files (e.g.,  `AccentButton.tsx`, `Header.tsx`, `App.tsx`) and the use of Typ
    [collaboration_discussion] BackendEngineer: I agree with the current plan. I will  focus on analyzing the Go code and markdown  file to understand their structure and purpose. I  will identify any issues,
    [collaboration_discussion] BackendEngineer: This workspace appears to be a React application.  It contains standard React components like `AccentButton.tsx` and  `Header.tsx`, as well as a context file `T
    [collaboration_discussion] BackendEngineer: ### Summary The provided code snippet is a  React component named `App` that is part of  a web application using React, Tauri, and Go.  The application includes
  --- end ---

agent discussion: total=9 counts={'SoftwareArchitect': 3, 'BackendEngineer': 6} (excluding generation_error)
  raw discussion posts (incl. errors): 12
  generation_error posts in channel: 3
  ok: @SoftwareArchitect — 3 message(s)
  ok: @BackendEngineer — 6 message(s)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 4
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='converged' msgs=3/15
```

### collab:plan-dependency-prose-regression [code]
```text
=== FAIL: plan-dependency-prose-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] BackendEngineer: ```json { "response": "Task understood. Creating files for  API documentation and code structure." } ```
    [collaboration_discussion] SoftwareArchitect: - Task 1: @SoftwareArchitect - Write collabs/8da16828-d253-47b0-b9ad-6efacdd80fa9/markdown_doc_structure.md detailing  the document structure and guidelines for
    [collaboration_discussion] BackendEngineer: The provided code is a simple Go program  with a `main` package and a `main` function.  It does not contain any specific logic or  dependencies beyond the stand
    [collaboration_discussion] SoftwareArchitect: The WORKSPACE CONTEXT indicates a React application with  TypeScript support. Key files and directories suggest the  use of React for building UI components and
    [collaboration_discussion] BackendEngineer: The provided code is written in Go, as  indicated by the `package main` declaration and the  use of the `func main()` function.
  --- end ---

agent discussion: total=5 counts={'BackendEngineer': 3, 'SoftwareArchitect': 2} (excluding generation_error)
  ok: @BackendEngineer — 3 message(s)
  ok: @SoftwareArchitect — 2 message(s)
  FAIL: @PlatformEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 9
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=1/4
```

### collab:resource-api-schema-regression [code]
```text
=== FAIL: resource-api-schema-regression ===


  --- transcript (agent messages) ---
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] Claude: **Claude** could not complete this turn: Sorry, I encountered an error while generating a response. Please try again.
    [collaboration_discussion] BackendEngineer: ```markdown # Task Analysis The task requires analyzing  the provided workspace context and creating a task  plan for defining and implementing the API document
  --- end ---

agent discussion: total=1 counts={'BackendEngineer': 1} (excluding generation_error)
  raw discussion posts (incl. errors): 4
  generation_error posts in channel: 3
  ok: @BackendEngineer — 1 message(s)
  FAIL: @FrontendEngineer — no collaboration_discussion (silent or shouldRespond blocked)
  FAIL: @Claude — no collaboration_discussion (silent or shouldRespond blocked)
  system turn handoffs in channel: 24
  pending file changes (hub): 0
  planning_discussion_ready=True phase='reviewing' discussion.status='timed_out' msgs=4/10
```

## Child artifacts (read for full context)
- /Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-collab-full-2026-07-06-2153-iter2.log

## Targeted verification (run after your fixes)
- python3 scripts/collab-scenarios.py --scenario collab-conversation-quality-regression
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

