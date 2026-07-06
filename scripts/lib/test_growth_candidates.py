"""Discover and score test-growth candidates for the test-growth loop."""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCENARIOS_DIR = ROOT / "scenarios"
TESTING_DIR = ROOT / "docs" / "testing"
INTERNAL_AGENT = ROOT / "internal" / "agent"
DESKTOP_SRC = ROOT / "desktop" / "src"

# Layer A routing tests that should pair with live chat scenarios when possible.
LAYER_A_ROUTING_TESTS = INTERNAL_AGENT / "chat_quality_router_test.go"
LAYER_A_COVERAGE_TESTS = INTERNAL_AGENT / "chat_quality_coverage_test.go"

SCENARIO_FAIL_RE = re.compile(r"^=== FAIL: (\S+)(?: \(solo leg\))? ===", re.MULTILINE)
LAYER_GATE_FAIL_RE = re.compile(r"^layer=(\S+)", re.MULTILINE)

# Known edge-case classes from docs/TESTING.md and CHAT_SCENARIOS.md.
EDGE_CASE_KEYWORDS: tuple[tuple[str, str], ...] = (
    ("closure", "thanks-closure / already-said-closure"),
    ("echo", "dm-backend-echo-followup"),
    ("workspace", "dm-backend-workspace / public-backend-theme-workspace"),
    ("interject", "dm-backend-interject-resume"),
    ("continuation", "dm-backend-deep-continuation / dm-assistant-continue-after-closure"),
    ("topic-switch", "dm-topic-switch"),
    ("verify", "verify-failure-one-repair / go-test-failure-repair"),
    ("boot-fix", "app-wont-boot-fix-like / tauri-make-start-all-missing"),
    ("destructive", "deny-destructive-command"),
    ("plan-mode", "plan-mode-no-write"),
)


@dataclass(frozen=True)
class GrowthCandidate:
    """A scored opportunity to add or strengthen tests."""

    id: str
    kind: str  # unit | scenario | layer_a | failure_repro | contract
    title: str
    description: str
    score: float
    target_paths: tuple[str, ...] = ()
    suggested_files: tuple[str, ...] = ()
    verify_cmds: tuple[tuple[str, ...], ...] = ()
    source: str = ""
    metadata: dict = field(default_factory=dict)


def _load_json(path: Path) -> dict | None:
    try:
        with path.open(encoding="utf-8") as f:
            data = json.load(f)
        return data if isinstance(data, dict) else None
    except (OSError, json.JSONDecodeError):
        return None


def _scenario_dirs() -> list[tuple[Path, str]]:
    out: list[tuple[Path, str]] = []
    for name in ("chat", "implement", "collab", "conversation", "learning", "parity"):
        d = SCENARIOS_DIR / name
        if d.is_dir():
            out.append((d, name))
    return out


def _list_scenarios(kind: str) -> list[tuple[str, Path, dict]]:
    results: list[tuple[str, Path, dict]] = []
    for directory, prefix in _scenario_dirs():
        if kind and prefix != kind:
            continue
        for path in sorted(directory.glob("*.json")):
            scenario = _load_json(path)
            if scenario is None:
                continue
            name = (scenario.get("name") or path.stem).strip()
            results.append((name, path, scenario))
    return results


def _read_text(path: Path) -> str:
    if not path.is_file():
        return ""
    return path.read_text(encoding="utf-8", errors="replace")


def _go_packages_without_tests() -> list[GrowthCandidate]:
    """Find internal packages with source files but no *_test.go."""
    candidates: list[GrowthCandidate] = []
    internal = ROOT / "internal"
    if not internal.is_dir():
        return candidates

    for pkg_dir in sorted(internal.rglob("*")):
        if not pkg_dir.is_dir():
            continue
        if "testdata" in pkg_dir.parts:
            continue
        go_files = [p for p in pkg_dir.glob("*.go") if not p.name.endswith("_test.go")]
        test_files = list(pkg_dir.glob("*_test.go"))
        if not go_files or test_files:
            continue
        rel = pkg_dir.relative_to(ROOT).as_posix()
        pkg_import = rel.replace("/", ".")
        candidates.append(
            GrowthCandidate(
                id=f"unit:missing:{rel}",
                kind="unit",
                title=f"Add Go unit tests for {rel}",
                description=f"Package `{rel}` has {len(go_files)} source file(s) but no *_test.go.",
                score=40.0 + min(len(go_files) * 5, 25),
                target_paths=(rel,),
                suggested_files=(f"{rel}/{pkg_dir.name}_test.go",),
                verify_cmds=(("go", "test", f"./{rel}/...", "-count=1"),),
                source="coverage-gap",
            )
        )
    return candidates


def _chat_scenarios_missing_layer_a() -> list[GrowthCandidate]:
    """Live chat scenarios without a mention in Layer A routing/coverage tests."""
    router_text = _read_text(LAYER_A_ROUTING_TESTS)
    coverage_text = _read_text(LAYER_A_COVERAGE_TESTS)
    combined = router_text + coverage_text
    candidates: list[GrowthCandidate] = []

    for name, path, scenario in _list_scenarios("chat"):
        tags = scenario.get("tags") if isinstance(scenario.get("tags"), list) else []
        if "regression" not in tags:
            continue
        slug = path.stem.replace("-", "_")
        if slug in combined or name in combined or path.stem in combined:
            continue
        candidates.append(
            GrowthCandidate(
                id=f"layer_a:chat:{path.stem}",
                kind="layer_a",
                title=f"Add Layer A test for chat scenario `{name}`",
                description=(
                    f"Regression chat scenario `{name}` has no companion case in "
                    f"`internal/agent/chat_quality_router_test.go` or `chat_quality_coverage_test.go`."
                ),
                score=55.0,
                target_paths=(path.relative_to(ROOT).as_posix(),),
                suggested_files=(
                    "internal/agent/chat_quality_router_test.go",
                    "internal/agent/chat_quality_coverage_test.go",
                ),
                verify_cmds=(
                    ("go", "test", "./internal/agent/...", "-count=1", "-run", "ChatQuality"),
                ),
                source="layer-a-gap",
                metadata={"scenario": name, "scenario_kind": "chat"},
            )
        )
    return candidates


def _scenario_assertion_gaps() -> list[GrowthCandidate]:
    """Scenarios with weak assertion coverage (no none_match on regression scenarios)."""
    candidates: list[GrowthCandidate] = []

    for name, path, scenario in _list_scenarios(""):
        tags = scenario.get("tags") if isinstance(scenario.get("tags"), list) else []
        if "regression" not in tags:
            continue
        steps = scenario.get("steps") if isinstance(scenario.get("steps"), list) else []
        has_none_match = False
        for step in steps:
            if not isinstance(step, dict):
                continue
            if step.get("none_match"):
                has_none_match = True
                break
            fq = step.get("for_question")
            if isinstance(fq, dict) and fq.get("none_match"):
                has_none_match = True
                break
        expect = scenario.get("expect_deliverables")
        if isinstance(expect, list):
            for item in expect:
                if isinstance(item, dict):
                    fq = item.get("for_question")
                    if isinstance(fq, dict) and fq.get("none_match"):
                        has_none_match = True
                        break
        if has_none_match:
            continue
        rel = path.relative_to(ROOT).as_posix()
        prefix = rel.split("/")[1] if "/" in rel else "unknown"
        candidates.append(
            GrowthCandidate(
                id=f"scenario:assertion:{path.stem}",
                kind="scenario",
                title=f"Strengthen assertions on `{name}`",
                description=(
                    f"Regression scenario `{name}` has no `none_match` guardrails. "
                    "Add negative assertions for known failure modes."
                ),
                score=50.0,
                target_paths=(rel,),
                suggested_files=(rel,),
                verify_cmds=_scenario_verify_cmds(prefix, name),
                source="assertion-gap",
                metadata={"scenario": name, "scenario_kind": prefix},
            )
        )
    return candidates


def _scenario_verify_cmds(kind: str, scenario_name: str) -> tuple[tuple[str, ...], ...]:
    py = ("python3",)
    if kind == "chat":
        return (
            ("make", "test-scenario-assert"),
            (*py, "scripts/chat-scenarios.py", "--scenario", scenario_name),
        )
    if kind == "implement":
        return (
            ("make", "test-scenario-assert"),
            (*py, "scripts/implement-scenarios.py", "--scenario", scenario_name),
        )
    if kind == "collab":
        return (
            ("make", "test-scenario-assert"),
            (*py, "scripts/collab-scenarios.py", "--scenario", scenario_name),
        )
    return (("make", "test-scenario-assert"),)


def _failures_from_artifacts(limit: int = 20) -> list[GrowthCandidate]:
    """Parse recent docs/testing artifacts for failed scenarios worth reproducing."""
    if not TESTING_DIR.is_dir():
        return []
    candidates: list[GrowthCandidate] = []
    seen: set[str] = set()

    paths = sorted(
        list(TESTING_DIR.glob("layer-gate-*.log"))
        + list(TESTING_DIR.glob("layer-fix-loop-*.log"))
        + list(TESTING_DIR.glob("test-everything-*.log")),
        key=lambda p: p.stat().st_mtime,
        reverse=True,
    )[:limit]

    for log_path in paths:
        text = _read_text(log_path)
        layer_match = LAYER_GATE_FAIL_RE.search(text)
        layer = layer_match.group(1) if layer_match else ""
        for match in SCENARIO_FAIL_RE.finditer(text):
            scenario = match.group(1)
            key = f"{layer}:{scenario}"
            if key in seen:
                continue
            seen.add(key)
            kind = "implement" if "implement" in log_path.name or layer == "implement" else "chat"
            if "collab" in log_path.name or layer.startswith("collab"):
                kind = "collab"
            candidates.append(
                GrowthCandidate(
                    id=f"failure_repro:{kind}:{scenario}",
                    kind="failure_repro",
                    title=f"Convert failure `{scenario}` to stronger test",
                    description=(
                        f"Scenario `{scenario}` failed in `{log_path.name}`. "
                        "Add or tighten a test that captures this failure class."
                    ),
                    score=70.0,
                    target_paths=(f"scenarios/{kind}/{scenario}.json",),
                    suggested_files=(f"scenarios/{kind}/{scenario}.json",),
                    verify_cmds=_scenario_verify_cmds(kind, scenario),
                    source=str(log_path.relative_to(ROOT)),
                    metadata={"scenario": scenario, "scenario_kind": kind, "artifact": log_path.name},
                )
            )
    return candidates


def _edge_case_coverage_gaps() -> list[GrowthCandidate]:
    """Suggest new scenarios for documented edge-case classes not covered."""
    existing_names: set[str] = set()
    for name, _, _ in _list_scenarios(""):
        existing_names.add(name)

    candidates: list[GrowthCandidate] = []
    for keyword, examples in EDGE_CASE_KEYWORDS:
        covered = any(keyword in n for n in existing_names)
        if covered:
            continue
        candidates.append(
            GrowthCandidate(
                id=f"scenario:edge:{keyword}",
                kind="scenario",
                title=f"Add scenario for edge case: {keyword}",
                description=(
                    f"No scenario covers the `{keyword}` edge-case class "
                    f"(see docs: {examples}). Add a live scenario with strong assertions."
                ),
                score=45.0,
                suggested_files=(f"scenarios/chat/{keyword}-regression.json",),
                verify_cmds=(("make", "test-scenario-assert"),),
                source="edge-case-doc",
                metadata={"edge_case": keyword},
            )
        )
    return candidates


def discover_candidates(*, limit: int = 50) -> list[GrowthCandidate]:
    """Gather and rank all test-growth candidates."""
    all_candidates: list[GrowthCandidate] = []
    all_candidates.extend(_failures_from_artifacts())
    all_candidates.extend(_chat_scenarios_missing_layer_a())
    all_candidates.extend(_scenario_assertion_gaps())
    all_candidates.extend(_go_packages_without_tests())
    all_candidates.extend(_edge_case_coverage_gaps())

    deduped: dict[str, GrowthCandidate] = {}
    for c in all_candidates:
        existing = deduped.get(c.id)
        if existing is None or c.score > existing.score:
            deduped[c.id] = c

    ranked = sorted(deduped.values(), key=lambda c: (-c.score, c.id))
    return ranked[:limit]


def pick_candidate(
    candidates: list[GrowthCandidate],
    *,
    skip_ids: set[str] | None = None,
    kind: str | None = None,
) -> GrowthCandidate | None:
    """Return the highest-scored candidate not in skip_ids."""
    skip = skip_ids or set()
    for c in candidates:
        if c.id in skip:
            continue
        if kind and c.kind != kind:
            continue
        return c
    return None


def format_candidate_list(candidates: list[GrowthCandidate]) -> str:
    lines = ["# Test growth candidates", ""]
    if not candidates:
        lines.append("(none found)")
        return "\n".join(lines) + "\n"
    for i, c in enumerate(candidates, 1):
        lines.append(f"{i}. **[{c.kind}]** {c.title} (score={c.score:.0f})")
        lines.append(f"   - id: `{c.id}`")
        lines.append(f"   - {c.description}")
        if c.suggested_files:
            lines.append(f"   - suggested: {', '.join(c.suggested_files)}")
        lines.append("")
    return "\n".join(lines) + "\n"
