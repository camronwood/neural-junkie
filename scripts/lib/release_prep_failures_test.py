#!/usr/bin/env python3
"""Unit tests for release-prep failure parsing."""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.release_prep_failures import (  # noqa: E402
    FailureKind,
    _extract_scenarios_from_text,
    build_agent_prompt,
    parse_release_prep_report,
)


FIXTURE = SCRIPTS_DIR.parent / "docs" / "testing" / "release-prep-2026-06-27-2204.md"

COLLAB_TAIL = """
>>> python3 scripts/collab-scenarios.py --scenario execution-no-stack-commands

=== scenario: execution-no-stack-commands ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@Assistant @PlatformEngineer
  started collab abc → collab-abc
  ✓ [1] wait_phase: phase=planning
  ✗ [2] wait_phase: timeout waiting for phase 'reviewing' (last='planning')
=== FAIL: execution-no-stack-commands ===
"""

COLLAB_PARTICIPATION_TAIL = """
=== scenario: plan-dependency-prose-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios agents=@BackendEngineer @SoftwareArchitect
  started collab abc → collab-abc
  ✗ [2] wait_discussion: agent discussion
agent discussion: total=0 counts={}
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
=== FAIL: plan-dependency-prose-regression ===
"""

COLLAB_BATCHED_FAIL_TAIL = """
=== scenario: collab-conversation-quality-regression ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios
  ✓ cleanup: cancelled

=== scenario: collab-generation-error-resilience ===
  hub=http://127.0.0.1:18765 channel=collab-scenarios
  ✓ cleanup: cancelled

=== FAIL: collab-conversation-quality-regression ===
  FAIL: no collaboration redirect
=== FAIL: collab-generation-error-resilience ===
  FAIL: @SoftwareArchitect — no collaboration_discussion (silent or shouldRespond blocked)
"""

CHAT_TAIL = """
=== scenario: thanks-closure ===
  hub=http://127.0.0.1:18765
  channel=chat-scenarios agent=Assistant
  ✗ [6] assert_reply_count: reply count since start: got 3 want 2
=== FAIL: thanks-closure ===
"""


class ReleasePrepFailuresTest(unittest.TestCase):
    def test_extract_collab_scenario_from_tail(self) -> None:
        found = _extract_scenarios_from_text(
            COLLAB_TAIL,
            "http://127.0.0.1:18765",
            default_stage="collab-scenario-regression",
        )
        names = {f.name for f in found}
        self.assertIn("collab:execution-no-stack-commands", names)
        collab = next(f for f in found if f.name == "collab:execution-no-stack-commands")
        self.assertEqual(collab.rerun_cmd[:3], ["python3", "scripts/collab-scenarios.py", "--scenario"])
        self.assertEqual(collab.kind, FailureKind.CODE)

    def test_collab_participation_failures_are_code_not_flake(self) -> None:
        found = _extract_scenarios_from_text(
            COLLAB_PARTICIPATION_TAIL,
            "http://127.0.0.1:18765",
            default_stage="collab-scenarios-all",
        )
        collab = next(f for f in found if f.name == "collab:plan-dependency-prose-regression")
        self.assertEqual(collab.kind, FailureKind.CODE)

    def test_collab_batched_fail_blocks_at_log_tail(self) -> None:
        found = _extract_scenarios_from_text(
            COLLAB_BATCHED_FAIL_TAIL,
            "http://127.0.0.1:18765",
            default_stage="collab-scenarios-all",
        )
        names = {f.name for f in found}
        self.assertIn("collab:collab-conversation-quality-regression", names)
        self.assertIn("collab:collab-generation-error-resilience", names)
        self.assertTrue(all(f.kind == FailureKind.CODE for f in found))

    def test_extract_chat_scenario_from_tail(self) -> None:
        found = _extract_scenarios_from_text(
            CHAT_TAIL,
            "http://127.0.0.1:18765",
            default_stage="chat-scenarios-regression",
        )
        self.assertTrue(any(f.name == "chat:thanks-closure" for f in found))

    def test_parse_live_report_includes_collab_scenarios(self) -> None:
        if not FIXTURE.is_file():
            self.skipTest(f"missing fixture {FIXTURE}")
        report = parse_release_prep_report(FIXTURE)
        collab = [f for f in report.failures if f.name.startswith("collab:")]
        self.assertGreater(len(collab), 0, "expected collab scenario failures from stage tails")

    def test_parse_live_release_prep_report(self) -> None:
        if not FIXTURE.is_file():
            self.skipTest(f"missing fixture {FIXTURE}")
        report = parse_release_prep_report(FIXTURE)
        self.assertTrue(report.overall_fail)
        self.assertIn("test-everything", report.failed_phases)
        self.assertTrue(any(p.name.startswith("test-everything") for p in report.child_artifacts) or report.child_artifacts)
        self.assertGreater(len(report.failures), 0)

    def test_build_prompt_includes_rules(self) -> None:
        if not FIXTURE.is_file():
            self.skipTest(f"missing fixture {FIXTURE}")
        report = parse_release_prep_report(FIXTURE)
        prompt = build_agent_prompt(report)
        self.assertIn("Do NOT weaken test assertions", prompt)
        self.assertIn("test-everything", prompt)

    def test_classify_model_benchmark(self) -> None:
        text = "# Release prep\n| `model-benchmark` | FAIL |"
        path = Path(self._fixture_text(text))
        report = parse_release_prep_report(path)
        kinds = {f.name: f.kind for f in report.failures if f.name == "model-benchmark"}
        self.assertEqual(kinds.get("model-benchmark"), FailureKind.MODEL_BENCHMARK)

    def _fixture_text(self, text: str) -> str:
        import tempfile

        with tempfile.NamedTemporaryFile("w", suffix=".md", delete=False, encoding="utf-8") as f:
            f.write(text)
            return f.name


if __name__ == "__main__":
    raise SystemExit(unittest.main())
