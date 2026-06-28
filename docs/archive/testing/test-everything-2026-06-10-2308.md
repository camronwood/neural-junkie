# test-everything — 2026-06-10-2308 UTC

- Hub: `http://127.0.0.1:18765`
- Full collab sweep (`FULL=1`): `False`
- Skip live: `False`
- Overall: **FAIL** (7/8 stages)

## Stage summary

| Stage | Status | Duration |
|-------|--------|----------|
| `test-all` | OK | 93s |
| `test-conversation-contract` | OK | 7s |
| `test-collab-plan` | OK | 1s |
| `test-scenario-assert` | OK | 0s |
| `collab-smoke` | OK | 1s |
| `learning-lora-smoke` | OK | 3s |
| `collab-preflight` | OK | 0s |
| `implement-scenarios` | FAIL | 2471s |

## Artifacts

- Full log: `/Users/camronwood/development/projects/neural-junkie/docs/testing/test-everything-2026-06-10-2308.log`

## Failures (tail)

### implement-scenarios (exit 1)

```text
=== implement: ask-mode-no-write ===
  ✓ [1] send: sent
  ✓ [2] wait_reply: reply from BackendEngineer
  ✓ [3] assert_no_file_change: no file changes
=== PASS: ask-mode-no-write ===


=== implement: continuation-go-ahead ===
  ✓ [1] send: sent
  ✓ [2] wait_reply: reply from BackendEngineer (ok)
  ✓ [3] send: sent
  ✓ [4] wait_reply: reply from BackendEngineer (ok)
  ✓ [5] assert_messages: message assertions ok
  ✓ [6] assert_file_exists: core/sample/main.go
=== PASS: continuation-go-ahead ===


=== implement: general-workspace-implement ===
  ✓ [1] send: sent
  ✗ [2] wait_reply: timeout waiting for FrontendEngineer
=== FAIL: general-workspace-implement ===


=== implement: go-handler ===
  ✓ [1] send: sent
  ✓ [2] wait_reply: reply from BackendEngineer (ok)
  ✓ [3] assert_messages: message assertions ok
  ✓ [4] assert_file_exists: core/sample/main.go
=== PASS: go-handler ===


=== implement: react-theme-multi-file ===
  ✓ [1] send: sent
  ✓ [2] wait_reply: reply from FrontendEngineer (ok)
  ✓ [3] assert_messages: message assertions ok
  ✓ [4] assert_file_exists: tailwind.config.js
  ✓ [5] assert_file_exists: src/App.tsx
=== PASS: react-theme-multi-file ===


=== implement: react-theme-toggle ===
  ✓ [1] send: sent
  ✗ [2] wait_reply: timeout waiting for FrontendEngineer
=== FAIL: react-theme-toggle ===


=== implement: theme-toggle ===
  ✓ [1] send: sent
  ✓ [2] wait_reply: reply from FrontendEngineer (ok)
  ✓ [3] assert_messages: message assertions ok
  ✓ [4] assert_file_exists: src/theme.css
=== PASS: theme-toggle ===


=== implement: vite-boot-fix-corrupt-appjs ===
  ✓ [1] send: sent
  ✓ [2] send: sent
  ✗ [3] wait_reply: timeout waiting for SoftwareArchitect
=== FAIL: vite-boot-fix-corrupt-appjs ===
```

