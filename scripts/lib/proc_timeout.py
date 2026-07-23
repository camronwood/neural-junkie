"""Kill subprocess trees on wall-clock timeout (layer-climb / layer-gate)."""

from __future__ import annotations

import os
import signal
import subprocess
import time
from typing import Callable


def kill_process_tree(proc: subprocess.Popen, *, grace_s: float = 5.0) -> None:
    """SIGTERM the process group (or process), then SIGKILL after grace_s."""
    if proc.poll() is not None:
        return
    pid = proc.pid
    try:
        os.killpg(pid, signal.SIGTERM)
    except (ProcessLookupError, PermissionError, OSError):
        try:
            proc.terminate()
        except ProcessLookupError:
            return
    deadline = time.time() + max(0.1, grace_s)
    while time.time() < deadline:
        if proc.poll() is not None:
            return
        time.sleep(0.2)
    try:
        os.killpg(pid, signal.SIGKILL)
    except (ProcessLookupError, PermissionError, OSError):
        try:
            proc.kill()
        except ProcessLookupError:
            pass
    try:
        proc.wait(timeout=2)
    except subprocess.TimeoutExpired:
        pass


def wait_with_timeout(
    proc: subprocess.Popen,
    timeout_s: float | None,
    *,
    on_tick: Callable[[float], None] | None = None,
    tick_s: float = 5.0,
) -> tuple[int, bool]:
    """Wait for proc; return (exit_code, timed_out).

    When timeout_s is None/<=0, wait forever. Timed-out processes are killed.
    """
    if timeout_s is None or timeout_s <= 0:
        return int(proc.wait()), False
    deadline = time.time() + timeout_s
    next_tick = time.time() + tick_s
    while True:
        rc = proc.poll()
        if rc is not None:
            return int(rc), False
        now = time.time()
        if now >= deadline:
            kill_process_tree(proc)
            try:
                rc = int(proc.wait(timeout=3))
            except subprocess.TimeoutExpired:
                rc = 124
            return rc if rc is not None else 124, True
        if on_tick is not None and now >= next_tick:
            on_tick(max(0.0, deadline - now))
            next_tick = now + tick_s
        time.sleep(0.5)
