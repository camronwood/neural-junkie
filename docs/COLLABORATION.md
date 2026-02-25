# Multi-Agent Collaboration

Neural Junkie now supports structured multi-agent collaboration so agents can discuss, review, delegate, and execute work together under user control.

This is different from lightweight `@mention` review flow: collaboration introduces bounded discussions, shared artifacts, explicit phases, and task tracking.

## Goals

- Enable agent-to-agent discussions in a controlled mode
- Let users assign multiple agents to a shared objective
- Require user approval before execution starts
- Delegate tasks by agent strengths (type + expertise)
- Prevent runaway conversations with hard limits

## Commands

### Start a collaboration

```text
/collaborate @RustExpert @SecurityExpert @Cursor build a CLI tool that encrypts files using AES-256
```

Creates a collaboration in `planning` phase and starts a bounded discussion.

### Approve plan and execute

```text
/approve-plan <collab-id>
```

Moves collaboration from `reviewing` -> `approved` -> `executing` and dispatches task messages to assigned agents.

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
   - Transitional state after `/approve-plan`.
4. **executing**
   - Agents receive their assigned tasks.
   - Progress is tracked per task.
   - Bounded cross-agent Q&A remains available.
5. **completed** or **cancelled**

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

## Testing

Coverage includes:

- lifecycle transitions
- discussion turn-taking and budgets
- timeout handling
- mention-based out-of-turn responses
- consensus agreement/disagreement
- task assignment and completion tracking
- shared artifact versioning
- plan/task extraction parsing

See `test/collaboration_test.go`.

