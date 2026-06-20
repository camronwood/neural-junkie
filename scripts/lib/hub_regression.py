"""Helpers for live regression harnesses (hub health, optional restart)."""

from __future__ import annotations

import os
import subprocess
import time
import urllib.error
from pathlib import Path

from lib import collab_hub as hub


def wait_for_hub(base: str, timeout_s: float = 120.0) -> bool:
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        try:
            if hub.check_health(base):
                return True
        except urllib.error.URLError:
            pass
        time.sleep(2.0)
    return False


def stop_hub(repo_root: Path) -> None:
    subprocess.run(
        ["make", "stop"],
        cwd=repo_root,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    time.sleep(2.0)


def start_regression_hub(
    repo_root: Path,
    *,
    env: dict[str, str] | None = None,
) -> subprocess.Popen[bytes] | None:
    """Start make server-regression in background; returns Popen or None on failure."""
    merged = os.environ.copy()
    if env:
        merged.update(env)
    try:
        proc = subprocess.Popen(
            ["make", "server-regression"],
            cwd=repo_root,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            start_new_session=True,
            env=merged,
        )
    except OSError:
        return None
    return proc


def restart_regression_hub(
    repo_root: Path,
    hub_url: str,
    timeout_s: float = 120.0,
    *,
    env: dict[str, str] | None = None,
) -> bool:
    stop_hub(repo_root)
    if start_regression_hub(repo_root, env=env) is None:
        return False
    return wait_for_hub(hub_url, timeout_s=timeout_s)
