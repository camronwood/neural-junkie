"""Verification and acceptance checks for test-growth loop iterations."""

from __future__ import annotations

import subprocess
from dataclasses import dataclass, field
from pathlib import Path

from lib.test_growth_candidates import GrowthCandidate
from lib.test_growth_guardrails import GuardrailResult, check_edit_guardrails, git_changed_files

ROOT = Path(__file__).resolve().parents[2]


@dataclass
class VerifyResult:
    accepted: bool
    checks: list[tuple[str, bool, str]] = field(default_factory=list)
    verify_runs: list[tuple[list[str], int]] = field(default_factory=list)
    guardrails: GuardrailResult | None = None
    repair_handoff: bool = False
    rejection_reason: str = ""


def run_cmd(cmd: list[str], *, cwd: Path, env: dict | None = None) -> tuple[int, str]:
    merged = dict(__import__("os").environ)
    if env:
        merged.update(env)
    merged["PYTHONUNBUFFERED"] = "1"
    proc = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, env=merged)
    out = (proc.stdout or "") + (proc.stderr or "")
    return proc.returncode, out


def _default_verify_cmds(changed: list[str], candidate: GrowthCandidate | None) -> list[list[str]]:
    cmds: list[list[str]] = [["make", "test-scenario-assert"]]

    has_go_test = any(p.endswith("_test.go") for p in changed)
    has_py_test = any("_test.py" in p for p in changed)
    has_frontend = any(".test." in p or ".spec." in p for p in changed)
    has_scenario = any(p.startswith("scenarios/") and p.endswith(".json") for p in changed)

    if has_go_test:
        pkg_dirs: set[str] = set()
        for p in changed:
            if p.endswith("_test.go"):
                parts = Path(p).parts
                if len(parts) >= 2:
                    pkg_dirs.add(str(Path(*parts[:-1])))
        for pkg in sorted(pkg_dirs):
            cmds.append(["go", "test", f"./{pkg}/...", "-count=1"])
        if not pkg_dirs:
            cmds.append(["go", "test", "./internal/agent/...", "-count=1"])

    if has_py_test:
        cmds.append(
            [
                "python3",
                "-m",
                "unittest",
                "discover",
                "-s",
                "scripts/lib",
                "-p",
                "*_test.py",
            ]
        )

    if has_frontend:
        cmds.append(["bash", "-lc", "cd desktop && npm test"])

    if candidate and candidate.verify_cmds:
        for cmd in candidate.verify_cmds:
            if list(cmd) not in cmds:
                cmds.append(list(cmd))
    elif has_scenario and candidate and candidate.metadata.get("scenario"):
        kind = candidate.metadata.get("scenario_kind", "chat")
        name = candidate.metadata["scenario"]
        if kind == "chat":
            cmds.append(["python3", "scripts/chat-scenarios.py", "--scenario", name])
        elif kind == "implement":
            cmds.append(["python3", "scripts/implement-scenarios.py", "--scenario", name])
        elif kind == "collab":
            cmds.append(["python3", "scripts/collab-scenarios.py", "--scenario", name])

    return cmds


def _has_meaningful_change(changed: list[str]) -> bool:
    meaningful = [
        p
        for p in changed
        if not p.startswith("docs/testing/test-growth-")
        and (
            "_test." in p
            or p.endswith("_test.go")
            or ".test." in p
            or ".spec." in p
            or (p.startswith("scenarios/") and p.endswith(".json"))
        )
    ]
    return len(meaningful) > 0


def verify_iteration(
    cwd: Path,
    *,
    candidate: GrowthCandidate | None,
    hub_url: str = "http://127.0.0.1:18765",
    stability_runs: int = 1,
    skip_live: bool = False,
) -> VerifyResult:
    """Run acceptance checks after agent pass."""
    result = VerifyResult(accepted=False)
    changed = git_changed_files(cwd)

    if not changed:
        result.rejection_reason = "no files changed"
        result.checks.append(("changes_present", False, "no repo changes"))
        return result

    result.checks.append(("changes_present", True, f"{len(changed)} file(s) changed"))

    guard = check_edit_guardrails(cwd, changed_paths=changed)
    result.guardrails = guard
    guard_ok = guard.ok
    result.checks.append(
        (
            "guardrails",
            guard_ok,
            "; ".join(guard.violations) if guard.violations else "ok",
        )
    )

    if guard.product_code_touched and not guard.test_files_touched and not guard.scenario_files_touched:
        result.repair_handoff = True
        result.rejection_reason = "product defect exposed — hand off to layer-fix-loop"
        result.checks.append(("repair_handoff", True, "product code only"))
        return result

    meaningful = _has_meaningful_change(changed)
    result.checks.append(
        (
            "meaningful_change",
            meaningful,
            "test or scenario files touched" if meaningful else "no test/scenario files",
        )
    )
    if not meaningful:
        result.rejection_reason = "changes are not test/scenario improvements"
        return result

    # Contract validation always
    rc, out = run_cmd(["make", "test-scenario-assert"], cwd=cwd)
    result.verify_runs.append((["make", "test-scenario-assert"], rc))
    contract_ok = rc == 0
    result.checks.append(("scenario_contract", contract_ok, out.strip()[-500:] if not contract_ok else "ok"))
    if not contract_ok:
        result.rejection_reason = "scenario contract validation failed"
        return result

    cmds = _default_verify_cmds(changed, candidate)
    env = {"NEURAL_JUNKIE_HUB_URL": hub_url}

    all_cmds_ok = True
    for cmd in cmds:
        if skip_live and any(
            part in ("chat-scenarios.py", "implement-scenarios.py", "collab-scenarios.py")
            for part in cmd
        ):
            result.checks.append(("live_skipped", True, " ".join(cmd)))
            continue
        runs = stability_runs if any("scenarios.py" in part for part in cmd) else 1
        for run_idx in range(runs):
            rc, out = run_cmd(cmd, cwd=cwd, env=env if "scenarios.py" in " ".join(cmd) else None)
            result.verify_runs.append((cmd, rc))
            if rc != 0:
                all_cmds_ok = False
                label = f"verify:{ ' '.join(cmd)}"
                if runs > 1:
                    label += f" run{run_idx + 1}/{runs}"
                result.checks.append((label, False, out.strip()[-800:]))
                break
        if not all_cmds_ok:
            break

    if not all_cmds_ok:
        result.rejection_reason = "targeted verification failed"
        return result

    if guard.violations:
        result.rejection_reason = "guardrail violations: " + "; ".join(guard.violations)
        return result

    result.accepted = True
    result.checks.append(("accepted", True, "all checks passed"))
    return result
