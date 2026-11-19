# Away operations — unattended ship bar

Hands-off path for a **personal project with real users**: overnight box runs real-user
gates + desktop UI clicks, writes `docs/testing/away-morning-YYYY-MM-DD.md`, optionally
fixes FAILs, cuts RC, and after **two consecutive greens** cuts stable.

Human morning triage is **not** required. See [PRE_STABLE_RELIABILITY.md](PRE_STABLE_RELIABILITY.md).

## Recommended unattended command

```bash
make overnight NJ_OVERNIGHT_TARGET=away \
  AWAY_GATE=full MORNING_RC=1 AWAY_FIX_ON_FAIL=1 AUTO_STABLE_CUT=1
```

Launchd:

- Overnight: [packaging/away/launchd-away-overnight.plist.example](../packaging/away/launchd-away-overnight.plist.example)
- Daytime `away-agent`: [packaging/away/launchd-away-agent.plist.example](../packaging/away/launchd-away-agent.plist.example)

## Schedule (America/Chicago)

| Time | Action |
|------|--------|
| 22:00 | `make overnight NJ_OVERNIGHT_TARGET=away` with full env above |
| 22:00–~05:00 | `test-all` → **user-flows** → **layer-climb** → **desktop-e2e** (`AWAY_DEADLINE_CT=05:00`) |
| On FAIL | `AWAY_FIX_ON_FAIL=1` → `layer-fix-loop` + `away-agent` |
| On PASS | `MORNING_RC=1` → RC tag; `AUTO_STABLE_CUT=1` → stable when 2× green |
| Daytime | `make away-agent` drains `agent-ready` issues |

## Overnight box prerequisites

- Checkout on `pre-stable/reliability` worktree (or `main` after merge).
- Ollama + ≤14B regression models; GUI session logged in (for Playwright).
- Prevent sleep (`caffeinate` is used by `overnight.sh`).
- Cursor CLI + `gh` for `AWAY_FIX_ON_FAIL` / daytime `away-agent`.

## Commands

```bash
make overnight NJ_OVERNIGHT_TARGET=away AWAY_GATE=full MORNING_RC=1 AWAY_FIX_ON_FAIL=1 AUTO_STABLE_CUT=1
make layer-gate LAYER=user-flows
make desktop-e2e
make ship-bar
make ship-bar-package
make auto-cut-stable              # dry-run
make auto-cut-stable EXECUTE=1    # cut when ready
make away-morning-rc DRY_RUN=1
make away-agent
```

Env knobs: `AWAY_DEADLINE_CT=05:00`, `AWAY_GATE=user-flows|full`, `SKIP_DESKTOP_E2E=1`,
`MORNING_RC=1`, `AWAY_FIX_ON_FAIL=1`, `AUTO_STABLE_CUT=1`, `CONSECUTIVE_GREEN=2`,
`STABLE_CUT_TAG=v1.2.0`.

## Halt

- Close overnight tmux (`tmux kill-session -t nj-overnight`) or kill the nohup PID.
- Unload launchd agents if installed.

## RC vs beta vs stable

- **RC** (`v1.2.0-rc.N`): `MORNING_RC=1` when gates green — no Homebrew, no `updater/beta` promote.
- **Stable** (`v1.2.0`): `AUTO_STABLE_CUT=1` after two consecutive all-green away mornings + package verify.
- Interactive Gate 5 click-through is replaced by `make ship-bar` / `ship-bar-package` (see KNOWN_ISSUES #18).

## Related

- [AGENTS.md](../AGENTS.md) — unsupervised agent rules
- [DESKTOP_E2E.md](DESKTOP_E2E.md) — Playwright / tauri-driver
- [USER_FLOW_SCENARIOS.md](USER_FLOW_SCENARIOS.md) — product journeys
- [TESTING.md](TESTING.md) — layer climb / release-prep
- [PRE_STABLE_RELIABILITY.md](PRE_STABLE_RELIABILITY.md) — feature freeze + ship bar
