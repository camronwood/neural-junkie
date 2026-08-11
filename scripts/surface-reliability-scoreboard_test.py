#!/usr/bin/env python3
import importlib.util
import unittest
from pathlib import Path
from types import SimpleNamespace

_SPEC = importlib.util.spec_from_file_location(
    "surface_reliability_scoreboard",
    Path(__file__).with_name("surface-reliability-scoreboard.py"),
)
_MOD = importlib.util.module_from_spec(_SPEC)
assert _SPEC.loader is not None
_SPEC.loader.exec_module(_MOD)
stamp_status = _MOD.stamp_status
memory_status = _MOD.memory_status
session_status = _MOD.session_status


class ScoreboardGatesTest(unittest.TestCase):
    def test_stamp_dual_gate(self):
        self.assertTrue(stamp_status({"action_accuracy": 0.91, "misstamp_rate": 0.02})["ok"])
        self.assertFalse(stamp_status({"action_accuracy": 0.89, "misstamp_rate": 0.0})["ok"])
        self.assertFalse(stamp_status({"action_accuracy": 0.99, "misstamp_rate": 0.06})["ok"])

    def test_memory_dual_gate(self):
        self.assertTrue(memory_status({"hit_rate": 1.0, "forbidden_hit_rate": 0.0})["ok"])
        self.assertFalse(memory_status({"hit_rate": 0.8, "forbidden_hit_rate": 0.0})["ok"])

    def test_session_requires_pass_at_1(self):
        ok = session_status(
            'EVAL_JSON:{"scenario":"dm-work-surface-plan-stickiness","passed_at_1":true}\n',
            SimpleNamespace(returncode=0),
        )
        self.assertTrue(ok["ok"])
        retry = session_status(
            "=== FAIL: dm-work-surface-plan-stickiness ===\n"
            "=== PASS: dm-work-surface-plan-stickiness ===\n"
            'EVAL_JSON:{"scenario":"dm-work-surface-plan-stickiness","passed_at_1":false}\n',
            SimpleNamespace(returncode=0),
        )
        self.assertFalse(retry["ok"])
        self.assertEqual(retry["failed"], ["dm-work-surface-plan-stickiness"])


if __name__ == "__main__":
    unittest.main()
