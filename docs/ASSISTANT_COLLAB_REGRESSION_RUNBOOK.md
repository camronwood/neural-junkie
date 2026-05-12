# Assistant + Collaboration Regression Runbook

Use this runbook when Assistant reminders/tasks appear conversational but do not persist, or when collaborations look active in UI but agents do not take turns.

## Scope

- Assistant deterministic actions (reminders/tasks) must persist to storage.
- Assistant reminders must fire and clear/deactivate correctly.
- Collaboration startup must create discussion activity (not stuck at `Messages: 0/...`).

## 0) Preconditions

- Server expected at `http://localhost:18765`.
- You can run local CLI commands (`curl`, `make`, `lsof`).
- If you use `make refresh`, confirm no stale process still owns `:18765`.

## 1) Quick Health + Log Baseline

```bash
curl -s "http://localhost:18765/api/health"
```

Expected:
- JSON with `"status":"ok"`.
- `uptime_secs` should reflect the process you think is running.

Then inspect recent server logs:

```bash
tail -n 200 /tmp/chat-server.log
```

Look for:
- `Assistant` startup lines.
- DM rebind lines (`DM rebind: Assistant -> dm-camron-assistant`).
- Collaboration creation/turn lines.
- Any `bind: address already in use` (stale runtime risk).

## 2) Assistant Persistence Probe (Deterministic Path)

Send a controlled reminder message:

```bash
curl -s -X POST "http://localhost:18765/api/send" \
  -H "Content-Type: application/json" \
  -d '{"channel":"dm-camron-assistant","content":"set a reminder for 25s from now to run persistence probe","type":"question","from":{"name":"Camron","type":"human"}}'
```

Immediately verify state:

```bash
curl -s "http://localhost:18765/api/assistant/state?channel=dm-camron-assistant"
```

Pass criteria:
- `reminders` contains a new active reminder with the expected content/channel.

Fail signals:
- Assistant says "I've set a reminder..." but `reminders` is empty.
- Logs show LLM response path only, or save errors.

## 3) Reminder Fire/Clear Probe

Wait past trigger time and re-check:

```bash
sleep 30
curl -s "http://localhost:18765/api/assistant/state?channel=dm-camron-assistant"
```

Pass criteria:
- One-time reminder no longer appears as active in API output.
- Optional: corresponding reminder message appears in channel history.

## 4) Collaboration Probe

If collaboration capacity is full, cancel stale plans first:

```bash
curl -s -X POST "http://localhost:18765/api/send" \
  -H "Content-Type: application/json" \
  -d '{"channel":"unit-testing","content":"/cancel-plan <collab_id>","type":"question","from":{"name":"Camron","type":"human"}}'
```

Start a fresh collaboration:

```bash
curl -s -X POST "http://localhost:18765/api/send" \
  -H "Content-Type: application/json" \
  -d '{"channel":"unit-testing","content":"/collaborate @RustExpert @ReactExpert run collaboration probe","type":"question","from":{"name":"Camron","type":"human"}}'
```

Pass criteria in logs:
- Collaboration created.
- Participants join/listen on the channel.
- "Found unanswered message ... processing" or equivalent turn activity appears.

## 5) If Results Do Not Match Source (Stale Runtime Playbook)

1. Stop anything on `:18765`:

```bash
lsof -ti :18765 | xargs kill -9 2>/dev/null || true
```

2. Restart from current source:

```bash
make refresh
```

3. Re-run Sections 1-4 with the exact same payloads.

Important:
- If logs show `bind: address already in use`, restart did not take effect.
- If running inside a restricted sandbox, storage writes to `~/.neural-junkie/...` may fail (`operation not permitted`). Use an unsandboxed runtime for persistence checks.

## 6) Known Good Signals

- Assistant state endpoint shows newly created reminders/tasks right after controlled probe.
- Reminder clears/deactivates after trigger.
- Collaboration logs show created session plus participant processing activity.
- No contradictory "success" chat response without persisted state.

## 7) Minimal Report Template

- `Health`: pass/fail (+ key fields)
- `Assistant create`: pass/fail (+ state payload snippet)
- `Assistant fire/clear`: pass/fail
- `Collaboration create`: pass/fail
- `Collaboration turn activity`: pass/fail (+ log evidence)
- `Runtime consistency`: pass/fail (`bind`/stale/sandbox notes)
