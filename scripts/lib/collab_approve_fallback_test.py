"""Unit tests: approve_file_changes must not greenwash without [FILE_CHANGE]."""

from __future__ import annotations

import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

SCRIPTS = Path(__file__).resolve().parents[1]
ROOT = SCRIPTS.parent
sys.path.insert(0, str(SCRIPTS))

from lib.collab_edge_scenarios import COLLAB_EDGE_SCENARIOS  # noqa: E402


def _load_collab_scenarios():
    path = SCRIPTS / "collab-scenarios.py"
    spec = importlib.util.spec_from_file_location("collab_scenarios_mod", path)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


class EdgeScenarioFallbackFlagsTest(unittest.TestCase):
    def test_edge_approve_steps_disallow_discussion_fallback(self) -> None:
        for name in COLLAB_EDGE_SCENARIOS:
            path = ROOT / "scenarios" / "collab" / f"{name}.json"
            data = json.loads(path.read_text())
            for i, step in enumerate(data.get("steps") or []):
                if step.get("action") != "approve_file_changes":
                    continue
                self.assertFalse(
                    bool(step.get("from_discussion_fallback", False)),
                    f"{name} step[{i}] must not use from_discussion_fallback "
                    "(real [FILE_CHANGE] contract for edge layer)",
                )


class ApproveFileChangesFallbackTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.mod = _load_collab_scenarios()

    def test_no_pending_change_fallback_false_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            ws = Path(tmp)
            collab_id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
            (ws / "collabs" / collab_id).mkdir(parents=True)
            ctx = self.mod.ScenarioContext("http://127.0.0.1:18765", {"name": "t"})
            ctx.collab_channel = "collab:t"
            ctx.collab_id = collab_id
            ctx.workspace_root = str(ws)

            step = {
                "action": "approve_file_changes",
                "path_match": "findings.md",
                "target_rel": f"collabs/{collab_id}/findings.md",
                "from_discussion_fallback": False,
                "min_approved": 1,
                "timeout": "1s",
            }

            with mock.patch.object(
                self.mod.hub,
                "wait_and_approve_file_changes",
                return_value=(0, []),
            ), mock.patch.object(
                self.mod,
                "env_or_automation_bool",
                return_value=False,
            ):
                ok, detail = self.mod.step_approve_file_changes(ctx, step)

            self.assertFalse(ok, detail)
            self.assertIn("no file change approved", detail)

    def test_fallback_true_can_materialize_from_discussion(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            ws = Path(tmp)
            collab_id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
            (ws / "collabs" / collab_id).mkdir(parents=True)
            ctx = self.mod.ScenarioContext("http://127.0.0.1:18765", {"name": "t"})
            ctx.collab_channel = "collab:t"
            ctx.collab_id = collab_id
            ctx.workspace_root = str(ws)

            step = {
                "action": "approve_file_changes",
                "path_match": "findings.md",
                "target_rel": f"collabs/{collab_id}/findings.md",
                "from_discussion_fallback": True,
                "min_approved": 1,
                "timeout": "1s",
            }

            with mock.patch.object(
                self.mod.hub,
                "wait_and_approve_file_changes",
                return_value=(0, []),
            ), mock.patch.object(
                self.mod.hub,
                "list_messages",
                return_value=[],
            ), mock.patch.object(
                self.mod.hub,
                "write_loose_file_change_from_messages",
                return_value=f"collabs/{collab_id}/findings.md",
            ), mock.patch.object(
                self.mod,
                "env_or_automation_bool",
                return_value=False,
            ):
                ok, detail = self.mod.step_approve_file_changes(ctx, step)

            self.assertTrue(ok, detail)
            self.assertIn("discussion fallback", detail)


if __name__ == "__main__":
    unittest.main()
