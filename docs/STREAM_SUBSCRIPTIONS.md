# Stream subscriptions

Long-lived **MQTT** and **Kafka** consumers in the hub that dispatch matched messages to one of three actions:

1. **Runbook** — instantiate + start a runbook definition (Step Functions–style)
2. **Channel** — post a chat message into a hub channel (agents can react)
3. **Webhook** — HTTP POST to an outbound URL (optionally via a webhook connector)

## Concepts

| Piece | Storage | Role |
|-------|---------|------|
| Broker connector (`mqtt` / `kafka`) | `~/.neural-junkie/connectors.json` | Broker URL(s), auth secret |
| Subscription | `~/.neural-junkie/stream/subscriptions.json` | Topic + match → action |
| Stream manager | In-process | Starts one consumer per enabled subscription |

## UI

**Settings → Streams**

1. Create an MQTT or Kafka connector under **Integrations** (or Connectors section).
2. Add a subscription: protocol, connector, topic, optional JSON match / debounce, action.
3. Use **Test** to inject a synthetic payload without a broker.
4. **Restart** reloads workers after connector changes if needed (saves already restart).

## MQTT connector config

| Key | Description |
|-----|-------------|
| `broker_url` | e.g. `tcp://localhost:1883` or `ssl://host:8883` |
| `client_id` | Optional; defaults to `neural-junkie-<subscription-id>` |
| `username` | Optional |
| secret | Password |

## Kafka connector config

| Key | Description |
|-----|-------------|
| `brokers` | Comma-separated `host:9092` list |
| `group_id` | Optional consumer group; defaults to `neural-junkie-<subscription-id>` |
| `username` | Optional SASL PLAIN user |
| secret | SASL password |

## Match filter

Optional dotted JSON path (supports `status` or `nested.code`):

- `op: equals` (default) or `contains`
- Empty path → every message matches

## Actions

### Runbook

Requires `definition_id` and `agent_ids`. Inputs always include `topic` and `payload` (plus `key` for Kafka). Optional `input_map` maps JSON paths to extra input keys.

If the hub is at the concurrent collaboration cap (3), the event is **skipped** (logged; `last_error` / skip count on status).

### Channel

Posts from author `Stream` into `hub_channel`. Template placeholders: `{{payload}}`, `{{topic}}`, `{{key}}`.

### Webhook

POST JSON body. Use `webhook_connector_id` and/or `url_override`. Empty `body_template` sends `{"topic","key","payload"}`.

## HTTP API

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/stream/status` | Manager running + per-sub stats |
| POST | `/api/stream/restart` | Restart all consumers |
| GET/POST | `/api/stream/subscriptions` | List / create |
| PUT/DELETE | `/api/stream/subscriptions/:id` | Update / delete |
| POST | `/api/stream/subscriptions/:id/test` | Body `{ "payload": "...", "topic": "..." }` |

Local-only + mutation auth (same as other settings APIs).

## Out of scope (MVP)

Durable DLQ UI, raising the collaboration concurrency cap, pack-shipped subscriptions, EventBridge/SQS natives.
