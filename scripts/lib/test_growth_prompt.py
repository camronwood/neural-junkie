"""Build Cursor agent prompts for test-growth loop iterations."""

from __future__ import annotations

from lib.test_growth_candidates import GrowthCandidate
from lib.test_growth_guardrails import guardrail_rules_text


def build_test_growth_prompt(
    candidate: GrowthCandidate,
    *,
    iteration: int,
    max_iterations: int,
) -> str:
    verify_lines = []
    for cmd in candidate.verify_cmds:
        verify_lines.append(f"- {' '.join(cmd)}")
    if not verify_lines:
        verify_lines = [
            "- make test-scenario-assert",
            "- go test for any new *_test.go files you add",
        ]

    lines = [
        f"You are improving Neural Junkie test coverage — **test-growth loop iteration {iteration}/{max_iterations}**.",
        "",
        "Objective: **strengthen the test suite** — add or tighten tests. This is NOT a repair loop.",
        "If your new test exposes a product bug, stop and report it — do NOT patch product code to greenwash.",
        "",
        guardrail_rules_text(),
        "---",
        "",
        f"## Selected candidate: `{candidate.id}`",
        "",
        f"**Kind:** {candidate.kind}",
        f"**Title:** {candidate.title}",
        f"**Score:** {candidate.score:.0f}",
        f"**Source:** {candidate.source or '(discovery)'}",
        "",
        candidate.description,
        "",
    ]

    if candidate.target_paths:
        lines.extend(["**Target paths:**", ""])
        for p in candidate.target_paths:
            lines.append(f"- `{p}`")
        lines.append("")

    if candidate.suggested_files:
        lines.extend(["**Suggested files to create or edit:**", ""])
        for p in candidate.suggested_files:
            lines.append(f"- `{p}`")
        lines.append("")

    if candidate.metadata:
        lines.extend(["**Metadata:**", ""])
        for k, v in candidate.metadata.items():
            lines.append(f"- {k}: {v}")
        lines.append("")

    lines.extend(
        [
            "## Your task",
            "",
            "1. Implement ONE focused test improvement for this candidate.",
            "2. Prefer minimal, high-signal changes over broad rewrites.",
            "3. For new live scenarios: copy a similar JSON from `scenarios/`, set tags/assertions, "
            "and add Layer A routing coverage when routing-related.",
            "4. Run the verification commands below before finishing.",
            "5. Summarize: what gap you closed, files changed, commands run.",
            "",
            "## Targeted verification (run after your changes)",
            "",
            *verify_lines,
            "",
        ]
    )

    return "\n".join(lines)


def build_agent_invoke_message(prompt_path: str, *, repo_root: str) -> str:
    return (
        "Neural Junkie test-growth loop.\n\n"
        f"Read the full brief at `{prompt_path}` in this repo and implement ONE test improvement.\n"
        "Add or tighten tests only — do NOT weaken assertions or patch product code to pass.\n"
        "Run verification commands listed in that file when done.\n"
        "Summarize what test gap you closed and which commands you ran."
    )
