"""Full boot for live regression — Ollama, models, hub, roster, hygiene (one command)."""

from __future__ import annotations

import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
ROOT = SCRIPTS_DIR.parent

sys.path.insert(0, str(SCRIPTS_DIR))

from lib.hub_cleanup import clean_hub_for_regression, wait_for_agent_roster  # noqa: E402
from lib.hub_regression import (  # noqa: E402
    hub_is_healthy,
    restart_regression_hub,
    stop_hub,
    wait_for_hub,
)
from lib.regression_models import enforce_regression_agent_models  # noqa: E402
from lib.release_prep_env import apply_release_prep_env, provision_hub_automation_key, release_prep_env  # noqa: E402

DEFAULT_OLLAMA_HEALTH_URL = "http://127.0.0.1:11434/api/tags"
BOOT_DONE_ENV = "NJ_BOOT_DONE"


def _env_truthy(name: str) -> bool:
    return os.environ.get(name, "").strip().lower() in ("1", "true", "yes")


def should_skip_boot() -> bool:
    if _env_truthy("NJ_REQUIRE_FULL_BOOT"):
        return False
    if _env_truthy("SKIP_BOOT") or _env_truthy(BOOT_DONE_ENV):
        return True
    return False


def ollama_healthy(url: str | None = None) -> bool:
    health = (url or os.environ.get("OLLAMA_HEALTH_URL") or DEFAULT_OLLAMA_HEALTH_URL).strip()
    try:
        with urllib.request.urlopen(health, timeout=5.0) as resp:
            return resp.status == 200
    except (urllib.error.URLError, TimeoutError, OSError):
        return False


def require_ollama(root: Path) -> bool:
    if ollama_healthy():
        print(f"OK: Ollama ({os.environ.get('OLLAMA_HEALTH_URL', DEFAULT_OLLAMA_HEALTH_URL)})")
        return True
    print(">>> Starting Ollama...")
    script = root / "scripts" / "ensure-ollama.sh"
    if not script.is_file():
        print(f"FAIL: missing {script}", file=sys.stderr)
        return False
    proc = subprocess.run(["bash", str(script)], cwd=root, check=False)
    if proc.returncode != 0 and not ollama_healthy():
        print("WARN: ensure-ollama.sh exited non-zero; checking health anyway", file=sys.stderr)
    deadline = time.time() + 45.0
    while time.time() < deadline:
        if ollama_healthy():
            print("OK: Ollama is up")
            return True
        time.sleep(1.0)
    print(
        "FAIL: Ollama required. Install from https://ollama.ai or run: make ensure-ollama",
        file=sys.stderr,
    )
    return False


def ensure_models_ready(root: Path) -> bool:
    suite = os.environ.get("BENCHMARK_SUITE", "release").strip() or "release"
    keep_alive = os.environ.get("NJ_OVERNIGHT_KEEP_ALIVE", "24h").strip() or "24h"
    cmd = [
        sys.executable,
        str(root / "scripts" / "ensure-ollama-models-ready.py"),
        "--warm",
        "--smoke",
        "--keep-alive",
        keep_alive,
        "--suite",
        suite,
    ]
    if not _env_truthy("NO_PULL") or _env_truthy("PULL"):
        cmd.append("--pull-missing")
    models = os.environ.get("BENCHMARK_MODELS", "").strip()
    if models:
        cmd.extend(["--benchmark-models", models])
    if _env_truthy("SKIP_BENCHMARK"):
        cmd.append("--skip-benchmark")
    if _env_truthy("BENCHMARK_ALLOW_LARGE"):
        cmd.append("--allow-large-models")
    print(f">>> Warming Ollama models (suite={suite})...")
    proc = subprocess.run(cmd, cwd=root, check=False)
    return proc.returncode == 0


def clean_environment(root: Path) -> bool:
    print(">>> Stopping Neural Junkie processes...")
    stop_hub(root)
    time.sleep(2.0)

    print(">>> Restoring scenario fixtures from git...")
    proc = subprocess.run(
        ["git", "checkout", "--", "scenarios/fixtures/"],
        cwd=root,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    if proc.returncode != 0:
        print("WARN: git checkout scenarios/fixtures/ failed (continuing)", file=sys.stderr)

    orphan = root / "scenarios" / "fixtures" / "react-vite-corrupt-appjs" / "src" / "App.js"
    if orphan.is_file():
        orphan.unlink(missing_ok=True)
        print(f"  removed orphan {orphan}")

    print(">>> Removing fixture collab runtime dirs...")
    subprocess.run(
        [sys.executable, str(root / "scripts" / "cleanup-test-artifacts.py"), "--fixture-collabs"],
        cwd=root,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    return True


def start_regression_hub_stack(root: Path, hub_url: str, *, no_restart: bool) -> bool:
    base = hub_url.rstrip("/")
    env = release_prep_env(root)
    os.environ.update(env)
    provision_hub_automation_key(root)

    if no_restart and hub_is_healthy(base):
        print(f"OK: reusing healthy hub at {base}")
        return True

    print(">>> Starting regression hub (in-process specialists)...")
    if not restart_regression_hub(root, base, timeout_s=180.0, env=env):
        print("FAIL: hub restart failed", file=sys.stderr)
        return False
    if not wait_for_hub(base, timeout_s=30.0):
        print("FAIL: hub not healthy after restart", file=sys.stderr)
        return False
    print(f"OK: hub healthy at {base}")
    return True


def wait_for_roster(hub_url: str) -> bool:
    print(">>> Waiting for agent roster...")
    ok, missing = wait_for_agent_roster(hub_url.rstrip("/"), timeout_s=240.0)
    if not ok:
        print(f"FAIL: agents still offline after 240s: {', '.join(missing)}", file=sys.stderr)
        return False
    print("OK: required agents online")
    return True


def pin_regression_models(hub_url: str, root: Path) -> bool:
    """Switch in-process agents to ≤14B models (hub config may point at 27B+)."""
    print(">>> Pinning regression agent models (≤14B)...")
    ok, detail = enforce_regression_agent_models(hub_url.rstrip("/"), root)
    if not ok:
        print(f"FAIL: {detail}", file=sys.stderr)
        return False
    print(f"OK: {detail}")
    return True


def hub_hygiene(root: Path, hub_url: str, *, label: str) -> bool:
    print(">>> Hub hygiene (pending file changes + scenario channels)...")
    if not clean_hub_for_regression(root, hub_url.rstrip("/"), label=label):
        print("FAIL: hub hygiene failed", file=sys.stderr)
        return False
    print("OK: hub hygiene complete")
    return True


def run_release_prep_ready(root: Path, hub_url: str) -> bool:
    print(">>> release-prep-ready (smoke scenarios)...")
    proc = subprocess.run(
        [
            sys.executable,
            str(root / "scripts" / "release-prep-ready.py"),
            "--hub",
            hub_url.rstrip("/"),
            "--no-restart-hub",
        ],
        cwd=root,
        env={**os.environ, "NEURAL_JUNKIE_RATE_LIMIT": "0"},
        check=False,
    )
    return proc.returncode == 0


def ensure_ollama_stack(root: Path) -> bool:
    """Start Ollama when down and warm/pull regression models."""
    if not require_ollama(root):
        return False
    return ensure_models_ready(root)


def boot_regression_stack(
    root: Path,
    hub_url: str,
    *,
    label: str = "live regression",
    clean: bool | None = None,
    ollama: bool = True,
    models: bool = True,
    hub: bool = True,
    roster: bool = True,
    hygiene: bool = True,
    ready_smoke: bool = False,
    no_restart_hub: bool = False,
) -> bool:
    """Boot Ollama + models + regression hub before live scenario harnesses."""
    if should_skip_boot():
        print("SKIP: regression boot (SKIP_BOOT/NJ_BOOT_DONE set)")
        return True

    apply_release_prep_env(root)
    os.environ.setdefault("NEURAL_JUNKIE_HUB_URL", hub_url.rstrip("/"))
    os.environ.setdefault("NEURAL_JUNKIE_RATE_LIMIT", "0")

    do_clean = clean if clean is not None else not _env_truthy("SKIP_BOOT_CLEAN")
    no_restart = no_restart_hub or _env_truthy("NO_RESTART_HUB")

    print(f"\n=== Regression boot ({label}) ===")

    if do_clean and not clean_environment(root):
        return False
    if ollama and not require_ollama(root):
        return False
    if models and not ensure_models_ready(root):
        return False
    if hub and not start_regression_hub_stack(root, hub_url, no_restart=no_restart):
        return False
    if roster and not wait_for_roster(hub_url):
        return False
    if hub and not pin_regression_models(hub_url, root):
        return False
    if _env_truthy("NJ_REGRESSION_SLIM_ROSTER") or _env_truthy("NJ_REGRESSION_CLAUDE_CLOUD"):
        from lib.regression_collab import apply_collab_regression_tuning

        print(">>> Collab regression tuning (slim roster / cloud Claude)...")
        ok, detail = apply_collab_regression_tuning(hub_url)
        if not ok:
            print(f"FAIL: {detail}", file=sys.stderr)
            return False
        print(f"OK: {detail}")
    if hygiene and not hub_hygiene(root, hub_url, label=label):
        return False
    if ready_smoke and not run_release_prep_ready(root, hub_url):
        return False

    os.environ[BOOT_DONE_ENV] = "1"
    print(f"=== Regression boot complete ({label}) ===\n")
    return True


def restart_hub_for_live_run(root: Path, hub_url: str, *, label: str = "live regression") -> bool:
    """Force hub restart to clear poisoned in-process agent / impl-session state."""
    print(f"\n>>> Hub restart ({label})...")
    apply_release_prep_env(root)
    os.environ.pop(BOOT_DONE_ENV, None)
    base = hub_url.rstrip("/")
    env = release_prep_env(root)
    provision_hub_automation_key(root)
    if not restart_regression_hub(root, base, timeout_s=180.0, env=env):
        print("FAIL: hub restart failed", file=sys.stderr)
        return False
    if not wait_for_hub(base, timeout_s=30.0):
        print("FAIL: hub not healthy after restart", file=sys.stderr)
        return False
    if not wait_for_roster(base):
        return False
    if not pin_regression_models(base, root):
        return False
    if _env_truthy("NJ_REGRESSION_SLIM_ROSTER") or _env_truthy("NJ_REGRESSION_CLAUDE_CLOUD"):
        from lib.regression_collab import apply_collab_regression_tuning

        print(">>> Collab regression tuning (slim roster / cloud Claude)...")
        ok, detail = apply_collab_regression_tuning(base)
        if not ok:
            print(f"FAIL: {detail}", file=sys.stderr)
            return False
        print(f"OK: {detail}")
    if not hub_hygiene(root, base, label=label):
        return False
    os.environ[BOOT_DONE_ENV] = "1"
    print(f"OK: hub restarted for {label}\n")
    return True


def maybe_boot_regression(
    hub_url: str,
    *,
    root: Path = ROOT,
    label: str = "live regression",
    clean: bool | None = None,
    ready_smoke: bool = False,
    no_restart_hub: bool = False,
) -> bool:
    """Convenience wrapper for live test entry points."""
    return boot_regression_stack(
        root,
        hub_url,
        label=label,
        clean=clean,
        ready_smoke=ready_smoke,
        no_restart_hub=no_restart_hub,
    )
