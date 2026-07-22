"""Backward-compatible telemetry envelope for live evaluation runners."""

from __future__ import annotations

import time
from typing import Any

SCHEMA_VERSION = 1


def new_run(kind: str, scenario: str) -> dict[str, Any]:
    return {
        "schema_version": SCHEMA_VERSION,
        "kind": kind,
        "scenario": scenario,
        "started_monotonic": time.monotonic(),
        "attempts": 0,
        "passed_at_1": False,
        "eventual_pass": False,
        "retry_reasons": [],
        "nudge_reasons": [],
        "actual_provider": None,
        "actual_model": None,
        "validation_failures": [],
        "escalation_reasons": [],
        "repair_attempts": None,
        "tool_calls": None,
        "ttft_ms": None,
    }


def record_reason(run: dict[str, Any], field: str, reason: Any) -> None:
    text = str(reason or "").strip()
    values = run.setdefault(field, [])
    if text and isinstance(values, list) and text not in values:
        values.append(text)


def _first(mapping: dict[str, Any], names: tuple[str, ...]) -> Any:
    for name in names:
        value = mapping.get(name)
        if value not in (None, ""):
            return value
    return None


def absorb_metadata(run: dict[str, Any], metadata: Any) -> None:
    """Copy only bounded quality telemetry from persisted message metadata."""
    if not isinstance(metadata, dict):
        return
    outcome = metadata.get("implementation_session_outcome")
    sources = [metadata]
    if isinstance(outcome, dict):
        sources.append(outcome)

    for source in sources:
        provider = _first(
            source,
            (
                "actual_provider",
                "routing_provider_id",
                "provider_id",
                "task_provider_id",
                "provider",
            ),
        )
        model = _first(
            source,
            (
                "actual_model",
                "routing_model",
                "routing_tool_model",
                "model_id",
                "model",
                "model_tag",
            ),
        )
        if provider is not None:
            run["actual_provider"] = str(provider)[:120]
        if model is not None:
            run["actual_model"] = str(model)[:200]

        for key in ("validation_failures", "validation_errors"):
            raw = source.get(key)
            if isinstance(raw, list):
                for item in raw:
                    record_reason(run, "validation_failures", item)
            elif raw:
                record_reason(run, "validation_failures", raw)
        validation = source.get("validation_result")
        if isinstance(validation, dict) and validation.get("passed") is False:
            record_reason(
                run,
                "validation_failures",
                validation.get("reason") or validation.get("error") or "validation failed",
            )
        if source.get("verify_failed") is True:
            record_reason(run, "validation_failures", "implementation verification failed")

        for key in ("escalation_reasons", "model_escalation_reasons"):
            raw = source.get(key)
            if isinstance(raw, list):
                for item in raw:
                    record_reason(run, "escalation_reasons", item)
            elif raw:
                record_reason(run, "escalation_reasons", raw)
        if source.get("escalation_reason"):
            record_reason(run, "escalation_reasons", source["escalation_reason"])
        attempts = source.get("routing_attempts")
        if isinstance(attempts, list):
            for index, attempt in enumerate(attempts):
                if not isinstance(attempt, dict):
                    continue
                if index == len(attempts) - 1:
                    if attempt.get("provider_id"):
                        run["actual_provider"] = str(attempt["provider_id"])[:120]
                    if attempt.get("model"):
                        run["actual_model"] = str(attempt["model"])[:200]
                failure = attempt.get("failure_reason")
                if failure:
                    record_reason(run, "escalation_reasons", failure)
        failure_evidence = source.get("routing_failure_evidence")
        if isinstance(failure_evidence, list):
            for item in failure_evidence:
                record_reason(run, "validation_failures", item)

        repairs = _first(source, ("repair_attempts", "repairs"))
        if isinstance(repairs, (int, float)):
            run["repair_attempts"] = int(repairs)
        tools = _first(source, ("tool_calls", "tool_call_count"))
        if isinstance(tools, (int, float)):
            run["tool_calls"] = int(tools)
        ttft = source.get("ttft_ms")
        if isinstance(ttft, (int, float)):
            run["ttft_ms"] = float(ttft)

        tool_steps = source.get("tool_steps")
        if isinstance(tool_steps, list):
            run["tool_calls"] = max(int(run.get("tool_calls") or 0), len(tool_steps))


def absorb_messages(run: dict[str, Any], messages: list[dict[str, Any]]) -> None:
    for message in messages:
        if isinstance(message, dict):
            absorb_metadata(run, message.get("metadata"))


def finish(run: dict[str, Any], *, eventual_pass: bool) -> dict[str, Any]:
    attempts = max(1, int(run.get("attempts") or 0))
    result = {
        key: value
        for key, value in run.items()
        if key != "started_monotonic"
    }
    result["attempts"] = attempts
    result["passed_at_1"] = bool(run.get("passed_at_1"))
    result["eventual_pass"] = bool(eventual_pass)
    result["retry_count"] = len(result.get("retry_reasons") or [])
    result["nudge_count"] = len(result.get("nudge_reasons") or [])
    result["escalation_count"] = len(result.get("escalation_reasons") or [])
    result["wall_duration_ms"] = round(
        max(0.0, time.monotonic() - float(run.get("started_monotonic") or time.monotonic()))
        * 1000,
        3,
    )
    return result


def metrics_payload(telemetry: dict[str, Any], extra: dict[str, Any] | None = None) -> dict[str, Any]:
    payload = dict(extra or {})
    for key in (
        "passed_at_1",
        "eventual_pass",
        "attempts",
        "retry_count",
        "retry_reasons",
        "nudge_count",
        "nudge_reasons",
        "actual_provider",
        "actual_model",
        "validation_failures",
        "escalation_count",
        "escalation_reasons",
        "repair_attempts",
        "tool_calls",
        "ttft_ms",
        "wall_duration_ms",
    ):
        if key not in payload or payload[key] is None:
            payload[key] = telemetry.get(key)
    return payload
