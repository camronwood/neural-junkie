"""Smoke tests for deliverable judge (cloud-first, Ollama fallback)."""

from __future__ import annotations

import os

from lib.ollama_judge_auth import check_ollama_judge

HUB_JUDGE_PROMPT = (
    "You are an independent automated test harness judge.\n"
    "Reply with exactly two lines:\n"
    "Line 1: PASS\n"
    "Line 2: ok"
)


def _judge_provider() -> str:
    return os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "gemini").strip().lower()


def _judge_mode() -> str:
    return os.environ.get("NJ_DELIVERABLE_JUDGE_MODE", "hub").strip().lower()


def _ollama_fallback_enabled() -> bool:
    return os.environ.get("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA", "1").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def _ensure_gemini_judge_model(*, timeout_s: float) -> tuple[bool, str]:
    """Probe fast → pro → fast-light when unset; verify configured model when pinned."""
    if _judge_provider() != "gemini":
        return True, "not gemini provider"
    from lib.gemini_judge_auth import select_gemini_judge_model
    from lib.release_prep_env import ROOT, explicit_gemini_judge_model

    explicit = explicit_gemini_judge_model(ROOT)
    model, ok, detail = select_gemini_judge_model(
        timeout_s=min(timeout_s, 35.0),
        explicit_model=explicit,
        retry_quota=False,
    )
    if ok and model:
        os.environ["NJ_DELIVERABLE_JUDGE_GEMINI_MODEL"] = model
        return True, detail
    return False, detail


def _cloud_judge_smoke(hub_url: str, *, timeout_s: float) -> tuple[bool, str]:
    provider = _judge_provider()
    mode = _judge_mode()

    if provider == "gemini":
        ensured, edetail = _ensure_gemini_judge_model(timeout_s=timeout_s)
        if not ensured:
            return False, edetail

    if provider == "gemini" and mode == "cli":
        from lib.gemini_judge_auth import check_gemini_judge

        model = (os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
        return check_gemini_judge(timeout_s=timeout_s, model=model or None)

    if mode == "hub" and hub_url.strip():
        from lib import collab_hub as hub

        agent = (os.environ.get("NJ_DELIVERABLE_JUDGE_AGENT") or "").strip().lstrip("@")
        if not agent:
            agent = "Cursor" if provider == "cursor" else "Gemini"
        base = hub_url.rstrip("/")
        ok, missing = hub.verify_agents_online(base, [agent])
        if not ok:
            return False, f"judge agent offline: {', '.join(missing)}"
        try:
            from lib.deliverable_judge import hub_judge_deliverable
        except ImportError:
            from deliverable_judge import hub_judge_deliverable  # type: ignore[no-redef]
        ok, detail = hub_judge_deliverable(
            hub_base=base,
            agent_name=agent,
            prompt=HUB_JUDGE_PROMPT,
            timeout_s=timeout_s,
        )
        if ok:
            return True, f"hub {agent} judge OK ({detail})"
        return False, detail

    if provider == "gemini":
        from lib.gemini_judge_auth import check_gemini_judge

        model = (os.environ.get("NJ_DELIVERABLE_JUDGE_GEMINI_MODEL") or "").strip()
        return check_gemini_judge(timeout_s=timeout_s, model=model or None)

    return False, f"unsupported cloud judge provider={provider!r} mode={mode!r}"


def check_deliverable_judge_smoke(hub_url: str = "", *, timeout_s: float = 90.0) -> tuple[bool, str]:
    """Try cloud deliverable judge; fall back to local Ollama when enabled."""
    provider = _judge_provider()
    mode = _judge_mode()

    if provider == "ollama" or mode == "ollama":
        return check_ollama_judge(timeout_s=timeout_s)

    cloud_ok, cloud_detail = _cloud_judge_smoke(hub_url, timeout_s=timeout_s)
    if cloud_ok:
        return True, cloud_detail

    if _ollama_fallback_enabled():
        ok, detail = check_ollama_judge(timeout_s=timeout_s)
        if ok:
            return True, f"cloud unavailable ({cloud_detail}); ollama fallback OK ({detail})"

    return False, cloud_detail
