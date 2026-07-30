# Layer gate — parity — 2026-07-30-1438 UTC

layer=parity
hub=http://127.0.0.1:18765
Overall: **FAIL** (0/1 stages)

## Stage summary

| Stage | Status | Duration | Exit |
|-------|--------|----------|------|
| `test-parity-stable-restart` | FAIL | 259s | -15 |

## Child artifacts

- `/Users/camronwood/development/projects/neural-junkie/docs/testing/layer-gate-parity-2026-07-30-1438.log`

## Failures (tail)

### test-parity-stable-restart (exit -15)

```text
>>> python3 scripts/implement-scenarios-stable.py --runs 3 --min-pass 14 --restart-between --hub http://127.0.0.1:18765


>>> [parity-stable preflight] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)

>>> [parity-stable run-1 preflight] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)

>>> [parity-stable run-2 preflight] fixture collabs + hub channel cleanup
  fixture collabs: already clean
  cleared 5 scenario channel(s)

RESULT test-parity-stable-restart: FAIL (exit -15, 259s)
```

