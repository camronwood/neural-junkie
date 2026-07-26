# Model benchmark suite

Compare **local Ollama coder models** against the same NJ live scenarios and produce ranked benchmarks.

## Prerequisites

`make model-benchmark` **auto-boots** Ollama, pulls/warms the suite roster, starts the regression hub, and waits for agents — same path as `implement-scenarios` / `layer-gate`.

Skip boot when the stack is already up:

```bash
SKIP_BOOT=1 make model-benchmark SUITE=standard
```

Optional: `make collab-preflight` for a faster smoke check. Install models you want to compare (or let the default `--pull` / boot warm handle it):

```bash
ollama pull qwen2.5-coder:14b
ollama pull qwen3.5:9b
ollama pull gemma3:12b
```

## Quick smoke benchmark (default)

Runs **5 implement + 2 chat** scenarios against the **quick roster** (6 models, ≤~9 GB footprint; ~15–45 min per model):

```bash
make model-benchmark
# or
PYTHONUNBUFFERED=1 make model-benchmark SUITE=quick VERBOSE=1
```

## Suites

| Suite | What it runs |
|-------|----------------|
| `quick` | 5 implement (`go-handler`, `theme-toggle`, `ask-mode-no-write`, `go-test-failure-repair`, `typescript-compile-error-fix`) + 2 chat DMs; ≤~9 GB |
| `release` | Release gate (winners only) — 6 implement + 2 chat + arena `logic-set`; isolated deliverable judge; min model pass rate |
| `standard` | Broader implement set + chat `regression` tag + arena `logic-set` + external `humaneval-25` |
| `implement` | Full `scenarios/implement/` |
| `chat-regression` | All chat scenarios with `regression` tag |
| `collab` | Collab core track (`collab: "core"`) |
| `user-flows` | Full real-world product journeys (~7; trip, games, APIs, websites, boot fix) |
| `user-flows-quick` | Lighter user-flow sample (trip, blackjack, Node CRUD, boot fix) |
| `arena` | Arena track: `logic-set`, `connect4-smoke` |
| `cad` | CAD compile track: `cad-compile` |
| `external` | HumanEval-25 calibration |

```bash
make model-benchmark-list
make model-benchmark SUITE=standard
make model-benchmark SUITE=release
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
| `model-benchmark-<suite>-<timestamp>.json` | Machine-readable full results (hardware, scenario catalog, judge verdicts, metrics) |
| `model-benchmark-<suite>-<timestamp>.tsv` | One row per scenario × model (includes token/TTFT/quality columns) |

After each run, results are merged into **`docs/data/model-benchmarks.json`** for the public site:

```bash
make publish-model-benchmarks
```

Publishing also regenerates **`docs/data/model-capability-profiles.json`** — ranked Ollama model lists per task class (`implement`, `chat`, `collab_light`, `utility`, `ask_mode`, `implement_heavy`, plus `arena_logic` / `cad_compile` when those tracks appear). The hub loads these at startup for **benchmark model routing** (Settings → Collab routing → *Benchmark model routing*).

Override profile path: `NEURAL_JUNKIE_CAPABILITY_PROFILES=/path/to/profiles.json`

**Website:** [Model benchmarks](https://camronwood.github.io/neural-junkie/benchmarks/) (`docs/benchmarks/index.html` on GitHub Pages).

Each published run includes:

- **Hardware** — RAM tier from `GET /api/system/hardware` at benchmark start (when hub is local)
- **Scenario catalog** — description, agent, and whether an LLM deliverable judge runs
- **Judge verdicts** — Pass/Fail/Warn + reason from the independent judge on `llm_judge` implement scenarios
- **Track lists** — implement / chat / collab / arena / cad / external as present

## Default roster

Configured in `scripts/config/model-benchmark-models.json`. Cap is **memory footprint** (`size_hint_gb`), not parameter count — default **≤~9 GB** (about the footprint of Qwen 2.5 Coder 14B). Larger in-memory weights get too slow for routine tests.

**Quick suite roster:**

1. `deepseek-coder:6.7b` — lightweight reference (~3.8 GB)
2. `nj-bonsai:27b` — PrismML 1-bit Bonsai (~3.5 GB; 27B params, small footprint)
3. `codegemma:7b` — Google code-focused instruct
4. `qwen3.5:9b` — fast modern baseline
5. `gemma3:12b` — Google Gemma 3 mid-tier (~8.1 GB)
6. `qwen2.5-coder:14b` — NJ-proven specialist (~9 GB; defines the speed ceiling)

Catalog models above ~9 GB (`codestral:22b` ~13 GB, `devstral:24b` ~14 GB) require `--allow-large-models` / `BENCHMARK_ALLOW_LARGE=1`. Ternary Bonsai (~7.2 GB) is under the default cap but not in the quick suite list.

```bash
make model-benchmark SUITE=quick PULL=1
# include heavier catalog weights:
make model-benchmark SUITE=standard BENCHMARK_ALLOW_LARGE=1 SKIP_MISSING=1
```

Or pull/run the large catalog slice:

```bash
make pull-benchmark-models SUITE=standard PULL_ALL=1
make model-benchmark SUITE=standard BENCHMARK_ALLOW_LARGE=1 SKIP_MISSING=1
```

Edit that file or pass `--models` to change the roster.

## Notes

- Each model run calls `POST /api/agents/switch-all-providers` with `{provider: ollama, model: <tag>}`.
- Arena / CAD / external tracks pass `--model <runtime_tag>` and `--hub`.
- Full **6-model × quick suite** can take several hours on a laptop — start with `MODELS='qwen2.5-coder:14b,nj-bonsai:27b'` or a shorter list.
- Release suite (`SUITE=release`) isolates the deliverable judge from SUTs and requires every active model to meet `--min-model-pass-rate` (default 0.5) plus the winner gate.
