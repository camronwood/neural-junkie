"""Unit tests for SUT self-improve parsers and LoRA row emit (no hub)."""

from __future__ import annotations

import json
import tempfile
import unittest
from pathlib import Path

from sut_episode import validate_sut_episode
from sut_human import parse_human_response
from sut_judge import parse_judge_response
from sut_lora_emit import append_jsonl, rows_from_failure


class SutHumanParseTest(unittest.TestCase):
    def test_user_message(self) -> None:
        t = parse_human_response("USER_MESSAGE: Remember ThemeSettings please\n")
        self.assertEqual(t.kind, "message")
        self.assertIn("ThemeSettings", t.text)

    def test_stop(self) -> None:
        t = parse_human_response("STOP: Goal satisfied\n")
        self.assertEqual(t.kind, "stop")
        self.assertIn("satisfied", t.text.lower())


class SutJudgeParseTest(unittest.TestCase):
    def test_pass(self) -> None:
        v = parse_judge_response(
            "PASS\nSCORE: 0.91\nFAILURE_KIND: none\nREASON: Retained entity\n"
        )
        self.assertTrue(v.passed)
        self.assertAlmostEqual(v.score or 0, 0.91, places=2)

    def test_fail_with_gold(self) -> None:
        v = parse_judge_response(
            "FAIL\n"
            "SCORE: 0.20\n"
            "FAILURE_KIND: memory\n"
            "GOLD_OUTPUT: Keep referring to ThemeSettings in the Appearance section.\n"
            "REASON: Dropped the component name\n"
        )
        self.assertFalse(v.passed)
        self.assertEqual(v.failure_kind, "memory")
        self.assertIn("ThemeSettings", v.gold_output)


class SutLoraEmitTest(unittest.TestCase):
    def test_rows_and_jsonl(self) -> None:
        transcript = [
            {"role": "user", "content": "Design ThemeSettings"},
            {"role": "assistant", "content": "Sure, here is a button"},
            {"role": "user", "content": "Add a11y for it"},
        ]
        rows = rows_from_failure(
            episode_name="dm-long-horizon-entity-retention",
            transcript=transcript,
            gold_output="Add keyboard a11y to ThemeSettings…",
            target_agent="FrontendEngineer",
        )
        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["source_kind"], "sut_eval")
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "rows.jsonl"
            n = append_jsonl(path, rows)
            self.assertEqual(n, 1)
            loaded = json.loads(path.read_text(encoding="utf-8").strip())
            self.assertEqual(loaded["output"], rows[0]["output"])


class SutEpisodeContractTest(unittest.TestCase):
    def test_seed_episode_valid(self) -> None:
        root = Path(__file__).resolve().parents[2]
        path = root / "scenarios" / "sut" / "dm-long-horizon-entity-retention.json"
        data = json.loads(path.read_text(encoding="utf-8"))
        errs = validate_sut_episode("sut/dm-long-horizon-entity-retention.json", data)
        self.assertEqual(errs, [])


if __name__ == "__main__":
    unittest.main()
