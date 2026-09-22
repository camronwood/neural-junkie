# AGENTS.md — unsupervised / overnight coding agents

Rules for Cursor agents running on the overnight box (`make overnight NJ_OVERNIGHT_TARGET=away`, `make away-agent`, layer fix-loops).

## Mission

Overnight **away** is **report-only** by default: run real-user gates and write `docs/testing/away-morning-YYYY-MM-DD.md` for morning triage. Fix-loops / `agent-ready` are optional daytime tools — not part of the default overnight path.

When you *do* run an unsupervised fix (`make away-agent` / `layer-fix-loop`): find bugs the way a **real user** would, fix them, open a small PR labeled `agent-pr`, and let unit CI auto-merge.

## Do

- Prefer tools when workspace is shared: list/read/`run_command` before asking the human to paste files.
- Fix root causes; keep or strengthen tests and scenarios.
- Keep PRs small and scoped to the failure brief or GitHub issue.
- Run `make test-all` (and `make test-race` if Go changed) before pushing.
- Stop and label `agent-blocked` when blocked on secrets, product judgment, or flaky infra.

## Do not

- Weaken, skip, or delete failing scenarios/E2E specs to go green.
- Create release tags, edit `updater/beta`, or bump Homebrew.
- Commit secrets, `.env`, or API keys.
- Drive-by refactors unrelated to the issue/failure.
- Put illustrative shell in \`\`\`bash\`\`\` fences (triggers false Run / wait cues). Use prose or a non-bash fence for examples.
- Claim Electron/`BrowserWindow`/`main.js` without inspecting the workspace.

## Labels

| Label | Meaning |
|-------|---------|
| `agent-ready` | Safe unsupervised improvement (queue) |
| `agent-working` | Claimed by away-agent-loop |
| `agent-pr` | PR from overnight/away agent (auto-merge eligible) |
| `agent-blocked` | Needs a human |

## Issue body template

Write outcomes a user would recognize:

> As a user I …  
> Acceptance: …  
> Out of scope: …

## Where to edit

Before changing code, use the ownership map: [docs/AGENT_CODEMAP.md](docs/AGENT_CODEMAP.md) (edit-here-for-X, naming traps, slash command files, do-not-touch).

## Verify

See [docs/AWAY_OPERATIONS.md](docs/AWAY_OPERATIONS.md) and [docs/TESTING.md](docs/TESTING.md).
