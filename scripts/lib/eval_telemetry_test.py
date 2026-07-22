from __future__ import annotations

import unittest

from eval_telemetry import absorb_metadata, finish, metrics_payload, new_run, record_reason


class EvalTelemetryTest(unittest.TestCase):
    def test_collects_available_runtime_metadata(self) -> None:
        run = new_run("implement", "example")
        run["attempts"] = 2
        record_reason(run, "retry_reasons", "timeout")
        record_reason(run, "nudge_reasons", "silent agent")
        absorb_metadata(
            run,
            {
                "routing_provider_id": "ollama",
                "routing_model": "coder:14b",
                "routing_attempts": [
                    {"provider_id": "ollama", "model": "coder:7b", "failure_reason": "invalid tool call"},
                    {"provider_id": "ollama", "model": "coder:14b"}
                ],
                "tool_steps": [{"name": "read"}, {"name": "edit"}],
                "implementation_session_outcome": {
                    "repair_attempts": 1,
                    "validation_failures": ["first verification failed"],
                    "escalation_reason": "tool model required",
                    "ttft_ms": 25,
                },
            },
        )
        report = finish(run, eventual_pass=True)
        payload = metrics_payload(report)
        self.assertFalse(payload["passed_at_1"])
        self.assertTrue(payload["eventual_pass"])
        self.assertEqual(payload["retry_reasons"], ["timeout"])
        self.assertEqual(payload["nudge_reasons"], ["silent agent"])
        self.assertEqual(payload["actual_provider"], "ollama")
        self.assertEqual(payload["actual_model"], "coder:14b")
        self.assertEqual(payload["tool_calls"], 2)
        self.assertEqual(payload["repair_attempts"], 1)
        self.assertEqual(payload["validation_failures"], ["first verification failed"])
        self.assertEqual(
            payload["escalation_reasons"],
            ["invalid tool call", "tool model required"],
        )


if __name__ == "__main__":
    unittest.main()
