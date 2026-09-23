# Runbook action tasks

Runbook tasks default to **agent** steps (`collaboration_task` prompts). Set `kind: "action"` to run deterministic hub steps without an LLM turn.

## Action types

| Type | Purpose |
|------|---------|
| `http_get` | Fetch a URL (allowlist + SSRF guards) |
| `http_post` | POST JSON to a URL |
| `webhook` | POST payload to a webhook URL (requires approval) |
| `web_search` | Query web search (stub unless provider configured) |
| `sms` | SMS via HTTP POST to a connector URL (requires approval) |
| `email` | Send email via SMTP connector (requires approval) |
| `slack_message` | Post a message to a Slack channel (requires connected Slack bridge) |
| `shell` | Run a shell command in the collab working directory |
| `wait_human` | Pause until the user approves the task |
| `git_status` / `git_diff` | Local git inspection in the collab working directory |
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

### `sms` config (HTTP notify — no vendor SDK)

SMS is a first-class action that **POSTs** to a URL from an **SMS connector** (or inline `url`). Typical endpoints: Twilio Messages API, TextBelt, Zapier/Make catch hooks.

| Field | Required | Description |
|-------|----------|-------------|
| `to` | yes | Destination number (E.164 preferred) |
| `body` | yes | Message text |
| `url` | via connector | POST target |
| `from` | no | Sender id / From number |
| `format` | no | `form` (default, Twilio-style) or `json` |

Connector (`type: sms`): `config.url`, optional `from` / `format` / `header_name`; `secret` becomes `Authorization` (Bearer unless already `Basic …` / `Bearer …`).

Webhook/SMS/email pause for **desktop approval**, then execute once. In the Collaboration panel task row, use **Approve** (not Task Management’s plan Resume) — it calls `POST /api/collaborations/:id/tasks/:taskId/approve`.

### `email` config (SMTP connector)

| Field | Required | Description |
|-------|----------|-------------|
| `to` | yes | Recipient |
| `subject` | yes* | Subject (*or body) |
| `body` | yes* | Plain-text body |
| `host` / `port` / `username` / `from` / `password` | via connector | SMTP settings |

Connector (`type: email`): `config.host`, `port` (default 587), `username`, optional `from`; `secret` is the SMTP password. Port `465` uses implicit TLS; otherwise `net/smtp` (STARTTLS when advertised).

Example:

```json
{
  "kind": "action",
  "title": "Email on-call",
  "action": {
    "type": "email",
    "connector_id": "<email-connector-id>",
    "config": {
      "to": "camronwood@gmail.com",
      "subject": "Runbook {{task.title}}",
      "body": "Health check failed: {{tasks.health-check.output}}"
    }
  }
}
```

## Security

- Host allowlist and private-IP blocking for HTTP / SMS URLs
- Webhook/SMS/email/Slack/non-allowlisted HTTP should use tool approval (desktop)
- Do not store secrets in runbook JSON — use connector profile IDs (Settings → Connectors)

## Testing

Unit tests cover action runners and approve → execute for gated notify.

Live scenario harness (hub must be running on `127.0.0.1:18765`):

```text
make runbook-scenario SCENARIO=health-check-branch
make runbook-scenario SCENARIO=notify-webhook-approve
make runbook-scenario SCENARIO=notify-sms-http
```

Opt-in live email (requires an `email` SMTP connector in `~/.neural-junkie/connectors.json`):

```text
RUNBOOK_LIVE_EMAIL=1 make runbook-scenario SCENARIO=notify-email-smtp
```

## Templates

The core app ships one **generic starter** in `assets/runbook-templates/` (`health-check-alert`). Customer-specific runbooks belong in packs or your user library. List via `GET /api/runbook-definitions` (or legacy `/api/runbook-templates`).

User definitions, run inputs, connector profiles, and run history are documented in [RUNBOOKS_V2.md](RUNBOOKS_V2.md).
