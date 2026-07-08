"""Invoke Cursor CLI agent for release-prep fix loops."""

from __future__ import annotations

import os
import shutil
import subprocess
import sys
import threading
from pathlib import Path

DEFAULT_AGENT_TIMEOUT_S = 18_000  # 5h — release-prep fixes often exceed 3h

# subprocess exit codes for signals / our timeout wrapper
AGENT_EXIT_TIMEOUT = 124
RECOVERABLE_AGENT_EXITS = frozenset({AGENT_EXIT_TIMEOUT, 130, 137, 143})


def agent_exit_label(rc: int) -> str:
    if rc == AGENT_EXIT_TIMEOUT:
        return "timed out (fix-loop limit)"
    if rc == 130:
        return "SIGINT (interrupted — Ctrl+C?)"
    if rc == 137:
        return "SIGKILL (killed — OOM or force quit?)"
    if rc == 143:
        return "SIGTERM (terminated — sleep, tmux kill, or parent stopped?)"
    return f"exit {rc}"


def build_agent_invoke_message(prompt_path: Path, *, repo_root: Path) -> str:
    """Small argv-safe prompt; agent reads the full brief from disk."""
    resolved = prompt_path.resolve()
    try:
        rel = resolved.relative_to(repo_root.resolve())
        path_str = str(rel)
    except ValueError:
        path_str = str(resolved)
    return (
        "Neural Junkie release-prep fix loop.\n\n"
        f"Read the full failure brief at `{path_str}` in this repo and fix code issues per its rules.\n"
        "Priority: CI failures (`make test-all`, `make test-conversation-contract`) before live scenarios.\n"
        "Do NOT weaken test assertions. Run verification commands listed in that file when done.\n"
        "Summarize what you changed and which commands you ran."
    )


def agent_binary() -> str | None:
    return shutil.which("agent")


def invoke_cursor_agent(
    prompt: str,
    *,
    cwd: Path,
    model: str | None = None,
    timeout_s: int = DEFAULT_AGENT_TIMEOUT_S,
    api_key: str | None = None,
    log_path: Path | None = None,
) -> tuple[int, str]:
    """Run `agent -p` headless with workspace trust. Returns (exit_code, combined_output)."""
    binary = agent_binary()
    if not binary:
        return 127, "Cursor CLI 'agent' not found on PATH (curl https://cursor.com/install -fsS | bash)"

    cmd = [
        binary,
        "--trust",
        "--force",
        "-p",
        "--output-format",
        "text",
    ]
    if model:
        cmd.extend(["--model", model])
    cmd.append(prompt)

    env = os.environ.copy()
    key = api_key or env.get("CURSOR_API_KEY", "").strip()
    if key:
        env["CURSOR_API_KEY"] = key

    print(f"\n>>> [cursor-agent] cwd={cwd} timeout={timeout_s}s", flush=True)
    if log_path:
        print(f">>> [cursor-agent] streaming log: {log_path}", flush=True)
        log_path.parent.mkdir(parents=True, exist_ok=True)

    proc = subprocess.Popen(
        cmd,
        cwd=cwd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        env=env,
        bufsize=1,
    )
    chunks: list[str] = []
    log_file = log_path.open("w", encoding="utf-8") if log_path else None

    # Fail fast on auth issues so overnight fix loops don't burn hours before
    # discovering Cursor needs re-login. Opt-out with NJ_SKIP_CURSOR_AGENT_AUTH_PREFLIGHT=1.
    auth_preflight = os.environ.get("NJ_SKIP_CURSOR_AGENT_AUTH_PREFLIGHT", "").strip().lower() not in ("1", "true", "yes")
    auth_failed = False

    def _drain() -> None:
        assert proc.stdout is not None
        for line in proc.stdout:
            chunks.append(line)
            sys.stdout.write(line)
            sys.stdout.flush()
            if log_file:
                log_file.write(line)
                log_file.flush()
            nonlocal auth_failed
            if auth_preflight and not auth_failed:
                low = line.lower()
                if "actionrequirederror" in low or ("authentication error" in low and "try logging out" in low):
                    auth_failed = True
                    try:
                        proc.kill()
                    except Exception:
                        pass

    reader = threading.Thread(target=_drain, daemon=True)
    reader.start()
    rc = 1
    try:
        rc = proc.wait(timeout=timeout_s)
    except subprocess.TimeoutExpired:
        proc.kill()
        reader.join(timeout=5.0)
        if log_file:
            log_file.write(f"\n\n[fix-loop] Cursor agent timed out after {timeout_s}s\n")
            log_file.flush()
            log_file.close()
        partial = "".join(chunks)
        msg = partial.strip() or f"Cursor agent timed out after {timeout_s}s (no stdout captured)"
        return AGENT_EXIT_TIMEOUT, msg
    finally:
        reader.join(timeout=2.0)
        if log_file and not log_file.closed:
            exit_rc = proc.returncode if proc.returncode is not None else rc
            if exit_rc != 0:
                log_file.write(f"\n\n[fix-loop] agent {agent_exit_label(exit_rc)}\n")
                log_file.flush()
            log_file.close()

    out = "".join(chunks)
    rc = proc.returncode if proc.returncode is not None else rc
    if auth_failed:
        return 2, out.strip() or "ActionRequiredError: authentication error (log out/in to Cursor)"
    if rc != 0 and not out.strip():
        out = f"agent {agent_exit_label(rc)} with no stdout captured"
    return rc, out


def try_cursor_sdk_agent(
    prompt: str,
    *,
    cwd: Path,
    model: str = "composer-2.5",
    api_key: str | None = None,
) -> tuple[int, str] | None:
    """Use cursor_sdk when installed; return None to fall back to CLI."""
    try:
        from cursor_sdk import Agent, AgentOptions, CursorAgentError, LocalAgentOptions
    except ImportError:
        return None

    key = api_key or os.environ.get("CURSOR_API_KEY", "").strip()
    if not key:
        return None

    try:
        result = Agent.prompt(
            prompt,
            AgentOptions(
                api_key=key,
                model=model,
                local=LocalAgentOptions(cwd=str(cwd)),
            ),
        )
    except CursorAgentError as err:
        return 1, f"CursorAgentError: {err.message} (retryable={err.is_retryable})"

    text = getattr(result, "result", "") or str(result)
    status = getattr(result, "status", "finished")
    if status == "error":
        return 2, text or "run finished with status=error"
    return 0, text


def run_fix_agent(
    prompt: str,
    *,
    cwd: Path,
    model: str | None = None,
    prefer_sdk: bool = False,
    timeout_s: int = DEFAULT_AGENT_TIMEOUT_S,
    log_path: Path | None = None,
) -> tuple[int, str]:
    if prefer_sdk:
        sdk = try_cursor_sdk_agent(prompt, cwd=cwd, model=model or "composer-2.5")
        if sdk is not None:
            return sdk
    return invoke_cursor_agent(
        prompt,
        cwd=cwd,
        model=model,
        timeout_s=timeout_s,
        log_path=log_path,
    )
