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

## Security

- Host allowlist and private-IP blocking for HTTP actions
- Webhook/SMS/non-allowlisted HTTP should use tool approval (desktop)
- Do not store secrets in runbook JSON — use connector profile IDs (future Settings UI)

## Templates

Bundled templates live under `assets/runbook-templates/`. List via `GET /api/runbook-templates` and instantiate with `POST /api/runbook-templates/:name/instantiate`.
