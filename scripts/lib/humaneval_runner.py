"""Pure helpers for HumanEval-style external calibration runs."""

from __future__ import annotations

import ast
import json
import re
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
DEFAULT_PROBLEMS_PATH = SCRIPTS_DIR / "config" / "humaneval-25.json"

_CODE_FENCE_RE = re.compile(r"```(?:python|py)?\s*\n(.*?)```", re.DOTALL | re.IGNORECASE)


def load_problems(path: Path | None = None) -> tuple[str, list[dict[str, Any]]]:
    """Load curated problem set. Returns (license_note, problems)."""
    p = path or DEFAULT_PROBLEMS_PATH
    data = json.loads(p.read_text(encoding="utf-8"))
    if isinstance(data, list):
        return "", [x for x in data if isinstance(x, dict)]
    note = str(data.get("license_note") or "")
    problems = data.get("problems") or data.get("items") or []
    if not isinstance(problems, list):
        raise ValueError(f"invalid problems list in {p}")
    return note, [x for x in problems if isinstance(x, dict)]


def filter_problems(
    problems: list[dict[str, Any]],
    scenario: str | None,
) -> list[dict[str, Any]]:
    """Select all problems, or a single id / comma list of ids."""
    if not scenario or scenario.strip().lower() in {"", "all", "*"}:
        return list(problems)
    wanted = {s.strip() for s in scenario.split(",") if s.strip()}
    selected = [p for p in problems if str(p.get("id") or "").strip() in wanted]
    missing = wanted - {str(p.get("id") or "").strip() for p in selected}
    if missing:
        raise ValueError(f"unknown problem id(s): {', '.join(sorted(missing))}")
    return selected


def extract_python_code(text: str, entry_point: str | None = None) -> str:
    """Pull a Python function body/module from an LLM response."""
    raw = (text or "").strip()
    if not raw:
        return ""

    fence = _CODE_FENCE_RE.search(raw)
    if fence:
        raw = fence.group(1).strip()
    else:
        # Drop leading prose until a def/import appears.
        lines = raw.splitlines()
        start = 0
        for i, line in enumerate(lines):
            s = line.strip()
            if s.startswith("def ") or s.startswith("import ") or s.startswith("from "):
                start = i
                break
        raw = "\n".join(lines[start:]).strip()

    # Trim trailing markdown fencing leftovers.
    if raw.endswith("```"):
        raw = raw[: raw.rfind("```")].strip()

    if entry_point:
        # Prefer the function matching entry_point when multiple defs exist.
        try:
            tree = ast.parse(raw)
        except SyntaxError:
            return raw
        keep_names = {entry_point}
        # Keep helpers that don't collide with candidate naming.
        for node in tree.body:
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                if node.name == entry_point or not node.name.startswith("check"):
                    keep_names.add(node.name)
            elif isinstance(node, (ast.Import, ast.ImportFrom, ast.Assign, ast.AnnAssign, ast.ClassDef)):
                pass
        # If entry_point is present, keep whole module (helpers often needed).
        names = {
            n.name
            for n in tree.body
            if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef))
        }
        if entry_point in names:
            return raw
    return raw


def build_harness_source(candidate_code: str, test_source: str, entry_point: str) -> str:
    """Compose a temp module that defines candidate + runs check(candidate)."""
    return (
        "# auto-generated HumanEval harness\n"
        "from __future__ import annotations\n"
        f"{candidate_code.rstrip()}\n\n"
        f"{test_source.rstrip()}\n\n"
        f"check({entry_point})\n"
        "print('OK')\n"
    )


def run_harness(
    candidate_code: str,
    test_source: str,
    entry_point: str,
    *,
    timeout_s: float = 5.0,
    python_exe: str | None = None,
) -> tuple[bool, str]:
    """Execute candidate against check(candidate) in a subprocess with timeout."""
    if not candidate_code.strip():
        return False, "empty candidate code"
    if not entry_point.strip():
        return False, "missing entry_point"
    src = build_harness_source(candidate_code, test_source, entry_point)
    exe = python_exe or sys.executable
    with tempfile.TemporaryDirectory(prefix="nj-humaneval-") as tmp:
        path = Path(tmp) / "harness.py"
        path.write_text(src, encoding="utf-8")
        try:
            proc = subprocess.run(
                [exe, str(path)],
                cwd=tmp,
                capture_output=True,
                text=True,
                timeout=timeout_s,
                check=False,
            )
        except subprocess.TimeoutExpired:
            return False, f"timeout after {timeout_s}s"
        if proc.returncode == 0 and "OK" in (proc.stdout or ""):
            return True, "passed"
        err = (proc.stderr or proc.stdout or "").strip()
        if not err:
            err = f"exit {proc.returncode}"
        # Keep failures short for log lines.
        first = err.splitlines()[-1] if err.splitlines() else err
        return False, first[:240]


def ollama_generate(
    ollama_base: str,
    model: str,
    prompt: str,
    *,
    timeout_s: float = 180.0,
) -> dict[str, Any]:
    """POST /api/generate (non-stream). Raises urllib.error.URLError/HTTPError."""
    url = f"{ollama_base.rstrip('/')}/api/generate"
    body = json.dumps(
        {
            "model": model,
            "prompt": prompt,
            "stream": False,
            "options": {"temperature": 0.0},
        }
    ).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=timeout_s) as resp:
        data = json.loads(resp.read().decode("utf-8"))
    if not isinstance(data, dict):
        return {"response": str(data)}
    return data


def tokens_from_ollama(response: dict[str, Any]) -> tuple[int | None, int | None]:
    """Map Ollama generate fields to (prompt_tokens, completion_tokens)."""
    prompt = response.get("prompt_eval_count")
    completion = response.get("eval_count")
    try:
        prompt_n = int(prompt) if prompt is not None else None
    except (TypeError, ValueError):
        prompt_n = None
    try:
        completion_n = int(completion) if completion is not None else None
    except (TypeError, ValueError):
        completion_n = None
    return prompt_n, completion_n


def coding_prompt(problem: dict[str, Any]) -> str:
    entry = str(problem.get("entry_point") or "solution")
    prompt = str(problem.get("prompt") or "").rstrip()
    return (
        "Complete the following Python function. "
        "Reply with ONLY valid Python code for the function "
        f"(and helpers if needed). Do not include tests. Entry point: {entry}.\n\n"
        f"{prompt}\n"
    )


def emit_metrics_line(metrics: dict[str, Any]) -> None:
    print("METRICS_JSON:" + json.dumps(metrics, separators=(",", ":")))
