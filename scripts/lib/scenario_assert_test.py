"""Unit tests for scenario_assert helpers."""

from __future__ import annotations

import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from deliverable_judge import (
    cloud_judge_tripped,
    judge_deliverable,
    parse_judge_response,
    reset_cloud_judge_circuit,
)
from scenario_assert import (
    check_contains_all,
    check_file_deliverable,
    check_text_patterns,
    expand_deliverable_steps,
    looks_like_read_only_inspection_command,
    looks_like_stack_tool_command,
    merge_deliverable_step,
    scenario_question,
)


class StackToolCommandTest(unittest.TestCase):
    def test_detects_stack_tools(self) -> None:
        for cmd in ("docker-compose up -d", "npm install", "kubectl get pods", "make build"):
            self.assertTrue(looks_like_stack_tool_command(cmd), cmd)

    def test_allows_read_only(self) -> None:
        for cmd in ("cat README.md", "grep schema resource-api/", "find . -name '*.md'"):
            self.assertFalse(looks_like_stack_tool_command(cmd), cmd)


class ReadOnlyInspectionCommandTest(unittest.TestCase):
    def test_detects_read_only(self) -> None:
        for cmd in ("cat collabs/abc/out.md", "grep schema internal/", "git status", "ls -la"):
            self.assertTrue(looks_like_read_only_inspection_command(cmd), cmd)

    def test_rejects_write_commands(self) -> None:
        for cmd in ("npm install", "rm -rf node_modules", "curl -X POST http://x"):
            self.assertFalse(looks_like_read_only_inspection_command(cmd), cmd)


class DeliverableAssertTest(unittest.TestCase):
    def test_check_file_deliverable_exists_and_matches(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "src" / "theme.css"
            path.parent.mkdir(parents=True)
            path.write_text(":root { --dark: #111; }\n.dark { color: white; }\n", encoding="utf-8")
            ok, detail = check_file_deliverable(
                root=tmp,
                rel="src/theme.css",
                spec={
                    "for_question": {"contains_all": ["dark"], "any_match": ["--dark|\\.dark"]},
                },
            )
            self.assertTrue(ok, detail)

    def test_check_file_deliverable_rejects_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            ok, detail = check_file_deliverable(root=tmp, rel="missing.go", spec={})
            self.assertFalse(ok)
            self.assertIn("missing", detail)

    def test_check_file_deliverable_absent(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            stale = Path(tmp) / "src" / "App.js"
            stale.parent.mkdir(parents=True)
            stale.write_text("bad", encoding="utf-8")
            ok, detail = check_file_deliverable(
                root=tmp,
                rel="src/App.js",
                spec={"must_exist": False},
            )
            self.assertFalse(ok)
            self.assertIn("absent", detail.lower())

    def test_merge_deliverable_step_inherits_llm_judge(self) -> None:
        scenario = {
            "expect_deliverables": [
                {
                    "path": "core/sample/main.go",
                    "llm_judge": True,
                    "for_question": {"contains_all": ["HelloWorld"]},
                }
            ]
        }
        step = {"action": "assert_file_exists", "path": "core/sample/main.go", "contains": "HelloWorld"}
        merged = merge_deliverable_step(scenario, step)
        self.assertTrue(merged.get("llm_judge"))
        self.assertEqual(merged.get("contains"), "HelloWorld")
        self.assertEqual(merged["for_question"]["contains_all"], ["HelloWorld"])

    def test_expand_deliverable_steps_skips_duplicates(self) -> None:
        scenario = {
            "expect_deliverables": [{"path": "a.txt"}, {"path": "b.txt"}],
            "steps": [{"action": "assert_file_exists", "path": "a.txt"}],
        }
        extra = expand_deliverable_steps(scenario)
        self.assertEqual(len(extra), 1)
        self.assertEqual(extra[0]["path"], "b.txt")

    def test_scenario_question_from_send(self) -> None:
        scenario = {
            "steps": [
                {"action": "send", "content": "implement theme toggle"},
                {"action": "wait_reply"},
            ]
        }
        self.assertEqual(scenario_question(scenario), "implement theme toggle")

    def test_parse_judge_response(self) -> None:
        ok, reason = parse_judge_response("PASS\nGrounded in main.go")
        self.assertTrue(ok)
        self.assertIn("main.go", reason)
        ok, reason = parse_judge_response("FAIL\nGeneric stub only")
        self.assertFalse(ok)
        self.assertIn("stub", reason.lower())

    def test_cloud_judge_falls_back_to_ollama(self) -> None:
        reset_cloud_judge_circuit()
        with mock.patch.dict(
            os.environ,
            {
                "NJ_DELIVERABLE_JUDGE_PROVIDER": "gemini",
                "NJ_DELIVERABLE_JUDGE_MODE": "hub",
                "NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA": "1",
            },
        ):
            with mock.patch(
                "deliverable_judge.hub_judge_deliverable",
                return_value=(False, "Sorry, I encountered an error while generating a response."),
            ):
                with mock.patch(
                    "deliverable_judge.ollama_judge_deliverable",
                    return_value=(True, "substantive pass"),
                ):
                    ok, detail = judge_deliverable(
                        question="write findings",
                        rel_path="findings.md",
                        file_body="# Findings\nmain.go",
                        hub_base="http://127.0.0.1:18765",
                    )
        self.assertTrue(ok)
        self.assertIn("ollama/", detail)
        self.assertTrue(cloud_judge_tripped("gemini"))

    def test_cloud_judge_circuit_skips_hub_on_second_call(self) -> None:
        reset_cloud_judge_circuit()
        with mock.patch.dict(
            os.environ,
            {
                "NJ_DELIVERABLE_JUDGE_PROVIDER": "gemini",
                "NJ_DELIVERABLE_JUDGE_MODE": "hub",
                "NJ_DELIVERABLE_JUDGE_FALLBACK_OLLAMA": "1",
            },
        ):
            hub = mock.patch(
                "deliverable_judge.hub_judge_deliverable",
                return_value=(False, "quota exceeded for gemini-2.5-flash"),
            )
            ollama = mock.patch(
                "deliverable_judge.ollama_judge_deliverable",
                return_value=(True, "substantive pass"),
            )
            with hub as hub_mock, ollama as ollama_mock:
                ok1, _ = judge_deliverable(
                    question="write findings",
                    rel_path="findings.md",
                    file_body="# Findings\nmain.go",
                    hub_base="http://127.0.0.1:18765",
                )
                ok2, detail2 = judge_deliverable(
                    question="write findings",
                    rel_path="findings2.md",
                    file_body="# Findings\nmain.go",
                    hub_base="http://127.0.0.1:18765",
                )
        self.assertTrue(ok1)
        self.assertTrue(ok2)
        self.assertEqual(hub_mock.call_count, 1)
        self.assertEqual(ollama_mock.call_count, 2)
        self.assertIn("cloud circuit open", detail2)

    def test_ollama_judge_routes_locally(self) -> None:
        reset_cloud_judge_circuit()
        with mock.patch.dict(
            os.environ,
            {"NJ_DELIVERABLE_JUDGE_PROVIDER": "ollama", "NJ_DELIVERABLE_JUDGE_MODE": "ollama"},
        ):
            with mock.patch(
                "deliverable_judge.ollama_judge_deliverable",
                return_value=(True, "independent pass"),
            ):
                ok, detail = judge_deliverable(
                    question="write findings",
                    rel_path="findings.md",
                    file_body="# Findings\nmain.go",
                    hub_base="http://127.0.0.1:18765",
                )
        self.assertTrue(ok)
        self.assertIn("ollama/", detail)

    def test_contains_all(self) -> None:
        ok, _ = check_contains_all("hello world", ["hello", "world"])
        self.assertTrue(ok)
        ok, detail = check_contains_all("hello", ["hello", "world"])
        self.assertFalse(ok)
        self.assertIn("world", detail)


if __name__ == "__main__":
    unittest.main()
