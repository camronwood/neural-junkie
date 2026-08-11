# Surface reliability — 2026-08-11-1544

Overall: **FAIL**

| Gate | Status | Detail |
|------|--------|--------|
| `stamp` | PASS | action=0.959 misstamp=0.027 (action>=0.9 misstamp<=0.05) |
| `memory` | PASS | hit=1.000 forbidden=0.000 (hit>=0.9 forbidden<=0.05) |
| `session` | FAIL | pass=3 fail=1 (all work-surface scenarios PASS @1) |

Artifacts:

- semantic_eval: `docs/testing/semantic-eval-2026-08-11-1544.json`
- memory_eval: `docs/testing/memory-eval-2026-08-11-1544.json`
- session_log: `docs/testing/work-surface-2026-08-11-1544.log`
