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

## Commands

### Start a collaboration

```text
/collaborate @RustExpert @SecurityExpert @Cursor build a CLI tool that encrypts files using AES-256
```

Optional limits (must appear **before** the first `@mention`; omitted values use defaults **3** rounds and **20** agent messages, then the server clamps to hard caps):

```text
/collaborate --rounds 5 --messages 40 @RustExpert @SecurityExpert design the auth flow
```

Creates a collaboration in `planning` phase and starts a bounded discussion.

**Dedicated channel:** On success the hub auto-creates a channel named `collab-<collaboration-uuid>` (type `collaboration`). Seeds, agent discussion, plan updates, and execution tasks for that session are isolated **there**. Your `/collaborate` line stays in the channel where you typed the command (for example `#general`). The desktop app switches to the new room after send and lists it under **Collaborations** in the sidebar; **Open collaboration** in the task panel also jumps to that channel.

### Approve plan and execute

```text
/approve-plan <collab-id>
```

Moves collaboration from `reviewing` -> `approved` -> `executing`, creates the on-disk collaboration sandbox, and lists assigned tasks in chat. **Task prompts are not sent to agents until you confirm the workspace** (desktop **Continue** on that channel, or `/ack-collab-workspace <collab-id>`). This keeps execution from racing ahead of the app registering the sandbox as a workspace.

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

## Files, workspace, and approvals during execution

Agents create or edit files by emitting **`[FILE_CHANGE]`** blocks in their replies (the same machine-readable format as normal channels). **Plain discussion or markdown code fences alone do not write to disk** until a proposal is emitted and you approve it in the desktop **Pending changes** flow.

- **Workspace sharing:** The hub resolves file proposals against the **workspace path** carried in message metadata. Collaboration traffic usually happens in `collab-…` channels, while your IDE workspace is often attached to messages on `#general` or your project channel. The app now **falls back** to the most recent `workspace_context` from other channels the agent has seen, so proposals still register when you had sharing enabled from the project window.
- **Paths must stay under the shared root:** Requests like “write under `~/development/test-site-001`” only work if that directory is the shared workspace (or added as a workspace) and sharing is on. Otherwise agents should explain the limitation and ask you to add that folder or use paths **relative** to the current workspace root.
- **Collaboration sandbox:** When a plan is approved and execution starts, the hub creates `~/.neural-junkie/collaborations/<collaboration-id>/`, attaches it as `workspace_context` on `collaboration_task` messages **after you confirm**, and snapshots expose `working_directory` plus `workspace_acknowledged`. **Agents do not receive task prompts until you confirm:** use the desktop **Continue** dialog on the collaboration channel, or run `/ack-collab-workspace <id>`.
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
| Max rounds | 3 | 10 |
| Max turns per agent per round | 1 | 3 |
| Max total messages | 20 | 50 |
| Wall-clock timeout | 5 min | 30 min |
| Max concurrent collaborations | 3 | n/a |
| Max tasks per collaboration | 10 | n/a |

When a bound is reached, the discussion is ended and the system keeps what was produced.

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

