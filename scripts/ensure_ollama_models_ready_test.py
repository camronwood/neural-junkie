"""Tests for Ollama model readiness helpers used by overnight."""

from __future__ import annotations

import argparse
import importlib.util
import sys
import unittest
from pathlib import Path
from unittest import mock

SCRIPTS = Path(__file__).resolve().parent
_SPEC = importlib.util.spec_from_file_location(
    "ensure_ollama_models_ready",
    SCRIPTS / "ensure-ollama-models-ready.py",
)
assert _SPEC and _SPEC.loader
ready = importlib.util.module_from_spec(_SPEC)
sys.modules["ensure_ollama_models_ready"] = ready
_SPEC.loader.exec_module(ready)


class ResolveTagsTest(unittest.TestCase):
    def test_release_suite_adds_benchmark_roster(self) -> None:
        args = argparse.Namespace(
            models="qwen2.5-coder:14b",
            skip_benchmark=False,
            suite="release",
            benchmark_models=None,
            allow_large_models=False,
        )
        warm, pull = ready.resolve_tags(args)
        self.assertEqual(warm, ["qwen2.5-coder:14b"])
        self.assertIn("qwen2.5-coder:14b", pull)
        self.assertIn("gemma3:12b", pull)

    def test_skip_benchmark_omits_suite_roster(self) -> None:
        args = argparse.Namespace(
            models="qwen2.5-coder:14b",
            skip_benchmark=True,
            suite="release",
            benchmark_models=None,
            allow_large_models=False,
        )
        warm, pull = ready.resolve_tags(args)
        self.assertEqual(warm, ["qwen2.5-coder:14b"])
        self.assertEqual(pull, ["qwen2.5-coder:14b"])


class SuiteMissingFailFastTest(unittest.TestCase):
    def test_missing_suite_model_fails_without_pull(self) -> None:
        """NO_PULL overnight must not boot green when gemma3:12b is absent."""
        argv = [
            "ensure-ollama-models-ready.py",
            "--models",
            "qwen2.5-coder:14b",
            "--suite",
            "release",
            "--warm",
        ]
        installed = {"qwen2.5-coder:14b", "qwen2.5-coder:14b:latest"}

        with mock.patch.object(ready.sys, "argv", argv), mock.patch.object(
            ready, "installed_tags", return_value=installed
        ), mock.patch.object(ready, "_ollama_get", return_value={"models": []}), mock.patch.object(
            ready, "warm_tag", return_value=(True, "ok")
        ), mock.patch.object(ready, "model_is_installed") as mis:

            def _mis(_installed, tag, pull_tag=""):
                return tag.startswith("qwen2.5-coder")

            mis.side_effect = _mis
            rc = ready.main()
        self.assertEqual(rc, 1)


if __name__ == "__main__":
    unittest.main()
