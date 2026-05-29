# Multi-Agent Collaboration

Neural Junkie now supports structured multi-agent collaboration so agents can discuss, review, delegate, and execute work together under user control.

This is different from lightweight `@mention` review flow: collaboration introduces bounded discussions, shared artifacts, explicit phases, and task tracking.

## Goals

- Enable agent-to-agent discussions in a controlled mode
- Let users assign multiple agents to a shared objective
- Require user approval before execution starts
- Delegate tasks by agent strengths (type + expertise)
- Prevent runaway conversations with hard limits

## Smart routing (execution tasks)

When **Collaboration smart routing** is enabled (Desktop **Settings → AI Providers**), the hub may choose a different **configured** provider for each `collaboration_task` message (after workspace ack) using a static heuristic (for example cheaper local models for short wording-style tasks, higher-tier models for security-related text, and the lowest-cost configured tier when the task includes user images and the assignee supports vision). Normal chat and per-agent defaults are unchanged.

**In-process specialists only:** separate specialist processes (`make agents` / `cmd/agent`) do not load hub multi-provider routing in v1.

## Two paths: collaborate vs runbook

| Path | Entry | Who defines tasks | Execution |
|------|--------|-------------------|-----------|
| **Collaborate** | `/collaborate` | Agents during bounded planning discussion | Hub orchestrates task DAG after `/approve-plan` |
| **Runbook** | Desktop **RB** button, `/runbook`, or `POST /api/runbooks` | You in the Runbook builder (or import markdown) | Same orchestrator after **Start execution** |

Both paths share the same execution engine: dependency-aware waves, workspace ack, file-change approvals, and sandbox/worktree modes.

## Task dependencies (DAG orchestration)

During planning, agents can declare dependencies on task lines:

```markdown
- Task 1: @BackendEngineer - Scaffold API changes
- Task 2: @SecurityReviewer - Threat model
- Task 3: @CodeReviewer - Integration tests
  - depends: 1, 2
```

After approval, the hub dispatches only **ready** tasks (all dependencies completed). When a task completes, the next wave unlocks automatically. Upstream task output is included in downstream `collaboration_task` prompts.

Task statuses during execution: `pending`, `in_progress`, `completed`, `blocked`. The desktop collaboration panel shows **Waiting on**, **Ready**, or **Dispatched** per task.

## Runbook builder (desktop)

1. Click **RB** in the chat toolbar or run `/runbook @Agent1 @Agent2 goal description`.
2. In the Runbook builder: define tasks, assign agents (or **Auto** suggest), set **Depends on** checkboxes.
3. **Import from markdown** (parse-plan API) or **Save draft**.
4. **Submit for review** → **Start execution** (same workspace gate as `/collaborate`).

### Graph view (task DAG)

Open the visual task graph from:

- **Runbook builder** — **Graph** (draft/reviewing): drag nodes, connect edges (upstream → downstream = dependency), edit title/assignee in the inspector, **Auto-layout**, **Save & close** (same as Save draft).
- **Collaboration panel** — **View graph** (any collaboration with tasks, including `/collaborate` and executing runbooks): read-only topology with live status; pan/zoom and optional node drag for layout only.

Edges mean “target waits for source.” Cycles are rejected before save (client and server). Layout positions are stored on the collaboration (`graph_layout` via PUT runbook) with `localStorage` fallback.

### Execution policy (runbook / collaboration)

| Field | Values | Effect |
|-------|--------|--------|
| `blocked_upstream_policy` | `block` (default), `skip_branch`, `fail_run` | How blocked upstream tasks affect downstream readiness |
| `max_concurrent_tasks` | 0 = unlimited | Cap prompts dispatched per wave |
| `max_execution_messages` | default 100 | Agent chat cap during execution |
| `strict_task_status` | default true for runbooks | Only `TASK_STATUS:` lines update task state (no keyword guessing) |

### Task controls (executing)

Desktop collaboration panel and REST:

- `POST /api/collaborations/:id/tasks/:taskId/complete` — mark done and unlock DAG wave
- `POST .../skip`, `.../redispatch`, `.../reassign`, `.../approve`
- `POST /api/collaborations/:id/pause` / `resume` — pause dispatch without cancelling

### Stop vs pause dispatch vs `/pause-agent`

| Control | Where | Effect |
|---------|--------|--------|
| **Stop** (typing bar) | Any channel while agents are thinking/streaming | Cancels in-flight generations, holds the channel until you send a message (`POST /api/channels/:channel/interject`) |
| **Pause dispatch** | Collaboration panel (executing) | Stops new task waves only; agents already replying continue |
| **`/pause-agent`** | Slash command | Marks an agent paused in the roster; does not abort an active LLM call |

See [RUNBOOK_ACTIONS.md](RUNBOOK_ACTIONS.md) for **action** task types (HTTP, webhook, web search, SMS).

### Runbook HTTP API

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/runbooks` | Create draft runbook |
| GET | `/api/runbooks/:id` | Snapshot |
| PUT | `/api/runbooks/:id` | Update draft/reviewing |
| POST | `/api/runbooks/:id/suggest-assignee` | Auto-assign heuristic |
| POST | `/api/runbooks/:id/parse-plan` | Import tasks from markdown |
| POST | `/api/runbooks/:id/submit` | `draft` → `reviewing` |
| POST | `/api/runbooks/:id/start` | Approve and begin execution |

## Commands

### Start a runbook (user-built DAG)

```text
/runbook @BackendEngineer @SecurityReviewer ship the auth refactor
```

Creates a collaboration in **draft** phase. Use the desktop Runbook builder to add tasks and dependencies, then submit and start.

### Start a collaboration

```text
/collaborate @SoftwareArchitect @SecurityReviewer @Cursor design a secure file-encryption workflow
```

Optional limits (must appear **before** the first `@mention`; omitted values use defaults **2** rounds and **12** agent messages, then the server clamps to hard caps):

```text
/collaborate --rounds 5 --messages 40 @SoftwareArchitect @SecurityReviewer design the auth flow
```

Optional: attach a **high-level** view of your open editor workspace (file tree only, no open tab bodies) for repo-wide planning:

```text
/collaborate --workspace @SoftwareArchitect @CodeReviewer review the layout of this repo

**Desktop command form** — pick **Active workspace** (default), **Choose folder**, or **No workspace** (research-only; adds `--no-workspace`). Choose folder sends `--repo /path` and binds that repo instead of whatever is open in the editor.

/collaborate --no-workspace @Assistant @SoftwareArchitect compare API schema standards (no repo)

/collaborate --repo /Users/you/projects/phoenix @BackendEngineer @SecurityReviewer audit auth middleware
```

Optional: run execution in a **git worktree** (isolated branch + full repo copy) instead of an empty sandbox:

```text
/collaborate --worktree --workspace @BackendEngineer @SecurityReviewer refactor the auth middleware
```

Without `--workspace`, planning uses only the collaboration goal — agents are not given your open project files. This avoids general questions (for example “who is the better Rust programmer?”) being interpreted as questions about whatever repo you had open.

### Git worktree execution (`--worktree`)

When you pass `--worktree`, approved plans execute in a **git worktree** under your collaboration assets folder (`<assets-root>/worktrees/<collab-id>/`) on branch `nj/collab-<id-prefix>`. Your main checkout is not modified.

| Flag / step | Behavior |
|---|---|
| `--worktree --workspace` | Binds the source repo from message `workspace_path` at create time |
| `--worktree` only | Source repo is chosen at the desktop **Continue** gate from the **active workspace** (must be a git repo) |
| `/approve-plan` | Creates the worktree when the source repo is already known; otherwise waits for gate |
| **Continue** (desktop) | Sends `source_repo_path` if needed, creates worktree, registers it as a workspace, acks the hub |
| Complete / cancel | Worktree directory is removed (branch is kept for you to merge) |

Merge from your main checkout when done, for example: `git merge nj/collab-abc12345`.

Creates a collaboration in `planning` phase and starts a bounded discussion.

**Dedicated channel:** On success the hub auto-creates a channel named `collab-<collaboration-uuid>` (type `collaboration`). Seeds, agent discussion, plan updates, and execution tasks for that session are isolated **there**. Your `/collaborate` line stays in the channel where you typed the command (for example `#general`). The desktop app switches to the new room after send and lists it under **Collaborations** in the sidebar; **Open collaboration** in the task panel also jumps to that channel.

### Submit plan for review (end planning early)

```text
/submit-plan <collab-id>
```

Moves `planning` → `reviewing` when you are ready (without waiting for consensus or discussion limits). Agent planning discussion stops; a facilitator posts the pre-approval session summary. Desktop: **Submit for review** in the collaboration panel.

### Approve plan and execute

```text
/approve-plan <collab-id>
```

Moves collaboration from `reviewing` -> `approved` -> `executing`, creates the on-disk collaboration workspace (`collabs/<id>/` under your project when a source workspace is bound), and lists assigned tasks in chat. Desktop: **Approve & start** (same as `/resume-plan` while reviewing).

**Sandbox + bound project repo (`--workspace` / `--repo`):** approving the plan **auto-confirms the workspace** and dispatches `collaboration_task` prompts immediately (no separate **Continue** step).

**Worktree or no bound repo:** task prompts are held until you confirm the workspace (desktop **Continue** on that channel, or `/ack-collab-workspace <collab-id>`).

On approve, the hub also **normalizes** the task list (drops vague/duplicate lines, caps at 10 tasks) and shows **plan validation** warnings in chat and the collaboration panel.

### Confirm collaboration workspace (after approve)

```text
/ack-collab-workspace <collab-id>
```

Marks the execution sandbox as ready and delivers `collaboration_task` messages to assignees. The desktop app calls the same step via HTTP when you click **Continue** in the gate dialog.

### Revise plan

```text
/revise-plan <collab-id> <feedback>
```

Returns collaboration to planning with user feedback and starts a new bounded discussion round.

### Cancel collaboration

```text
/cancel-plan <collab-id>
```

Moves collaboration to `cancelled`.

### Mark collaboration complete

```text
/complete-collab <collab-id>
/complete-collab <collab-id> --force
```

Closes a collaboration in phase `completed`, stops execution Q&A (`discussion` → `converged`), and broadcasts a status message.

- Without `--force`, open tasks (pending / in_progress / blocked) block completion; the hub lists them and asks you to confirm with `--force`.
- With `--force`, any open tasks are marked `completed` (output: `Closed by user`) and the collaboration finalizes.

Desktop: **Mark collaboration done** in the collaboration panel or task management drawer (confirm when tasks are still open).

### Mark one task complete

```text
/collab-task-done <collab-id> <task#>
```

Marks a single task by **1-based** index (as shown in the plan) or task ID prefix. When that was the last open task, the collaboration auto-completes.

### Inspect status

```text
/collab-status
/collab-status <collab-id>
```

Shows active collaborations or details for one collaboration.

## Lifecycle

1. **planning**
   - Agents discuss in bounded rounds.
   - Shared artifact (`plan`) is updated.
   - Tasks are proposed and assigned.
2. **reviewing**
   - Plan is presented to the user.
   - The **last agent who spoke** in planning delivers a **session summary** recap to you (research-only sessions included).
   - **`/approve-plan` and Approve & start are blocked** until that recap is posted (or the hub times out and posts a fallback summary).
   - User approves, revises, or cancels.
3. **approved**
   - Transitional state after `/approve-plan` (before execution starts).
   - If execution fails to start, the collaboration can remain here; use **Start execution** in the UI or run `/approve-plan <id>` again (the server treats a second approve as a no-op and retries the transition to `executing`).
4. **executing**
   - Agents receive their assigned tasks (`collaboration_task` messages). Each message carries `task_assigned_to` so the assignee responds even when execution-phase discussion turn order would not otherwise select them.
   - Tasks are parsed from the plan markdown when possible (structured list lines or task headings with `@Agent` mentions). If **no** tasks are found when execution starts, the hub creates **one default pending task per participant** (goal + plan excerpt) so work still fans out to every agent.
   - Progress is tracked per task.
   - Bounded cross-agent Q&A remains available.
5. **completed** or **cancelled**
   - **Automatic:** when every task is `completed` (agents report `TASK_STATUS: completed`, plan handoff lines like `Task N — Complete`, or `/collab-task-done`).
   - **Manual:** `/complete-collab` or UI **Mark collaboration done**.
   - Before the collaboration is marked **completed**, the **last speaker** during execution (fallback: planning recap agent, then first participant) posts a **final session summary** — accomplishments, research, open questions, and next steps — even when no files were changed.

### Session recaps (two checkpoints)

| When | Who | What you see |
|------|-----|----------------|
| Planning → **reviewing** | Last planning speaker | Pre-approval summary in the collab channel; pinned in the collaboration panel. Unlocks `/approve-plan`. |
| All tasks done / **complete-collab** | Last execution speaker | Final summary in channel + completion status; panel shows `session_recap`. |

If an agent does not respond within ~90 seconds, the hub generates a fallback summary so sessions do not hang.

### Task status signaling (execution)

Assignees should end execution replies with a machine-readable line when work is done:

```text
TASK_STATUS: completed
```

Optional: `TASK_STATUS: blocked` with a short reason. The hub also infers completion from plan handoff lines (`Task 2 (@Agent) — Complete`) during execution.

## Files, workspace, and approvals during execution

Agents create or edit files by emitting **`[FILE_CHANGE]`** blocks in their replies (the same machine-readable format as normal channels). **Plain discussion or markdown code fences alone do not write to disk** until a proposal is emitted and you approve it in the desktop **Pending changes** flow.

- **Workspace sharing:** The hub resolves file proposals against the **workspace path** carried in message metadata. Collaboration traffic usually happens in `collab-…` channels, while your IDE workspace is often attached to messages on `#general` or your project channel. The app now **falls back** to the most recent `workspace_context` from other channels the agent has seen, so proposals still register when you had sharing enabled from the project window.
- **Paths must stay under the shared root:** Requests like “write under `~/development/test-site-001`” only work if that directory is the shared workspace (or added as a workspace) and sharing is on. Otherwise agents should explain the limitation and ask you to add that folder or use paths **relative** to the current workspace root.
- **Collaboration sandbox (default):** When a plan is approved and execution starts, the hub creates `<assets-root>/<collaboration-id>/` (default assets root: `~/.neural-junkie/collaborations`), attaches it as `workspace_context` on `collaboration_task` messages **after you confirm**, and snapshots expose `working_directory` plus `workspace_acknowledged`.
- **Git worktree (`--worktree`):** Execution uses `<assets-root>/worktrees/<collaboration-id>/` on branch `nj/collab-<prefix>`. Agents receive a real project tree in `workspace_context`. Source repo binding uses `--workspace` at start and/or the desktop active workspace at **Continue** (`POST /api/collaboration-workspace-ack` with optional `source_repo_path`).
- Set the parent folder in **Settings → AI Providers → Collaboration output folder** (`collaboration.assets_root` in `config.json`) or via `NEURAL_JUNKIE_COLLAB_ASSETS_DIR`. **Agents do not receive task prompts until you confirm:** use the desktop **Continue** dialog on the collaboration channel, or run `/ack-collab-workspace <id>`.
- **Shell commands:** During execution, agents should put runnable commands in fenced **bash** code blocks; the desktop surfaces **Run** and passes the collaboration sandbox as the working directory when executing suggestions.

## One executing collaboration per channel

The server enforces **at most one collaboration in `executing` phase per chat channel**. Other phases (`planning`, `reviewing`, `approved`, etc.) can overlap across collaborations; the constraint applies when work actually moves into execution.

When a collaboration transitions into `executing` (after `/approve-plan` and the hub’s transition to execution), `CollaborationManager.TransitionToExecuting` **automatically cancels** any other collaboration on the **same channel** that was already `executing`. That collaboration’s execution discussion is set to cancelled so only the new run remains active.

The desktop app warns before the user would implicitly stop a run:

- **Approve / resume from UI** (`CollaborationPanel` or task management): if another collaboration in the current channel is already executing, a **confirm** dialog names both collaborations and explains that continuing **stops the current run** and proceeds with the selected plan.
- **`/collaborate` in the composer**: if something is already executing in the channel, a **confirm** explains that you can still start a new plan, and that **when you approve the new plan**, the current execution will be stopped so only one collaboration runs at a time.

Other channels are unaffected: two collaborations can execute **in parallel** on different channels (including two different auto-created `collab-…` rooms).

## Bounded Discussion Safeguards

All discussions enforce hard caps:

| Safeguard | Default | Hard Max |
|---|---:|---:|
| Max rounds | 2 | 10 |
| Max turns per agent per round | 1 | 3 |
| Max total messages | 12 | 50 |
| Wall-clock timeout | 5 min | 30 min |
| Max concurrent collaborations | 3 | n/a |
| Max tasks per collaboration | 10 | n/a |

When a bound is reached, the discussion is ended and the system keeps what was produced.

## Delivery checklist (quality outcomes)

Use this when you want **files and finished tasks**, not a long planning thread:

1. **Agent mix** — match specialists to the work (architect for schema docs, backend for code, assistant for summaries). **Any** assignee can ship files via `[FILE_CHANGE]` in the IDE — you do not need a CLI agent for markdown or code tasks.
2. **Task shape** — **3–6** lines like `- Task 1: @SoftwareArchitect - Write collabs/<collab-id>/schema.md …` (verb + path). Avoid meta lines (“document findings”, “specific actions”).
3. **Workspace** — use `--workspace` (or runbook with a bound repo) so deliverables land under `collabs/<collab-id>/` in your project.
4. **Approve** — sandbox + bound repo: tasks dispatch on approve. Worktree: approve, then **Continue**.
5. **Files** — agents must emit **`[FILE_CHANGE]`**; approve proposals in **Pending changes**. `TASK_STATUS: completed` alone does not write to disk; file-shaped tasks may stay **in progress** until a proposal or existing file is detected.

**Harness:** `make collab-scenario SCENARIO=delivery-sandbox-auto-ack` (auto-ack on approve), `execute-deliverable` (end-to-end file), `resource-api-schema-planning` (plan quality).

## Collaboration Data Model

Implemented in `internal/collaboration/`:

- `Collaboration`
  - ID, title, description, phase
  - participants (`CollaborationAgent`)
  - plan artifact (`SharedArtifact`)
  - task list (`CollaborationTask`)
- `DiscussionSession`
  - round-robin turn tracking
  - per-round budgets
  - total message and timeout enforcement
- `SharedArtifact`
  - versioned markdown content
  - edit history (`ArtifactEdit`)
- `ConsensusState`
  - per-agent: `undecided` / `agrees` / `disagrees`

## How Agent-to-Agent Messaging Works

Outside collaboration mode, the anti-loop guard still prevents agents from replying to other agents by default.

Inside collaboration mode:

- `shouldRespond()` allows responses to agent messages if:
  - the agent is a participant, and
  - it is their turn, or
  - they are explicitly @mentioned
- discussion budget checks still apply

This preserves safety while enabling real collaboration.

## Prompt Behavior in Collaboration

When `collaboration_id` is present in message metadata, prompts include:

- collaboration goal and current phase
- participant roles and strengths
- shared plan artifact content/version
- explicit collaboration instructions:
  - build on others' ideas
  - @mention participants when needed
  - signal agreement/disagreement clearly

## Consensus Detection

Consensus is tracked using:

- **Signal-based checks** (e.g., "I agree", "I have concerns")
- **Heuristic checks** (e.g., all agents responded with no substantive changes)

If all agents agree, collaboration can move to user review.
If disagreement persists at discussion limits, the system escalates decision-making back to the user.

## Frontend Support

Desktop updates include:

- new message types for collaboration events
- collaboration badges on collaboration messages
- `CollaborationPanel` showing:
  - phase
  - participants + roles
  - task status/progress
  - plan artifact
  - approve/revise/cancel controls
- confirmation when approving or resuming would replace another collaboration already executing in the channel; same idea when sending `/collaborate` while one is executing

## Agent lanes (minimize overlap)

During `/collaborate`, each participant gets **YOUR LANE** and **PEER LANES** in the system prompt: what they own, what to defer, and what to avoid. Planning tasks should have **one primary assignee** per deliverable.

| Agent (examples) | Owns | Defers to |
|------------------|------|-----------|
| **Assistant** | Goal clarity, plan synthesis, task sequencing, assigned summary files | Deep schema design → architect when not assigned |
| **SoftwareArchitect** | API/schema shape, registration approach, doc standards, assigned schema markdown | App code → backend when not assigned; CI/deploy → platform |
| **PlatformEngineer** | CI/CD, packaging, deploy, observability for artifacts | Schema/doc narrative → architect when not assigned |
| **Gemini** (CLI) | Same as other agents for assigned deliverables; optional for external CLI tooling | Architecture → architect when not assigned |

**PlatformEngineer** and **SoftwareArchitect** both belong in the same collab when the goal spans **design + delivery** — they should not duplicate each other's tasks. The harness scenario `resource-api-schema-planning` encodes your original prompt with explicit lanes.

## Live scenario harness (dev loop)

For tuning multi-agent **conversation** and planning quality with **real Ollama agents** (not mocks), use JSON scenarios and a live hub.

**Prerequisites**

- Hub running (`make gui` or `make server`)
- Ollama serving models used by your agents
- Fewer than three active collaborations (runner auto-cancels prior runs on `collab-scenarios`)

**Commands**

```bash
# List scenarios
python3 scripts/collab-scenarios.py --list

# Fast iteration (7b agents)
make collab-scenario SCENARIO=planning-two-agent

# Heavier agents
make collab-scenario SCENARIO=planning-two-agent PROFILE=realistic

# All planning + isolation scenarios
make collab-scenarios

# Execution + fixture repo (slow; needs task completion)
make collab-scenario SCENARIO=execute-deliverable PROFILE=fast

# Matrix sweep before/after prompt changes
make collab-scenario-matrix
```

**Scenarios** live under `scenarios/collab/*.json`. Each file defines agents (or uses `PROFILE`), a `/collaborate` goal, ordered **steps** (`wait_phase`, `wait_discussion`, `assert_messages`, `assert_plan`, `approve_file_changes`, …), and `cleanup`.

Execution scenarios use **`approve_file_changes`** with **`require_hub_approval`** to apply pending hub file proposals (`POST /api/file-changes/approve/{id}`) before `assert_files`. **`assert_files`** can require grounded content (`any_match` / `none_match` on file body) and reject `TASK_STATUS` lines in deliverables. Set **`NJ_SCENARIO_ALLOW_FILE_FALLBACK=1`** only for local debugging when agents omit canonical `[FILE_CHANGE]` proposals.

**Scenarios**

| Name | Purpose |
|------|---------|
| `planning-two-agent` | Fast planning quality (2 tasks, reviewing) |
| `multi-collab-isolation` | Planning while another collab executes |
| `reject-collabs-subfolder` | Workspace guard for `collabs/<uuid>` paths |
| `execute-deliverable` | Fixture repo execution + grounded `findings.md` |
| `resource-api-schema-planning` | Your resource API schema prompt (`@Assistant @Gemini @PlatformEngineer`) |
| `delivery-sandbox-auto-ack` | Approve on `--workspace` sandbox auto-acks and dispatches tasks (no `workspace_ack` step) |

**Environment**

- `NEURAL_JUNKIE_HUB_URL` — default `http://127.0.0.1:18765`
- `NJ_SCENARIO_PROFILE` — `fast` (`@ChatModerator @Assistant`) or `realistic` (`@SoftwareArchitect @BackendEngineer`)
- `NJ_COLLAB_SCENARIO_AGENTS` — override mentions
- `NEURAL_JUNKIE_SCENARIO_REPO` — optional repo root for `resource-api-schema-planning` (`--workspace` metadata when set)
- `NJ_SCENARIO_ALLOW_FILE_FALLBACK` — allow discussion-only file write fallback (dev)
- `NEURAL_JUNKIE_RATE_LIMIT=0` — start the hub with rate limiting disabled when running many scenarios locally (`make collab-scenario-matrix`); otherwise POST `/api/send` may return HTTP 429

**Interpreting failures**

- The runner prints per-step pass/fail and dumps the last agent transcript lines on failure.
- `wait_discussion` prints a **diagnosis** block: per-agent message counts, silent agents, and handoff counts.
- Common exact failures:
  - **HTTP 429 on approve or /collaborate:** hub rate limit exhausted from polling — restart with `NEURAL_JUNKIE_RATE_LIMIT=0 make server` or wait one minute.
  - **Silent agent (e.g. Gemini):** turn handoffs with `collab_internal_event` were ignored before v1.0.0-beta.19+ fix — restart the hub after upgrading.
  - **PlatformEngineer JSON/kubectl in planning:** scenario `none_match` fails with the agent name; hub should rewrite raw tool JSON during planning when the workspace is not k8s-heavy — restart after upgrade. `@SoftwareArchitect` is also valid for the same goal.
  - **`Grounding: I loaded` in planning-two-agent:** usually means the agent scanned your open editor workspace during a no-repo collab — fixed when no source workspace is bound; restart hub after upgrade.
  - **`multi-collab-isolation` setup cancelled:** do not run scenarios in parallel; the isolation blocker must stay executing on `collab-scenario-blocker`.
  - **`any_match` keywords:** use a single combined regex in scenario JSON (e.g. `schema|standardiz|registr`) — each list entry must match somewhere in agent messages.
  - **`require_hub_approval` with approved=0:** agent used chat-only `[FILE_CHANGE] path` without a hub proposal — upgrade hub for loose FILE_CHANGE parsing or approve in Pending changes manually.
- Use `make debug-collab COLAB=<id8> LIVE=1` or `python3 scripts/debug-collab.py messages --channel collab-... --live` for full history.
- `KEEP=1` leaves the collab active: `make collab-scenario SCENARIO=planning-two-agent KEEP=1`

**CI:** `make collab-smoke` (in-process, no agents). Live scenarios are local-only.

## Testing

Coverage includes:

- lifecycle transitions
- transition to executing cancels a prior executing collaboration on the same channel (and does not cancel across channels)
- discussion turn-taking and budgets
- timeout handling
- mention-based out-of-turn responses
- consensus agreement/disagreement
- task assignment and completion tracking
- shared artifact versioning
- plan/task extraction parsing

See `test/collaboration_test.go`.

