"""Helpers for live regression harnesses (hub health, crash recovery, restart)."""

from __future__ import annotations

import os
import subprocess
import sys
import time
import urllib.error
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path

from lib import collab_hub as hub

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_HUB_LOG = Path("/tmp/nj-hub.log")
HUB_RECOVERY_LOG_ENV = "NEURAL_JUNKIE_HUB_RECOVERY_LOG"
DEFAULT_MAX_RECOVERY_ATTEMPTS = 3


@dataclass
class HubRecoveryEvent:
    utc: str
    context: str
    attempts: int
    recovered: bool
    detail: str
    log_tail: str = ""

    def to_log_block(self) -> str:
        lines = [
            f"--- hub recovery {self.utc} UTC ---",
            f"context: {self.context}",
            f"attempts: {self.attempts}",
            f"recovered: {self.recovered}",
            f"detail: {self.detail}",
        ]
        if self.log_tail.strip():
            lines.append("hub log tail:")
            lines.append(self.log_tail.rstrip())
        lines.append("")
        return "\n".join(lines)

    def to_md_line(self) -> str:
        status = "recovered" if self.recovered else "FAILED"
        return f"- `{self.utc}` **{status}** — {self.context} ({self.attempts} attempt(s)): {self.detail}"


@dataclass
class HubRecoveryJournal:
    events: list[HubRecoveryEvent] = field(default_factory=list)

    def record(self, event: HubRecoveryEvent) -> None:
        self.events.append(event)
        _append_recovery_log(event)

    def format_md(self) -> str:
        if not self.events:
            return ""
        lines = ["## Hub recovery events", ""]
        lines.extend(e.to_md_line() for e in self.events)
        lines.append("")
        return "\n".join(lines)

    def format_log(self) -> str:
        if not self.events:
            return ""
        return "\n".join(e.to_log_block() for e in self.events)


def recovery_log_path() -> Path | None:
    raw = os.environ.get(HUB_RECOVERY_LOG_ENV, "").strip()
    return Path(raw) if raw else None


def _append_recovery_log(event: HubRecoveryEvent) -> None:
    path = recovery_log_path()
    if path is None:
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as f:
        f.write(event.to_log_block())


def read_recovery_log_text() -> str:
    path = recovery_log_path()
    if path is None or not path.is_file():
        return ""
    return path.read_text(encoding="utf-8")


def read_hub_log_tail(path: Path = DEFAULT_HUB_LOG, *, max_lines: int = 40) -> str:
    if not path.is_file():
        return f"(no hub log at {path})"
    try:
        lines = path.read_text(encoding="utf-8", errors="replace").splitlines()
    except OSError as e:
        return f"(could not read {path}: {e})"
    if not lines:
        return f"(empty hub log at {path})"
    tail = lines[-max_lines:]
    return "\n".join(tail)


def hub_is_healthy(base: str) -> bool:
    try:
        return hub.check_health(base) is not None
    except (urllib.error.URLError, OSError, TimeoutError):
        return False


def wait_for_hub(base: str, timeout_s: float = 120.0) -> bool:
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        if hub_is_healthy(base):
            return True
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
    if not wait_for_hub(hub_url, timeout_s=timeout_s):
        return False
    try:
        from lib.hub_auth import refresh_hub_auth_after_restart

        refresh_hub_auth_after_restart(hub_url)
    except Exception:
        pass
    return True


def recover_regression_hub(
    repo_root: Path,
    hub_url: str,
    *,
    context: str = "",
    max_attempts: int = DEFAULT_MAX_RECOVERY_ATTEMPTS,
    timeout_s: float = 120.0,
    env: dict[str, str] | None = None,
    journal: HubRecoveryJournal | None = None,
) -> bool:
    """Document hub crash and try up to max_attempts restarts before giving up."""
    if hub_is_healthy(hub_url):
        return True

    log_tail = read_hub_log_tail()
    label = context or "unspecified"
    print(f"\n⚠ hub crash detected [{label}] at {hub_url}", file=sys.stderr)
    print(f"  documenting crash; up to {max_attempts} restart attempt(s)", file=sys.stderr)

    last_detail = "unknown"
    for attempt in range(1, max_attempts + 1):
        print(f"  → recovery attempt {attempt}/{max_attempts} (make stop && make server-regression)", file=sys.stderr)
        if restart_regression_hub(repo_root, hub_url, timeout_s=timeout_s, env=env):
            detail = f"hub healthy after restart attempt {attempt}"
            print(f"  ✓ {detail}", file=sys.stderr)
            event = HubRecoveryEvent(
                utc=datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M%S"),
                context=label,
                attempts=attempt,
                recovered=True,
                detail=detail,
                log_tail=log_tail,
            )
            if journal is not None:
                journal.record(event)
            else:
                _append_recovery_log(event)
            time.sleep(3.0)
            return True
        last_detail = f"restart attempt {attempt} did not yield healthy hub within {timeout_s:.0f}s"
        print(f"  ✗ {last_detail}", file=sys.stderr)
        time.sleep(5.0)

    detail = f"recovery exhausted after {max_attempts} attempt(s): {last_detail}"
    print(f"  ✗ {detail}", file=sys.stderr)
    event = HubRecoveryEvent(
        utc=datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M%S"),
        context=label,
        attempts=max_attempts,
        recovered=False,
        detail=detail,
        log_tail=log_tail,
    )
    if journal is not None:
        journal.record(event)
    else:
        _append_recovery_log(event)
    return False


def ensure_hub_with_recovery(
    repo_root: Path,
    hub_url: str,
    *,
    context: str = "",
    max_attempts: int = DEFAULT_MAX_RECOVERY_ATTEMPTS,
    timeout_s: float = 120.0,
    env: dict[str, str] | None = None,
    journal: HubRecoveryJournal | None = None,
) -> bool:
    """Return True when hub is healthy, recovering automatically if needed."""
    if hub_is_healthy(hub_url):
        return True
    return recover_regression_hub(
        repo_root,
        hub_url,
        context=context,
        max_attempts=max_attempts,
        timeout_s=timeout_s,
        env=env,
        journal=journal,
    )


def output_suggests_hub_crash(text: str) -> bool:
    needles = (
        "hub not healthy",
        "hub unhealthy",
        "hub down",
        "Remote end closed connection",
        "RemoteDisconnected",
        "Connection refused",
        "Connection reset",
        "urlopen error",
    )
    lower = (text or "").lower()
    return any(n.lower() in lower for n in needles)
