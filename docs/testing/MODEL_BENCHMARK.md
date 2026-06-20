# Model benchmark suite

Compare **local Ollama coder models** against the same NJ live scenarios and produce ranked benchmarks.

## Prerequisites

```bash
make server-regression    # terminal 1
make agents               # terminal 1b
make collab-preflight     # optional sanity check
```

Install models you want to compare (or use `PULL=1`):

```bash
ollama pull qwen2.5-coder:14b
ollama pull codestral:22b
ollama pull qwen3.5:9b
```

## Quick smoke benchmark (default)

Runs **3 implement + 2 chat** scenarios against the **quick roster** (7 models, ≤24B; ~15–45 min per model):

```bash
make model-benchmark
# or
PYTHONUNBUFFERED=1 make model-benchmark SUITE=quick VERBOSE=1
```

## Suites

| Suite | What it runs |
|-------|----------------|
| `quick` | `go-handler`, `theme-toggle`, `ask-mode-no-write` + 2 chat DMs |
| `standard` | 4 implement scenarios + all chat tagged `regression` |
| `implement` | Full `scenarios/implement/` |
| `chat-regression` | All chat scenarios with `regression` tag |

```bash
make model-benchmark-list
make model-benchmark SUITE=standard
```

## Custom model list

```bash
make model-benchmark MODELS='qwen2.5-coder:14b,qwen3.5:9b'
```

Skip models not installed (no error):

```bash
make model-benchmark SKIP_MISSING=1
```

Pull missing models via hub before each model run:

```bash
make model-benchmark PULL=1
```

## Output

Reports land in `docs/testing/`:

| File | Contents |
|------|----------|
| `model-benchmark-<suite>-<timestamp>.md` | Ranked summary + per-scenario matrix |
| `model-benchmark-<suite>-<timestamp>.json` | Machine-readable full results |
| `model-benchmark-<suite>-<timestamp>.tsv` | One row per scenario × model |

After each run, results are merged into **`docs/data/model-benchmarks.json`** for the public site:

```bash
make publish-model-benchmarks
```

Publishing also regenerates **`docs/data/model-capability-profiles.json`** — ranked Ollama model lists per task class (`implement`, `chat`, `collab_light`, `utility`, `ask_mode`, `implement_heavy`). The hub loads these at startup for **benchmark model routing** (Settings → Collab routing → *Benchmark model routing*).

Override profile path: `NEURAL_JUNKIE_CAPABILITY_PROFILES=/path/to/profiles.json`

**Website:** [Model benchmarks](https://camronwood.github.io/neural-junkie/benchmarks/) (`docs/benchmarks/index.html` on GitHub Pages).

## Default roster

Configured in `scripts/config/model-benchmark-models.json`:

1. `qwen2.5-coder:14b` — NJ-proven specialist
2. `codestral:22b` — Mistral code
3. `devstral:24b` — Mistral agentic coder
4. `qwen3.5:9b` — fast modern baseline
5. `deepseek-coder:6.7b` — lightweight reference
6. `codegemma:7b` — Google code-focused instruct
7. `gemma3:12b` — Google Gemma 3 mid-tier (~12B class)

Edit that file or pass `--models` to change the roster.

## Notes

- Each model run calls `POST /api/agents/switch-all-providers` with `{provider: ollama, model: <tag>}`.
- Full **5-model × quick suite** can take several hours on a laptop — start with `MODELS='qwen2.5-coder:14b,qwen3.5:9b'` or `SUITE=quick` on two models.
- For release gates, keep using `make implement-scenarios` with your chosen default model.
