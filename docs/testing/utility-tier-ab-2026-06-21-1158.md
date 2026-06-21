# Utility tier A/B — 2026-06-21-1158 UTC

Candidates: `qwen3.5:9b` (current `UtilityOllamaModel`) vs `gemma3:12b` (release smoke co-winner).

## Release smoke benchmark (reference)

- `gemma3:12b`: 5/5 in `model-benchmark-release-2026-06-21-0301` (3m20s)
- `qwen2.5-coder:14b`: 5/5 (coding specialists — not utility)
- `qwen3.5:9b`: not in release suite; quick suite often 4/5

## Rules classifier baseline (model-agnostic)

- `go test ./internal/routing/...`: **PASS**

```text
ok  	github.com/camronwood/neural-junkie/internal/routing	0.302s
ok  	github.com/camronwood/neural-junkie/internal/routing/capabilities	0.451s
```

## Live LLM classifier (`GET /api/debug/routing-classify`)

- Hub: `http://127.0.0.1:18765`
- Reachable: **no** (needs `NEURAL_JUNKIE_DEBUG=1` on hub)

| Query | Expected | OK | domain | cost_tier | source |
|-------|----------|----|--------|-----------|--------|
| security oauth | security | ✗ | — | — | — |
| Fix React component state bug | frontend | ✗ | — | — | — |
| Design REST API endpoint for users | backend | ✗ | — | — | — |
| fix typo in README | cheap | ✗ | — | — | — |
| Explain this protein pathway | biology | ✗ | — | — | — |
| Kubernetes deployment rollout | devops | ✗ | — | — | — |

## Decision (manual)

| Outcome | Action |
|---------|--------|
| Gemma ≥ Qwen on classifier + tools | Switch `UtilityOllamaModel` → `gemma3:12b` |
| Gemma wins smoke only | Keep Qwen utility; document Gemma as optional Assistant upgrade |
| Mixed | Split roles if config supports |

## Next steps for full A/B

1. Run hub with `NEURAL_JUNKIE_DEBUG=1` and Ollama models pulled.
2. Temporarily set `ClassifierModel` / Assistant model to each candidate; rerun this script.
3. Compare session summary quality and Bio/CAD tool loop manually or via harness.
