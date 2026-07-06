# Runbook action tasks

Runbook tasks default to **agent** steps (`collaboration_task` prompts). Set `kind: "action"` to run deterministic hub steps without an LLM turn.

## Action types

| Type | Purpose |
|------|---------|
| `http_get` | Fetch a URL (allowlist + SSRF guards) |
| `http_post` | POST JSON to a URL |
| `webhook` | POST payload to a webhook URL |
| `web_search` | Query web search (stub unless provider configured) |
| `sms` | SMS notify (disabled unless enabled in server config) |
| `slack_message` | Post a message to a Slack channel (requires connected Slack bridge) |
| `mcp_tool` | Reserved; use agent + MCP for tool calls in v1 |

## Output format

Action tasks store JSON in `task.output`:

```json
{
  "summary": "HTTP 200 https://…",
  "action_type": "http_get",
  "data": { "status_code": 200, "body": "…" }
}
```

Downstream **conditional edges** can match `on_output` with `contains` or `regex`.

### `slack_message` config

| Field | Required | Description |
|-------|----------|-------------|
| `channel_id` | yes | Slack channel ID (e.g. `C01234567`) |
| `text` | yes | Message body; supports `{{task.title}}`, `{{task.description}}`, `{{collab.description}}` |
| `thread_ts` | no | Reply in an existing thread |
| `username` | no | Bot display name override |

Example:

```json
{
  "kind": "action",
  "title": "Notify #eng",
  "action": {
    "type": "slack_message",
    "config": {
      "channel_id": "C01234567",
      "text": "Runbook step **{{task.title}}** finished."
    }
  }
}
```

## Security

- Host allowlist and private-IP blocking for HTTP actions
- Webhook/SMS/Slack/non-allowlisted HTTP should use tool approval (desktop)
- Do not store secrets in runbook JSON — use connector profile IDs (future Settings UI)

## Templates

The core app ships one **generic starter** in `assets/runbook-templates/` (`health-check-alert`). Customer-specific runbooks belong in packs or your user library. List via `GET /api/runbook-definitions` (or legacy `/api/runbook-templates`).

User definitions, run inputs, connector profiles, and run history are documented in [RUNBOOKS_V2.md](RUNBOOKS_V2.md).
