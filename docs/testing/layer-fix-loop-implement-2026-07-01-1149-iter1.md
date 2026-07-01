# layer fix loop — implement — iteration 1 — 2026-07-01-1149-iter1 UTC

layer=implement
summary=/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-implement-2026-07-01-1149-iter1.md
fix_branch=release-prep/layer-implement-2026-07-01-1149
git_commit=d40ad89
layer_gate_rc=1
agent_rc=0

## Cursor agent output

```text
## Summary

Two fixes to unblock release-prep / layer gate:

### 1. Missing import in `implement-scenarios.py` (layer gate failure)

The gate failed immediately on the first implement scenario with:

`NameError: name 'release_prep_env' is not defined`

`ensure_hub_ready()` calls `release_prep_env(ROOT)` but the function was never imported. Added the same import used by `collab-scenarios.py` and `chat-scenarios.py`:

```python
from lib.release_prep_env import release_prep_env  # noqa: E402
```

### 2. Flaky `hub_auth_test` under local automation key

`test_ensure_hub_session_caches_token` failed locally because `~/.neural-junkie/automation.key` exists, so `ensure_hub_session()` short-circuits and returns `""` instead of exercising the session path. The test now mocks `load_automation_api_key` to return `""` so it stays isolated without weakening any assertion.

## Verification commands run

| Command | Result |
|---------|--------|
| `make test-all` | **PASS** (~136s) |
| `make test-conversation-contract` | **FAIL** → fixed → **PASS** |
| `python3 -m py_compile scripts/implement-scenarios.py scripts/lib/hub_auth_test.py` | **PASS** |

Live `implement-scenarios` was not run (per your CI-first priority; the gate failure was a harness import bug, not a scenario assertion failure). If you want, I can run `make layer-gate LAYER=implement` next to confirm 20/20 on a live hub.
```

