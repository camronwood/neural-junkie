"""Unit tests for release_prep_layers."""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.release_prep_layers import (  # noqa: E402
    CLIMB_ORDER,
    LAYER_ORDER,
    QUARANTINE_ORDER,
    SOAK_ORDER,
    build_layer_agent_prompt,
    get_layer,
    parse_layer_gate_report,
)


class ReleasePrepLayersTest(unittest.TestCase):
    def test_layer_order_complete(self) -> None:
        for name in LAYER_ORDER:
            spec = get_layer(name)
            self.assertEqual(spec.name, name)
            self.assertIn(spec.tier, ("climb", "soak"))

    def test_climb_order_is_tier_a(self) -> None:
        self.assertEqual(CLIMB_ORDER, ("ci", "implement", "collab-core", "chat"))
        for name in CLIMB_ORDER:
            self.assertEqual(get_layer(name).tier, "climb")

    def test_soak_and_quarantine(self) -> None:
        self.assertIn("chat-full", SOAK_ORDER)
        self.assertIn("bundle", QUARANTINE_ORDER)
        self.assertIn("user-flows", QUARANTINE_ORDER)
        self.assertEqual(get_layer("bundle").tier, "quarantine")
        self.assertEqual(get_layer("chat").stages[0].cmd[-1], "canary")
        self.assertEqual(get_layer("chat-full").stages[0].cmd[-1], "regression")

    def test_parse_layer_gate_report(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            log = Path(tmp) / "layer-gate-implement-test.log"
            log.write_text(
                "=== scenario: go-handler ===\n  ✗ wait_reply: timeout\n=== FAIL: go-handler ===\n",
                encoding="utf-8",
            )
            summary = Path(tmp) / "layer-gate-implement-test.md"
            summary.write_text(
                "\n".join(
                    [
                        "# Layer gate — implement — test UTC",
                        "layer=implement",
                        "Overall: **FAIL** (0/1 stages)",
                        "",
                        "## Stage summary",
                        "",
                        "| Stage | Status | Duration | Exit |",
                        "|-------|--------|----------|------|",
                        "| `implement-scenarios` | FAIL | 10s | 1 |",
                        "",
                        "## Child artifacts",
                        f"- `{log}`",
                        "",
                        "### implement-scenarios (exit 1)",
                        "```text",
                        "tail",
                        "```",
                    ]
                ),
                encoding="utf-8",
            )
            report = parse_layer_gate_report(summary, hub_url="http://127.0.0.1:18765")
            self.assertTrue(report.overall_fail)
            names = {f.name for f in report.failures}
            self.assertIn("implement:go-handler", names)

    def test_build_layer_prompt_mentions_layer(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            summary = Path(tmp) / "layer-gate-ci-test.md"
            summary.write_text(
                "Overall: **FAIL**\n### test-all (exit 1)\n```text\nvet failed\n```\n",
                encoding="utf-8",
            )
            report = parse_layer_gate_report(summary)
            prompt = build_layer_agent_prompt(report, layer="ci")
            self.assertIn("layer gate: ci", prompt)


if __name__ == "__main__":
    unittest.main()
