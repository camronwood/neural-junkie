# Layer gate — parity — 2026-07-17-2055 UTC

layer=parity
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `test-parity-stable-restart` | FAIL | 0s | 2 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-parity-2026-07-17-2055.log`

## Failures (tail)

### test-parity-stable-restart (exit 2)

```text
>>> python3 scripts/implement-scenarios-stable.py --runs 3 --min-pass 20 --restart-between --hub http://127.0.0.1:18765 --verbose

usage: implement-scenarios-stable.py [-h] [--runs RUNS] [--min-pass MIN_PASS]
                                     [--hub HUB] [--log-dir LOG_DIR]
                                     [--restart-between]
implement-scenarios-stable.py: error: unrecognized arguments: --verbose

RESULT test-parity-stable-restart: FAIL (exit 2, 0s)
```

