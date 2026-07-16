"""Unit tests for model benchmark report helpers."""

from __future__ import annotations

import sys
import unittest
import json
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parents[1]
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.model_benchmark import (  # noqa: E402
    ModelBenchmarkResult,
    ScenarioResult,
    SuiteTracks,
    derive_capability_profiles,
    filter_models_by_max_size_gb,
    format_duration,
    load_models,
    model_is_installed,
    model_pull_tag,
    model_requires_hf_import,
    model_runtime_tag,
    parse_judge_from_output,
    parse_metrics_from_output,
    render_markdown_report,
    resolve_suite_max_size_gb,
    resolve_suite_scenarios,
)


class TestModelBenchmark(unittest.TestCase):
    def test_format_duration(self) -> None:
        self.assertEqual(format_duration(45), "45s")
        self.assertEqual(format_duration(125), "2m05s")

    def test_model_is_installed(self) -> None:
        installed = {"qwen2.5-coder:14b", "qwen3.5:9b"}
        self.assertTrue(model_is_installed(installed, "qwen2.5-coder:14b"))
        self.assertFalse(model_is_installed(installed, "codestral:22b"))

    def test_model_pull_and_runtime_tags(self) -> None:
        ornith = {
            "tag": "nj-ornith:9b",
            "pull_tag": "hf.co/deepreinforce-ai/Ornith-1.0-9B-GGUF:Q4_K_M",
        }
        self.assertEqual(
            model_pull_tag(ornith),
            "hf.co/deepreinforce-ai/Ornith-1.0-9B-GGUF:Q4_K_M",
        )
        installed = {"hf.co/deepreinforce-ai/Ornith-1.0-9B-GGUF:Q4_K_M"}
        self.assertTrue(model_is_installed(installed, ornith["tag"], pull_tag=ornith["pull_tag"]))
        self.assertEqual(model_runtime_tag(ornith, installed), ornith["pull_tag"])
        installed_with_brand = {"nj-ornith:9b"}
        self.assertEqual(model_runtime_tag(ornith, installed_with_brand), "nj-ornith:9b")

    def test_bonsai_under_default_memory_cap(self) -> None:
        models = load_models()
        by_id = {str(m.get("id") or ""): m for m in models}
        self.assertIn("bonsai-27b", by_id)
        self.assertIn("ternary-bonsai-27b", by_id)
        bonsai = by_id["bonsai-27b"]
        self.assertEqual(bonsai["tag"], "nj-bonsai:27b")
        self.assertFalse(model_requires_hf_import(bonsai))
        self.assertEqual(model_pull_tag(bonsai), "hf.co/prism-ml/Bonsai-27B-gguf:Q1_0")
        ternary = by_id["ternary-bonsai-27b"]
        self.assertTrue(model_requires_hf_import(ternary))
        self.assertEqual(model_pull_tag(ternary), "")
        self.assertEqual(ternary.get("hf_filename"), "Ternary-Bonsai-27B-Q2_0.gguf")
        self.assertLessEqual(float(bonsai["size_hint_gb"]), 9.0)
        capped = {str(m.get("id") or "") for m in filter_models_by_max_size_gb(models, 9.0)}
        self.assertIn("bonsai-27b", capped)
        self.assertIn("ternary-bonsai-27b", capped)
        self.assertNotIn("codestral-22b", capped)
        self.assertNotIn("devstral-24b", capped)
        tiny = {str(m.get("id") or "") for m in filter_models_by_max_size_gb(models, 3.0)}
        self.assertNotIn("bonsai-27b", tiny)

    def test_ollama_installed_tags_from_object(self) -> None:
        from lib.model_benchmark import ollama_installed_tags
        import unittest.mock as mock

        with mock.patch("lib.collab_hub.hub_request", return_value=(200, {"models": ["codestral:22b", "devstral:24b"]})):
            tags = ollama_installed_tags("http://127.0.0.1:18765")
        self.assertIn("codestral:22b", tags)
        self.assertIn("devstral:24b", tags)

    def test_resolve_quick_suite(self) -> None:
        suite = {
            "implement": ["go-handler", "theme-toggle"],
            "chat": ["dm-backend-workspace"],
        }
        tracks = resolve_suite_scenarios(suite)
        self.assertIsInstance(tracks, SuiteTracks)
        self.assertEqual(tracks.implement, ["go-handler", "theme-toggle"])
        self.assertEqual(tracks.chat, ["dm-backend-workspace"])
        self.assertEqual(tracks.collab, [])
        self.assertEqual(tracks.arena, [])
        # backward compat unpack
        impl, chat = tracks
        self.assertEqual(impl, tracks.implement)
        self.assertEqual(chat, tracks.chat)

    def test_resolve_collab_core_suite(self) -> None:
        from lib.collab_core_scenarios import collab_core_scenarios

        tracks = resolve_suite_scenarios({"collab": "core", "implement": [], "chat": []})
        self.assertEqual(tracks.collab, collab_core_scenarios())
        self.assertFalse(tracks.implement)
        self.assertTrue(tracks.has_any())

    def test_resolve_suite_max_size_fallback(self) -> None:
        self.assertEqual(resolve_suite_max_size_gb({}), 9.0)

    def test_parse_judge_warn_and_score(self) -> None:
        verdict, reason = parse_judge_from_output("  ✓ judge:warn:SCORE: 0.85\nthin deliverable")
        self.assertIs(verdict, False)
        self.assertIn("SCORE: 0.85", reason)
        from lib.model_benchmark import parse_quality_score

        self.assertEqual(parse_quality_score(reason), 0.85)

    def test_parse_metrics_json(self) -> None:
        out = 'ok\nMETRICS_JSON:{"prompt_tokens":10,"completion_tokens":20,"ttft_ms":12.5,"tok_per_s":33.1,"repair_attempts":1}\n'
        m = parse_metrics_from_output(out)
        self.assertIsNotNone(m)
        assert m is not None
        self.assertEqual(m.prompt_tokens, 10)
        self.assertEqual(m.completion_tokens, 20)
        self.assertEqual(m.repair_attempts, 1)

    def test_render_markdown_includes_winner(self) -> None:
        r = ModelBenchmarkResult(
            model_id="a",
            model_tag="qwen2.5-coder:14b",
            title="Qwen 2.5 Coder 14B",
            size_hint_gb=9,
            scenarios=[
                ScenarioResult("go-handler", "implement", True, 120.0),
                ScenarioResult("dm-backend-workspace", "chat", True, 30.0),
            ],
            total_duration_s=150.0,
        )
        md = render_markdown_report(
            suite_name="quick",
            suite_desc="smoke",
            hub_url="http://127.0.0.1:18765",
            results=[r],
            implement_names=["go-handler"],
            chat_names=["dm-backend-workspace"],
        )
        self.assertIn("qwen2.5-coder:14b", md)
        self.assertIn("winner", md)

    def test_to_dict_pass_rate_helpers(self) -> None:
        r = ModelBenchmarkResult(
            model_id="a",
            model_tag="qwen2.5-coder:14b",
            title="Qwen",
            size_hint_gb=9,
            scenarios=[
                ScenarioResult(
                    "a",
                    "implement",
                    True,
                    1.0,
                    structural_passed=True,
                    quality_passed=True,
                    capability_passed=True,
                ),
                ScenarioResult(
                    "b",
                    "implement",
                    True,
                    1.0,
                    structural_passed=True,
                    quality_passed=False,
                ),
            ],
        )
        d = r.to_dict()
        self.assertEqual(d["structural_pass_rate"], 1.0)
        self.assertEqual(d["quality_pass_rate"], 0.5)
        self.assertEqual(d["composite_pass_rate"], 0.5)

    def test_derive_capability_profiles(self) -> None:
        fixture_path = ROOT / "docs/archive/testing/model-benchmark-quick-2026-06-19-0639.json"
        if not fixture_path.is_file():
            fixture_path = ROOT / "docs/testing/model-benchmark-quick-2026-06-29-1755.json"
        fixture = json.loads(fixture_path.read_text(encoding="utf-8"))
        run_id = "quick-2026-06-19-0639" if "2026-06-19" in fixture_path.name else fixture_path.stem.replace("model-benchmark-", "")
        catalog = {"runs": [{"id": run_id, **fixture}]}
        roster = [
            {"tag": "qwen2.5-coder:14b", "params_b": 14},
            {"tag": "deepseek-coder:6.7b", "params_b": 6.7},
            {"tag": "qwen3.5:9b", "params_b": 9},
        ]
        profiles = derive_capability_profiles(catalog, roster)
        self.assertEqual(profiles["source_run_id"], run_id)
        self.assertTrue(profiles["task_classes"]["implement"])
        self.assertIn("qwen3.5:9b", profiles["task_classes"]["utility"])
        self.assertNotIn("qwen2.5-coder:14b", profiles["task_classes"]["utility"])
        self.assertIn("deepseek-coder:6.7b", profiles["task_classes"]["ask_mode"])
        if "2026-06-19" in fixture_path.name:
            self.assertEqual(profiles["task_classes"]["implement"][0], "qwen2.5-coder:14b")


if __name__ == "__main__":
    unittest.main()
