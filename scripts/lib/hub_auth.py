"""Hub authentication helpers for scripts, CI, and release-prep automation."""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from pathlib import Path

AUTOMATION_KEY_PATH = Path.home() / ".neural-junkie" / "automation.key"
_DEFAULT_BOOTSTRAP_FILE = Path.home() / ".neural-junkie" / "bootstrap.token"

_session_token: str | None = None


def bootstrap_token() -> str:
    tok = (os.environ.get("NEURAL_JUNKIE_BOOTSTRAP_TOKEN") or "").strip()
    if tok:
        return tok
    custom = (os.environ.get("NEURAL_JUNKIE_BOOTSTRAP_TOKEN_FILE") or "").strip()
    path = Path(custom) if custom else _DEFAULT_BOOTSTRAP_FILE
    if path.is_file():
        return path.read_text(encoding="utf-8").strip()
    return ""


def load_automation_api_key() -> str:
    key = (os.environ.get("NEURAL_JUNKIE_API_KEY") or "").strip()
    if key:
        return key
    if AUTOMATION_KEY_PATH.is_file():
        return AUTOMATION_KEY_PATH.read_text(encoding="utf-8").strip()
    return ""


def _hub_network_headers() -> dict[str, str]:
    headers: dict[str, str] = {}
    hub_token = (os.environ.get("NEURAL_JUNKIE_HUB_TOKEN") or "").strip()
    if hub_token:
        headers["X-NJ-Hub-Token"] = hub_token
    return headers


def hub_auth_headers() -> dict[str, str]:
    """Headers for hub HTTP calls (API key, hub token, or cached session)."""
    api_key = load_automation_api_key()
    if api_key:
        return {"Authorization": f"Bearer {api_key}", **_hub_network_headers()}
    headers = dict(_hub_network_headers())
    if _session_token:
        headers["X-NJ-Session"] = _session_token
    return headers


def ensure_hub_session(base: str, username: str = "automation") -> str:
    """POST /api/auth/session once per process; returns token (empty when API key is set)."""
    global _session_token
    if load_automation_api_key():
        return ""
    if _session_token:
        return _session_token
    url = f"{base.rstrip('/')}/api/auth/session"
    body = json.dumps({"username": username}).encode()
    headers = {"Content-Type": "application/json", **hub_auth_headers()}
    req = urllib.request.Request(url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        raise RuntimeError(f"hub session failed: {e.code} {e.read().decode()}") from e
    token = (data.get("token") or "").strip()
    if not token:
        raise RuntimeError("hub session response missing token")
    _session_token = token
    return token


def ensure_hub_auth_headers(base: str) -> dict[str, str]:
    """Session or API key headers for hub_request-style callers."""
    ensure_hub_session(base)
    return hub_auth_headers()


def ensure_automation_api_key(base: str, name: str = "release-prep") -> str:
    """Create and persist a member API key when missing (requires bootstrap token)."""
    existing = load_automation_api_key()
    if existing:
        return existing
    boot = bootstrap_token()
    if not boot:
        raise RuntimeError(
            "missing automation API key and bootstrap token "
            "(set NEURAL_JUNKIE_API_KEY or NEURAL_JUNKIE_BOOTSTRAP_TOKEN)"
        )
    net = _hub_network_headers()
    sess_url = f"{base.rstrip('/')}/api/auth/session"
    sess_body = json.dumps({"username": "automation-admin", "role": "admin"}).encode()
    sess_headers = {"Content-Type": "application/json", "X-NJ-Bootstrap": boot, **net}
    req = urllib.request.Request(sess_url, data=sess_body, headers=sess_headers, method="POST")
    with urllib.request.urlopen(req, timeout=30) as resp:
        sess = json.loads(resp.read().decode())
    admin_token = (sess.get("token") or "").strip()
    if not admin_token:
        raise RuntimeError("bootstrap admin session missing token")

    key_url = f"{base.rstrip('/')}/api/auth/api-keys"
    key_body = json.dumps({"name": name, "role": "member"}).encode()
    key_headers = {"Content-Type": "application/json", "X-NJ-Session": admin_token, **net}
    req = urllib.request.Request(key_url, data=key_body, headers=key_headers, method="POST")
    with urllib.request.urlopen(req, timeout=30) as resp:
        created = json.loads(resp.read().decode())
    raw = (created.get("api_key") or "").strip()
    if not raw:
        raise RuntimeError("api key create response missing api_key")

    AUTOMATION_KEY_PATH.parent.mkdir(parents=True, exist_ok=True)
    AUTOMATION_KEY_PATH.write_text(raw + "\n", encoding="utf-8")
    AUTOMATION_KEY_PATH.chmod(0o600)
    os.environ["NEURAL_JUNKIE_API_KEY"] = raw
    return raw
