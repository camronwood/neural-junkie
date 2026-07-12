# Model Arena pack

Official domain pack: **ArenaExpert**, verifiable chess / Connect Four / logic puzzles, and a live **Arena workbench**.

## Install

1. Desktop **Settings → Domain packs → Pack store** → **Model Arena** → **Install** → **Enable**
2. **Domain packs → Tools → Model Arena** → **Install chess dependencies** (one-time venv + `python-chess`). Connect Four and logic puzzles do not need this step.

Optional manual install:

```bash
python3 -m pip install -r ~/.neural-junkie/packs/model-arena/requirements-sidecar.txt
```

Optional: set `stockfish_path` in Domain packs tools for chess engine eval (Stockfish is not bundled).

## Use

- Toolbar **Arena** chip opens the workbench modal
- Open `*.nj-arena.json` in the file explorer for a dedicated tab
- Chat: `/create-expert arena ArenaExpert` then `@ArenaExpert play connect4 against qwen2.5-coder:14b`

## Architecture

| Layer | Role |
|-------|------|
| ArenaExpert | Hub tools: `arena_create_session`, `arena_get_state`, `arena_make_move`, `arena_submit_answer` |
| Hub match runner | `POST /api/arena/match/step` and `/api/arena/match/run` — prompts any configured provider/model |
| Pack sidecar | Ground truth: legal moves, FEN, Connect Four grid, logic answer checking, leaderboard |

Pack repo: [neural-junkie-pack-model-arena](https://github.com/camronwood/neural-junkie-pack-model-arena)

See also [PACKS.md](./PACKS.md).
