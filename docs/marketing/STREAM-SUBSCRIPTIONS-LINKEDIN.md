# LinkedIn article — Stream subscriptions (publish copy)

**Format:** LinkedIn **Article** (long-form). Use the cover image below when publishing.

**LinkedIn paste tips:** Copy from **"PASTE START"** through **"PASTE END"** only. Do not include `---` lines — LinkedIn renders them as awkward rules with huge gaps. Use LinkedIn's title field for the headline; skip the first `#` line if the editor duplicates it. One blank line between paragraphs is enough.

**Cover image:** `assets/neural-junkie-stream-subscriptions-1200.png` (1200×627)

**Regenerate cover:** `./scripts/compose-stream-subscriptions-article.sh`

**Suggested title (pick one):**
- Streams In. Agents Out: MQTT and Kafka Triggers for Local Runbooks
- We Didn't Put Agents on the Bus. We Put the Bus Next to the Hub.
- Step Functions Style, Local First: Stream Subscriptions in Neural Junkie

**Feed post teaser:**
> MQTT and Kafka don't need a chatbot sitting on the topic. Neural Junkie adds long-lived stream subscriptions that match messages and fire a runbook, post into a hub channel, or call a webhook — Settings UI included.

**Hashtags:** `#AI #LocalAI #MultiAgent #MQTT #Kafka #DeveloperTools #OpenSource #LocalFirst`

**Link:** https://github.com/camronwood/neural-junkie/releases/latest

**Website:** https://camronwood.github.io/neural-junkie/articles/stream-subscriptions.html

**Download CTA:** https://camronwood.github.io/neural-junkie/download.html

**Related articles:** [The Hub](HUB-LINKEDIN.md) · [Collaboration](COLLABORATION-LINKEDIN.md) · [Runbooks docs](../STREAM_SUBSCRIPTIONS.md)

---

## PASTE START

Most multi-agent demos start when a human types.

The real world often starts when **a message arrives**: a device telemetry spike on MQTT, an ops event on Kafka, an alert someone already publishes to a bus you run.

The wrong answer is to park an LLM on that topic and let it improvise forever.

The right shape is closer to **AWS Step Functions**: a long-lived consumer decides *that something happened*, then a **finite workflow** decides *what to do*.

That is what **stream subscriptions** add to Neural Junkie.

## Two pieces (don't fuse them)

1. **Long-lived stream processor** — stays connected to MQTT or Kafka, tracks offset/group membership, filters noise, debounces bursts.
2. **Action** — what happens when a message is worth caring about:
   - **Runbook** — start a saved SOP (agents + HTTP actions + approvals)
   - **Hub channel** — post into a channel so always-on agents can react
   - **Webhook** — forward to something else you already operate

Runbooks stay runbooks. Streams stay streams. The subscription is the binding between them.

## Why not "just put an agent on Kafka"?

High-volume buses and generative agents are a bad couple:

- Agents are expensive and stateful; buses are chatty and idempotent
- You want offsets, retries, and consumer groups — not prompt cache
- You often need a **human-shaped SOP** (triage → investigate → notify), not a freeform chat loop

So the hub owns a small **stream manager** (same idea as our Slack Socket Mode bridge): start on boot, restart from Settings, one worker per enabled subscription.

## What ships

**Connectors** for broker credentials (`mqtt` / `kafka`) — same encrypted connector store we use for webhooks.

**Subscriptions** saved under `~/.neural-junkie/stream/`:

- Protocol + connector + topic
- Optional JSON field match (`status` equals `alert`, or `contains`)
- Optional debounce window
- Action: runbook, channel, or webhook

**Settings → Streams** for status, CRUD, restart, and a **Test** button that injects a synthetic payload without a broker.

Docs: [STREAM_SUBSCRIPTIONS.md](https://github.com/camronwood/neural-junkie/blob/main/docs/STREAM_SUBSCRIPTIONS.md)

## Runbook path (Step Functions style)

When the action is a runbook, the hub reuses the same path as `POST /api/runbook-definitions/:id/trigger`:

instantiate → submit → start

Inputs always include `topic` and `payload` (plus Kafka `key`). You can map JSON fields into named runbook inputs.

If you're already at the concurrent collaboration cap, we **skip and surface the error** on the subscription status instead of queueing an unbounded backlog. Honest beats silent.

## Channel and webhook paths

Not every event deserves a full DAG.

- **Channel** posts as author `Stream` into a hub channel — mention an agent if you want a specialist to wake up.
- **Webhook** POSTs outbound (connector auth optional) when another system should own the next hop.

Same matcher. Same manager. Different consequence.

## The product bet

Local-first multi-agent work is not only chat UX.

It is an orchestrator that can sit next to the buses you already trust — device MQTT in the lab, Kafka in the platform — and still enforce **human-visible workflows**, approvals, and agent pools you control.

Streams in. Agents out — on purpose.

## Try it

1. Open **Settings → Integrations** and add an MQTT or Kafka connector.
2. Open **Settings → Streams**, create a subscription, pick an action.
3. Hit **Test** with a JSON payload, then point the connector at a real broker when you're ready.

Neural Junkie is open source. Download the latest release, or build from source, and wire your bus to a runbook instead of another zombie listener.

## PASTE END
