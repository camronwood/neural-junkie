"""Helpers for multi-model live hub benchmarks."""

from __future__ import annotations

import json
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
ROOT = SCRIPTS_DIR.parent
CONFIG_DIR = SCRIPTS_DIR / "config"
MODELS_CONFIG = CONFIG_DIR / "model-benchmark-models.json"
SUITES_CONFIG = CONFIG_DIR / "model-benchmark-suites.json"

sys.path.insert(0, str(SCRIPTS_DIR))
from lib import collab_hub as hub  # noqa: E402


@dataclass
class ScenarioResult:
    name: str
    kind: str
    passed: bool
    duration_s: float
    detail: str = ""


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

    def to_dict(self) -> dict[str, Any]:
        d = asdict(self)
        d["passed_count"] = self.passed
        d["total_count"] = self.total
        d["pass_rate"] = round(self.pass_rate, 4)
        return d


def load_json_config(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def load_models(config_path: Path | None = None) -> list[dict[str, Any]]:
    data = load_json_config(config_path or MODELS_CONFIG)
    models = data.get("models") or []
    if not isinstance(models, list):
        raise ValueError("models config must contain a models array")
    return [m for m in models if isinstance(m, dict) and (m.get("tag") or "").strip()]


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


def filter_models_by_max_params(
    models: list[dict[str, Any]],
    max_params_b: float,
    *,
    allow_unknown: bool = False,
) -> list[dict[str, Any]]:
    """Drop models above max_params_b (and unknown sizes unless allow_unknown)."""
    kept: list[dict[str, Any]] = []
    for model in models:
        params = model_params_b(model)
        if params is None:
            if allow_unknown:
                kept.append(model)
            continue
        if params <= max_params_b:
            kept.append(model)
    return kept


def resolve_suite_max_params_b(suite: dict[str, Any], config_path: Path | None = None) -> float:
    raw = suite.get("max_params_b")
    if raw is not None:
        try:
            return float(raw)
        except (TypeError, ValueError):
            pass
    data = load_json_config(config_path or MODELS_CONFIG)
    default = data.get("max_params_b_default")
    if default is not None:
        try:
            return float(default)
        except (TypeError, ValueError):
            pass
    return 24.0


def resolve_suite_model_tags(suite: dict[str, Any]) -> list[str]:
    raw = suite.get("models") or []
    if not isinstance(raw, list):
        return []
    return [str(x).strip() for x in raw if str(x).strip()]


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


def resolve_suite_scenarios(suite: dict[str, Any]) -> tuple[list[str], list[str]]:
    implement_raw = suite.get("implement") or []
    chat_raw = suite.get("chat") or []
    chat_tag = (suite.get("chat_tag") or "").strip() or None

    if implement_raw == "all":
        implement = list_implement_scenarios()
    else:
        implement = [str(x).strip() for x in implement_raw if str(x).strip()]

    if chat_tag:
        chat = list_chat_scenarios(chat_tag)
    else:
        chat = [str(x).strip() for x in chat_raw if str(x).strip()]

    return implement, chat


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


def model_is_installed(installed: set[str], tag: str) -> bool:
    tag = tag.strip()
    if not tag:
        return False
    if tag in installed:
        return True
    base = tag.split(":", 1)[0]
    return any(name == tag or name.startswith(f"{base}:") for name in installed)


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


def run_script_scenario(
    script: str,
    scenario: str,
    hub_url: str,
    *,
    extra_args: list[str] | None = None,
) -> tuple[bool, float, str]:
    cmd = [sys.executable, str(SCRIPTS_DIR / script), "--scenario", scenario, "--hub", hub_url]
    if extra_args:
        cmd.extend(extra_args)
    t0 = time.monotonic()
    proc = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True, check=False)
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
    return passed, elapsed, detail


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
) -> str:
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
        "| Rank | Model | Pass | Rate | Implement | Chat | Total time | Notes |",
        "|------|-------|------|------|-----------|------|------------|-------|",
    ]

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
        rate = f"{r.pass_rate * 100:.0f}%"
        notes = ""
        if r.model_tag == ranked[0].model_tag and r.passed == r.total and r.total:
            notes = "winner"
        lines.append(
            f"| {i} | `{r.model_tag}` | {r.passed}/{r.total} | {rate} | {impl_s} | {chat_s} | {format_duration(r.total_duration_s)} | {notes} |"
        )

    if skipped:
        lines.extend(["", "## Skipped models", ""])
        for r in skipped:
            lines.append(f"- `{r.model_tag}` — {r.skip_reason}")

    lines.extend(["", "## Per-scenario matrix", ""])
    all_scenarios = [( "implement", n) for n in implement_names] + [("chat", n) for n in chat_names]
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

    lines.extend(["", "## Scenario lists", "", f"- **Implement:** {', '.join(implement_names) or '(none)'}", f"- **Chat:** {', '.join(chat_names) or '(none)'}", ""])
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
) -> tuple[Path, Path, Path]:
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
        ),
        encoding="utf-8",
    )

    payload = {
        "suite": suite_name,
        "description": suite_desc,
        "hub": hub_url,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "implement_scenarios": implement_names,
        "chat_scenarios": chat_names,
        "results": [r.to_dict() for r in results],
    }
    json_path.write_text(json.dumps(payload, indent=2), encoding="utf-8")

    tsv_lines = ["model_tag\tscenario_kind\tscenario\tpassed\tduration_s\tdetail"]
    for r in results:
        for s in r.scenarios:
            tsv_lines.append(
                f"{r.model_tag}\t{s.kind}\t{s.name}\t{int(s.passed)}\t{s.duration_s:.1f}\t{s.detail.replace(chr(9), ' ')}"
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
        if kind and name:
            out[f"{kind}/{name}"] = bool(s.get("passed"))
    return out


def _pass_rates(result: dict[str, Any]) -> tuple[float, float, float]:
    scenarios = [s for s in (result.get("scenarios") or []) if isinstance(s, dict)]
    impl = [s for s in scenarios if s.get("kind") == "implement"]
    chat = [s for s in scenarios if s.get("kind") == "chat"]
    impl_rate = sum(1 for s in impl if s.get("passed")) / len(impl) if impl else 0.0
    chat_rate = sum(1 for s in chat if s.get("passed")) / len(chat) if chat else 0.0
    overall = sum(1 for s in scenarios if s.get("passed")) / len(scenarios) if scenarios else 0.0
    return impl_rate, chat_rate, overall


def latest_quick_run(catalog: dict[str, Any]) -> dict[str, Any] | None:
    runs = [r for r in (catalog.get("runs") or []) if isinstance(r, dict)]
    quick = [r for r in runs if str(r.get("suite") or "").strip() == "quick"]
    if not quick:
        return None
    quick.sort(key=lambda r: r.get("generated_at") or "", reverse=True)
    return quick[0]


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

    run = latest_quick_run(catalog) if suite == "quick" else None
    if run is None:
        runs = [r for r in (catalog.get("runs") or []) if isinstance(r, dict) and r.get("suite") == suite]
        runs.sort(key=lambda r: r.get("generated_at") or "", reverse=True)
        run = runs[0] if runs else None
    if run is None:
        raise ValueError(f"no benchmark run found for suite {suite!r}")

    active = [
        r
        for r in (run.get("results") or [])
        if isinstance(r, dict)
        and not r.get("skipped")
        and str(r.get("model_tag") or "").strip() in roster_tags
    ]

    def rank_tags(*, kind: str | None = None, params_max: float | None = None, params_min: float | None = None) -> list[str]:
        candidates = active
        if params_max is not None or params_min is not None:
            filtered: list[dict[str, Any]] = []
            for r in candidates:
                tag = str(r.get("model_tag") or "").strip()
                params = model_params_b(roster_map.get(tag, {"tag": tag}))
                if params is None:
                    continue
                if params_max is not None and params > params_max:
                    continue
                if params_min is not None and params <= params_min:
                    continue
                filtered.append(r)
            candidates = filtered
        ranked = sorted(candidates, key=lambda r: _model_rank_key(r, kind=kind), reverse=True)
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
        "task_classes": {
            "implement": rank_tags(kind="implement"),
            "chat": rank_tags(kind="chat"),
            "collab_light": rank_tags(params_max=9.0),
            "utility": rank_tags(params_max=9.0),
            "ask_mode": ask_mode_rank(),
            "implement_heavy": rank_tags(kind="implement", params_min=14.0),
        },
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
