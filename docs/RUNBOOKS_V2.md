# Runbooks v2

Runbooks v2 evolves user-authored task DAGs from ephemeral collaboration drafts into **persisted definitions** and **replayable executions** — local-first, approval-gated, multi-agent operational workflows.

See also: [COLLABORATION.md](COLLABORATION.md), [RUNBOOK_ACTIONS.md](RUNBOOK_ACTIONS.md).

## Concepts

| Concept | Description |
|---------|-------------|
| **RunbookDefinition** | Named, versioned workflow: tasks, graph, policies, inputs, default agent pool |
| **RunExecution** | One run of a definition — implemented as a `Collaboration` with `source: runbook` plus run metadata |
| **Connector profile** | Named integration credential (Slack, webhook, HTTP auth) referenced by action tasks — never stored in definition JSON |

**v1:** User draws a task DAG; execution is a collaboration session.  
**v2:** User owns definitions (repeatable SOPs) and executions (runs with inputs, trace, replay).

## Data model

### RunbookDefinition

```json
{
  "id": "health-check-alert",
  "version": 2,
  "title": "Health check and alert",
  "description": "Fetch API health, summarize, notify on failure.",
  "source": "user",
  "agent_ids": ["agent-uuid"],
  "execution_policy": { "blocked_upstream_policy": "block", "strict_task_status": true },
  "graph_layout": {},
  "inputs": [
    { "key": "health_url", "type": "string", "label": "Health URL", "default": "https://httpbin.org/status/200", "required": true }
  ],
  "tasks": []
}
```

**Source values:** `bundled`, `user`, `pack`.

### RunInputSpec

| Field | Type | Notes |
|-------|------|-------|
| `key` | string | Input name; referenced as `{{inputs.key}}` |
| `type` | string | `string`, `number`, `bool`, `slack_channel` |
| `label` | string | Form label |
| `default` | string | Default when not provided |
| `required` | bool | Validated at start |

### Execution metadata (on Collaboration)

| Field | Purpose |
|-------|---------|
| `definition_id` | Link to library definition |
| `definition_version` | Pinned version at run start |
| `run_inputs` | `map[string]string` values |
| `run_number` | Monotonic per definition |

## Persistence

```
~/.neural-junkie/
  runbook-library/
    {definition-id}/
      manifest.json       # id, title, latest_version, updated_at
      v1.json
      v2.json
  runbook-runs/
    index.json            # lightweight run index for history UI
  connectors.json         # encrypted connector profiles
```

**Definition resolution order:**

1. `~/.neural-junkie/runbook-library/*/v*.json` (user)
2. Enabled pack runbooks (`runbooks_glob` in pack.yaml)
3. `<collab_assets_root>/runbook-templates`
4. Repo `assets/runbook-templates/`

## HTTP API

### Definitions

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/runbook-definitions` | List all definitions (all sources) |
| GET | `/api/runbook-definitions/:id` | Load latest or `?version=N` |
| POST | `/api/runbook-definitions` | Create user definition |
| PUT | `/api/runbook-definitions/:id` | Save new version |
| DELETE | `/api/runbook-definitions/:id` | Delete user definition |
| POST | `/api/runbook-definitions/:id/instantiate` | Create draft collab from definition |

Legacy `GET /api/runbook-templates` remains; lists bundled templates only.

### Runs

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/runbook-runs?definition_id=` | List runs |
| GET | `/api/runbook-runs/:collabId` | Run snapshot + task outputs |
| POST | `/api/runbook-runs/:collabId/replay` | Re-run with same inputs |

### Connectors

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/connectors` | List profiles (secrets redacted) |
| POST | `/api/connectors` | Create profile |
| PUT | `/api/connectors/:id` | Update profile |
| DELETE | `/api/connectors/:id` | Delete profile |

### Runbooks (execution, unchanged + extensions)

| Method | Path | Extension |
|--------|------|-----------|
| POST | `/api/runbooks/:id/start` | Body may include `inputs: { key: value }` |

## Interpolation

Action configs and task descriptions support:

| Token | Source |
|-------|--------|
| `{{inputs.key}}` | Run inputs |
| `{{tasks.<taskId>.output}}` | Upstream task output |
| `{{tasks.<taskId>.data.status_code}}` | Parsed action JSON |
| `{{task.title}}`, `{{task.description}}`, `{{collab.description}}` | v1 tokens |

## Conditional edges

Tasks may declare:

- `dependency_edges[]` — per-edge conditions (`always`, `on_status`, `on_output`)
- `dependency_groups[]` — OR/AND join semantics (`mode: all|any`)

See `internal/collaboration/edges.go`.

## Migration

- Existing draft runbooks (in-memory collaborations) unchanged.
- **Save to library** promotes a draft to a persisted definition.
- Bundled JSON templates remain compatible; `RunbookTemplate` fields map to `RunbookDefinition`.

## Phasing

| Phase | Focus |
|-------|-------|
| **v2.0** | Library, template picker, graph conditions, run inputs, history, connectors |
| **v2.1** | Real actions (mcp_tool, shell, wait_human), pack browser, scenario harness |
| **v2.2** | Triggers (slash, Slack, CLI), scheduled runs via cron |

## Triggers (v2.2)

| Trigger | Usage |
|---------|--------|
| Hub slash | `/runbook-run <definition-id> @Agent1 [--input key=value]` |
| Slack | @mention the bot with the same slash command in a bound channel |
| HTTP | `POST /api/runbook-definitions/:id/trigger` with `{ agent_ids, channel, inputs }` |
| CLI | `python3 scripts/nj_runbook.py list\|run\|history` |

## Scheduled runs

Runbooks are local-first; schedule with cron or launchd invoking the CLI:

```bash
# Every day at 9am — health check with custom URL
0 9 * * * cd /path/to/neural-junkie && python3 scripts/nj_runbook.py run health-check-alert --agent Assistant --input health_url=https://example.com/health
```

The hub does not include a built-in scheduler in v2.2; use OS-level scheduling against `nj_runbook.py` or the trigger HTTP endpoint.
