"""Model Arena routes and deterministic challenge engines."""
from __future__ import annotations

import json
import os
import re
import subprocess
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

try:
    import chess  # type: ignore
except Exception:  # pragma: no cover - optional until setup installs python-chess
    chess = None

ROWS = 6
COLS = 7

DEFAULT_PUZZLES: list[dict[str, Any]] = [
    {
        "id": "kk-001",
        "difficulty": "easy",
        "title": "One truth teller",
        "prompt": "On an island, Ada says: 'Babbage is a knight.' Babbage says: 'Ada and I are different types.' Knights always tell the truth and knaves always lie. What is Ada?",
        "answer": "knave",
        "accepted": ["ada is a knave", "knave"],
        "explanation": "If Ada is a knight, Babbage is a knight, but then Babbage's statement is false. If Ada is a knave, her statement is false, so Babbage is a knave, but then Babbage's statement is false and he can be a knave. Ada is a knave.",
    },
    {
        "id": "kk-002",
        "difficulty": "easy",
        "title": "Same type",
        "prompt": "Ada says: 'Babbage and I are the same type.' Babbage says: 'Ada is a knave.' What is Ada?",
        "answer": "knave",
        "accepted": ["ada is a knave", "knave"],
        "explanation": "Ada cannot be a knight because Babbage would be a knight and falsely call Ada a knave. Ada is a knave.",
    },
    {
        "id": "kk-003",
        "difficulty": "medium",
        "title": "Three islanders",
        "prompt": "Ada says: 'Babbage is a knave.' Babbage says: 'Curie is a knight.' Curie says: 'Ada and I are different types.' What is Curie?",
        "answer": "knight",
        "accepted": ["curie is a knight", "knight"],
        "explanation": "Curie and Babbage are knights and Ada is a knave. Ada's claim that Babbage is a knave is false, Babbage truthfully identifies Curie, and Curie truthfully says she differs from Ada.",
    },
    {
        "id": "kk-004",
        "difficulty": "medium",
        "title": "The matching claim",
        "prompt": "Ada says: 'Babbage is a knave.' Babbage says: 'Ada and I are the same type.' What is Ada?",
        "answer": "knight",
        "accepted": ["ada is a knight", "knight"],
        "explanation": "Ada is a knight and Babbage is a knave. Ada truthfully identifies Babbage, while Babbage falsely claims they are the same type.",
    },
    {
        "id": "kk-005",
        "difficulty": "hard",
        "title": "Exactly one knight",
        "prompt": "Ada says: 'Babbage is a knave.' Babbage says: 'Exactly one of us is a knight.' What is Babbage?",
        "answer": "knight",
        "accepted": ["babbage is a knight", "knight"],
        "explanation": "If Ada were a knight, Babbage would be a knave whose statement was true, which is impossible. Therefore Ada is a knave and Babbage is a knight; exactly one is a knight.",
    },
]


def _json(handler: Any, code: int, payload: dict[str, Any]) -> None:
    handler._json(code, payload)


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


def _expand(path: str) -> Path:
    return Path(os.path.expanduser(path.strip()))


def _sessions_dir(settings: dict[str, str]) -> Path:
    root = settings.get("arena_sessions_dir") or "~/.neural-junkie/arena/sessions"
    path = _expand(root)
    path.mkdir(parents=True, exist_ok=True)
    return path


def _session_path(settings: dict[str, str], session_id: str) -> Path:
    safe = re.sub(r"[^a-zA-Z0-9_-]", "", session_id)
    return _sessions_dir(settings) / f"{safe}.json"


def _leaderboard_path(settings: dict[str, str]) -> Path:
    return _sessions_dir(settings) / "leaderboard.json"


def _load_json(path: Path, default: Any) -> Any:
    if not path.is_file():
        return default
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return default


def _save_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True), encoding="utf-8")


def _load_session(settings: dict[str, str], session_id: str) -> dict[str, Any] | None:
    path = _session_path(settings, session_id)
    if not path.is_file():
        return None
    return _load_json(path, None)


def _save_session(settings: dict[str, str], session: dict[str, Any]) -> dict[str, Any]:
    session["updated_at"] = _now()
    _save_json(_session_path(settings, session["id"]), session)
    return session


def _challenges() -> list[dict[str, Any]]:
    return [
        {
            "id": "chess",
            "title": "Chess",
            "description": "Legal-move chess with optional Stockfish evaluation.",
            "players": 2,
            "move_format": "uci",
            "available": chess is not None,
            "setup_hint": "" if chess is not None else "Install python-chess in the pack sidecar environment.",
        },
        {
            "id": "connect4",
            "title": "Connect Four",
            "description": "Fast tactical board game on a 6x7 grid.",
            "players": 2,
            "move_format": "column",
            "available": True,
        },
        {
            "id": "logic",
            "title": "Logic puzzles",
            "description": "Knights-and-knaves deduction with exact answer checking. Two models can duel on the same puzzle.",
            "players": 2,
            "move_format": "answer",
            "available": True,
        },
    ]


def _puzzles() -> list[dict[str, Any]]:
    return DEFAULT_PUZZLES


def _public_puzzles() -> list[dict[str, Any]]:
    return [
        {
            "id": p["id"],
            "difficulty": p["difficulty"],
            "title": p["title"],
            "prompt": p["prompt"],
        }
        for p in _puzzles()
    ]


def _find_puzzle(puzzle_id: str | None) -> dict[str, Any]:
    puzzles = _puzzles()
    if puzzle_id:
        for p in puzzles:
            if p["id"] == puzzle_id:
                return p
    return puzzles[0]


def _empty_connect4() -> list[list[str]]:
    return [["" for _ in range(COLS)] for _ in range(ROWS)]


def _connect4_legal(board: list[list[str]]) -> list[int]:
    return [c for c in range(COLS) if board[0][c] == ""]


def _connect4_drop(board: list[list[str]], col: int, token: str) -> tuple[list[list[str]], int]:
    if col < 0 or col >= COLS:
        raise ValueError("column out of range")
    if board[0][col] != "":
        raise ValueError("column is full")
    nxt = [row[:] for row in board]
    for row in range(ROWS - 1, -1, -1):
        if nxt[row][col] == "":
            nxt[row][col] = token
            return nxt, row
    raise ValueError("column is full")


def _connect4_winner(board: list[list[str]]) -> str:
    directions = [(1, 0), (0, 1), (1, 1), (1, -1)]
    for r in range(ROWS):
        for c in range(COLS):
            token = board[r][c]
            if not token:
                continue
            for dr, dc in directions:
                ok = True
                for i in range(1, 4):
                    rr, cc = r + dr * i, c + dc * i
                    if rr < 0 or rr >= ROWS or cc < 0 or cc >= COLS or board[rr][cc] != token:
                        ok = False
                        break
                if ok:
                    return token
    return ""


def _connect4_status(board: list[list[str]]) -> tuple[str, str]:
    winner = _connect4_winner(board)
    if winner:
        return "finished", "red" if winner == "R" else "yellow"
    if not _connect4_legal(board):
        return "draw", "draw"
    return "active", ""


def _chess_state_from_board(board: Any, settings: dict[str, str]) -> dict[str, Any]:
    legal = [m.uci() for m in board.legal_moves]
    status = "active"
    result = ""
    if board.is_checkmate():
        status = "finished"
        result = "black" if board.turn == chess.WHITE else "white"
    elif board.is_stalemate() or board.is_insufficient_material() or board.can_claim_draw():
        status = "draw"
        result = "draw"
    payload = {
        "fen": board.fen(),
        "ascii": str(board),
        "turn": "white" if board.turn == chess.WHITE else "black",
        "legal_moves": legal,
        "status": status,
        "result": result,
    }
    eval_payload = _stockfish_eval(board.fen(), settings)
    if eval_payload:
        payload["engine_eval"] = eval_payload
    return payload


def _stockfish_eval(fen: str, settings: dict[str, str]) -> dict[str, Any] | None:
    stockfish = settings.get("stockfish_path", "").strip()
    if not stockfish:
        return None
    path = _expand(stockfish)
    if not path.is_file():
        return {"error": "stockfish_path does not exist"}
    try:
        proc = subprocess.run(
            [str(path)],
            input=f"uci\nisready\nposition fen {fen}\ngo depth 8\nquit\n",
            text=True,
            capture_output=True,
            timeout=5,
            check=False,
        )
    except Exception as exc:
        return {"error": str(exc)}
    best = ""
    score = ""
    for line in proc.stdout.splitlines():
        if " score " in line:
            score = line.strip()
        if line.startswith("bestmove "):
            best = line.split()[1]
    return {"bestmove": best, "raw_score": score}


def _session_public(session: dict[str, Any]) -> dict[str, Any]:
    public = dict(session)
    if public.get("challenge") == "logic" and "puzzle" in public:
        puzzle = dict(public["puzzle"])
        puzzle.pop("answer", None)
        puzzle.pop("accepted", None)
        public["puzzle"] = puzzle
    return public


def _create_session(body: dict[str, Any], settings: dict[str, str]) -> dict[str, Any]:
    challenge = str(body.get("challenge") or "chess").strip().lower()
    if challenge not in {"chess", "connect4", "logic"}:
        raise ValueError("challenge must be chess, connect4, or logic")
    session: dict[str, Any] = {
        "id": uuid.uuid4().hex,
        "challenge": challenge,
        "created_at": _now(),
        "updated_at": _now(),
        "players": {
            "white": body.get("white") or body.get("red") or "human",
            "black": body.get("black") or body.get("yellow") or "model",
            "human_color": body.get("human_color", ""),
        },
        "moves": [],
        "status": "active",
        "result": "",
    }
    if challenge == "chess":
        if chess is None:
            raise ValueError("chess challenge requires python-chess")
        board = chess.Board(str(body.get("fen") or chess.STARTING_FEN))
        session["state"] = _chess_state_from_board(board, settings)
    elif challenge == "connect4":
        board = _empty_connect4()
        status, result = _connect4_status(board)
        session["state"] = {
            "board": board,
            "turn": "red",
            "legal_moves": _connect4_legal(board),
            "status": status,
            "result": result,
        }
    else:
        custom_prompt = str(body.get("custom_prompt") or "").strip()
        custom_answer = str(body.get("custom_answer") or "").strip()
        if custom_prompt or custom_answer:
            if not custom_prompt:
                raise ValueError("custom_prompt is required for a custom logic puzzle")
            if not custom_answer:
                raise ValueError("custom_answer is required to score a custom logic puzzle")
            if len(custom_prompt) > 10_000:
                raise ValueError("custom_prompt must be 10,000 characters or fewer")
            if len(custom_answer) > 500:
                raise ValueError("custom_answer must be 500 characters or fewer")
            puzzle = {
                "id": f"custom-{uuid.uuid4().hex[:12]}",
                "difficulty": "custom",
                "title": "Custom puzzle",
                "prompt": custom_prompt,
                "answer": custom_answer,
                "accepted": [custom_answer],
                "explanation": "Scored against the expected answer supplied by the user.",
            }
        else:
            puzzle = _find_puzzle(str(body.get("puzzle_id") or ""))
        session["puzzle"] = puzzle
        session["state"] = {
            "puzzle_id": puzzle["id"],
            "difficulty": puzzle["difficulty"],
            "prompt": puzzle["prompt"],
            "status": "active",
            "result": "",
        }
    return _save_session(settings, session)


def _record_logic_result(
    leaderboard: dict[str, Any],
    session: dict[str, Any],
    players: dict[str, Any],
    status: str,
) -> None:
    answers = session.get("answers") or {}
    if session.get("answer", {}).get("duel"):
        for seat, entry in answers.items():
            model = str(players.get(seat) or "").strip()
            if not model or model == "human":
                continue
            row = leaderboard["models"].setdefault(
                model,
                {"wins": 0, "losses": 0, "draws": 0, "logic_correct": 0, "logic_total": 0, "illegal_moves": 0},
            )
            row["logic_total"] += 1
            if entry.get("correct"):
                row["logic_correct"] += 1
            duel_result = session.get("result")
            if duel_result == seat:
                row["wins"] += 1
            elif duel_result == "draw":
                row["draws"] += 1
            elif duel_result not in {"", seat}:
                row["losses"] += 1
        return
    model_ids = [str(v) for v in players.values() if str(v).strip() and str(v) != "human"]
    for model in model_ids:
        row = leaderboard["models"].setdefault(
            model,
            {"wins": 0, "losses": 0, "draws": 0, "logic_correct": 0, "logic_total": 0, "illegal_moves": 0},
        )
        row["logic_total"] += 1
        if status == "correct":
            row["logic_correct"] += 1


def _record_result(settings: dict[str, str], session: dict[str, Any]) -> None:
    result = session.get("result") or session.get("state", {}).get("result")
    status = session.get("status") or session.get("state", {}).get("status")
    if status not in {"finished", "draw", "correct", "incorrect"}:
        return
    leaderboard = _load_json(_leaderboard_path(settings), {"models": {}, "sessions": []})
    if session["id"] in leaderboard.get("sessions", []):
        return
    players = session.get("players", {})
    if session["challenge"] == "logic":
        _record_logic_result(leaderboard, session, players, status)
        leaderboard.setdefault("sessions", []).append(session["id"])
        _save_json(_leaderboard_path(settings), leaderboard)
        return
    model_ids = [str(v) for v in players.values() if str(v).strip() and str(v) != "human"]
    for model in model_ids:
        row = leaderboard["models"].setdefault(
            model,
            {"wins": 0, "losses": 0, "draws": 0, "logic_correct": 0, "logic_total": 0, "illegal_moves": 0},
        )
        if result == "draw":
            row["draws"] += 1
        elif result in {"white", "red"} and players.get("white") == model:
            row["wins"] += 1
        elif result in {"black", "yellow"} and players.get("black") == model:
            row["wins"] += 1
        else:
            row["losses"] += 1
    leaderboard.setdefault("sessions", []).append(session["id"])
    _save_json(_leaderboard_path(settings), leaderboard)


def _make_move(session: dict[str, Any], body: dict[str, Any], settings: dict[str, str]) -> dict[str, Any]:
    challenge = session["challenge"]
    if challenge == "chess":
        if chess is None:
            raise ValueError("chess challenge requires python-chess")
        board = chess.Board(session["state"]["fen"])
        move_text = str(body.get("move") or body.get("uci") or "").strip().lower()
        if not move_text and body.get("from") and body.get("to"):
            move_text = f"{body['from']}{body['to']}{body.get('promotion', '')}".lower()
        try:
            move = chess.Move.from_uci(move_text)
        except ValueError as exc:
            raise ValueError(f"invalid UCI move: {move_text}") from exc
        if move not in board.legal_moves:
            raise ValueError(f"illegal move {move_text}")
        board.push(move)
        session["moves"].append({"move": move.uci(), "by": body.get("by", ""), "at": _now()})
        session["state"] = _chess_state_from_board(board, settings)
    elif challenge == "connect4":
        col = int(body.get("column", -1))
        state = session["state"]
        token = "R" if state.get("turn") == "red" else "Y"
        board, row = _connect4_drop(state["board"], col, token)
        status, result = _connect4_status(board)
        next_turn = "yellow" if state.get("turn") == "red" else "red"
        session["moves"].append({"column": col, "row": row, "by": body.get("by", ""), "at": _now()})
        session["state"] = {
            "board": board,
            "turn": next_turn if status == "active" else "",
            "legal_moves": _connect4_legal(board) if status == "active" else [],
            "status": status,
            "result": result,
        }
    else:
        raise ValueError("logic sessions use /answer")
    session["status"] = session["state"]["status"]
    session["result"] = session["state"].get("result", "")
    _record_result(settings, session)
    return _save_session(settings, session)


def _logic_model_seats(players: dict[str, Any]) -> list[str]:
    seats: list[str] = []
    for seat in ("white", "black"):
        tag = str(players.get(seat) or "").strip()
        if tag and tag != "human":
            seats.append(seat)
    return seats


def _answer_is_correct(puzzle: dict[str, Any], answer: str) -> bool:
    answer = str(answer or "").strip().lower()
    accepted = [str(a).strip().lower() for a in puzzle.get("accepted", [])]
    return answer == str(puzzle.get("answer", "")).lower() or answer in accepted


def _finalize_logic_session(session: dict[str, Any]) -> None:
    puzzle = session["puzzle"]
    players = session.get("players", {})
    answers = session.get("answers", {})
    seats = _logic_model_seats(players)
    if not seats:
        seats = ["white"]

    scored: dict[str, bool] = {}
    for seat in seats:
        entry = answers.get(seat) or {}
        scored[seat] = bool(entry.get("correct"))

    if len(seats) == 1:
        seat = seats[0]
        correct = scored.get(seat, False)
        session["status"] = "correct" if correct else "incorrect"
        session["result"] = session["status"]
        session["answer"] = answers.get(seat, {})
        return

    correct_seats = [seat for seat in seats if scored.get(seat)]
    if len(correct_seats) == 1:
        session["status"] = "finished"
        session["result"] = correct_seats[0]
    elif len(correct_seats) >= 2:
        session["status"] = "draw"
        session["result"] = "draw"
    else:
        session["status"] = "finished"
        session["result"] = "draw"

    session["answer"] = {
        "duel": True,
        "answers": answers,
        "winner": session["result"],
        "explanation": puzzle.get("explanation", ""),
    }


def _submit_answer(session: dict[str, Any], body: dict[str, Any], settings: dict[str, str]) -> dict[str, Any]:
    if session["challenge"] != "logic":
        raise ValueError("answer endpoint is only for logic sessions")
    puzzle = session["puzzle"]
    players = session.get("players", {})
    seat = str(body.get("seat") or "").strip().lower()
    if not seat:
        by = str(body.get("by") or "").strip()
        for candidate in ("white", "black"):
            if str(players.get(candidate) or "").strip() == by:
                seat = candidate
                break
    if not seat:
        seat = "white"

    answer_text = str(body.get("answer") or "").strip()
    correct = _answer_is_correct(puzzle, answer_text)
    answers = dict(session.get("answers") or {})
    answers[seat] = {
        "model": body.get("by", ""),
        "seat": seat,
        "answer": body.get("answer", ""),
        "correct": correct,
        "at": _now(),
    }
    session["answers"] = answers

    pending = [s for s in _logic_model_seats(players) if s not in answers]
    if not pending and not _logic_model_seats(players):
        pending = [] if seat in answers else ["white"]

    if pending:
        session["status"] = "active"
        session["result"] = ""
        session["state"]["status"] = "active"
        session["state"]["result"] = ""
        session["state"]["pending_seats"] = pending
        return _save_session(settings, session)

    _finalize_logic_session(session)
    session["state"]["status"] = session["status"]
    session["state"]["result"] = session["result"]
    session["state"].pop("pending_seats", None)
    _record_result(settings, session)
    return _save_session(settings, session)


def _run_eval(settings: dict[str, str]) -> dict[str, Any]:
    puzzles = _puzzles()
    return {
        "ok": True,
        "logic_puzzles": len(puzzles),
        "connect4_smoke": {
            "legal_first_moves": list(range(COLS)),
            "expected_board": f"{ROWS}x{COLS}",
        },
        "note": "Model calls are orchestrated by the hub match runner; sidecar eval provides deterministic fixtures.",
    }


def handle_get(handler: Any, path: str, settings: dict[str, str], pack_dir: str) -> None:
    if path in {"/api/arena", "/api/arena/"}:
        _json(handler, 200, {"ok": True, "challenges": _challenges()})
        return
    if path == "/api/arena/challenges":
        _json(handler, 200, {"challenges": _challenges(), "puzzles": _public_puzzles()})
        return
    if path == "/api/arena/leaderboard":
        _json(handler, 200, _load_json(_leaderboard_path(settings), {"models": {}, "sessions": []}))
        return
    match = re.fullmatch(r"/api/arena/sessions/([A-Za-z0-9_-]+)", path)
    if match:
        session = _load_session(settings, match.group(1))
        if not session:
            _json(handler, 404, {"error": "session not found"})
            return
        _json(handler, 200, _session_public(session))
        return
    _json(handler, 404, {"error": "not found"})


def handle_post(handler: Any, path: str, body: dict, settings: dict[str, str], pack_dir: str) -> None:
    try:
        if path == "/api/arena/sessions":
            _json(handler, 200, _session_public(_create_session(body, settings)))
            return
        if path == "/api/arena/eval/run":
            _json(handler, 200, _run_eval(settings))
            return
        move_match = re.fullmatch(r"/api/arena/sessions/([A-Za-z0-9_-]+)/move", path)
        if move_match:
            session = _load_session(settings, move_match.group(1))
            if not session:
                _json(handler, 404, {"error": "session not found"})
                return
            _json(handler, 200, _session_public(_make_move(session, body, settings)))
            return
        answer_match = re.fullmatch(r"/api/arena/sessions/([A-Za-z0-9_-]+)/answer", path)
        if answer_match:
            session = _load_session(settings, answer_match.group(1))
            if not session:
                _json(handler, 404, {"error": "session not found"})
                return
            _json(handler, 200, _session_public(_submit_answer(session, body, settings)))
            return
    except ValueError as exc:
        _json(handler, 400, {"error": str(exc)})
        return
    except Exception as exc:  # pragma: no cover - user-facing guard at sidecar boundary
        _json(handler, 500, {"error": str(exc)})
        return
    _json(handler, 404, {"error": "not found"})
