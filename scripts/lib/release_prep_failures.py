"""Parse release-prep / test-everything artifacts and plan targeted reruns."""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from enum import Enum
from pathlib import Path

PHASE_ROW_RE = re.compile(r"^\|\s*`([^`]+)`\s*\|\s*(FAIL|OK|SKIPPED)", re.MULTILINE)
STAGE_ROW_RE = PHASE_ROW_RE
ARTIFACT_BULLET_RE = re.compile(r"^- `([^`]+)`", re.MULTILINE)
OVERALL_FAIL_RE = re.compile(r"Overall:\s+\*\*FAIL\*\*", re.IGNORECASE)
SCENARIO_START_RE = re.compile(r"^=== scenario: (\S+) ===", re.MULTILINE)
SCENARIO_FAIL_RE = re.compile(r"^=== FAIL: (\S+)(?: \(solo leg\))? ===", re.MULTILINE)
STAGE_INVOCATION_RE = re.compile(
    r"^>>> (?:\[(?P<bracket>[^\]]+)\]|python3 scripts/(?P<script>[\w-]+)\.py(?: .*)?)$"
)
CHANNEL_COLLAB_RE = re.compile(r"channel=collab-scenarios\b")
CHANNEL_CHAT_RE = re.compile(r"channel=(?:chat-scenarios|dm-[\w-]+)\b|agent=\w+")
CHANNEL_IMPLEMENT_RE = re.compile(r"channel=implement-scenarios\b")
COLLAB_AGENTS_RE = re.compile(r"agents=@")

FLAKE_MARKERS = (
    "could not complete this turn",
    "timeout waiting for phase",
    "timeout waiting for",
    "send failed (401)",
    "send failed (0)",
    "Sorry, I encountered an error",
    "wait_discussion attempt",
    "RemoteDisconnected",
    "quota",
    "429",
    "rate limit",
)

INFRA_MARKERS = (
    "hub not healthy",
    "hub unhealthy",
    "hub recovery exhausted",
    "hub restart failed",
    "send failed (401)",
    "panic:",
    "OOM",
    "out of memory",
)

CI_STAGES = {
    "test-all": ["make", "test-all"],
    "test-conversation-contract": ["make", "test-conversation-contract"],
    "test-collab-plan": ["make", "test-collab-plan"],
    "test-scenario-assert": ["make", "test-scenario-assert"],
    "collab-smoke": ["make", "collab-smoke"],
    "learning-lora-smoke": ["make", "learning-lora-smoke"],
}


class FailureKind(str, Enum):
    CODE = "code"
    FLAKE = "flake"
    INFRA = "infra"
    MODEL_BENCHMARK = "model_benchmark"
    UNKNOWN = "unknown"


@dataclass
class ParsedFailure:
    name: str
    source: str
    kind: FailureKind
    detail: str = ""
    rerun_cmd: list[str] = field(default_factory=list)


@dataclass
class ReleasePrepFailureReport:
    summary_path: Path
    overall_fail: bool
    failed_phases: list[str] = field(default_factory=list)
    child_artifacts: list[Path] = field(default_factory=list)
    failures: list[ParsedFailure] = field(default_factory=list)
    rerun_cmds: list[list[str]] = field(default_factory=list)

    @property
    def agent_candidates(self) -> list[ParsedFailure]:
        return [f for f in self.failures if f.kind in (FailureKind.CODE, FailureKind.UNKNOWN)]

    @property
    def retry_only(self) -> list[ParsedFailure]:
        return [f for f in self.failures if f.kind in (FailureKind.FLAKE, FailureKind.INFRA)]


def _read(path: Path) -> str:
    if not path.is_file():
        return ""
    return path.read_text(encoding="utf-8", errors="replace")


def _classify(name: str, detail: str, *, phase: str = "") -> FailureKind:
    if phase == "model-benchmark" or name.startswith("model-benchmark"):
        return FailureKind.MODEL_BENCHMARK
    lower = detail.lower()
    # Infra / flake markers win even for live scenario names so fix-loop can
    # re-boot/rerun instead of thrashing product code on timeouts and hub outages.
    if any(m.lower() in lower for m in INFRA_MARKERS):
        return FailureKind.INFRA
    if any(m.lower() in lower for m in FLAKE_MARKERS):
        return FailureKind.FLAKE
    # Live scenario harness assertion failures default to product regressions.
    if name.startswith("implement:") or name.startswith("collab:") or name.startswith("chat:"):
        return FailureKind.CODE
    if name in CI_STAGES or name.startswith("test-"):
        return FailureKind.CODE
    return FailureKind.UNKNOWN


def _scenario_rerun(kind: str, scenario: str, hub_url: str) -> list[str]:
    py = "python3"
    if kind == "implement":
        return [
            py,
            "scripts/implement-scenarios.py",
            "--scenario",
            scenario,
            "--hub",
            hub_url,
        ]
    if kind == "collab":
        return [py, "scripts/collab-scenarios.py", "--scenario", scenario]
    if kind == "chat":
        return [py, "scripts/chat-scenarios.py", "--scenario", scenario]
    return []


SCRIPT_HARNESS = {
    "implement-scenarios": "implement",
    "collab-scenarios": "collab",
    "chat-scenarios": "chat",
    "conversation-scenarios-regression": "chat",
}

STAGE_HARNESS = {
    "implement-scenarios": "implement",
    "collab-scenarios-all": "collab",
    "collab-scenarios-core": "collab",
    "collab-scenario-regression": "collab",
    "chat-scenarios-regression": "chat",
    "conversation-scenarios-regression": "chat",
}


def _harness_from_block(block: str, *, stage_hint: str | None = None) -> str:
    if CHANNEL_COLLAB_RE.search(block) or COLLAB_AGENTS_RE.search(block):
        return "collab"
    if CHANNEL_IMPLEMENT_RE.search(block):
        return "implement"
    if CHANNEL_CHAT_RE.search(block):
        return "chat"
    if stage_hint and stage_hint in STAGE_HARNESS:
        return STAGE_HARNESS[stage_hint]
    head = block[:1500].lower()
    if "scripts/collab-scenarios" in head or "collab-scenarios.py" in head:
        return "collab"
    if "scripts/chat-scenarios" in head or "chat-scenarios.py" in head:
        return "chat"
    if "scripts/implement-scenarios" in head or "implement-scenarios.py" in head:
        return "implement"
    if "collaboration_discussion" in head or "started collab" in head:
        return "collab"
    return "unknown"


def _extract_scenarios_from_text(
    text: str,
    hub_url: str,
    *,
    default_stage: str | None = None,
) -> list[ParsedFailure]:
    """Walk log/tail text; track stage invocations and parse failed scenario blocks."""
    out: list[ParsedFailure] = []
    seen: set[str] = set()
    current_stage = default_stage
    current_harness = STAGE_HARNESS.get(default_stage or "", "")

    lines = text.splitlines()
    i = 0
    while i < len(lines):
        line = lines[i]
        inv = STAGE_INVOCATION_RE.match(line.strip())
        if inv:
            if inv.group("bracket"):
                current_stage = inv.group("bracket")
            elif inv.group("script"):
                current_stage = inv.group("script")
            current_harness = STAGE_HARNESS.get(current_stage, SCRIPT_HARNESS.get(current_stage, ""))

        start = SCENARIO_START_RE.match(line.strip())
        if start:
            scenario = start.group(1)
            block_lines = [line]
            j = i + 1
            failed = False
            fail_detail_end = i
            while j < len(lines):
                block_lines.append(lines[j])
                if SCENARIO_FAIL_RE.match(lines[j].strip()) and scenario in lines[j]:
                    failed = True
                    fail_detail_end = j
                    break
                if SCENARIO_START_RE.match(lines[j].strip()) or STAGE_INVOCATION_RE.match(lines[j].strip()):
                    break
                j += 1
            if failed:
                block = "\n".join(block_lines)
                prefix = _harness_from_block(block, stage_hint=current_stage)
                if prefix == "unknown" and current_harness:
                    prefix = current_harness
                if prefix != "unknown":
                    key = f"{prefix}:{scenario}"
                    if key not in seen:
                        seen.add(key)
                        detail = "\n".join(block_lines[: fail_detail_end - i + 1])
                        kind = _classify(key, detail)
                        out.append(
                            ParsedFailure(
                                name=key,
                                source=prefix,
                                kind=kind,
                                detail=detail[-2000:],
                                rerun_cmd=_scenario_rerun(prefix, scenario, hub_url),
                            )
                        )
            i = j
            continue
        i += 1
    out.extend(
        _extract_orphan_scenario_failures(
            text,
            hub_url,
            default_stage=default_stage,
            seen=seen,
        )
    )
    return out


def _extract_orphan_scenario_failures(
    text: str,
    hub_url: str,
    *,
    default_stage: str | None = None,
    seen: set[str] | None = None,
) -> list[ParsedFailure]:
    """Parse trailing ``=== FAIL:`` blocks (collab-scenarios batches these after the run loop)."""
    if seen is None:
        seen = set()
    harness = STAGE_HARNESS.get(default_stage or "", "") or SCRIPT_HARNESS.get(default_stage or "", "")
    if not harness:
        if CHANNEL_COLLAB_RE.search(text) or "collab-scenarios" in text:
            harness = "collab"
        elif CHANNEL_IMPLEMENT_RE.search(text):
            harness = "implement"
        elif CHANNEL_CHAT_RE.search(text) or "chat-scenarios" in text:
            harness = "chat"
    if not harness or harness == "unknown":
        return []

    out: list[ParsedFailure] = []
    matches = list(SCENARIO_FAIL_RE.finditer(text))
    for idx, match in enumerate(matches):
        scenario = match.group(1)
        start = match.start()
        end = matches[idx + 1].start() if idx + 1 < len(matches) else len(text)
        detail = text[start:end].strip()
        key = f"{harness}:{scenario}"
        if key in seen:
            continue
        seen.add(key)
        kind = _classify(key, detail)
        out.append(
            ParsedFailure(
                name=key,
                source=harness,
                kind=kind,
                detail=detail[-2000:],
                rerun_cmd=_scenario_rerun(harness, scenario, hub_url),
            )
        )
    return out


def _extract_scenarios_from_log(text: str, hub_url: str) -> list[ParsedFailure]:
    return _extract_scenarios_from_text(text, hub_url)


def _extract_scenarios_from_stage_tails(text: str, hub_url: str) -> list[ParsedFailure]:
    """Parse scenario failures embedded in test-everything.md failure tail sections."""
    out: list[ParsedFailure] = []
    for match in re.finditer(
        r"### ([\w-]+) \(exit \d+\)\s*\n\s*```text\s*\n(.*?)```",
        text,
        re.DOTALL,
    ):
        stage = match.group(1)
        if stage not in STAGE_HARNESS:
            continue
        out.extend(
            _extract_scenarios_from_text(
                match.group(2),
                hub_url,
                default_stage=stage,
            )
        )
    return out


def _parse_test_everything(path: Path, hub_url: str) -> list[ParsedFailure]:
    text = _read(path)
    if not text:
        return []
    failures: list[ParsedFailure] = []
    for stage, status in STAGE_ROW_RE.findall(text):
        if status != "FAIL":
            continue
        tail = ""
        block = re.search(
            rf"### {re.escape(stage)} \(exit \d+\)\s*\n\s*```text\s*\n(.*?)```",
            text,
            re.DOTALL,
        )
        if block:
            tail = block.group(1)
        kind = _classify(stage, tail)
        cmd = list(CI_STAGES.get(stage, []))
        if not cmd and stage == "collab-preflight":
            cmd = ["python3", "scripts/collab-preflight.py", "--hub", hub_url]
        elif not cmd and stage in ("implement-scenarios",):
            cmd = ["python3", "scripts/implement-scenarios.py", "--all", "--hub", hub_url]
        elif not cmd and stage in ("chat-scenarios-regression",):
            cmd = [
                "python3",
                "scripts/chat-scenarios.py",
                "--all",
                "--tag",
                "regression",
            ]
        elif not cmd and stage in ("conversation-scenarios-regression",):
            cmd = ["python3", "scripts/conversation-scenarios-regression.py"]
        elif not cmd and stage in ("collab-scenario-regression", "collab-scenarios-all", "collab-scenarios-core"):
            cmd = []  # filled from scenario-level parse below
        failures.append(
            ParsedFailure(
                name=stage,
                source="test-everything",
                kind=kind,
                detail=tail[-2000:],
                rerun_cmd=cmd,
            )
        )

    log_path = None
    for line in text.splitlines():
        if line.startswith("- Full log:"):
            raw = line.split("`")
            if len(raw) >= 2:
                log_path = Path(raw[1])
            break
    if log_path and log_path.is_file():
        failures.extend(_extract_scenarios_from_log(_read(log_path), hub_url))
    failures.extend(_extract_scenarios_from_stage_tails(text, hub_url))
    return failures


def _parse_parity_stable(path: Path, hub_url: str) -> list[ParsedFailure]:
    text = _read(path)
    if not text:
        return []
    failures = _extract_scenarios_from_log(text, hub_url)
    if not failures and "OVERALL: FAIL" in text:
        failures.append(
            ParsedFailure(
                name="test-parity-stable-restart",
                source="parity-stable",
                kind=FailureKind.UNKNOWN,
                detail=text[-2000:],
                rerun_cmd=[
                    "python3",
                    "scripts/implement-scenarios-stable.py",
                    "--runs",
                    "1",
                    "--min-pass",
                    "20",
                    "--restart-between",
                    "--hub",
                    hub_url,
                ],
            )
        )
    return failures


def parse_release_prep_report(summary_path: Path, *, hub_url: str = "http://127.0.0.1:18765") -> ReleasePrepFailureReport:
    """Parse a release-prep-*.md summary and child artifacts."""
    text = _read(summary_path)
    report = ReleasePrepFailureReport(
        summary_path=summary_path,
        overall_fail=bool(OVERALL_FAIL_RE.search(text)),
    )
    report.failed_phases = [name for name, status in PHASE_ROW_RE.findall(text) if status == "FAIL"]

    for match in ARTIFACT_BULLET_RE.finditer(text):
        p = Path(match.group(1))
        if p.is_file():
            report.child_artifacts.append(p)

    all_failures: list[ParsedFailure] = []
    for phase in report.failed_phases:
        block = re.search(
            rf"### {re.escape(phase)} \(exit \d+\)\s*\n\s*```text\s*\n(.*?)```",
            text,
            re.DOTALL,
        )
        tail = block.group(1) if block else ""
        kind = _classify(phase, tail, phase=phase)
        cmd: list[str] = []
        if phase == "test-everything":
            cmd = ["python3", "scripts/test-everything.py", "--full", "--continue-on-fail", "--hub", hub_url]
        elif phase == "test-parity-stable-restart":
            cmd = [
                "python3",
                "scripts/implement-scenarios-stable.py",
                "--runs",
                "1",
                "--min-pass",
                "20",
                "--restart-between",
                "--hub",
                hub_url,
            ]
        elif phase == "model-benchmark":
            cmd = [
                "python3",
                "scripts/model-benchmark-suite.py",
                "--hub",
                hub_url,
                "--suite",
                "quick",
            ]
        all_failures.append(
            ParsedFailure(
                name=phase,
                source="release-prep",
                kind=kind,
                detail=tail[-2000:],
                rerun_cmd=cmd,
            )
        )

    for artifact in report.child_artifacts:
        name = artifact.name
        if name.startswith("test-everything-") and name.endswith(".md"):
            all_failures.extend(_parse_test_everything(artifact, hub_url))
        elif name.startswith("parity-stable-restart-") and name.endswith(".log"):
            all_failures.extend(_parse_parity_stable(artifact, hub_url))

    deduped: dict[str, ParsedFailure] = {}
    for f in all_failures:
        existing = deduped.get(f.name)
        if existing is None or (f.rerun_cmd and not existing.rerun_cmd):
            deduped[f.name] = f
    report.failures = list(deduped.values())
    report.rerun_cmds = _prioritize_rerun_cmds(report.failures)
    return report


def _prioritize_rerun_cmds(failures: list[ParsedFailure]) -> list[list[str]]:
    """Build a focused verification list: CI first, then failed scenarios, not full sweeps."""
    seen: set[tuple[str, ...]] = set()
    ordered: list[list[str]] = []

    def add(cmd: list[str]) -> None:
        if not cmd:
            return
        key = tuple(cmd)
        if key in seen:
            return
        seen.add(key)
        ordered.append(cmd)

    failed_names = {f.name for f in failures}
    for stage in (
        "test-all",
        "test-conversation-contract",
        "test-collab-plan",
        "test-scenario-assert",
        "collab-smoke",
        "learning-lora-smoke",
    ):
        if stage in failed_names:
            add(CI_STAGES.get(stage, []))

    scenario_failures = [f for f in failures if ":" in f.name and f.rerun_cmd]
    code_first = sorted(scenario_failures, key=lambda f: (f.kind != FailureKind.CODE, f.name))
    for f in code_first:
        add(f.rerun_cmd)

    for f in failures:
        if f.name in CI_STAGES or ":" in f.name:
            continue
        if f.name == "model-benchmark":
            continue
        if f.rerun_cmd and f.name not in ("test-everything", "implement-scenarios", "collab-scenarios-all"):
            add(f.rerun_cmd)

    if not ordered:
        for f in failures:
            add(f.rerun_cmd)
    return ordered


def build_agent_prompt(report: ReleasePrepFailureReport, *, max_items: int = 24) -> str:
    """Build a Cursor agent prompt from a parsed failure report."""
    candidates = report.agent_candidates
    code_items = [f for f in candidates if f.kind == FailureKind.CODE]
    other_items = [f for f in candidates if f.kind != FailureKind.CODE]
    selected = (code_items + other_items)[:max_items]
    omitted = max(0, len(candidates) - len(selected))

    lines = [
        "You are fixing failures from a Neural Junkie release-prep test run.",
        "",
        "Rules (mandatory):",
        "- Triage product/hub/agent behavior first, harness second (docs/TESTING.md).",
        "- Do NOT weaken test assertions or scenario contracts to greenwash flakes.",
        "- Prefer minimal, focused fixes in the neural-junkie repo.",
        "- After edits, run the targeted verification commands listed below.",
        "- Summarize what you changed and which commands you ran.",
        "",
        f"Release prep summary: {report.summary_path}",
        f"Failed phases: {', '.join(report.failed_phases) or '(none parsed)'}",
        "",
        "## Failures to address",
        "",
    ]
    for f in selected:
        lines.append(f"### {f.name} [{f.kind.value}]")
        if f.detail.strip():
            lines.append("```text")
            lines.append(f.detail.strip()[-4000:])
            lines.append("```")
        lines.append("")

    if omitted:
        lines.append(f"({omitted} more agent-candidate failures omitted — read child artifacts.)")
        lines.append("")

    if report.retry_only:
        lines.extend(
            [
                "## Likely flakes/infra (retry only — do not hack code to pass)",
                "",
                ", ".join(f.name for f in report.retry_only),
                "",
            ]
        )

    if report.child_artifacts:
        lines.append("## Child artifacts (read for full context)")
        for p in report.child_artifacts:
            lines.append(f"- {p}")
        lines.append("")

    if report.rerun_cmds:
        lines.append("## Targeted verification (run after your fixes)")
        for cmd in report.rerun_cmds:
            lines.append(f"- {' '.join(cmd)}")
        lines.append("")

    return "\n".join(lines)
