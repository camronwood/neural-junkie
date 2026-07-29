"""Release-prep layer gates — test and fix one layer at a time before the full gate.

See docs/TEST_PORTFOLIO.md for tiers (climb / soak / quarantine).
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path

from lib.release_prep_failures import (
    ARTIFACT_BULLET_RE,
    OVERALL_FAIL_RE,
    PHASE_ROW_RE,
    ParsedFailure,
    ReleasePrepFailureReport,
    _extract_scenarios_from_text,
    _prioritize_rerun_cmds,
    build_agent_prompt,
)

# Tier A — daily / pre-tag / layer-climb default
CLIMB_ORDER: tuple[str, ...] = (
    "ci",
    "implement",
    "collab-core",
    "chat",
)

# Tier B — overnight soak (explicit layer-gate; not default climb)
SOAK_ORDER: tuple[str, ...] = (
    "chat-full",
    "collab",
    "collab-full",
    "parity",
)

# Tier C — invokable but not climb/soak defaults
QUARANTINE_ORDER: tuple[str, ...] = (
    "bundle",
    "user-flows",
)

# Default list + climb path (excludes quarantine)
LAYER_ORDER: tuple[str, ...] = CLIMB_ORDER + SOAK_ORDER

# Non-optional implement gate size after twin collapse (docs/TEST_PORTFOLIO.md)
IMPLEMENT_GATE_MIN_PASS = 14


@dataclass(frozen=True)
class LayerStage:
    name: str
    cmd: list[str]


@dataclass(frozen=True)
class LayerSpec:
    name: str
    description: str
    requires_hub: bool
    stages: tuple[LayerStage, ...]
    next_layer: str = ""
    est_minutes: int = 30
    tier: str = "climb"  # climb | soak | quarantine


def _hub_url_placeholder() -> str:
    return "{hub_url}"


LAYERS: dict[str, LayerSpec] = {
    "ci": LayerSpec(
        name="ci",
        description="CI smoke — vet, Go, desktop tsc, contracts (no live hub)",
        requires_hub=False,
        est_minutes=5,
        tier="climb",
        stages=(
            LayerStage("test-all", ["make", "test-all"]),
            LayerStage("test-conversation-contract", ["make", "test-conversation-contract"]),
        ),
        next_layer="implement",
    ),
    "implement": LayerSpec(
        name="implement",
        description="Implementation session scenarios (non-optional file gates; optional soak twins skipped)",
        requires_hub=True,
        est_minutes=15,
        tier="climb",
        stages=(
            LayerStage(
                "implement-scenarios",
                ["python3", "scripts/implement-scenarios.py", "--all", "--hub", _hub_url_placeholder()],
            ),
        ),
        next_layer="collab-core",
    ),
    "collab-core": LayerSpec(
        name="collab-core",
        description="Collab participation + planning core (~8 scenarios)",
        requires_hub=True,
        est_minutes=30,
        tier="climb",
        stages=(
            LayerStage("collab-scenarios-core", ["make", "collab-scenarios-core"]),
        ),
        next_layer="chat",
    ),
    "chat": LayerSpec(
        name="chat",
        description="Chat canary + conversation regression (Tier A)",
        requires_hub=True,
        est_minutes=30,
        tier="climb",
        stages=(
            LayerStage(
                "chat-scenarios-canary",
                ["python3", "scripts/chat-scenarios.py", "--all", "--tag", "canary"],
            ),
            LayerStage(
                "conversation-scenarios-regression",
                ["python3", "scripts/conversation-scenarios-regression.py", "--chat-only"],
            ),
        ),
        next_layer="",
    ),
    "chat-full": LayerSpec(
        name="chat-full",
        description="Full chat regression tag (Tier B soak)",
        requires_hub=True,
        est_minutes=120,
        tier="soak",
        stages=(
            LayerStage(
                "chat-scenarios-regression",
                ["python3", "scripts/chat-scenarios.py", "--all", "--tag", "regression"],
            ),
            LayerStage(
                "conversation-scenarios-regression",
                ["python3", "scripts/conversation-scenarios-regression.py", "--chat-only"],
            ),
        ),
        next_layer="collab",
    ),
    "collab": LayerSpec(
        name="collab",
        description="Collab edge regression (thinned website/findings twins)",
        requires_hub=True,
        est_minutes=75,
        tier="soak",
        stages=(
            LayerStage("collab-scenario-regression", ["make", "collab-scenario-regression"]),
        ),
        next_layer="collab-full",
    ),
    "collab-full": LayerSpec(
        name="collab-full",
        description="Full collab-scenarios sweep (24 scenarios)",
        requires_hub=True,
        est_minutes=120,
        tier="soak",
        stages=(
            LayerStage("collab-scenarios-all", ["make", "collab-scenarios-all"]),
        ),
        next_layer="parity",
    ),
    "parity": LayerSpec(
        name="parity",
        description="Implement stability — 3× sweep with hub restart (not scenarios/parity/)",
        requires_hub=True,
        est_minutes=45,
        tier="soak",
        stages=(
            LayerStage(
                "test-parity-stable-restart",
                [
                    "python3",
                    "scripts/implement-scenarios-stable.py",
                    "--runs",
                    "3",
                    "--min-pass",
                    str(IMPLEMENT_GATE_MIN_PASS),
                    "--restart-between",
                    "--hub",
                    _hub_url_placeholder(),
                ],
            ),
        ),
        next_layer="",
    ),
    "bundle": LayerSpec(
        name="bundle",
        description="Quarantine — overlaps implement+chat; prefer climb layers",
        requires_hub=True,
        est_minutes=90,
        tier="quarantine",
        stages=(
            LayerStage(
                "regression-bundle",
                ["python3", "scripts/regression-bundle.py", "--hub", _hub_url_placeholder()],
            ),
        ),
        next_layer="",
    ),
    "user-flows": LayerSpec(
        name="user-flows",
        description="Quarantine — product journeys until 2 consecutive green overnight runs",
        requires_hub=True,
        est_minutes=240,
        tier="quarantine",
        stages=(
            LayerStage(
                "user-flow-scenarios",
                ["python3", "scripts/user-flow-scenarios.py", "--all"],
            ),
        ),
        next_layer="",
    ),
}


def list_layers(*, include_quarantine: bool = False) -> list[LayerSpec]:
    order = LAYER_ORDER + (QUARANTINE_ORDER if include_quarantine else ())
    return [LAYERS[name] for name in order if name in LAYERS]


def list_climb_layers() -> list[LayerSpec]:
    return [LAYERS[name] for name in CLIMB_ORDER if name in LAYERS]


def get_layer(name: str) -> LayerSpec:
    key = name.strip().lower()
    if key not in LAYERS:
        known = ", ".join(sorted(LAYERS))
        raise ValueError(f"unknown layer {name!r} — choose from: {known}")
    return LAYERS[key]


def resolve_stage_cmd(cmd: list[str], *, hub_url: str) -> list[str]:
    out: list[str] = []
    for part in cmd:
        if part == _hub_url_placeholder():
            out.append(hub_url.rstrip("/"))
        else:
            out.append(part)
    return out


def layer_report_paths(testing_dir: Path, layer: str, stamp: str) -> tuple[Path, Path]:
    """Return (summary.md, full.log) paths for a layer gate run."""
    base = f"layer-gate-{layer}-{stamp}"
    return testing_dir / f"{base}.md", testing_dir / f"{base}.log"


def parse_layer_gate_report(summary_path: Path, *, hub_url: str = "http://127.0.0.1:18765") -> ReleasePrepFailureReport:
    """Parse layer-gate-*.md and companion log into a failure report."""
    text = summary_path.read_text(encoding="utf-8", errors="replace") if summary_path.is_file() else ""
    report = ReleasePrepFailureReport(
        summary_path=summary_path,
        overall_fail=bool(OVERALL_FAIL_RE.search(text)),
    )
    report.failed_phases = [name for name, status in PHASE_ROW_RE.findall(text) if status == "FAIL"]

    log_path: Path | None = None
    for match in ARTIFACT_BULLET_RE.finditer(text):
        p = Path(match.group(1))
        if p.is_file():
            report.child_artifacts.append(p)
            if p.suffix == ".log":
                log_path = p

    layer_match = re.search(r"^layer=(\S+)", text, re.MULTILINE)
    layer_name = layer_match.group(1) if layer_match else ""

    all_failures = []
    for stage in report.failed_phases:
        block = re.search(
            rf"### {re.escape(stage)} \(exit \d+\)\s*\n\s*```text\s*\n(.*?)```",
            text,
            re.DOTALL,
        )
        tail = block.group(1) if block else ""
        all_failures.extend(
            _extract_scenarios_from_text(tail, hub_url, default_stage=stage)
        )

    if log_path and log_path.is_file():
        log_text = log_path.read_text(encoding="utf-8", errors="replace")
        default_stage = report.failed_phases[0] if report.failed_phases else ""
        all_failures.extend(
            _extract_scenarios_from_text(log_text, hub_url, default_stage=default_stage)
        )

    deduped: dict[str, ParsedFailure] = {}
    for f in all_failures:
        existing = deduped.get(f.name)
        if existing is None or (f.rerun_cmd and not existing.rerun_cmd):
            deduped[f.name] = f
    report.failures = list(deduped.values())

    if not report.failures and report.overall_fail and log_path and log_path.is_file():
        default_stage = report.failed_phases[0] if report.failed_phases else ""
        report.failures = _extract_scenarios_from_text(
            log_path.read_text(encoding="utf-8", errors="replace"),
            hub_url,
            default_stage=default_stage,
        )

    report.rerun_cmds = _prioritize_rerun_cmds(report.failures)
    if layer_name:
        report.summary_path = summary_path
    return report


def build_layer_agent_prompt(report: ReleasePrepFailureReport, *, layer: str) -> str:
    """Layer-scoped agent brief."""
    spec = get_layer(layer)
    base = build_agent_prompt(report)
    header = (
        f"You are fixing failures from Neural Junkie **layer gate: {layer}**.\n"
        f"Layer goal: {spec.description}\n"
        f"Do not weaken assertions. Fix product/hub/agent behavior first (docs/TESTING.md).\n"
        f"After edits, run the targeted verification commands in this brief.\n"
        "\n---\n\n"
    )
    return header + base.replace(
        "You are fixing failures from a Neural Junkie release-prep test run.\n",
        "",
        1,
    )
