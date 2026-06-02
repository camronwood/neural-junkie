# LinkedIn — Neural Junkie v1.0.0-beta.21

**Cover image:** `assets/neural-junkie-beta21-ad-1200.png` (1200×627)

**Regenerate:**

```bash
chmod +x ./scripts/compose-beta21-ad.sh
./scripts/compose-beta21-ad.sh
./scripts/sync-gallery.sh
```

**Download:** https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.21

---

## Headline on image

**SLACK INBOX. CHAT YOU CAN TRUST.**

Sub: Context model v2, workspace visibility, live scenario regressions.

---

## Feed post (short)

> **Neural Junkie v1.0.0-beta.21** is out.
>
> **Slack** — personal inbox DM to your hub, selective channel forwarding, optional away-mode replies in your own DMs.
>
> **Chat quality** — conversation mode + turn intent v2, deterministic “can you see my workspace?” answers (no fake `go get` packages), echo/closure fixes.
>
> **Testing** — expanded chat/collab/learning JSON scenarios, `make test-all`, and docs in `docs/TESTING.md` so regressions show up before manual clicking.
>
> macOS / Windows / Linux: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.21
>
> #AI #MultiAgent #DeveloperTools #OpenSource #Slack

---

## LinkedIn (longer)

> Multi-agent tools are easy to demo and hard to regression-test. **beta.21** focuses on **real channels** and **conversation quality you can verify**.
>
> **Slack**
> - Personal inbox — message the NJ bot from mobile; routes to a private hub channel and your chosen agent
> - Forwarding rules — `@mention`, `nj:` prefix, or reaction emoji; replies land in the **original Slack thread**
> - Away mode (opt-in) — user-token replies in your 1:1 DMs when you’re marked away
>
> **Context model v2** ([CONTEXT_MODEL.md](https://github.com/camronwood/neural-junkie/blob/main/docs/CONTEXT_MODEL.md))
> - Composer **Chat / Code** mode with `conversation_mode` metadata
> - Turn intent: casual, closure, task, substantive — with DM persona and thread-scoped history
> - ~32KB context budget; session persist slimming for workspace payloads
>
> **Chat regressions**
> - Workspace visibility shortcut + streaming fallback when models ignore the question
> - Live scenarios: `make chat-scenarios-regression` (workspace, echo, closure)
>
> **Collab & CI**
> - Deterministic agent order for round-robin task assignment
> - Collab auto-approve deliverables, idle watchdog, dependency-prose parser tests
> - `docs/TESTING.md` — isolated Go tests, scenario runners, full checklist
>
> Open source beta: https://github.com/camronwood/neural-junkie/releases/tag/v1.0.0-beta.21

---

## Suggested first comment

Pin the release link and: “If workspace visibility or Slack routing misbehaves on your setup, open an issue with hub logs — we added scenario repros for the chat cases we kept hitting.”
