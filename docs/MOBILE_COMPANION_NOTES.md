# Mobile companion app — future notes

**Captured:** July 2026  
**Status:** Exploratory reference for a future build. Not a roadmap commitment.

These notes capture a possible **separate mobile app** for Neural Junkie: a smaller, phone-native companion that focuses on chat, lightweight specialists, personal memory, and **local sync with desktop NJ**.

The goal is not to shrink the current desktop workspace onto a phone. The goal is to build a "little brother" app that borrows NJ's ideas while respecting phone constraints.

## Product thesis

Build a separate mobile app that:

- feels like NJ
- runs a small local model on-device
- works offline for core chat and personal workflows
- syncs with desktop NJ **over the local network**
- avoids desktop-only features like terminals, workspace editing, PTY sessions, and full repo workspaces

Possible product names:

- `NJ Pocket`
- `NJ Mobile`
- `NJ Companion`

## Why a separate app

Current NJ is desktop-shaped:

- Desktop client is Tauri + React and assumes a hub/WebSocket workflow.
- Local model management assumes an Ollama-style runtime.
- Several core surfaces are desktop-specific: file explorer, Monaco editor, terminal, workspace mutation, and remote dev flows.

Relevant references:

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [SECURITY.md](SECURITY.md)
- [IDE_V4.md](IDE_V4.md)

A phone app should reuse:

- agent/persona concepts
- pack vocabulary
- prompt conventions
- message formats
- settings language
- local-first product framing

It should not try to reuse the entire desktop shell.

## Recommended v1 scope

Ship a **chat-first, offline-first** mobile product.

### Include

- direct chat with `Assistant`
- 3 to 6 lightweight specialist personas
- saved conversations / lightweight threads
- notes, tasks, reminders
- optional voice / image input later
- optional handoff to desktop NJ

### Exclude

- code editor
- terminal
- file explorer
- workspace mutation
- repo-agent indexing
- SSH / devcontainer workflows
- full collaboration DAG execution

## Model strategy

Use **one shared local model** behind multiple personas.

That keeps the NJ "many specialists" feel without requiring multiple weights on the phone.

### Practical target

- 1B to 3B instruct model
- 4-bit quantization
- one default fast model tuned for latency, battery, and thermal limits

### Guidance

- `1B-2B` is the safest mobile tier.
- `3B` is the likely sweet spot on newer phones.
- `7B` may run on high-end devices, but should not define the default UX.

Multiple named experts should mostly be **prompt/persona layers**, not separate model downloads.

## Local sync with desktop NJ

Desktop NJ should be the **local sync host** and source of truth.

The mobile app should keep its own local database for offline use, then sync with the desktop when both devices are on the same network.

### Why local sync instead of cloud sync

- stays aligned with NJ's local-first posture
- avoids standing up hosted accounts / storage as a requirement
- keeps personal notes, tasks, and transcripts on user-controlled devices
- works for users who do not want cloud vendor lock-in

## Current NJ building blocks

Today's desktop app already has useful pieces for a future sync design:

- hub auth sessions via `POST /api/auth/session`
- hub token support for non-loopback access
- channel ACLs and local-only route protections
- SQLite message persistence in `~/.neural-junkie/messages.db`
- SQLite conversation memory in `~/.neural-junkie/memory.db`
- assistant data persisted as JSON records under `~/.neural-junkie/assistant/`
- channel export and MCP export flows

Important constraint:

- current hub sessions are **in-memory** and expire on restart, so they should not be the long-lived trust anchor for device pairing

References:

- [SECURITY.md](SECURITY.md)
- [MCP_EXPORTS.md](MCP_EXPORTS.md)
- `cmd/server/auth_handlers.go`
- `internal/hub/auth.go`
- `cmd/server/channel_history_handlers.go`

## Recommended sync architecture

Treat sync as **record replication**, not file copying and not SQLite file mirroring.

### Desktop responsibilities

- own the canonical store for synced data
- advertise local sync when enabled by the user
- issue pairing credentials to mobile devices
- provide snapshot + incremental sync APIs

### Mobile responsibilities

- maintain a local database for offline reads and writes
- queue local changes while offline
- pull changes from desktop when reachable
- upload local mutations when bidirectional sync is enabled

## Data to sync

### Good v1 candidates

- direct-message channels
- personal threads
- notes
- tasks
- reminders
- lightweight persona definitions
- selected exported experts / pack metadata

### Avoid in v1

- workspaces
- file trees
- pending file changes
- terminal state
- PTY sessions
- local model binaries copied from desktop
- repo indexes and code-intelligence caches

## Sync data model

Define syncable logical record types rather than shipping raw storage files.

Suggested common fields:

- `id`
- `type`
- `updated_at`
- `deleted_at` (tombstone)
- `source_device_id`
- `version`

Suggested record families:

- `conversation`
- `message`
- `thread`
- `task`
- `reminder`
- `note`
- `memory_entry`
- `persona`
- `agent_export_ref`

### Merge strategy

- messages: append-only, dedupe by stable ID
- tasks / reminders / notes: last-write-wins first, with tombstones
- memory entries: desktop-authoritative in v1

This keeps the first implementation simple while leaving room for richer conflict handling later.

## Pairing and trust model

Do **not** rely on the current user session token as the permanent pairing identity.

Recommended flow:

1. User enables **Local sync** on desktop.
2. Desktop shows a QR code with host discovery info + one-time pairing token.
3. Mobile scans the QR code on the same LAN.
4. Desktop exchanges the one-time token for a long-lived device credential.
5. Mobile stores that credential securely and uses it for future sync.

### Security requirements

- sync must be opt-in
- desktop should listen on LAN only while sync is enabled
- every sync request must authenticate as a trusted device
- user/session/channel ACL should still apply
- paired devices should be revocable from desktop settings

### Nice-to-have hardening

- device public/private key pairs
- signed sync requests
- optional local TLS or encrypted session transport
- per-device sync audit log

## Transport and discovery

Preferred UX:

- QR pairing for first connection
- mDNS / Bonjour for local discovery after pairing

Fallback UX:

- manual host entry
- desktop-generated short code

Platform caveats:

- iOS requires Local Network permission
- background sync is limited on iOS
- mobile wake/sleep behavior means sync must tolerate intermittent connectivity

## Sync protocol phases

### Phase 1: snapshot + pull

Start simple:

- mobile pairs with desktop
- desktop sends initial snapshot
- mobile stores local copy
- mobile periodically asks for changes since `cursor`

This is enough for:

- read-only transcript sync
- local notes/tasks/reminders mirror
- offline viewing with later refresh

### Phase 2: bidirectional sync

Add mobile-originated writes:

- create/update task
- create/update reminder
- save note
- start mobile conversation that syncs back to desktop

### Phase 3: richer handoff

- "Open on desktop"
- "Continue on phone"
- optional export/import of selected persona or history bundles

## Suggested API shape

Illustrative only:

```text
POST /api/mobile/pair/start
POST /api/mobile/pair/complete
GET  /api/mobile/sync/snapshot
GET  /api/mobile/sync/changes?cursor=...
POST /api/mobile/sync/push
POST /api/mobile/devices/revoke
GET  /api/mobile/devices
```

The important design choice is not the exact path names. It is keeping sync as a **small, explicit surface** instead of exposing the full desktop API to the phone.

## Implementation order

1. Mobile prototype with local-only storage and no sync
2. Pairing flow with desktop discovery
3. Read-only snapshot + incremental pull
4. Bidirectional sync for notes/tasks/reminders
5. DM/thread sync
6. Desktop handoff polish

This order proves product value before taking on full mobile/desktop convergence.

## Open questions

- Which mobile runtime is best for on-device inference?
- Should the mobile app have its own conversation memory store or only cache desktop memory?
- Should synced conversations be limited to DMs and personal channels in v1?
- How much of Assistant state should be editable on mobile vs view-only?
- Should pairing be per-user, per-device, or per-desktop install?

## Bottom line

If this ships, it should ship as:

> A separate NJ companion app with a small local model, offline-first chat, and explicit local sync to desktop NJ.

Not:

> The full desktop workspace squeezed onto a phone.

That narrower shape is much more realistic and better aligned with the current NJ architecture.
