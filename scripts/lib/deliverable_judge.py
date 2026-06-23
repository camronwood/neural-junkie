"""Cloud/local judges for scenario deliverable quality (independent of producing agent)."""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request
from typing import Any

try:
    from gemini_rate_limit import throttle_gemini_api_call
except ImportError:
    from lib.gemini_rate_limit import throttle_gemini_api_call  # type: ignore[no-redef]

DEFAULT_JUDGE_PROVIDER = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "gemini").strip().lower()
DEFAULT_JUDGE_AGENT = os.environ.get("NJ_DELIVERABLE_JUDGE_AGENT", "").strip()
DEFAULT_OLLAMA_URL = os.environ.get("OLLAMA_HOST", "http://127.0.0.1:11434").rstrip("/")
DEFAULT_OLLAMA_MODEL = os.environ.get("NJ_DELIVERABLE_JUDGE_MODEL", "qwen2.5-coder:14b")
JUDGE_MAX_CHARS = int(os.environ.get("NJ_DELIVERABLE_JUDGE_MAX_CHARS", "12000"))
JUDGE_TIMEOUT_S = float(os.environ.get("NJ_DELIVERABLE_JUDGE_TIMEOUT", "180"))
JUDGE_DM_USER = os.environ.get("NJ_DELIVERABLE_JUDGE_DM_USER", "DeliverableJudge").strip()

_QUOTA_RE = re.compile(
    r"quota|429|resource_exhausted|rate.?limit|exceeded your current quota|generativelanguage\.googleapis\.com",
    re.I,
)
_RETRY_DELAY_RE = re.compile(r"retry in ([\d.]+)s", re.I)

_PROVIDER_DEFAULT_AGENT = {
    "gemini": "Gemini",
    "cursor": "Cursor",
    "ollama": "",
}

# After a definitive cloud-judge failure, skip hub/CLI for the rest of the process.
_cloud_judge_tripped: dict[str, str] = {}


def reset_cloud_judge_circuit() -> None:
    """Clear tripped cloud providers (for tests)."""
    _cloud_judge_tripped.clear()


def cloud_judge_tripped(provider: str) -> bool:
    return provider.strip().lower() in _cloud_judge_tripped


def trip_cloud_judge(provider: str, detail: str) -> None:
    """Remember that cloud judging is unavailable for this provider."""
    key = provider.strip().lower()
    if not key or key in _cloud_judge_tripped:
        return
    reason = (detail or "cloud judge unavailable").strip()[:200]
    _cloud_judge_tripped[key] = reason
    print(
        f"[deliverable-judge] cloud judge disabled for {key} (using ollama): {reason[:120]}",
        file=sys.stderr,
    )


def _is_cloud_judge_agent_error(text: str) -> bool:
    if not text:
        return False
    lower = text.lower()
    if is_gemini_quota_error(text):
        return True
    if "sorry, i encountered an error" in lower:
        return True
    if "cli agent error" in lower:
        return True
    return False


def judge_skip_enabled() -> bool:
    return os.environ.get("NJ_DELIVERABLE_JUDGE_SKIP", "").strip().lower() in ("1", "true", "yes")


def ollama_fallback_enabled() -> bool:
    return os.environ.get("NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA", "1").strip().lower() in (
        "1",
        "true",
        "yes",
    )


def is_gemini_quota_error(text: str) -> bool:
    return bool(text and _QUOTA_RE.search(text))


def parse_retry_delay_s(text: str) -> float | None:
    if not text:
        return None
    m = _RETRY_DELAY_RE.search(text)
    if not m:
        return None
    try:
        return float(m.group(1))
    except ValueError:
        return None


def throttle_gemini_judge() -> None:
    """Pace Gemini judge calls to reduce free-tier RPM bursts."""
    throttle_gemini_api_call()


def _default_judge_provider() -> str:
    p = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "gemini").strip().lower()
    return p if p in ("gemini", "cursor", "ollama") else "gemini"


def _should_fallback_to_ollama(provider: str, detail: str) -> bool:
    if provider == "ollama" or not ollama_fallback_enabled():
        return False
    if provider not in ("gemini", "cursor"):
        return False
    if is_gemini_quota_error(detail):
        return True
    lower = detail.lower()
    if "timeout" in lower:
        return True
    if "offline" in lower or "not on path" in lower:
        return True
    if "unparseable judge response" in lower or "sorry, i encountered an error" in lower:
        return True
    if provider == "gemini" and "gemini exit" in lower:
        return True
    if "judge send failed" in lower or "judge agent offline" in lower:
        return True
    if "judge error" in lower:
        return True
    return False


def _trip_cloud_judge_on_failure(provider: str, detail: str) -> None:
    if _should_fallback_to_ollama(provider, detail):
        trip_cloud_judge(provider, detail)


def _judge_via_ollama(prompt: str, *, note: str) -> tuple[bool, str]:
    ok, detail = ollama_judge_deliverable(prompt=prompt)
    prefix = note.strip() or "fallback"
    return ok, f"{prefix} ollama/{DEFAULT_OLLAMA_MODEL}: {detail}"


def resolve_judge_agent(provider: str, spec_agent: str = "") -> str:
    if spec_agent.strip():
        return spec_agent.strip().lstrip("@")
    if DEFAULT_JUDGE_AGENT:
        return DEFAULT_JUDGE_AGENT.lstrip("@")
    return _PROVIDER_DEFAULT_AGENT.get(provider, "Gemini")


def build_judge_prompt(
    *,
    question: str,
    rel_path: str,
    file_body: str,
    criteria: str = "",
    max_chars: int = JUDGE_MAX_CHARS,
) -> str:
    body = file_body if len(file_body) <= max_chars else file_body[:max_chars] + "\n…[truncated]"
    extra = f"\nAdditional criteria:\n{criteria}\n" if criteria.strip() else ""
    return (
        "You are an independent automated test harness judge. You did NOT produce this file.\n"
        "Evaluate whether the deliverable substantively answers the user's request.\n\n"
        "The user asked:\n---\n"
        f"{question.strip()}\n"
        "---\n\n"
        f'The deliverable file is "{rel_path}" with this content:\n---\n'
        f"{body}\n"
        f"---{extra}\n\n"
        "Reject stubs, placeholders, unrelated boilerplate, or wrong-stack artifacts.\n"
        "Judge the deliverable file on substance and correctness, not whether it repeats the user's error log.\n\n"
        "Reply with exactly two lines:\n"
        "Line 1: PASS or FAIL\n"
        "Line 2: one short reason"
    )


def parse_judge_response(text: str) -> tuple[bool, str]:
    response = (text or "").strip()
    if not response:
        return False, "empty judge response"
    lines = [ln.strip() for ln in response.splitlines() if ln.strip()]
    first = (lines[0] if lines else response).upper()
    reason = lines[1] if len(lines) > 1 else response
    if first.startswith("PASS") or re.search(r"\bPASS\b", first):
        return True, reason or "pass"
    if first.startswith("FAIL") or re.search(r"\bFAIL\b", first):
        return False, reason or "fail"
    return False, f"unparseable judge response: {response[:200]}"


def _judge_spec_provider(spec: dict[str, Any] | bool | None) -> str:
    if isinstance(spec, dict):
        p = (spec.get("provider") or "").strip().lower()
        if p:
            return p
    mode = os.environ.get("NJ_DELIVERABLE_JUDGE_MODE", "").strip().lower()
    if mode in ("cli", "hub", "ollama"):
        if mode == "cli":
            return DEFAULT_JUDGE_PROVIDER if DEFAULT_JUDGE_PROVIDER in ("gemini", "cursor") else "gemini"
        if mode == "ollama":
            return "ollama"
    return _default_judge_provider()


def _judge_spec_agent(spec: dict[str, Any] | bool | None, provider: str) -> str:
    if isinstance(spec, dict):
        agent = (spec.get("agent") or "").strip()
        if agent:
            return agent.lstrip("@")
    return resolve_judge_agent(provider)


def _judge_spec_criteria(spec: dict[str, Any] | bool | None) -> str:
    if isinstance(spec, dict):
        return (spec.get("criteria") or "").strip()
    return ""


def _gemini_cli_env() -> dict[str, str] | None:
    try:
        from lib.gemini_judge_auth import gemini_cli_env
    except ImportError:
        from gemini_judge_auth import gemini_cli_env  # type: ignore[no-redef]
    return gemini_cli_env()


def ollama_judge_deliverable(
    *,
    prompt: str,
    model: str = DEFAULT_OLLAMA_MODEL,
    ollama_base: str = DEFAULT_OLLAMA_URL,
    timeout_s: float = JUDGE_TIMEOUT_S,
) -> tuple[bool, str]:
    payload = {
        "model": model,
        "prompt": prompt,
        "stream": False,
        "options": {"temperature": 0},
    }
    url = f"{ollama_base.rstrip('/')}/api/generate"
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout_s) as resp:
            data = json.loads(resp.read().decode("utf-8"))
    except urllib.error.URLError as exc:
        return False, f"ollama unreachable at {ollama_base}: {exc}"
    except TimeoutError:
        return False, f"ollama judge timeout ({timeout_s}s)"
    return parse_judge_response((data.get("response") or "").strip())


def cli_judge_deliverable(
    *,
    provider: str,
    prompt: str,
    work_dir: str = "",
    timeout_s: float = JUDGE_TIMEOUT_S,
) -> tuple[bool, str]:
    if provider == "gemini":
        binary = shutil.which("gemini")
        if not binary:
            return False, "gemini CLI not on PATH"
        cmd = [binary, "--output-format", "text", "-p", prompt]
        run_env = _gemini_cli_env()
    elif provider == "cursor":
        binary = shutil.which("agent")
        if not binary:
            return False, "cursor agent CLI not on PATH"
        cmd = [binary, "-p", "--output-format", "text", prompt]
        run_env = None
    else:
        return False, f"unsupported CLI judge provider {provider!r}"

    if provider == "gemini":
        throttle_gemini_judge()

    try:
        proc = subprocess.run(
            cmd,
            cwd=work_dir or None,
            capture_output=True,
            text=True,
            timeout=timeout_s,
            check=False,
            env=run_env,
        )
    except subprocess.TimeoutExpired:
        return False, f"{provider} CLI judge timeout ({timeout_s}s)"
    except OSError as exc:
        return False, f"{provider} CLI judge failed: {exc}"

    if proc.returncode != 0:
        err = (proc.stderr or proc.stdout or "").strip()
        detail = f"{provider} CLI exit {proc.returncode}: {err[:300]}"
        if provider == "gemini" and is_gemini_quota_error(err):
            delay = parse_retry_delay_s(err)
            if delay is not None and delay < 120:
                time.sleep(delay + 0.5)
                proc = subprocess.run(
                    cmd,
                    cwd=work_dir or None,
                    capture_output=True,
                    text=True,
                    timeout=timeout_s,
                    check=False,
                    env=run_env,
                )
                if proc.returncode == 0:
                    return parse_judge_response(proc.stdout or "")
        return False, detail
    return parse_judge_response(proc.stdout or "")


def hub_judge_deliverable(
    *,
    hub_base: str,
    agent_name: str,
    prompt: str,
    timeout_s: float = JUDGE_TIMEOUT_S,
) -> tuple[bool, str]:
    try:
        from lib import collab_hub as hub
    except ImportError:  # unittest cwd scripts/lib
        import collab_hub as hub  # type: ignore[no-redef]

    if agent_name.strip().lower() == "gemini":
        throttle_gemini_judge()

    base = hub_base.rstrip("/")
    ok, missing = hub.verify_agents_online(base, [agent_name])
    if not ok:
        return False, f"judge agent offline: {', '.join(missing)}"

    channel = hub.ensure_dm_channel(base, JUDGE_DM_USER, agent_name)
    if not channel:
        return False, f"could not open DM with judge agent {agent_name!r}"

    hub.clear_channel_history(base, channel)
    time.sleep(2.0)

    code, _ = hub.send_message(
        base,
        channel,
        prompt,
        from_name=JUDGE_DM_USER,
        metadata={
            "editor_mode": "ask",
            "conversation_mode": "chat",
            "deliverable_judge": True,
        },
    )
    if code != 200:
        return False, f"judge send failed ({code})"

    deadline = time.time() + timeout_s
    while time.time() < deadline:
        msgs = hub.list_messages(base, channel, 30)
        for msg in reversed(msgs):
            sender = msg.get("from") if isinstance(msg.get("from"), dict) else {}
            if (sender.get("name") or "").strip() != agent_name:
                continue
            if (sender.get("type") or "").strip().lower() == "human":
                continue
            text = (msg.get("content") or "").strip()
            if text:
                if _is_cloud_judge_agent_error(text):
                    return False, f"{agent_name} judge error: {text[:300]}"
                ok, detail = parse_judge_response(text)
                if not ok and _is_cloud_judge_agent_error(detail):
                    return False, f"{agent_name} judge error: {detail[:300]}"
                if ok:
                    return True, detail
                return False, detail
        time.sleep(2.0)
    return False, f"timeout waiting for {agent_name} judge ({timeout_s}s)"


def judge_deliverable(
    *,
    question: str,
    rel_path: str,
    file_body: str,
    criteria: str = "",
    llm_judge_spec: dict[str, Any] | bool | None = True,
    hub_base: str = "",
    work_dir: str = "",
) -> tuple[bool, str]:
    """Route deliverable judging: cloud-first (hub Gemini), Ollama fallback when enabled."""
    if judge_skip_enabled():
        return True, "skipped (NJ_DELIVERABLE_JUDGE_SKIP)"

    provider = _judge_spec_provider(llm_judge_spec)
    agent = _judge_spec_agent(llm_judge_spec, provider)
    criteria = criteria or _judge_spec_criteria(llm_judge_spec)
    prompt = build_judge_prompt(
        question=question,
        rel_path=rel_path,
        file_body=file_body,
        criteria=criteria,
    )

    mode = os.environ.get("NJ_DELIVERABLE_JUDGE_MODE", "hub").strip().lower()
    if mode not in ("hub", "cli", "ollama"):
        mode = "hub"

    if provider == "ollama" or mode == "ollama":
        ok, detail = ollama_judge_deliverable(prompt=prompt)
        return ok, f"ollama/{DEFAULT_OLLAMA_MODEL}: {detail}"

    if (
        provider in ("gemini", "cursor")
        and cloud_judge_tripped(provider)
        and ollama_fallback_enabled()
    ):
        ok, detail = ollama_judge_deliverable(prompt=prompt)
        return ok, f"cloud circuit open → ollama/{DEFAULT_OLLAMA_MODEL}: {detail}"

    if mode == "cli" or not hub_base.strip():
        ok, detail = cli_judge_deliverable(provider=provider, prompt=prompt, work_dir=work_dir)
        if ok:
            prefix = f"{provider}-cli"
            return ok, f"{prefix}: {detail}"
        if _should_fallback_to_ollama(provider, detail):
            _trip_cloud_judge_on_failure(provider, detail)
            return _judge_via_ollama(prompt, note="cloud judge error")
        if hub_base.strip():
            # fall through to hub when CLI fails and hub is available
            pass
        else:
            return ok, f"{provider}-cli: {detail}"

    if hub_base.strip():
        ok, detail = hub_judge_deliverable(
            hub_base=hub_base,
            agent_name=agent,
            prompt=prompt,
        )
        full_detail = f"{agent}@{hub_base}: {detail}"
        if ok:
            return ok, full_detail
        if _should_fallback_to_ollama(provider, detail):
            _trip_cloud_judge_on_failure(provider, detail)
            return _judge_via_ollama(prompt, note="cloud judge error")
        return False, full_detail

    ok, detail = cli_judge_deliverable(provider=provider, prompt=prompt, work_dir=work_dir)
    if not ok and _should_fallback_to_ollama(provider, detail):
        _trip_cloud_judge_on_failure(provider, detail)
        return _judge_via_ollama(prompt, note="cloud judge error")
    return ok, f"{provider}-cli: {detail}"
