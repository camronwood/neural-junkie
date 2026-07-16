#!/usr/bin/env python3
"""Unit tests for scripts/lib/humaneval_runner.py."""

from __future__ import annotations

import unittest

from lib.humaneval_runner import (
    build_harness_source,
    extract_python_code,
    filter_problems,
    load_problems,
    run_harness,
    tokens_from_ollama,
)


class HumanEvalRunnerTests(unittest.TestCase):
    def test_load_25(self) -> None:
        note, problems = load_problems()
        self.assertIn("HumanEval", note)
        self.assertEqual(len(problems), 25)
        self.assertTrue(all(p.get("id") and p.get("entry_point") and p.get("test") for p in problems))

    def test_filter_single(self) -> None:
        _, problems = load_problems()
        got = filter_problems(problems, "strlen")
        self.assertEqual(len(got), 1)
        self.assertEqual(got[0]["id"], "strlen")

    def test_extract_fence(self) -> None:
        raw = "Sure.\n```python\ndef strlen(string: str) -> int:\n    return len(string)\n```\n"
        code = extract_python_code(raw, "strlen")
        self.assertIn("def strlen", code)
        self.assertNotIn("```", code)

    def test_run_harness_pass_fail(self) -> None:
        _, problems = load_problems()
        p = next(x for x in problems if x["id"] == "strlen")
        ok, _ = run_harness(
            "def strlen(string: str) -> int:\n    return len(string)\n",
            p["test"],
            "strlen",
        )
        self.assertTrue(ok)
        ok2, detail = run_harness(
            "def strlen(string: str) -> int:\n    return 0\n",
            p["test"],
            "strlen",
        )
        self.assertFalse(ok2)
        self.assertTrue(detail)

    def test_build_harness_contains_check(self) -> None:
        src = build_harness_source("def f():\n    return 1\n", "def check(c):\n    assert c()==1\n", "f")
        self.assertIn("check(f)", src)

    def test_tokens_from_ollama(self) -> None:
        pt, ct = tokens_from_ollama({"prompt_eval_count": 11, "eval_count": 7})
        self.assertEqual((pt, ct), (11, 7))
        pt2, ct2 = tokens_from_ollama({})
        self.assertEqual((pt2, ct2), (None, None))


if __name__ == "__main__":
    unittest.main()
