"""Helpers for multi-model live hub benchmarks."""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterator

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
ROOT = SCRIPTS_DIR.parent
CONFIG_DIR = SCRIPTS_DIR / "config"
MODELS_CONFIG = CONFIG_DIR / "model-benchmark-models.json"
SUITES_CONFIG = CONFIG_DIR / "model-benchmark-suites.json"

sys.path.insert(0, str(SCRIPTS_DIR))
from lib import collab_hub as hub  # noqa: E402
from lib.collab_core_scenarios import collab_core_scenarios  # noqa: E402

SCRIPT_BY_KIND = {
    "implement": "implement-scenarios.py",
    "chat": "chat-scenarios.py",
    "collab": "collab-scenarios.py",
    "arena": "arena-benchmark.py",
    "cad": "cad-benchmark.py",
    "external": "external-humaneval.py",
}

_SCORE_RE = re.compile(r"SCORE:\s*([0-9]*\.?[0-9]+)", re.I)


@dataclass
class ScenarioMetrics:
    prompt_tokens: int | None = None
    completion_tokens: int | None = None
    ttft_ms: float | None = None
    tok_per_s: float | None = None
    tool_calls: int | None = None
    repair_attempts: int | None = None
    wall_duration_s: float | None = None
    cost_usd: float | None = None
    passed_at_1: bool | None = None
    eventual_pass: bool | None = None
    attempts: int | None = None
    retry_count: int | None = None
    retry_reasons: list[str] = field(default_factory=list)
    nudge_count: int | None = None
    nudge_reasons: list[str] = field(default_factory=list)
    actual_provider: str | None = None
    actual_model: str | None = None
    validation_failures: list[str] = field(default_factory=list)
    escalation_count: int | None = None
    escalation_reasons: list[str] = field(default_factory=list)


@dataclass
class ScenarioResult:
    name: str
    kind: str
    passed: bool
    duration_s: float
    detail: str = ""
    judge_passed: bool | None = None
    judge_reason: str = ""
    uses_llm_judge: bool = False
    metrics: ScenarioMetrics | None = None
    structural_passed: bool | None = None  # defaults to passed when None
    capability_passed: bool | None = None
    quality_score: float | None = None
    quality_passed: bool | None = None
    agent_efficiency: float | None = None

    def structural_ok(self) -> bool:
        if self.structural_passed is not None:
            return bool(self.structural_passed)
        return bool(self.passed)

    def composite_ok(self) -> bool:
        if not self.structural_ok():
            return False
        if self.quality_passed is not None and not self.quality_passed:
            return False
        if self.capability_passed is not None and not self.capability_passed:
            return False
        return True


@dataclass
class ModelBenchmarkResult:
    model_id: str
    model_tag: str
    title: str
    size_hint_gb: float | None
    skipped: bool = False
    skip_reason: str = ""
    scenarios: list[ScenarioResult] = field(default_factory=list)
    switch_duration_s: float = 0.0
    total_duration_s: float = 0.0

    @property
    def passed(self) -> int:
        return sum(1 for s in self.scenarios if s.passed)

    @property
    def total(self) -> int:
        return len(self.scenarios)

    @property
    def pass_rate(self) -> float:
        if not self.scenarios:
            return 0.0
        return self.passed / self.total

    @property
    def pass_at_1_rate(self) -> float:
        if not self.scenarios:
            return 0.0
        passed = sum(
            1
            for scenario in self.scenarios
            if (
                scenario.metrics.passed_at_1
                if scenario.metrics and scenario.metrics.passed_at_1 is not None
                else scenario.passed
            )
        )
        return passed / self.total

    @property
    def structural_pass_rate(self) -> float:
        if not self.scenarios:
            return 0.0
        return sum(1 for s in self.scenarios if s.structural_ok()) / self.total

    @property
    def quality_pass_rate(self) -> float:
        judged = [s for s in self.scenarios if s.quality_passed is not None]
        if not judged:
            return 0.0
        return sum(1 for s in judged if s.quality_passed) / len(judged)

    @property
    def composite_pass_rate(self) -> float:
        if not self.scenarios:
            return 0.0
        return sum(1 for s in self.scenarios if s.composite_ok()) / self.total

    def to_dict(self) -> dict[str, Any]:
        d = asdict(self)
        d["passed_count"] = self.passed
        d["total_count"] = self.total
        d["pass_rate"] = round(self.pass_rate, 4)
        d["pass_at_1_rate"] = round(self.pass_at_1_rate, 4)
        d["eventual_pass_rate"] = round(self.pass_rate, 4)
        d["structural_pass_rate"] = round(self.structural_pass_rate, 4)
        d["quality_pass_rate"] = round(self.quality_pass_rate, 4)
        d["composite_pass_rate"] = round(self.composite_pass_rate, 4)
        return d


@dataclass
class SuiteTracks:
    implement: list[str] = field(default_factory=list)
    chat: list[str] = field(default_factory=list)
    collab: list[str] = field(default_factory=list)
    arena: list[str] = field(default_factory=list)
    cad: list[str] = field(default_factory=list)
    external: list[str] = field(default_factory=list)

    def has_any(self) -> bool:
        return bool(
            self.implement or self.chat or self.collab or self.arena or self.cad or self.external
        )

    def __iter__(self) -> Iterator[list[str]]:
        """Unpack as (implement, chat) for older callers."""
        yield self.implement
        yield self.chat


def load_json_config(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def load_models(config_path: Path | None = None) -> list[dict[str, Any]]:
    data = load_json_config(config_path or MODELS_CONFIG)
    models = data.get("models") or []
    if not isinstance(models, list):
        raise ValueError("models config must contain a models array")
    return [m for m in models if isinstance(m, dict) and (m.get("tag") or "").strip()]


def model_requires_hf_import(model: dict[str, Any]) -> bool:
    """True when the model must be imported via HF GGUF (custom quants not ollama-pullable)."""
    return bool(model.get("requires_hf_import"))


def model_pull_tag(model: dict[str, Any]) -> str:
    """Ollama pull target; empty when requires_hf_import; else pull_tag or brand tag."""
    if model_requires_hf_import(model):
        return ""
    pull = str(model.get("pull_tag") or "").strip()
    if pull:
        return pull
    return str(model.get("tag") or "").strip()


def model_runtime_tag(model: dict[str, Any], installed: set[str]) -> str:
    """Tag to pass to Ollama for inference; prefers catalog tag when installed."""
    tag = str(model.get("tag") or "").strip()
    pull = model_pull_tag(model)
    if model_is_installed(installed, tag):
        return tag
    if pull and model_is_installed(installed, pull):
        return pull
    return tag or pull


def model_params_b(model: dict[str, Any]) -> float | None:
    raw = model.get("params_b")
    if raw is not None:
        try:
            return float(raw)
        except (TypeError, ValueError):
            return None
    tag = str(model.get("tag") or "")
    match = re.search(r":(\d+(?:\.\d+)?)b\b", tag, re.I)
    if match:
        try:
            return float(match.group(1))
        except ValueError:
            return None
    return None


def model_size_gb(model: dict[str, Any]) -> float | None:
    """On-disk / memory footprint hint used for the benchmark roster speed cap."""
    raw = model.get("size_hint_gb")
    if raw is not None:
        try:
            return float(raw)
        except (TypeError, ValueError):
            return None
    return None


def filter_models_by_max_size_gb(
    models: list[dict[str, Any]],
    max_size_gb: float,
    *,
    allow_unknown: bool = False,
) -> list[dict[str, Any]]:
    """Drop models whose size_hint_gb is above max_size_gb (unknown sizes optional)."""
    kept: list[dict[str, Any]] = []
    for model in models:
        size = model_size_gb(model)
        if size is None:
            if allow_unknown:
                kept.append(model)
            continue
        if size <= max_size_gb:
            kept.append(model)
    return kept


def filter_models_by_max_params(
    models: list[dict[str, Any]],
    max_params_b: float,
    *,
    allow_unknown: bool = False,
) -> list[dict[str, Any]]:
    """Deprecated name — filters by size_hint_gb, not parameter count."""
    return filter_models_by_max_size_gb(models, max_params_b, allow_unknown=allow_unknown)


def resolve_suite_max_size_gb(suite: dict[str, Any], config_path: Path | None = None) -> float:
    """Roster footprint cap in GB (default: ~Qwen 2.5 Coder 14B Q4 ≈ 9 GB)."""
    for key in ("max_size_gb", "max_params_b"):  # max_params_b = legacy suite key
        raw = suite.get(key)
        if raw is not None:
            try:
                return float(raw)
            except (TypeError, ValueError):
                pass
    data = load_json_config(config_path or MODELS_CONFIG)
    for key in ("max_size_gb_default", "max_params_b_default"):
        default = data.get(key)
        if default is not None:
            try:
                return float(default)
            except (TypeError, ValueError):
                pass
    return 9.0


def resolve_suite_max_params_b(suite: dict[str, Any], config_path: Path | None = None) -> float:
    """Deprecated alias for resolve_suite_max_size_gb."""
    return resolve_suite_max_size_gb(suite, config_path)


def resolve_suite_model_tags(suite: dict[str, Any]) -> list[str]:
    raw = suite.get("models") or []
    if not isinstance(raw, list):
        return []
    return [str(x).strip() for x in raw if str(x).strip()]


def parse_models_arg(
    raw: str,
    config_path: Path | None,
    *,
    max_size_gb: float | None = None,
    allow_large: bool = False,
    max_params_b: float | None = None,  # deprecated alias
) -> list[dict[str, Any]]:
    if max_size_gb is None:
        max_size_gb = max_params_b
    catalog = {m["tag"]: m for m in load_models(config_path)}
    tags = [t.strip() for t in raw.split(",") if t.strip()]
    out: list[dict[str, Any]] = []
    for tag in tags:
        if tag in catalog:
            out.append(catalog[tag])
        else:
            out.append(
                {
                    "id": tag.replace(":", "-"),
                    "tag": tag,
                    "title": tag,
                    "size_hint_gb": None,
                    "notes": "cli override",
                }
            )
    if max_size_gb is not None and not allow_large:
        out = filter_models_by_max_size_gb(out, max_size_gb, allow_unknown=True)
    return out


def resolve_benchmark_models(
    suite: dict[str, Any],
    *,
    models_arg: str | None,
    models_config: Path,
    suites_config: Path,
    allow_large: bool,
    max_size_gb: float | None = None,
    max_params_b: float | None = None,  # deprecated alias
) -> list[dict[str, Any]]:
    del suites_config  # reserved for suite-relative roster overrides
    if max_size_gb is None:
        max_size_gb = max_params_b
    cap = max_size_gb if max_size_gb is not None else resolve_suite_max_size_gb(suite, models_config)

    if models_arg:
        models = parse_models_arg(
            models_arg,
            models_config,
            max_size_gb=cap,
            allow_large=allow_large,
        )
    elif resolve_suite_model_tags(suite):
        models = parse_models_arg(
            ",".join(resolve_suite_model_tags(suite)),
            models_config,
            max_size_gb=cap,
            allow_large=allow_large,
        )
    else:
        models = load_models(models_config)
        if not allow_large:
            models = filter_models_by_max_size_gb(models, cap)

    if not models:
        raise ValueError(f"no models at or below {cap} GB footprint (use --allow-large-models to bypass cap)")
    return models


def load_suite(name: str, config_path: Path | None = None) -> dict[str, Any]:
    data = load_json_config(config_path or SUITES_CONFIG)
    suite = data.get(name)
    if not isinstance(suite, dict):
        known = ", ".join(sorted(k for k in data if k != "description"))
        raise ValueError(f"unknown suite {name!r}; known: {known}")
    return suite


def list_implement_scenarios() -> list[str]:
    scenarios_dir = ROOT / "scenarios" / "implement"
    return sorted(p.stem for p in scenarios_dir.glob("*.json"))


def list_collab_scenarios() -> list[str]:
    scenarios_dir = ROOT / "scenarios" / "collab"
    return sorted(p.stem for p in scenarios_dir.glob("*.json"))


def list_chat_scenarios(tag: str | None = None) -> list[str]:
    proc = subprocess.run(
        [sys.executable, str(SCRIPTS_DIR / "chat-scenarios.py"), "--list"],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    names: list[str] = []
    for line in (proc.stdout or "").splitlines():
        parts = line.split("\t", 1)
        if not parts or not parts[0].strip():
            continue
        name = parts[0].strip()
        if not tag:
            names.append(name)
            continue
        tags_part = parts[1].split(" (optional)", 1)[0] if len(parts) > 1 else ""
        have = {t.strip().lower() for t in tags_part.split(",") if t.strip() and t.strip() != "-"}
        if tag.strip().lower() in have:
            names.append(name)
    return names


def _string_list(raw: Any) -> list[str]:
    if not isinstance(raw, list):
        return []
    return [str(x).strip() for x in raw if str(x).strip()]


def resolve_suite_scenarios(suite: dict[str, Any]) -> SuiteTracks:
    implement_raw = suite.get("implement") or []
    chat_raw = suite.get("chat") or []
    chat_tag = (suite.get("chat_tag") or "").strip() or None

    if implement_raw == "all":
        implement = list_implement_scenarios()
    else:
        implement = _string_list(implement_raw)

    if chat_tag:
        chat = list_chat_scenarios(chat_tag)
    else:
        chat = _string_list(chat_raw)

    collab_raw = suite.get("collab")
    if collab_raw == "core":
        collab = collab_core_scenarios()
    elif collab_raw == "all":
        collab = list_collab_scenarios()
    else:
        collab = _string_list(collab_raw)

    return SuiteTracks(
        implement=implement,
        chat=chat,
        collab=collab,
        arena=_string_list(suite.get("arena")),
        cad=_string_list(suite.get("cad")),
        external=_string_list(suite.get("external")),
    )


def ollama_installed_tags(hub_url: str) -> set[str]:
    code, data = hub.hub_request(hub_url.rstrip("/"), "GET", "/api/ollama/models")
    if code != 200:
        return set()
    raw = data
    if isinstance(data, dict):
        raw = data.get("models")
    if not isinstance(raw, list):
        return set()
    out: set[str] = set()
    for item in raw:
        if isinstance(item, str):
            out.add(item.strip())
        elif isinstance(item, dict):
            name = (item.get("name") or "").strip()
            if name:
                out.add(name)
    return out


def model_is_installed(installed: set[str], tag: str, *, pull_tag: str = "") -> bool:
    tag = tag.strip()
    if not tag:
        return False
    if tag in installed:
        return True
    base = tag.split(":", 1)[0]
    if any(name == tag or name.startswith(f"{base}:") for name in installed):
        return True
    pull_tag = pull_tag.strip()
    if pull_tag and pull_tag != tag:
        return model_is_installed(installed, pull_tag)
    return False


def switch_all_ollama(hub_url: str, model_tag: str) -> tuple[bool, str]:
    body = {"provider": "ollama", "model": model_tag.strip()}
    code, data = hub.hub_request(hub_url.rstrip("/"), "POST", "/api/agents/switch-all-providers", body)
    if code != 200:
        detail = data if isinstance(data, str) else json.dumps(data)
        return False, f"switch-all HTTP {code}: {detail}"
    if isinstance(data, dict):
        return True, str(data.get("message") or "switched")
    return True, "switched"


def pull_ollama_model(hub_url: str, model_tag: str, timeout_s: float = 3600) -> tuple[bool, str]:
    """Best-effort pull via hub SSE endpoint (may take a long time)."""
    url = f"{hub_url.rstrip('/')}/api/ollama/pull"
    payload = json.dumps({"model": model_tag.strip()}).encode()
    req = urllib.request.Request(
        url,
        data=payload,
        headers={"Content-Type": "application/json", "Accept": "text/event-stream"},
        method="POST",
    )
    started = time.monotonic()
    last_err = ""
    try:
        with urllib.request.urlopen(req, timeout=timeout_s) as resp:
            while True:
                chunk = resp.readline()
                if not chunk:
                    break
                line = chunk.decode(errors="replace").strip()
                if line.startswith("data: "):
                    raw = line[6:].strip()
                    if not raw:
                        continue
                    try:
                        evt = json.loads(raw)
                    except json.JSONDecodeError:
                        continue
                    if evt.get("status") == "error" or evt.get("error"):
                        last_err = str(evt.get("error") or evt.get("status"))
                    if evt.get("status") == "success":
                        return True, f"pulled in {time.monotonic() - started:.0f}s"
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        return False, f"pull HTTP {e.code}: {body[:200]}"
    except (urllib.error.URLError, TimeoutError) as e:
        return False, str(e)
    if last_err:
        return False, last_err
    return True, f"pull stream ended ({time.monotonic() - started:.0f}s)"


def _optional_int(value: Any) -> int | None:
    if value is None:
        return None
    try:
        return int(value)
    except (TypeError, ValueError):
        return None


def _optional_float(value: Any) -> float | None:
    if value is None:
        return None
    try:
        return float(value)
    except (TypeError, ValueError):
        return None


def scenario_metrics_from_dict(data: dict[str, Any]) -> ScenarioMetrics:
    def strings(name: str) -> list[str]:
        raw = data.get(name)
        return [str(item) for item in raw] if isinstance(raw, list) else []

    return ScenarioMetrics(
        prompt_tokens=_optional_int(data.get("prompt_tokens")),
        completion_tokens=_optional_int(data.get("completion_tokens")),
        ttft_ms=_optional_float(data.get("ttft_ms")),
        tok_per_s=_optional_float(data.get("tok_per_s")),
        tool_calls=_optional_int(data.get("tool_calls")),
        repair_attempts=_optional_int(data.get("repair_attempts")),
        wall_duration_s=_optional_float(data.get("wall_duration_s")),
        cost_usd=_optional_float(data.get("cost_usd")),
        passed_at_1=data.get("passed_at_1") if isinstance(data.get("passed_at_1"), bool) else None,
        eventual_pass=data.get("eventual_pass") if isinstance(data.get("eventual_pass"), bool) else None,
        attempts=_optional_int(data.get("attempts")),
        retry_count=_optional_int(data.get("retry_count")),
        retry_reasons=strings("retry_reasons"),
        nudge_count=_optional_int(data.get("nudge_count")),
        nudge_reasons=strings("nudge_reasons"),
        actual_provider=str(data["actual_provider"]) if data.get("actual_provider") else None,
        actual_model=str(data["actual_model"]) if data.get("actual_model") else None,
        validation_failures=strings("validation_failures"),
        escalation_count=_optional_int(data.get("escalation_count")),
        escalation_reasons=strings("escalation_reasons"),
    )


def parse_metrics_from_output(output: str) -> ScenarioMetrics | None:
    """Parse METRICS_JSON: line into ScenarioMetrics."""
    for line in output.splitlines():
        if not line.startswith("METRICS_JSON:"):
            continue
        raw = line[len("METRICS_JSON:") :].strip()
        try:
            data = json.loads(raw)
        except json.JSONDecodeError:
            return None
        if isinstance(data, dict):
            return scenario_metrics_from_dict(data)
        return None
    return None


def parse_metrics_payload_from_output(output: str) -> dict[str, Any] | None:
    for line in output.splitlines():
        if not line.startswith("METRICS_JSON:"):
            continue
        raw = line[len("METRICS_JSON:") :].strip()
        try:
            data = json.loads(raw)
        except json.JSONDecodeError:
            return None
        return data if isinstance(data, dict) else None
    return None


def parse_quality_score(text: str) -> float | None:
    if not text:
        return None
    match = _SCORE_RE.search(text)
    if not match:
        return None
    try:
        return max(0.0, min(1.0, float(match.group(1))))
    except ValueError:
        return None


def parse_judge_from_output(output: str) -> tuple[bool | None, str]:
    """Extract deliverable judge verdict from implement scenario log lines."""
    for line in output.splitlines():
        lower = line.lower()
        for prefix, verdict in (
            ("judge:pass:", True),
            ("judge:fail:", False),
            ("judge:warn:", False),
        ):
            idx = lower.find(prefix)
            if idx >= 0:
                reason = line[idx + len(prefix) :].strip()
                return verdict, reason
        if "llm_judge:" in lower:
            idx = lower.find("llm_judge:")
            reason = line[idx + len("llm_judge:") :].strip()
            return False, reason
    return None, ""


def run_script_scenario(
    script: str,
    scenario: str,
    hub_url: str,
    *,
    kind: str = "",
    extra_args: list[str] | None = None,
    env: dict[str, str] | None = None,
) -> ScenarioResult:
    cmd = [sys.executable, str(SCRIPTS_DIR / script), "--scenario", scenario, "--hub", hub_url]
    if extra_args:
        cmd.extend(extra_args)
    run_env = None
    if env:
        run_env = {**os.environ, **env}
    t0 = time.monotonic()
    proc = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True, check=False, env=run_env)
    elapsed = time.monotonic() - t0
    output = (proc.stdout or "") + (proc.stderr or "")
    passed = proc.returncode == 0
    detail = ""
    for line in output.splitlines():
        if "FAIL:" in line or "PASS:" in line:
            detail = line.strip()
            break
    if not detail and not passed:
        detail = output.strip().splitlines()[-1] if output.strip() else f"exit {proc.returncode}"
    judge_passed, judge_reason = parse_judge_from_output(output)
    uses_judge = judge_passed is not None
    metrics = parse_metrics_from_output(output)
    payload = parse_metrics_payload_from_output(output) or {}
    quality_score = parse_quality_score(judge_reason) or parse_quality_score(output)
    if quality_score is None:
        quality_score = _optional_float(payload.get("quality_score"))
    capability_passed = payload.get("capability_passed")
    if capability_passed is not None:
        capability_passed = bool(capability_passed)
    quality_passed = judge_passed if judge_passed is not None else None
    if quality_passed is None and payload.get("quality_passed") is not None:
        quality_passed = bool(payload.get("quality_passed"))
    agent_efficiency = _optional_float(payload.get("agent_efficiency"))
    resolved_kind = (kind or "").strip()
    if not resolved_kind:
        for k, script_name in SCRIPT_BY_KIND.items():
            if script.endswith(script_name) or script == script_name:
                resolved_kind = k
                break
        if not resolved_kind:
            resolved_kind = "implement"
    return ScenarioResult(
        name=scenario,
        kind=resolved_kind,
        passed=passed,
        duration_s=elapsed,
        detail=detail,
        judge_passed=judge_passed,
        judge_reason=judge_reason,
        uses_llm_judge=uses_judge,
        metrics=metrics,
        structural_passed=passed,
        capability_passed=capability_passed,
        quality_score=quality_score,
        quality_passed=quality_passed,
        agent_efficiency=agent_efficiency,
    )


def fetch_hardware_snapshot(hub_url: str) -> dict[str, Any]:
    code, data = hub.hub_request(hub_url.rstrip("/"), "GET", "/api/system/hardware")
    if code != 200 or not isinstance(data, dict):
        return {}
    return {
        "total_memory_gb": data.get("total_memory_gb"),
        "total_memory_bytes": data.get("total_memory_bytes"),
        "tier": data.get("tier"),
    }


def resolve_judge_provider_note() -> str:
    provider = os.environ.get("NJ_DELIVERABLE_JUDGE_PROVIDER", "claude").strip().lower()
    model = os.environ.get("NJ_DELIVERABLE_JUDGE_MODEL", "").strip()
    if provider == "ollama":
        model = model or os.environ.get("NJ_DELIVERABLE_JUDGE_MODEL", "qwen2.5-coder:14b")
        return f"ollama/{model}" if model else "ollama"
    if model:
        return f"{provider}/{model}"
    return provider or "claude"


def load_scenario_meta(kind: str, name: str) -> dict[str, Any]:
    if kind == "implement":
        path = ROOT / "scenarios" / "implement" / f"{name}.json"
    elif kind == "chat":
        path = ROOT / "scenarios" / "chat" / f"{name}.json"
    elif kind == "collab":
        path = ROOT / "scenarios" / "collab" / f"{name}.json"
    elif kind in ("arena", "cad", "external"):
        descriptions = {
            "arena": f"Arena track scenario {name}",
            "cad": f"CAD compile track scenario {name}",
            "external": f"External calibration scenario {name}",
        }
        return {
            "name": name,
            "kind": kind,
            "description": descriptions.get(kind, ""),
            "llm_judge": False,
            "tags": [kind],
            "target_agent": "",
        }
    else:
        return {"name": name, "kind": kind, "description": "", "llm_judge": False, "tags": []}
    if not path.is_file():
        return {"name": name, "kind": kind, "description": "", "llm_judge": False, "tags": []}
    data = load_json_config(path)
    llm_judge = False
    for deliverable in data.get("expect_deliverables") or []:
        if isinstance(deliverable, dict) and deliverable.get("llm_judge"):
            llm_judge = True
            break
    return {
        "name": name,
        "kind": kind,
        "description": str(data.get("description") or "").strip(),
        "tags": [str(t) for t in (data.get("tags") or []) if str(t).strip()],
        "llm_judge": llm_judge,
        "target_agent": str(data.get("target_agent") or "").strip(),
    }


def build_scenario_catalog(
    implement_names: list[str] | SuiteTracks | None = None,
    chat_names: list[str] | None = None,
    *,
    collab_names: list[str] | None = None,
    arena_names: list[str] | None = None,
    cad_names: list[str] | None = None,
    external_names: list[str] | None = None,
    tracks: SuiteTracks | None = None,
) -> list[dict[str, Any]]:
    if isinstance(implement_names, SuiteTracks):
        tracks = implement_names
        implement_names = None
    if tracks is not None:
        implement_names = tracks.implement
        chat_names = tracks.chat
        collab_names = tracks.collab
        arena_names = tracks.arena
        cad_names = tracks.cad
        external_names = tracks.external
    catalog: list[dict[str, Any]] = []
    for name in implement_names or []:
        catalog.append(load_scenario_meta("implement", name))
    for name in chat_names or []:
        catalog.append(load_scenario_meta("chat", name))
    for name in collab_names or []:
        catalog.append(load_scenario_meta("collab", name))
    for name in arena_names or []:
        catalog.append(load_scenario_meta("arena", name))
    for name in cad_names or []:
        catalog.append(load_scenario_meta("cad", name))
    for name in external_names or []:
        catalog.append(load_scenario_meta("external", name))
    return catalog


def format_duration(seconds: float) -> str:
    seconds = max(0.0, seconds)
    if seconds < 60:
        return f"{seconds:.0f}s"
    mins, secs = divmod(int(seconds), 60)
    if mins < 60:
        return f"{mins}m{secs:02d}s"
    hours, mins = divmod(mins, 60)
    return f"{hours}h{mins:02d}m"


def render_markdown_report(
    *,
    suite_name: str,
    suite_desc: str,
    hub_url: str,
    results: list[ModelBenchmarkResult],
    implement_names: list[str],
    chat_names: list[str],
    collab_names: list[str] | None = None,
    arena_names: list[str] | None = None,
    cad_names: list[str] | None = None,
    external_names: list[str] | None = None,
    hardware: dict[str, Any] | None = None,
    judge_provider: str = "",
) -> str:
    collab_names = collab_names or []
    arena_names = arena_names or []
    cad_names = cad_names or []
    external_names = external_names or []
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    lines = [
        f"# Model benchmark — {suite_name}",
        "",
        f"**Run:** {stamp}  ",
        f"**Hub:** `{hub_url}`  ",
        f"**Suite:** {suite_desc}",
        "",
        "## Summary",
        "",
        "| Rank | Configured model | Pass@1 | Eventual | Implement | Chat | Total time | Notes |",
        "|------|------------------|--------|----------|-----------|------|------------|-------|",
    ]

    # Published/historical winner uses structural pass rate for continuity;
    # release exit gates use composite_pass_rate separately in the suite CLI.
    ranked = sorted(
        [r for r in results if not r.skipped and r.scenarios],
        key=lambda r: (-r.pass_rate, -r.passed, r.total_duration_s),
    )
    skipped = [r for r in results if r.skipped]

    for i, r in enumerate(ranked, 1):
        impl = [s for s in r.scenarios if s.kind == "implement"]
        chat = [s for s in r.scenarios if s.kind == "chat"]
        impl_s = f"{sum(1 for s in impl if s.passed)}/{len(impl)}" if impl else "—"
        chat_s = f"{sum(1 for s in chat if s.passed)}/{len(chat)}" if chat else "—"
        first_rate = f"{r.pass_at_1_rate * 100:.0f}%"
        eventual_rate = f"{r.pass_rate * 100:.0f}%"
        notes = ""
        if r.model_tag == ranked[0].model_tag and r.passed == r.total and r.total:
            notes = "winner"
        lines.append(
            f"| {i} | `{r.model_tag}` | {first_rate} | {r.passed}/{r.total} ({eventual_rate}) | {impl_s} | {chat_s} | {format_duration(r.total_duration_s)} | {notes} |"
        )

    if skipped:
        lines.extend(["", "## Skipped models", ""])
        for r in skipped:
            lines.append(f"- `{r.model_tag}` — {r.skip_reason}")

    lines.extend(["", "## Per-scenario matrix", ""])
    all_scenarios = (
        [("implement", n) for n in implement_names]
        + [("chat", n) for n in chat_names]
        + [("collab", n) for n in collab_names]
        + [("arena", n) for n in arena_names]
        + [("cad", n) for n in cad_names]
        + [("external", n) for n in external_names]
    )
    if all_scenarios:
        header = "| Scenario | " + " | ".join(r.model_tag for r in results if not r.skipped) + " |"
        sep = "|---|" + "|".join("---" for r in results if not r.skipped) + "|"
        lines.extend([header, sep])
        for kind, name in all_scenarios:
            cells = []
            for r in results:
                if r.skipped:
                    continue
                hit = next((s for s in r.scenarios if s.kind == kind and s.name == name), None)
                if not hit:
                    cells.append("—")
                elif hit.passed:
                    cells.append(f"✓ {format_duration(hit.duration_s)}")
                else:
                    cells.append(f"✗ {format_duration(hit.duration_s)}")
            lines.append(f"| {kind}/{name} | " + " | ".join(cells) + " |")

    lines.extend(
        [
            "",
            "## Scenario lists",
            "",
            f"- **Implement:** {', '.join(implement_names) or '(none)'}",
            f"- **Chat:** {', '.join(chat_names) or '(none)'}",
            f"- **Collab:** {', '.join(collab_names) or '(none)'}",
            f"- **Arena:** {', '.join(arena_names) or '(none)'}",
            f"- **CAD:** {', '.join(cad_names) or '(none)'}",
            f"- **External:** {', '.join(external_names) or '(none)'}",
            "",
        ]
    )
    if hardware:
        lines.extend(
            [
                "## Hardware",
                "",
                f"- **RAM:** {hardware.get('total_memory_gb', '?')} GB ({hardware.get('tier', '?')} tier)",
                "",
            ]
        )
    if judge_provider:
        lines.extend(["## Deliverable judge", "", f"- **Provider:** `{judge_provider}`", ""])
    return "\n".join(lines)


def write_reports(
    out_dir: Path,
    *,
    suite_name: str,
    suite_desc: str,
    hub_url: str,
    results: list[ModelBenchmarkResult],
    implement_names: list[str],
    chat_names: list[str],
    collab_names: list[str] | None = None,
    arena_names: list[str] | None = None,
    cad_names: list[str] | None = None,
    external_names: list[str] | None = None,
    hardware: dict[str, Any] | None = None,
    judge_provider: str = "",
) -> tuple[Path, Path, Path]:
    collab_names = collab_names or []
    arena_names = arena_names or []
    cad_names = cad_names or []
    external_names = external_names or []
    out_dir.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M")
    md_path = out_dir / f"model-benchmark-{suite_name}-{stamp}.md"
    json_path = out_dir / f"model-benchmark-{suite_name}-{stamp}.json"
    tsv_path = out_dir / f"model-benchmark-{suite_name}-{stamp}.tsv"

    md_path.write_text(
        render_markdown_report(
            suite_name=suite_name,
            suite_desc=suite_desc,
            hub_url=hub_url,
            results=results,
            implement_names=implement_names,
            chat_names=chat_names,
            collab_names=collab_names,
            arena_names=arena_names,
            cad_names=cad_names,
            external_names=external_names,
            hardware=hardware,
            judge_provider=judge_provider,
        ),
        encoding="utf-8",
    )

    hw_note = ""
    if hardware and hardware.get("total_memory_gb"):
        hw_note = f"{hardware['total_memory_gb']} GB RAM ({hardware.get('tier', '?')} tier)"

    tracks = SuiteTracks(
        implement=list(implement_names),
        chat=list(chat_names),
        collab=list(collab_names),
        arena=list(arena_names),
        cad=list(cad_names),
        external=list(external_names),
    )
    payload = {
        "suite": suite_name,
        "description": suite_desc,
        "hub": hub_url,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "hardware": hardware or {},
        "hardware_note": hw_note,
        "judge_provider": judge_provider,
        "scenario_catalog": build_scenario_catalog(tracks),
        "implement_scenarios": implement_names,
        "chat_scenarios": chat_names,
        "collab_scenarios": collab_names,
        "arena_scenarios": arena_names,
        "cad_scenarios": cad_names,
        "external_scenarios": external_names,
        "results": [r.to_dict() for r in results],
    }
    json_path.write_text(json.dumps(payload, indent=2), encoding="utf-8")

    tsv_lines = [
        "model_tag\tscenario_kind\tscenario\tpassed_at_1\teventual_pass\tattempts\tduration_s\tjudge_passed\tjudge_reason\t"
        "actual_provider\tactual_model\tretry_reasons\tnudge_reasons\tvalidation_failures\tescalation_reasons\t"
        "prompt_tokens\tcompletion_tokens\tttft_ms\ttok_per_s\ttool_calls\trepair_attempts\t"
        "quality_score\tstructural_passed\tdetail"
    ]
    for r in results:
        for s in r.scenarios:
            judge_passed = "" if s.judge_passed is None else str(int(s.judge_passed))
            m = s.metrics or ScenarioMetrics()
            structural = "" if s.structural_passed is None else str(int(s.structural_passed))
            quality = "" if s.quality_score is None else f"{s.quality_score:.4f}"
            tsv_lines.append(
                "\t".join(
                    [
                        r.model_tag,
                        s.kind,
                        s.name,
                        str(int(m.passed_at_1 if m.passed_at_1 is not None else s.passed)),
                        str(int(m.eventual_pass if m.eventual_pass is not None else s.passed)),
                        "" if m.attempts is None else str(m.attempts),
                        f"{s.duration_s:.1f}",
                        judge_passed,
                        s.judge_reason.replace("\t", " "),
                        m.actual_provider or "",
                        m.actual_model or "",
                        "; ".join(m.retry_reasons).replace("\t", " "),
                        "; ".join(m.nudge_reasons).replace("\t", " "),
                        "; ".join(m.validation_failures).replace("\t", " "),
                        "; ".join(m.escalation_reasons).replace("\t", " "),
                        "" if m.prompt_tokens is None else str(m.prompt_tokens),
                        "" if m.completion_tokens is None else str(m.completion_tokens),
                        "" if m.ttft_ms is None else f"{m.ttft_ms:.1f}",
                        "" if m.tok_per_s is None else f"{m.tok_per_s:.2f}",
                        "" if m.tool_calls is None else str(m.tool_calls),
                        "" if m.repair_attempts is None else str(m.repair_attempts),
                        quality,
                        structural,
                        s.detail.replace("\t", " "),
                    ]
                )
            )
    tsv_path.write_text("\n".join(tsv_lines) + "\n", encoding="utf-8")
    return md_path, json_path, tsv_path


def _roster_by_tag(roster: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    out: dict[str, dict[str, Any]] = {}
    for model in roster:
        tag = str(model.get("tag") or "").strip()
        if tag:
            out[tag] = model
    return out


def _model_rank_key(result: dict[str, Any], *, kind: str | None = None) -> tuple[float, int, float]:
    scenarios = result.get("scenarios") or []
    if kind:
        scenarios = [s for s in scenarios if isinstance(s, dict) and s.get("kind") == kind]
    total = len(scenarios)
    passed = sum(1 for s in scenarios if isinstance(s, dict) and s.get("passed"))
    rate = passed / total if total else 0.0
    duration = float(result.get("total_duration_s") or 0.0)
    return (rate, passed, -duration)


def _scenario_map(result: dict[str, Any]) -> dict[str, bool]:
    out: dict[str, bool] = {}
    for s in result.get("scenarios") or []:
        if not isinstance(s, dict):
            continue
        kind = str(s.get("kind") or "").strip()
        name = str(s.get("name") or "").strip()
        if kind == "external":
            continue
        if kind and name:
            out[f"{kind}/{name}"] = bool(s.get("passed"))
    return out


def _pass_rates(result: dict[str, Any]) -> tuple[float, float, float]:
    scenarios = [
        s
        for s in (result.get("scenarios") or [])
        if isinstance(s, dict) and s.get("kind") != "external"
    ]
    impl = [s for s in scenarios if s.get("kind") == "implement"]
    chat = [s for s in scenarios if s.get("kind") == "chat"]
    impl_rate = sum(1 for s in impl if s.get("passed")) / len(impl) if impl else 0.0
    chat_rate = sum(1 for s in chat if s.get("passed")) / len(chat) if chat else 0.0
    overall = sum(1 for s in scenarios if s.get("passed")) / len(scenarios) if scenarios else 0.0
    return impl_rate, chat_rate, overall


def _run_has_kind(run: dict[str, Any], kind: str) -> bool:
    for result in run.get("results") or []:
        if not isinstance(result, dict):
            continue
        for s in result.get("scenarios") or []:
            if isinstance(s, dict) and s.get("kind") == kind:
                return True
    for key in (
        f"{kind}_scenarios",
        "implement_scenarios" if kind == "implement" else "",
        "chat_scenarios" if kind == "chat" else "",
    ):
        if key and run.get(key):
            return True
    return False


def latest_quick_run(catalog: dict[str, Any]) -> dict[str, Any] | None:
    runs = [r for r in (catalog.get("runs") or []) if isinstance(r, dict)]
    quick = [r for r in runs if str(r.get("suite") or "").strip() == "quick"]
    if not quick:
        return None
    quick.sort(key=lambda r: r.get("generated_at") or "", reverse=True)
    return quick[0]


def latest_run_with_kinds(
    catalog: dict[str, Any],
    kinds: list[str],
    *,
    prefer_suite: str | None = "quick",
) -> dict[str, Any] | None:
    """Prefer a preferred-suite run when it has kinds; else latest run that has them."""
    runs = [r for r in (catalog.get("runs") or []) if isinstance(r, dict)]
    runs.sort(key=lambda r: r.get("generated_at") or "", reverse=True)
    preferred: list[dict[str, Any]] = []
    if prefer_suite:
        preferred = [r for r in runs if str(r.get("suite") or "").strip() == prefer_suite]
        for run in preferred:
            if all(_run_has_kind(run, k) for k in kinds):
                return run
            if not kinds:
                return run
    for run in runs:
        if all(_run_has_kind(run, k) for k in kinds):
            return run
    if preferred:
        return preferred[0]
    return runs[0] if runs else None


def derive_capability_profiles(
    catalog: dict[str, Any],
    roster: list[dict[str, Any]] | None = None,
    *,
    suite: str = "quick",
) -> dict[str, Any]:
    """Build ranked model lists per task class from the latest benchmark run."""
    roster = roster if roster is not None else load_models()
    roster_tags = {str(m.get("tag") or "").strip() for m in roster if str(m.get("tag") or "").strip()}
    roster_map = _roster_by_tag(roster)

    prefer = "quick" if suite == "quick" else suite
    run = latest_run_with_kinds(catalog, ["implement", "chat"], prefer_suite=prefer)
    if run is None and suite == "quick":
        run = latest_quick_run(catalog)
    if run is None:
        runs = [r for r in (catalog.get("runs") or []) if isinstance(r, dict) and r.get("suite") == suite]
        runs.sort(key=lambda r: r.get("generated_at") or "", reverse=True)
        run = runs[0] if runs else None
    if run is None:
        raise ValueError(f"no benchmark run found for suite {suite!r}")

    # Enrich with newest run that carries extra tracks when the day-to-day source lacks them.
    for kind in ("collab", "arena", "cad"):
        if not _run_has_kind(run, kind):
            richer = latest_run_with_kinds(catalog, [kind], prefer_suite=None)
            if richer is not None and _run_has_kind(richer, kind):
                # Prefer keeping quick for base ranking but merge awareness via per-kind ranks.
                pass

    active = [
        r
        for r in (run.get("results") or [])
        if isinstance(r, dict)
        and not r.get("skipped")
        and str(r.get("model_tag") or "").strip() in roster_tags
    ]

    def results_for_kind(kind: str) -> tuple[dict[str, Any], list[dict[str, Any]]]:
        source = run
        if not _run_has_kind(source, kind):
            alt = latest_run_with_kinds(catalog, [kind], prefer_suite=None)
            if alt is not None:
                source = alt
        rows = [
            r
            for r in (source.get("results") or [])
            if isinstance(r, dict)
            and not r.get("skipped")
            and str(r.get("model_tag") or "").strip() in roster_tags
        ]
        return source, rows

    def rank_tags(
        *,
        kind: str | None = None,
        params_max: float | None = None,
        params_min: float | None = None,
        candidates: list[dict[str, Any]] | None = None,
    ) -> list[str]:
        pool = candidates if candidates is not None else active
        if params_max is not None or params_min is not None:
            filtered: list[dict[str, Any]] = []
            for r in pool:
                tag = str(r.get("model_tag") or "").strip()
                params = model_params_b(roster_map.get(tag, {"tag": tag}))
                if params is None:
                    continue
                if params_max is not None and params > params_max:
                    continue
                if params_min is not None and params <= params_min:
                    continue
                filtered.append(r)
            pool = filtered
        ranked = sorted(pool, key=lambda r: _model_rank_key(r, kind=kind), reverse=True)
        return [str(r.get("model_tag") or "").strip() for r in ranked if str(r.get("model_tag") or "").strip()]

    def ask_mode_rank() -> list[str]:
        passed = [
            r
            for r in active
            if _scenario_map(r).get("implement/ask-mode-no-write") is True
        ]
        rest = [r for r in active if r not in passed]
        ordered = sorted(passed, key=lambda r: _model_rank_key(r, kind="implement"), reverse=True)
        ordered += sorted(rest, key=lambda r: _model_rank_key(r, kind="implement"), reverse=True)
        seen: set[str] = set()
        out: list[str] = []
        for r in ordered:
            tag = str(r.get("model_tag") or "").strip()
            if tag and tag not in seen:
                seen.add(tag)
                out.append(tag)
        return out

    has_collab = False
    for check_run in (catalog.get("runs") or []):
        if isinstance(check_run, dict) and _run_has_kind(check_run, "collab"):
            has_collab = True
            break
    if has_collab:
        _, collab_active = results_for_kind("collab")
        collab_light = rank_tags(kind="collab", candidates=collab_active) or rank_tags(params_max=9.0)
    else:
        collab_light = rank_tags(params_max=9.0)

    task_classes: dict[str, list[str]] = {
        "implement": rank_tags(kind="implement"),
        "chat": rank_tags(kind="chat"),
        "collab_light": collab_light,
        "utility": rank_tags(params_max=9.0),
        "ask_mode": ask_mode_rank(),
        "implement_heavy": rank_tags(kind="implement", params_min=14.0),
    }

    for kind, class_name in (("arena", "arena_logic"), ("cad", "cad_compile")):
        if any(isinstance(r, dict) and _run_has_kind(r, kind) for r in (catalog.get("runs") or [])):
            _, kind_active = results_for_kind(kind)
            task_classes[class_name] = rank_tags(kind=kind, candidates=kind_active)

    model_scores: dict[str, Any] = {}
    for r in active:
        tag = str(r.get("model_tag") or "").strip()
        if not tag:
            continue
        impl_rate, chat_rate, overall = _pass_rates(r)
        params = model_params_b(roster_map.get(tag, {"tag": tag}))
        model_scores[tag] = {
            "implement_pass_rate": round(impl_rate, 4),
            "chat_pass_rate": round(chat_rate, 4),
            "overall_pass_rate": round(overall, 4),
            "params_b": params,
            "scenarios": _scenario_map(r),
        }

    return {
        "updated_at": datetime.now(timezone.utc).isoformat(),
        "source_run_id": str(run.get("id") or ""),
        "source_suite": str(run.get("suite") or suite),
        "task_classes": task_classes,
        "model_scores": model_scores,
    }


def write_capability_profiles(
    profiles: dict[str, Any],
    *,
    docs_path: Path,
    embed_path: Path,
) -> None:
    payload = json.dumps(profiles, indent=2) + "\n"
    docs_path.parent.mkdir(parents=True, exist_ok=True)
    docs_path.write_text(payload, encoding="utf-8")
    embed_path.parent.mkdir(parents=True, exist_ok=True)
    embed_path.write_text(payload, encoding="utf-8")
