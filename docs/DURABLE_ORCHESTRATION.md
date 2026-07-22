# Durable orchestration

Neural Junkie persists collaboration runs, tasks, attempts, leases, human input,
results, events, workers, deployments, and automations in:

```text
~/.neural-junkie/orchestration.db
```

The existing session snapshot remains responsible for chat and desktop session
restoration. It is not the source of truth for task claims.

## Rollout mode

Durable state is shadow-written by default. Enable authoritative task claims:

```bash
NEURAL_JUNKIE_DURABLE_ORCHESTRATION=1 make server
```

Override the database location:

```bash
NEURAL_JUNKIE_ORCHESTRATION_DB=/absolute/path/orchestration.db make server
```

Authoritative mode enforces run-wide concurrency, attempt leases, retry budgets,
and claim-before-dispatch. Side effects marked `idempotency_required` fail
closed after an ambiguous lease loss and require operator reconciliation.

## Task execution options

Runbook tasks may set:

```json
{
  "options": {
    "max_retries": 3,
    "timeout_seconds": 300,
    "queue": "training",
    "capability_tags": ["gpu"],
    "cache_policy": "result",
    "cache_expiration_seconds": 3600,
    "refresh_cache": false,
    "idempotency_required": false
  }
}
```

Model and retrieval work must opt into result caching. Mutation tasks should
remain non-cacheable and set `idempotency_required`.

## Worker protocol

Remote workers authenticate with the existing hub token using
`X-NJ-Hub-Token` or `Authorization: Bearer`.

1. `POST /api/orchestration/workers/register`
2. `POST /api/orchestration/workers/heartbeat`
3. `POST /api/orchestration/work/claim`
4. `POST /api/orchestration/work/heartbeat`
5. `POST /api/orchestration/work/complete` or `/fail`

Claims return an attempt ID and secret lease token. The token is required for
heartbeat and terminal updates. Missing heartbeats expire the lease and either
schedule a policy-controlled retry or fail closed for protected side effects.

## Schedules and events

Create deployments with `POST /api/orchestration/deployments`. Supported local
schedules are:

- `@every 15m`
- `@daily 09:30`

An `event_filter` with `event_type` can trigger a deployment from durable
events. Wildcards use path-style matching, for example `attempt.*`.

Create automations with `POST /api/orchestration/automations`. Automation
delivery is deduplicated by automation and event ID.

## Human input gates

Pending gates are written to the RunStore and rehydrated on hub startup:

- user questions
- tool approvals
- file-change approvals
- git proposals
- collaboration task approvals
- plan approvals

Chat-native UX is unchanged. AWS write tools keep pack-side `confirm_token`
checks; those mutations also go through durable tool approvals when the hub
requires approval before dispatch.

## Observability

`GET /api/orchestration/run?run_id=<collaboration-id>` returns the run, tasks,
pending input, workers, and durable event history. Collaboration turn traces
also include this snapshot in the desktop routing trace panel.
