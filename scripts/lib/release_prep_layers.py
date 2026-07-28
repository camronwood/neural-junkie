"""Release-prep layer gates — test and fix one layer at a time before the full gate."""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import Path

from lib.release_prep_failures import (
    ARTIFACT_BULLET_RE,
    OVERALL_FAIL_RE,
    PHASE_ROW_RE,
    ParsedFailure,
    ReleasePrepFailureReport,
    _extract_scenarios_from_log,
    _extract_scenarios_from_text,
    _prioritize_rerun_cmds,
    build_agent_prompt,
)

LAYER_ORDER: tuple[str, ...] = (
    "ci",
    "implement",
    "chat",
    "collab",
    "collab-core",
    "collab-full",
    "bundle",
    "user-flows",
    "parity",
)


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


def _hub_url_placeholder() -> str:
    return "{hub_url}"


LAYERS: dict[str, LayerSpec] = {
    "ci": LayerSpec(
        name="ci",
        description="CI smoke — vet, Go, desktop tsc, contracts (no live hub)",
        requires_hub=False,
        est_minutes=15,
        stages=(
            LayerStage("test-all", ["make", "test-all"]),
            LayerStage("test-conversation-contract", ["make", "test-conversation-contract"]),
        ),
        next_layer="implement",
    ),
    "implement": LayerSpec(
        name="implement",
        description="Implementation session scenarios (target 20/20 PASS)",
        requires_hub=True,
        est_minutes=45,
        stages=(
            LayerStage(
                "implement-scenarios",
                ["python3", "scripts/implement-scenarios.py", "--all", "--hub", _hub_url_placeholder()],
            ),
        ),
        next_layer="chat",
    ),
    "chat": LayerSpec(
        name="chat",
        description="Chat + conversation regression (closure, DM, workspace)",
        requires_hub=True,
        # Live chat has many multi-turn DMs + flake retries; 90m still timed out at 5400s.
        est_minutes=120,
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
        description="Collab edge-case regression (plan parser, execution guards, full-completion paths, ~13 scenarios)",
        requires_hub=True,
        est_minutes=105,
        stages=(
            LayerStage("collab-scenario-regression", ["make", "collab-scenario-regression"]),
        ),
        next_layer="collab-core",
    ),
    "collab-core": LayerSpec(
        name="collab-core",
        description="Collab participation + planning core (~8 scenarios, ~45–90m; hub restart between)",
        requires_hub=True,
        est_minutes=75,
        stages=(
            LayerStage("collab-scenarios-core", ["make", "collab-scenarios-core"]),
        ),
        next_layer="collab-full",
    ),
    "collab-full": LayerSpec(
        name="collab-full",
        description="Full collab-scenarios sweep (24 scenarios, ~2–4h)",
        requires_hub=True,
        est_minutes=120,
        stages=(
            LayerStage("collab-scenarios-all", ["make", "collab-scenarios-all"]),
        ),
        next_layer="bundle",
    ),
    "bundle": LayerSpec(
        name="bundle",
        description="Regression bundle — implement + chat + conversation in one run",
        requires_hub=True,
        est_minutes=90,
        stages=(
            LayerStage(
                "regression-bundle",
                ["python3", "scripts/regression-bundle.py", "--hub", _hub_url_placeholder()],
            ),
        ),
        next_layer="user-flows",
    ),
    "user-flows": LayerSpec(
        name="user-flows",
        description="Real-world product journeys (trip research, games, APIs, websites, boot fix)",
        requires_hub=True,
        est_minutes=240,
        stages=(
            LayerStage(
                "user-flow-scenarios",
                ["python3", "scripts/user-flow-scenarios.py", "--all"],
            ),
        ),
        next_layer="parity",
    ),
    "parity": LayerSpec(
        name="parity",
        description="Implement stability — 3× sweep with hub restart (hardest layer)",
        requires_hub=True,
        est_minutes=180,
        stages=(
            LayerStage(
                "test-parity-stable-restart",
                [
                    "python3",
                    "scripts/implement-scenarios-stable.py",
                    "--runs",
                    "3",
                    "--min-pass",
                    "20",
                    "--restart-between",
                    "--hub",
                    _hub_url_placeholder(),
                ],
            ),
        ),
        next_layer="",
    ),
}


def list_layers() -> list[LayerSpec]:
    return [LAYERS[name] for name in LAYER_ORDER if name in LAYERS]


def get_layer(name: str) -> LayerSpec:
    key = name.strip().lower()
    if key not in LAYERS:
        known = ", ".join(LAYER_ORDER)
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
        report.summary_path = summary_path  # anchor for prompt
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
