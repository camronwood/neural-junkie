"""Edit allowlists and assertion-weakening detection for test-growth loop."""

from __future__ import annotations

import json
import re
import subprocess
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]

# Paths the test-growth agent may edit.
ALLOWED_PREFIXES = (
    "internal/",
    "desktop/src/",
    "desktop/src-tauri/src/",
    "scripts/lib/",
    "scenarios/",
    "test/",
    "cmd/",
)

ALLOWED_SUFFIXES = (
    "_test.go",
    "_test.py",
    ".test.ts",
    ".test.tsx",
    ".spec.ts",
    ".spec.tsx",
)

ALLOWED_EXACT = (
    "scripts/lib/scenario_assert.py",
    "scripts/lib/scenario_contract.py",
    "scripts/lib/scenario_flake_retry.py",
    "scripts/lib/scenario_wait.py",
)

# Product code prefixes — edits here trigger repair handoff unless explicitly allowed.
PRODUCT_PREFIXES = (
    "internal/agent/",
    "internal/hub/",
    "internal/routing/",
    "internal/collab/",
    "internal/mcp/",
    "internal/filechange/",
    "desktop/src/",
    "cmd/server/",
)

BLOCKED_PRODUCT_SUFFIXES = (".go", ".ts", ".tsx", ".rs")

ASSERTION_KEYS = frozenset({"any_match", "none_match", "contains_all", "contains", "content_any_match"})


@dataclass
class GuardrailResult:
    ok: bool
    violations: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    product_code_touched: list[str] = field(default_factory=list)
    test_files_touched: list[str] = field(default_factory=list)
    scenario_files_touched: list[str] = field(default_factory=list)


def _normalize(path: str) -> str:
    return path.replace("\\", "/").strip()


def is_allowed_edit_path(path: str) -> bool:
    p = _normalize(path)
    if p in ALLOWED_EXACT:
        return True
    if any(p.endswith(s) for s in ALLOWED_SUFFIXES):
        return True
    if p.startswith("scenarios/") and p.endswith(".json"):
        return True
    if p.startswith("scenarios/fixtures/"):
        return True
    if any(p.startswith(prefix) for prefix in ALLOWED_PREFIXES):
        if p.endswith((".go", ".ts", ".tsx", ".py", ".json", ".md")):
            if "_test." in p or p.endswith("_test.go") or "/scenarios/" in p:
                return True
            if p.startswith("scenarios/"):
                return True
            if p.endswith("_test.py") or ".test." in p or ".spec." in p:
                return True
    return False


def is_product_code_path(path: str) -> bool:
    p = _normalize(path)
    if not any(p.startswith(prefix) for prefix in PRODUCT_PREFIXES):
        return False
    if any(p.endswith(s) for s in BLOCKED_PRODUCT_SUFFIXES):
        if "_test." in p or p.endswith("_test.go"):
            return False
        return True
    return False


def git_changed_files(cwd: Path) -> list[str]:
    proc = subprocess.run(
        ["git", "status", "--porcelain"],
        cwd=cwd,
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        return []
    paths: list[str] = []
    for line in (proc.stdout or "").splitlines():
        if len(line) < 4:
            continue
        path = line[3:].strip()
        if " -> " in path:
            path = path.split(" -> ", 1)[1]
        if path:
            paths.append(path)
    return paths


def _extract_assertions(obj: object) -> dict[str, list]:
    """Collect assertion patterns from a scenario JSON tree."""
    found: dict[str, list] = {k: [] for k in ASSERTION_KEYS}

    def walk(node: object) -> None:
        if isinstance(node, dict):
            for key in ASSERTION_KEYS:
                val = node.get(key)
                if val is not None:
                    if isinstance(val, list):
                        found[key].extend(str(v) for v in val)
                    else:
                        found[key].append(str(val))
            for_question = node.get("for_question")
            if isinstance(for_question, dict):
                walk(for_question)
            for v in node.values():
                walk(v)
        elif isinstance(node, list):
            for item in node:
                walk(item)

    walk(obj)
    return found


def _load_scenario_json(path: Path) -> dict | None:
    try:
        with path.open(encoding="utf-8") as f:
            data = json.load(f)
        return data if isinstance(data, dict) else None
    except (OSError, json.JSONDecodeError):
        return None


def detect_assertion_weakening(
    cwd: Path,
    changed_paths: list[str],
) -> list[str]:
    """Compare scenario JSON before/after; flag removed or loosened assertions."""
    violations: list[str] = []
    for rel in changed_paths:
        p = _normalize(rel)
        if not (p.startswith("scenarios/") and p.endswith(".json")):
            continue
        full = cwd / rel
        if not full.is_file():
            continue
        proc = subprocess.run(
            ["git", "show", f"HEAD:{rel}"],
            cwd=cwd,
            capture_output=True,
            text=True,
            check=False,
        )
        if proc.returncode != 0:
            continue  # new file — not weakening
        try:
            before = json.loads(proc.stdout)
        except json.JSONDecodeError:
            continue
        after = _load_scenario_json(full)
        if after is None:
            continue
        before_assert = _extract_assertions(before)
        after_assert = _extract_assertions(after)

        for key in ASSERTION_KEYS:
            before_vals = set(before_assert.get(key, []))
            after_vals = set(after_assert.get(key, []))
            removed = before_vals - after_vals
            if removed and key in ("none_match", "contains_all"):
                violations.append(
                    f"{rel}: removed {key} patterns {sorted(removed)!r} (assertion weakening)"
                )
            if key == "any_match" and len(after_vals) > len(before_vals):
                # Widening any_match without removing none_match is suspicious but allowed with warning
                new = after_vals - before_vals
                if new and not (before_assert.get("none_match") or after_assert.get("none_match")):
                    violations.append(
                        f"{rel}: widened any_match without none_match guard ({sorted(new)!r})"
                    )

        # Soft chat-phrase removal is allowed (disk/metadata is the modern pass gate).
        # Do not flag dropping optional session prose any_match patterns.

        # Detect removal of expect_deliverables quality bars
        before_expect = before.get("expect_deliverables")
        after_expect = after.get("expect_deliverables")
        if isinstance(before_expect, list) and isinstance(after_expect, list):
            if len(after_expect) < len(before_expect):
                violations.append(f"{rel}: removed expect_deliverables entries")

        # Reject reintroducing phrase-only waits on implement/parity/user-flow implement.
        if any(
            rel.replace("\\", "/").startswith(prefix)
            for prefix in (
                "scenarios/implement/",
                "scenarios/parity/",
                "scenarios/user-flows/implement/",
            )
        ):
            for step in after.get("steps") or []:
                if not isinstance(step, dict) or step.get("action") != "wait_reply":
                    continue
                if step.get("until_any_match") and not (
                    step.get("until_file_match")
                    or step.get("until_file_exists")
                    or step.get("until_files_exist")
                    or step.get("until_file_absent")
                    or step.get("until_files_absent")
                    or step.get("until_metadata_keys")
                ):
                    violations.append(
                        f"{rel}: wait_reply uses until_any_match alone; use disk/metadata waits"
                    )
    return violations


def check_edit_guardrails(cwd: Path, *, changed_paths: list[str] | None = None) -> GuardrailResult:
    """Validate changed files against test-growth edit policy."""
    paths = changed_paths if changed_paths is not None else git_changed_files(cwd)
    result = GuardrailResult(ok=True)

    for rel in paths:
        p = _normalize(rel)
        if p.startswith("docs/testing/test-growth-"):
            continue  # loop artifacts
        if not is_allowed_edit_path(p):
            if is_product_code_path(p):
                result.product_code_touched.append(p)
                result.violations.append(f"product code edit not allowed in test-growth: {p}")
            else:
                result.violations.append(f"path not on test-growth allowlist: {p}")
        elif p.startswith("scenarios/") and p.endswith(".json"):
            result.scenario_files_touched.append(p)
        elif "_test." in p or p.endswith("_test.go") or ".test." in p or ".spec." in p:
            result.test_files_touched.append(p)

    weaken = detect_assertion_weakening(cwd, paths)
    result.violations.extend(weaken)

    if result.violations:
        result.ok = False
    if result.product_code_touched and not result.test_files_touched and not result.scenario_files_touched:
        result.warnings.append(
            "Only product code changed — hand off to layer-fix-loop instead of test-growth."
        )
    return result


def guardrail_rules_text() -> str:
    return "\n".join(
        [
            "## Edit allowlist (mandatory)",
            "",
            "You may ONLY edit:",
            "- Go test files (`*_test.go`) under `internal/`, `cmd/`, `test/`",
            "- Frontend tests (`*.test.ts`, `*.spec.ts`) under `desktop/src/`",
            "- Python test files under `scripts/lib/`",
            "- Scenario JSON under `scenarios/`",
            "- Scenario helpers: `scripts/lib/scenario_assert.py`, `scenario_contract.py`",
            "- Fixtures under `scenarios/fixtures/` when required for a new edge case",
            "",
            "Do NOT edit product/runtime code unless the iteration is explicitly handing off to repair.",
            "",
            "## Assertion policy (mandatory)",
            "",
            "- ADD tests or TIGHTEN assertions only.",
            "- Do NOT remove `none_match`, `contains_all`, or `expect_deliverables` quality bars.",
            "- Implement/parity waits: use `until_file_match` / `until_metadata_keys` — not `until_any_match` alone.",
            "- Soft chat-phrase `any_match` cleanup is OK; pass = disk + metadata.",
            "- Prefer `scripts/lib/scenario_assert.py` / `scenario_contract.py` / `scenario_wait.py` helpers.",
            "- Do NOT widen regex patterns to greenwash flakes.",
            "- New live scenarios must follow patterns in docs/CHAT_SCENARIOS.md and docs/TESTING.md.",
            "- Pair new chat scenarios with Layer A cases when the bug was routing-related.",
            "",
        ]
    )
