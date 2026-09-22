# Away operations — morning bug report by 7AM CT

Hands-off path: overnight box runs **real user scenarios** + desktop UI clicks, then writes
`docs/testing/away-morning-YYYY-MM-DD.md` for you to triage. **No Cursor fix-loop** and
**no `agent-ready` drain** by default.

Beta/stable (and RC) tags stay **manual** unless you opt in with `MORNING_RC=1`.

## Morning routine (you)

1. Open `docs/testing/away-morning-YYYY-MM-DD.md` — FAIL gates + artifact links.
2. Fix product bugs (chat / PR), or capture new user-flows / desktop-e2e from the report.
3. Optional: `make away-morning-rc DRY_RUN=1` then cut RC when you choose.
4. Promote to beta/stable only when you choose.

## Schedule (America/Chicago)

| Time | Action |
|------|--------|
| 22:00 | `make overnight NJ_OVERNIGHT_TARGET=away` (launchd example under [packaging/away/](../packaging/away/launchd-away-overnight.plist.example)) |
| 22:00–~05:00 | `test-all` → **user-flows** → climb canary → **desktop-e2e** (hard stop `AWAY_DEADLINE_CT=05:00`) |
| ~05:00 | Write morning bug report (no agent fix-loop) |
| 07:00 | You triage the report |

## Overnight box prerequisites

- Repo checkout on `main`.
- Ollama + ≤14B regression models; GUI session logged in (for Playwright).
- Prevent sleep (`caffeinate` is used by `overnight.sh`).
- Cursor CLI / `agent-ready` **not** required for report-only overnight.

## Commands

```bash
make overnight NJ_OVERNIGHT_TARGET=away
make layer-gate LAYER=user-flows   # real journeys only
make desktop-e2e
make away-morning-rc DRY_RUN=1     # optional RC when you want it
```

Env knobs: `AWAY_DEADLINE_CT=05:00`, `AWAY_GATE=user-flows|full`, `SKIP_DESKTOP_E2E=1`, `MORNING_RC=1` (opt-in RC).

## Halt

- Close overnight tmux (`tmux kill-session -t nj-overnight`) or kill the nohup PID.
- Unload launchd agent if installed.

## RC vs beta

- **RC** (`v1.2.0-rc.N`): opt-in via `MORNING_RC=1` or manual `make away-morning-rc` — no Homebrew, no `updater/beta` promote.
- **Beta / stable**: human `make release VERSION=…` after live confidence.

## Related

- [AGENTS.md](../AGENTS.md) — unsupervised agent rules (optional daytime / agent-ready)
- [DESKTOP_E2E.md](DESKTOP_E2E.md) — Playwright / tauri-driver
- [USER_FLOW_SCENARIOS.md](USER_FLOW_SCENARIOS.md) — product journeys
- [TESTING.md](TESTING.md) — layer climb / release-prep
