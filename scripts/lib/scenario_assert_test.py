"""Unit tests for scenario_assert helpers."""

from __future__ import annotations

import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from deliverable_judge import (
    build_judge_prompt,
    cloud_judge_tripped,
    judge_deliverable,
    parse_judge_response,
    reset_cloud_judge_circuit,
)
from scenario_assert import (
    check_contains_all,
    check_file_deliverable,
    check_min_markdown_bullets,
    check_text_patterns,
    count_markdown_bullets,
    expand_deliverable_steps,
    is_harness_control_send,
    looks_like_read_only_inspection_command,
    looks_like_stack_tool_command,
    merge_deliverable_step,
    scenario_question,
)

_REPO_ROOT = Path(__file__).resolve().parents[2]


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

    def test_check_file_deliverable_judge_fail_is_advisory(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "findings.md"
            path.write_text("# Findings\n- README.md is minimal\n", encoding="utf-8")
            with mock.patch(
                "scenario_assert.judge_deliverable",
                return_value=(False, "ollama/qwen: not substantive"),
            ):
                ok, detail = check_file_deliverable(
                    root=tmp,
                    rel="findings.md",
                    spec={"llm_judge": True, "min_bytes": 10},
                    question="summarize README",
                )
            self.assertTrue(ok)
            self.assertTrue(detail.startswith("judge:warn:"), detail)

    def test_check_file_deliverable_judge_pass(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "findings.md"
            path.write_text("# Findings\n- README.md is minimal\n", encoding="utf-8")
            with mock.patch(
                "scenario_assert.judge_deliverable",
                return_value=(True, "ollama/qwen: looks good"),
            ):
                ok, detail = check_file_deliverable(
                    root=tmp,
                    rel="findings.md",
                    spec={"llm_judge": True, "min_bytes": 10},
                    question="summarize README",
                )
            self.assertTrue(ok)
            self.assertTrue(detail.startswith("judge:pass:"), detail)

    def test_check_file_deliverable_judge_exception_is_advisory(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "findings.md"
            path.write_text("# Findings\n- README.md is minimal\n", encoding="utf-8")
            with mock.patch(
                "scenario_assert.judge_deliverable",
                side_effect=RuntimeError("ollama down"),
            ):
                ok, detail = check_file_deliverable(
                    root=tmp,
                    rel="findings.md",
                    spec={"llm_judge": True, "min_bytes": 10},
                    question="summarize README",
                )
            self.assertTrue(ok)
            self.assertTrue(detail.startswith("judge:warn:exception:"), detail)

    def test_check_file_deliverable_regex_still_hard_fail(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "findings.md"
            path.write_text("# Findings\n- wrong stuff only\n", encoding="utf-8")
            ok, detail = check_file_deliverable(
                root=tmp,
                rel="findings.md",
                spec={
                    "llm_judge": True,
                    "min_bytes": 10,
                    "for_question": {"any_match": ["README\\.md"]},
                },
                question="summarize README",
            )
            self.assertFalse(ok)
            self.assertNotIn("judge:", detail)

    def test_expand_deliverable_steps_skips_duplicates(self) -> None:
        scenario = {
            "expect_deliverables": [{"path": "a.txt"}, {"path": "b.txt"}],
            "steps": [{"action": "assert_file_exists", "path": "a.txt"}],
        }
        extra = expand_deliverable_steps(scenario)
        self.assertEqual(len(extra), 1)
        self.assertEqual(extra[0]["path"], "b.txt")


class ScenarioQuestionTest(unittest.TestCase):
    def test_harness_control_send_detection(self) -> None:
        self.assertTrue(is_harness_control_send("/resume-plan <collab-id>"))
        self.assertTrue(is_harness_control_send("/complete-collab <collab-id> --force"))
        self.assertTrue(is_harness_control_send("/cancel-plan abc12345"))
        self.assertFalse(is_harness_control_send("implement theme toggle"))
        self.assertFalse(is_harness_control_send("@BackendEngineer write findings.md"))

    def test_scenario_question_from_send(self) -> None:
        scenario = {
            "steps": [
                {"action": "send", "content": "implement theme toggle"},
                {"action": "wait_reply"},
            ]
        }
        self.assertEqual(scenario_question(scenario), "implement theme toggle")

    def test_scenario_question_prefers_collaborate_goal_over_send(self) -> None:
        scenario_path = _REPO_ROOT / "scenarios/collab/document-findings-execution.json"
        scenario = json.loads(scenario_path.read_text(encoding="utf-8"))
        goal = scenario["collaborate"]["goal"].strip()
        question = scenario_question(scenario)
        self.assertEqual(question, goal)
        self.assertIn("Document findings", question)
        self.assertNotIn("/resume-plan", question)

    def test_scenario_question_skips_harness_send_for_genuine_message(self) -> None:
        genuine = "@BackendEngineer Complete Task 1: write collabs/<collab-id>/findings.md."
        scenario = {
            "steps": [
                {"action": "send", "content": "/resume-plan <collab-id>"},
                {"action": "send", "content": genuine},
            ]
        }
        self.assertEqual(scenario_question(scenario), genuine)

    def test_scenario_question_description_when_only_harness_sends(self) -> None:
        description = "Task phrased as Document findings in collabs/<id>/findings.md."
        scenario = {
            "description": description,
            "steps": [
                {"action": "send", "content": "/resume-plan <collab-id>"},
                {"action": "send", "content": "/complete-collab <collab-id> --force"},
            ],
        }
        self.assertEqual(scenario_question(scenario), description)

    def test_build_judge_prompt_uses_collaborate_goal_not_harness_send(self) -> None:
        scenario_path = _REPO_ROOT / "scenarios/collab/document-findings-execution.json"
        scenario = json.loads(scenario_path.read_text(encoding="utf-8"))
        question = scenario_question(scenario)
        prompt = build_judge_prompt(
            question=question,
            rel_path="collabs/<collab-id>/findings.md",
            file_body="# Findings\nREADME and main.go\n",
        )
        self.assertIn("Document findings", prompt)
        self.assertNotIn("/resume-plan", prompt)


class DeliverableJudgeTest(unittest.TestCase):
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
        ok, _ = check_contains_all("see README.md and core/sample/main.go", [r"README\.md", r"core/sample/main\.go"])
        self.assertTrue(ok)
        ok, detail = check_contains_all("only README.md here", [r"README\.md", r"core/sample/main\.go"])
        self.assertFalse(ok)
        self.assertIn("missing contains_all", detail)
        self.assertIn("core/sample/main", detail)
        ok, detail = check_contains_all("text", ["("])
        self.assertFalse(ok)
        self.assertIn("invalid contains_all", detail)


class MarkdownBulletAssertTest(unittest.TestCase):
    def test_count_markdown_bullets(self) -> None:
        body = "- one\n* two\n+ three\n"
        self.assertEqual(count_markdown_bullets(body), 3)
        self.assertEqual(count_markdown_bullets("- one\n- two\n"), 2)
        ordered = "1. first\n2. second\n3. third\n"
        self.assertEqual(count_markdown_bullets(ordered), 3)

    def test_min_markdown_bullets_pass_and_fail(self) -> None:
        two = "- a\n- b\n"
        ok, _ = check_min_markdown_bullets(two, 2)
        self.assertTrue(ok)
        ok, detail = check_min_markdown_bullets(two, 3)
        self.assertFalse(ok)
        self.assertIn("2 < 3", detail)

    def test_check_file_deliverable_min_bullets_and_contains_all(self) -> None:
        good = "# Findings\n- README.md describes the repo.\n- core/sample/main.go prints HelloWorld.\n- Both are minimal Go fixtures.\n"
        bad_bullets = "# Findings\n- README.md only.\n- Missing second source.\n"
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "collabs" / "abc" / "findings.md"
            path.parent.mkdir(parents=True)
            spec = {
                "for_question": {
                    "min_markdown_bullets": 3,
                    "contains_all": [r"README\.md", r"core/sample/main\.go"],
                }
            }
            path.write_text(good, encoding="utf-8")
            ok, detail = check_file_deliverable(root=tmp, rel="collabs/abc/findings.md", spec=spec)
            self.assertTrue(ok, detail)

            path.write_text(bad_bullets, encoding="utf-8")
            ok, detail = check_file_deliverable(root=tmp, rel="collabs/abc/findings.md", spec=spec)
            self.assertFalse(ok)
            self.assertIn("min_markdown_bullets", detail)

            missing_source = "- README.md\n- line two\n- line three\n"
            path.write_text(missing_source, encoding="utf-8")
            ok, detail = check_file_deliverable(root=tmp, rel="collabs/abc/findings.md", spec=spec)
            self.assertFalse(ok)
            self.assertIn("missing contains_all", detail)
            self.assertIn("core/sample/main", detail)

    def test_check_file_deliverable_ordered_bullets(self) -> None:
        body = "1. README.md\n2. core/sample/main.go\n3. minimal repo\n"
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "findings.md"
            path.write_text(body, encoding="utf-8")
            ok, detail = check_file_deliverable(
                root=tmp,
                rel="findings.md",
                spec={"for_question": {"min_markdown_bullets": 3, "contains_all": [r"README\.md", r"core/sample/main\.go"]}},
            )
            self.assertTrue(ok, detail)


if __name__ == "__main__":
    unittest.main()
