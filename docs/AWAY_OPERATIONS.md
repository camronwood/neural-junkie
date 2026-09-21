# Away operations — morning RC by 7AM CT

Hands-off path: overnight box runs **real user scenarios**, fixes failures, drains `agent-ready` issues, auto-merges on green unit CI, then cuts a **`v*-rc.N`** so installers are ready by **07:00 America/Chicago**.

Beta/stable tags stay **manual**. Live climb/UI on the overnight machine is the quality bar; GitHub Actions only re-runs unit CI for merges and builds RC installers.

## Morning routine (you)

1. Open [GitHub Releases](https://github.com/camronwood/neural-junkie/releases) — newest `*-rc*` prerelease.
2. Read `docs/testing/away-morning-YYYY-MM-DD.md` for what ran / failed / blocked overnight.
3. Install and smoke. Promote to beta/stable only when you choose (after optional `make release-prep` / Gate 5).

Quiet night with no `main` delta → no new RC; status file explains.

## Schedule (America/Chicago)

| Time | Action |
|------|--------|
| 22:00 | `make overnight NJ_OVERNIGHT_TARGET=away` (launchd example under [packaging/away/](../packaging/away/launchd-away-overnight.plist.example)) |
| 22:00–~04:30 | `test-all` → **user-flows** → climb canary → **desktop-e2e** → fix PRs → `agent-ready` (hard stop `AWAY_DEADLINE_CT=05:00`) |
| ~05:00 | Wait for in-flight `agent-pr` merges |
| 05:00–05:30 | `scripts/away-morning-rc.sh` (also backup cron in `.github/workflows/auto-rc.yml` ~10:30 UTC) |
| 05:30–07:00 | `release.yml` builds multi-OS installers |
| 07:00 | You download the RC |

## Overnight box prerequisites

- Repo checkout on `main`, push rights, `gh` auth (PR + merge).
- Cursor CLI `agent` on PATH + `CURSOR_API_KEY` or `.cursor-api-key`.
- Ollama + ≤14B regression models; GUI session logged in (for Playwright).
- Prevent sleep (`caffeinate` is used by `overnight.sh`).
- Repo settings: **Allow auto-merge**; protect `main` with required `test.yml` check(s).

## Commands

```bash
make overnight NJ_OVERNIGHT_TARGET=away
make away-agent                    # one agent-ready issue
make desktop-e2e                   # UI click journeys
make away-morning-rc DRY_RUN=1     # print next tag
make layer-gate LAYER=user-flows   # real journeys only
```

Env knobs: `AWAY_DEADLINE_CT=05:00`, `AWAY_GATE=user-flows|full`, `SKIP_DESKTOP_E2E=1`, `MAX_ISSUES=3`, `MAX_ITER=3`, `MORNING_RC_FORCE=1`.

## Halt

- Remove `agent-ready` labels / close overnight tmux (`tmux kill-session -t nj-overnight`).
- Disable workflow **Auto RC** in GitHub Actions.
- Unload launchd agent if installed.

## RC vs beta

- **RC** (`v1.2.0-rc.N`): prerelease installers only — no Homebrew, no `updater/beta` promote.
- **Beta / stable**: human `make release VERSION=…` after live confidence.

## Related

- [AGENTS.md](../AGENTS.md) — unsupervised agent rules  
- [DESKTOP_E2E.md](DESKTOP_E2E.md) — Playwright / tauri-driver  
- [USER_FLOW_SCENARIOS.md](USER_FLOW_SCENARIOS.md) — product journeys  
- [TESTING.md](TESTING.md) — layer climb / release-prep  
