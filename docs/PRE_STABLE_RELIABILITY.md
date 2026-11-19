# Pre-stable reliability (unattended)

Branch / worktree: `pre-stable/reliability` (`.worktrees/pre-stable-reliability`).

**Feature freeze:** bugfix, harness honesty, docs, and ship-bar automation only on this line until `v1.2.0` is cut. No Share Agent polish / D5 / composition features land *on this freeze branch* after the initial platform backlog PRs merge — those use separate worktrees post-cut, or land before freeze if already in flight on this branch.

## Unattended overnight

```bash
make overnight NJ_OVERNIGHT_TARGET=away AWAY_GATE=full MORNING_RC=1 AWAY_FIX_ON_FAIL=1
```

Launchd example (full gate + morning RC + fix-on-fail):

- [packaging/away/launchd-away-overnight.plist.example](../packaging/away/launchd-away-overnight.plist.example)
- Daytime drain: [packaging/away/launchd-away-agent.plist.example](../packaging/away/launchd-away-agent.plist.example)

Env:

| Variable | Effect |
|----------|--------|
| `AWAY_GATE=full` | `test-all` → user-flows → `layer-climb CONTINUE=1` → desktop-e2e |
| `MORNING_RC=1` | Cut RC when gates green and HEAD has delta |
| `AWAY_FIX_ON_FAIL=1` | After FAIL report, run `layer-fix-loop` on first failing layer and/or `away-agent` |
| `AUTO_STABLE_CUT=1` | After **2 consecutive** all-green away mornings, run `scripts/auto-cut-stable-if-ready.sh` |

## Ship bar (replaces interactive Gate 5 clicks)

```bash
make ship-bar              # local automated trust gate
make ship-bar-package      # artifact / version / updater verify helpers
```

See [TESTING.md](TESTING.md) and [KNOWN_ISSUES.md](KNOWN_ISSUES.md) (`blocker-platform-smoke`).

## Operator time

Camron does **not** triage mornings or click Gate 5. Agents + scripts own FAIL → PR → re-overnight → RC → stable.
