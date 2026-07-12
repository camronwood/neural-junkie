#!/usr/bin/env python3
"""Fail-fast checks before live collab scenario sweeps."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / "scripts"))

from lib import collab_hub as hub  # noqa: E402
from lib.fixture_cleanup import preflight_regression_run  # noqa: E402
from lib.hub_auth import ensure_automation_api_key, hub_auth_headers, ensure_hub_session  # noqa: E402
from lib.release_prep_env import apply_release_prep_env  # noqa: E402

DEFAULT_ROSTER = [
    "BackendEngineer",
    "SoftwareArchitect",
    "PlatformEngineer",
    "FrontendEngineer",
    "SecurityReviewer",
]
GEMINI_SCENARIOS: tuple[str, ...] = ()  # no live collab scenarios require @Gemini (quota/flake)
EXPECTED_SCENARIO_COUNT = 25
HUB_LOG = Path("/tmp/nj-hub.log")
OLLAMA_TAGS_URL = "http://127.0.0.1:11434/api/tags"


def _fail(msg: str) -> None:
    print(f"FAIL: {msg}", file=sys.stderr)


def _warn(msg: str) -> None:
    print(f"WARN: {msg}", file=sys.stderr)


def _ok(msg: str) -> None:
    print(f"OK: {msg}")


def load_expected_ollama_models() -> list[str]:
    """Models required for live regression (≤14B; see regression_models.py)."""
    from lib.regression_models import regression_ollama_warm_tags

    return regression_ollama_warm_tags(ROOT)


def check_hub(base: str) -> bool:
    try:
        health = hub.check_health(base)
    except urllib.error.URLError as exc:
        _fail(f"hub not reachable at {base} ({exc}); start: make server-regression")
        return False
    if not health:
        _fail(f"hub not healthy at {base} (start: make server-regression)")
        return False
    _ok(f"hub healthy at {base}")
    return True


def check_hub_auth(base: str) -> bool:
    """Verify scripts can authenticate when NEURAL_JUNKIE_AUTH_REQUIRED=1."""
    if os.environ.get("NEURAL_JUNKIE_AUTH_REQUIRED", "").strip() != "1":
        _ok("hub auth strict mode off (NEURAL_JUNKIE_AUTH_REQUIRED not set)")
        return True
    try:
        if hub_auth_headers().get("Authorization", "").startswith("Bearer nj_"):
            _ok("automation API key configured for strict hub auth")
            return True
        ensure_hub_session(base)
        code, _ = hub.hub_request(base, "GET", "/api/messages?channel=general&limit=1")
        if code == 200:
            _ok("hub session auth OK for strict mode")
            return True
        if code == 401:
            try:
                ensure_automation_api_key(base)
                _ok("provisioned automation API key via bootstrap token")
                return True
            except RuntimeError as exc:
                _fail(f"hub auth failed (401) and could not provision API key: {exc}")
                return False
        _fail(f"hub auth check unexpected status {code}")
        return False
    except (urllib.error.URLError, RuntimeError, TimeoutError) as exc:
        _fail(f"hub auth check failed: {exc}")
        return False


def check_regression_log_hint() -> None:
    if not HUB_LOG.is_file():
        _warn(
            f"{HUB_LOG} missing — cannot verify regression env; use make server-regression"
        )
        return
    try:
        tail = HUB_LOG.read_text(encoding="utf-8", errors="replace")[-8000:]
    except OSError as exc:
        _warn(f"could not read {HUB_LOG}: {exc}")
        return
    has_rate = "NEURAL_JUNKIE_RATE_LIMIT=0" in tail or "RATE_LIMIT=0" in tail
    has_debug = "NEURAL_JUNKIE_DEBUG=1" in tail or "DEBUG=1" in tail
    if has_rate and has_debug:
        _ok("hub log suggests regression env (RATE_LIMIT=0, DEBUG=1)")
    else:
        _warn(
            "hub log may not be from make server-regression "
            "(expected NEURAL_JUNKIE_RATE_LIMIT=0 and NEURAL_JUNKIE_DEBUG=1 in /tmp/nj-hub.log)"
        )


def check_ollama(expected_models: list[str]) -> bool:
    try:
        with urllib.request.urlopen(OLLAMA_TAGS_URL, timeout=5) as resp:
            data = json.loads(resp.read().decode())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as exc:
        _fail(f"Ollama not reachable at 11434 ({exc}); run: ollama serve")
        return False

    names = {
        (m.get("name") or "").strip()
        for m in (data.get("models") or [])
        if isinstance(m, dict)
    }
    _ok(f"Ollama reachable ({len(names)} model(s) installed)")
    missing = [m for m in expected_models if m and m not in names]
    if missing:
        _warn(f"env.local models not pulled: {', '.join(missing)} — run: ollama pull <model>")
    return True


def _deliverable_judge_agent() -> str | None:
    provider = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "claude").strip().lower()
    if provider == "ollama":
        return None
    agent = (os.environ.get("NJ_DELIVERABLE_JUDGE_AGENT") or "").strip().lstrip("@")
    if not agent:
        if provider == "cursor":
            agent = "Cursor"
        elif provider == "claude":
            agent = "Claude"
        else:
            agent = "Gemini"
    return agent


def check_claude_for_testing(*, require: bool, timeout_s: float = 45.0) -> bool:
    """Probe Claude Code CLI before scenarios that use hub @Claude judge."""
    if os.environ.get("NJ_SKIP_CLAUDE_PROBE", "").strip().lower() in ("1", "true", "yes"):
        _ok("Claude probe skipped (NJ_SKIP_CLAUDE_PROBE)")
        return True

    provider = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "claude").strip().lower()
    if not require and provider == "ollama":
        _ok("Claude probe skipped (NJ_DELIVERABLE_JUDGE_PROVIDER=ollama)")
        return True
    if provider != "claude":
        _ok(f"Claude probe skipped (NJ_DELIVERABLE_JUDGE_PROVIDER={provider})")
        return True

    from lib.claude_judge_auth import ensure_claude_for_testing  # noqa: E402

    sel = ensure_claude_for_testing(timeout_s=timeout_s, smoke=False)
    if sel.ok:
        _ok(f"Claude ready: {sel.detail}")
        return True
    if require:
        _fail(f"Claude required for deliverable judge: {sel.detail}")
        return False
    _warn(f"Claude unavailable (non-fatal): {sel.detail}")
    return True


def check_gemini_for_testing(*, require: bool, timeout_s: float = 45.0) -> bool:
    """Probe all API keys × top Gemini models before collab scenarios that need @Gemini."""
    if os.environ.get("NJ_SKIP_GEMINI_PROBE", "").strip().lower() in ("1", "true", "yes"):
        _ok("Gemini probe skipped (NJ_SKIP_GEMINI_PROBE)")
        return True

    provider = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "claude").strip().lower()
    if not require and provider == "ollama":
        _ok("Gemini probe skipped (NJ_DELIVERABLE_JUDGE_PROVIDER=ollama)")
        return True
    if provider != "gemini":
        _ok(f"Gemini probe skipped (NJ_DELIVERABLE_JUDGE_PROVIDER={provider})")
        return True

    from lib.gemini_judge_auth import ensure_gemini_for_testing  # noqa: E402
    from lib.release_prep_env import explicit_gemini_judge_model  # noqa: E402

    sel = ensure_gemini_for_testing(
        root=ROOT,
        timeout_s=timeout_s,
        explicit_model=explicit_gemini_judge_model(ROOT),
        retry_quota=True,
        collab=require,
    )
    if sel.ok:
        _ok(f"Gemini ready: {sel.detail}")
        return True
    if require:
        _fail(f"Gemini required for collab testing: {sel.detail}")
        return False
    _warn(f"Gemini unavailable (non-fatal): {sel.detail}")
    return True


def check_deliverable_judge(base: str, *, skip_smoke: bool = False) -> bool:
    if os.environ.get("NJ_DELIVERABLE_JUDGE_SKIP", "").strip().lower() in ("1", "true", "yes"):
        _ok("deliverable judge skipped (NJ_DELIVERABLE_JUDGE_SKIP)")
        return True

    if skip_smoke:
        _ok("deliverable judge smoke skipped (already verified)")
        return True

    provider = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "claude").strip().lower()
    if provider != "ollama":
        agent = _deliverable_judge_agent()
        if agent:
            ok, missing = hub.verify_agents_online(base, [agent])
            if not ok:
                _fail(f"deliverable judge agent offline: {', '.join(missing)} (start hub with {agent} CLI on PATH)")
                return False
            _ok(f"deliverable judge agent online: {agent}")

    from lib.deliverable_judge_smoke import check_deliverable_judge_smoke  # noqa: E402

    ok, detail = check_deliverable_judge_smoke(base, timeout_s=90.0)
    if not ok:
        _fail(f"deliverable judge: {detail}")
        return False
    if "ollama fallback" in detail.lower():
        _warn(detail)
    else:
        _ok(detail)
    return True


def check_regression_agent_models(base: str) -> bool:
    from lib.regression_models import (
        MAX_REGRESSION_PARAMS_B,
        agents_over_regression_cap,
        enforce_regression_agent_models,
        resolve_regression_agent_model,
    )

    over = agents_over_regression_cap(base)
    if over:
        _warn(f"agents above {int(MAX_REGRESSION_PARAMS_B)}B — switching: {', '.join(over)}")
        ok, detail = enforce_regression_agent_models(base, ROOT)
        if not ok:
            _fail(detail)
            return False
        over = agents_over_regression_cap(base)
        if over:
            _fail(f"agents still above {int(MAX_REGRESSION_PARAMS_B)}B: {', '.join(over)}")
            return False
    model = resolve_regression_agent_model(ROOT)
    _ok(f"regression agent model ≤{int(MAX_REGRESSION_PARAMS_B)}B ({model})")
    return True


def check_agents(base: str, *, require_gemini: bool) -> bool:
    from lib.regression_collab import resolve_preflight_roster

    roster = resolve_preflight_roster()
    ok, missing = hub.verify_agents_online(base, roster)
    if not ok:
        _fail(f"agents offline: {', '.join(missing)}")
        return False
    label = "core roster" if roster != DEFAULT_ROSTER else "default roster"
    _ok(f"{label} online: {', '.join(roster)}")

    if require_gemini:
        gok, gmissing = hub.verify_agents_online(base, ["Gemini"])
        if not gok:
            _fail("Gemini agent offline (required for resource-api-schema-planning)")
            return False
        _ok("Gemini online")
    return True


def check_scenario_list() -> bool:
    proc = subprocess.run(
        [sys.executable, str(ROOT / "scripts" / "collab-scenarios.py"), "--list"],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        _fail(f"collab-scenarios.py --list failed: {proc.stderr.strip()}")
        return False
    names = [ln.strip() for ln in proc.stdout.splitlines() if ln.strip()]
    if len(names) != EXPECTED_SCENARIO_COUNT:
        _fail(f"expected {EXPECTED_SCENARIO_COUNT} scenarios, got {len(names)}")
        return False
    _ok(f"{len(names)} collab scenarios listed")
    return True


def main() -> int:
    p = argparse.ArgumentParser(description="Preflight for live collab sweeps")
    p.add_argument(
        "--hub",
        default=os.environ.get("NEURAL_JUNKIE_HUB_URL", hub.DEFAULT_HUB),
        help="Hub base URL",
    )
    p.add_argument(
        "--require-gemini",
        action="store_true",
        help="Require Gemini API probe (optional; no collab scenarios use @Gemini by default)",
    )
    p.add_argument(
        "--skip-judge-smoke",
        action="store_true",
        help="Skip deliverable judge smoke (hub judge already verified in release-prep)",
    )
    args = p.parse_args()
    apply_release_prep_env(ROOT)
    base = args.hub.rstrip("/")

    print(f"Collab preflight → {base}")
    if not check_hub(base):
        print("Preflight failed.", file=sys.stderr)
        return 1
    if not check_hub_auth(base):
        print("Preflight failed.", file=sys.stderr)
        return 1

    preflight_regression_run(ROOT, base, label="collab-preflight cleanup")

    checks = [
        check_ollama(load_expected_ollama_models()),
        check_gemini_for_testing(require=args.require_gemini),
        check_claude_for_testing(require=False),
        check_regression_agent_models(base),
        check_agents(base, require_gemini=args.require_gemini),
        check_deliverable_judge(base, skip_smoke=args.skip_judge_smoke),
        check_scenario_list(),
    ]
    check_regression_log_hint()

    if all(checks):
        print("Preflight passed.")
        return 0
    print("Preflight failed.", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
