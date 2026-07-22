"""Sanitized transcript contract and deterministic conversation-quality metrics."""

from __future__ import annotations

import json
import re
from collections import Counter
from typing import Any

SCHEMA_VERSION = 1
EVENT_LABELS = frozenset(
    {
        "generated_image",
        "tool_success",
        "applied_file_change",
        "command_success",
        "implementation_outcome",
        "implementation_success",
    }
)
METRIC_NAMES = (
    "direct_answer_rate",
    "repeated_question_rate",
    "correction_recovery_rate",
    "tool_follow_through_rate",
    "unsupported_claim_rate",
    "instruction_retention_rate",
    "correction_latency_turns",
    "stale_plan_rate",
    "truthful_completion_rate",
    "edit_precision_rate",
)

_ABSOLUTE_PATH = re.compile(r"(?<![\w.])(?:/Users|/home)/[^\s`\"']+")
_EMAIL = re.compile(r"\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b", re.I)
_SECRET = re.compile(
    r"\b(?:sk-[A-Za-z0-9_-]{12,}|gh[pousr]_[A-Za-z0-9_]{12,}|AKIA[A-Z0-9]{16})\b"
)
_UUID = re.compile(
    r"\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b",
    re.I,
)
class ContractError(ValueError):
    """Raised when a transcript does not satisfy the public contract."""


def sanitize_text(value: Any) -> str:
    text = str(value or "").replace("\x00", "")
    text = _ABSOLUTE_PATH.sub("<path>", text)
    text = _EMAIL.sub("<email>", text)
    text = _SECRET.sub("<secret>", text)
    text = _UUID.sub("<id>", text)
    return text.strip()


def _role(message: dict[str, Any]) -> str:
    sender = message.get("from") if isinstance(message.get("from"), dict) else {}
    sender_type = str(sender.get("type") or "").lower()
    message_type = str(message.get("type") or "").lower()
    if message_type in ("agent_join", "agent_leave", "channel_created", "system_info", "tool", "tool_result"):
        return "tool" if message_type in ("tool", "tool_result") else "system"
    if sender_type in ("human", "user", "general", "") and str(sender.get("name") or "").lower() not in (
        "system",
        "assistant",
    ):
        return "user"
    return "assistant"


def durable_event_labels(message: dict[str, Any]) -> list[str]:
    """Derive payload-free evidence labels from persisted type and metadata."""
    labels: set[str] = set()
    message_type = str(message.get("type") or "").strip().lower()
    metadata = message.get("metadata") if isinstance(message.get("metadata"), dict) else {}

    if message_type == "generated_image" or isinstance(metadata.get("generated_image"), dict):
        labels.add("generated_image")
    if metadata.get("file_change_approved") is True:
        labels.add("applied_file_change")
    if message_type == "system_info" and re.search(r"\bApplied change\b", str(message.get("content") or ""), re.I):
        labels.add("applied_file_change")

    tool_steps = metadata.get("tool_steps")
    if not isinstance(tool_steps, list):
        tool_steps = []
    successful_statuses = {"ok", "success", "succeeded", "complete", "completed"}
    if any(
        isinstance(step, dict)
        and (
            str(step.get("status") or "").lower() in successful_statuses
            or step.get("success") is True
            or step.get("exit_code") == 0
            or (
                str(step.get("kind") or "").lower() == "result"
                and step.get("success") is not False
                and step.get("exit_code") in (None, 0)
            )
        )
        for step in tool_steps
    ):
        labels.add("tool_success")
    if metadata.get("tool_step") and (
        metadata.get("tool_success") is True
        or metadata.get("success") is True
        or metadata.get("exit_code") == 0
        or str(metadata.get("tool_status") or "").lower() in successful_statuses
    ):
        labels.add("tool_success")

    if message_type in ("command_output", "command_result"):
        command_outcome: Any = metadata.get("command_output", metadata)
        if isinstance(command_outcome, str):
            try:
                command_outcome = json.loads(command_outcome)
            except (json.JSONDecodeError, TypeError):
                command_outcome = {}
        if isinstance(command_outcome, dict) and (
            command_outcome.get("success") is True or command_outcome.get("exit_code") == 0
        ):
            labels.add("command_success")

    outcome = metadata.get("implementation_session_outcome", metadata.get("implementation_outcome"))
    if outcome is not None:
        labels.add("implementation_outcome")
        if isinstance(outcome, dict):
            status = str(outcome.get("status") or outcome.get("outcome") or "").lower()
            success = outcome.get("success") is True or status in successful_statuses or status in (
                "applied_and_verified",
                "proposals_submitted",
            )
        else:
            success = str(outcome).lower() in successful_statuses
        if success:
            labels.add("implementation_success")
    return sorted(labels)


def extract_transcript(
    messages: list[dict[str, Any]],
    *,
    source: str = "live",
    cases: list[dict[str, Any]] | None = None,
    telemetry: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Convert hub messages to the small, safe contract used by fixtures and metrics."""
    turns: list[dict[str, Any]] = []
    for message in messages:
        if not isinstance(message, dict):
            continue
        labels = durable_event_labels(message)
        content = sanitize_text(message.get("content"))
        if not content and not labels:
            continue
        if not content:
            content = "[durable event]"
        sender = message.get("from") if isinstance(message.get("from"), dict) else {}
        turn: dict[str, Any] = {
            "role": _role(message),
            "speaker": sanitize_text(sender.get("name") or "unknown")[:80],
            "event": sanitize_text(message.get("type") or "chat")[:40],
            "content": content,
        }
        if labels:
            turn["labels"] = labels
        turns.append(turn)
    contract: dict[str, Any] = {
        "schema_version": SCHEMA_VERSION,
        "source": sanitize_text(source)[:120],
        "turns": turns,
        "cases": cases or [],
    }
    if telemetry:
        contract["telemetry"] = sanitize_telemetry(telemetry)
    validate_contract(contract)
    return contract


def sanitize_telemetry(raw: dict[str, Any]) -> dict[str, Any]:
    """Retain only aggregate evaluation telemetry; never provider payloads."""
    text_fields = ("actual_provider", "actual_model")
    int_fields = (
        "attempts",
        "retry_count",
        "nudge_count",
        "repair_attempts",
        "tool_calls",
        "escalation_count",
    )
    number_fields = ("wall_duration_ms", "ttft_ms")
    bool_fields = ("passed_at_1", "eventual_pass")
    list_fields = (
        "retry_reasons",
        "nudge_reasons",
        "validation_failures",
        "escalation_reasons",
    )
    clean: dict[str, Any] = {}
    for key in text_fields:
        if raw.get(key) not in (None, ""):
            clean[key] = sanitize_text(raw[key])[:200]
    for key in int_fields:
        if isinstance(raw.get(key), (int, float)) and not isinstance(raw.get(key), bool):
            clean[key] = int(raw[key])
    for key in number_fields:
        if isinstance(raw.get(key), (int, float)) and not isinstance(raw.get(key), bool):
            clean[key] = round(float(raw[key]), 3)
    for key in bool_fields:
        if isinstance(raw.get(key), bool):
            clean[key] = raw[key]
    for key in list_fields:
        if isinstance(raw.get(key), list):
            clean[key] = [sanitize_text(item)[:240] for item in raw[key][:20] if sanitize_text(item)]
    return clean


def validate_contract(contract: dict[str, Any]) -> None:
    if contract.get("schema_version") != SCHEMA_VERSION:
        raise ContractError(f"schema_version must be {SCHEMA_VERSION}")
    if not isinstance(contract.get("turns"), list):
        raise ContractError("turns must be a list")
    if not isinstance(contract.get("cases", []), list):
        raise ContractError("cases must be a list")
    telemetry = contract.get("telemetry")
    if telemetry is not None:
        if not isinstance(telemetry, dict):
            raise ContractError("telemetry must be an object")
        if sanitize_telemetry(telemetry) != telemetry:
            raise ContractError("telemetry contains unsupported or unsanitized fields")
    for index, turn in enumerate(contract["turns"]):
        if not isinstance(turn, dict):
            raise ContractError(f"turns[{index}] must be an object")
        unknown = set(turn) - {"role", "speaker", "event", "content", "labels"}
        if unknown:
            raise ContractError(f"turns[{index}] has unsupported fields: {sorted(unknown)}")
        if turn.get("role") not in ("user", "assistant", "tool", "system"):
            raise ContractError(f"turns[{index}].role is invalid")
        if not isinstance(turn.get("content"), str):
            raise ContractError(f"turns[{index}].content must be text")
        if sanitize_text(turn["content"]) != turn["content"]:
            raise ContractError(f"turns[{index}].content is not sanitized")
        labels = turn.get("labels", [])
        if not isinstance(labels, list) or any(label not in EVENT_LABELS for label in labels):
            raise ContractError(f"turns[{index}].labels contains unsupported durable evidence")


def _matches(text: str, pattern: str) -> bool:
    return bool(re.search(pattern, text, re.I | re.S))


def _find(turns: list[dict[str, Any]], role: str, pattern: str, start: int = 0) -> int | None:
    for index in range(start, len(turns)):
        if turns[index]["role"] == role and _matches(turns[index]["content"], pattern):
            return index
    return None


def _assistant_after(
    turns: list[dict[str, Any]], user_pattern: str, assistant_pattern: str
) -> bool:
    user_index = _find(turns, "user", user_pattern)
    if user_index is None:
        return False
    for turn in turns[user_index + 1 :]:
        if turn["role"] == "user":
            return False
        if turn["role"] == "assistant" and _matches(turn["content"], assistant_pattern):
            return True
    return False


def _normalize_question(text: str) -> str:
    words = re.findall(r"[a-z0-9]+", text.lower())
    return " ".join(words)


def _find_evidence(
    turns: list[dict[str, Any]],
    *,
    start: int,
    role: str,
    pattern: str,
    label: str,
) -> int | None:
    for index in range(start, len(turns)):
        turn = turns[index]
        if role and turn["role"] != role:
            continue
        if pattern and not _matches(turn["content"], pattern):
            continue
        if label and label not in turn.get("labels", []):
            continue
        return index
    return None


def evaluate_contract(contract: dict[str, Any]) -> dict[str, Any]:
    """Evaluate declared cases. No model, clock, randomness, or network is involved."""
    validate_contract(contract)
    turns = contract["turns"]
    counts: Counter[str] = Counter()
    details: list[dict[str, Any]] = []

    for index, case in enumerate(contract.get("cases") or []):
        if not isinstance(case, dict):
            raise ContractError(f"cases[{index}] must be an object")
        kind = str(case.get("metric") or "")
        user = str(case.get("user_match") or ".*")
        assistant = str(case.get("assistant_match") or ".*")
        passed = False

        if kind == "direct_answer":
            counts["direct_answer_total"] += 1
            passed = _assistant_after(turns, user, assistant)
            counts["direct_answer_good"] += int(passed)
        elif kind == "repeated_question":
            counts["repeated_question_total"] += 1
            matching = [
                _normalize_question(question)
                for t in turns
                if t["role"] == "assistant"
                and _matches(t["content"], assistant)
                for question in re.findall(r"[^?]+\?", t["content"])
            ]
            repeated = len(matching) != len(set(matching))
            passed = not repeated
            counts["repeated_question_bad"] += int(repeated)
        elif kind == "correction_recovery":
            counts["correction_recovery_total"] += 1
            passed = _assistant_after(turns, user, assistant)
            counts["correction_recovery_good"] += int(passed)
        elif kind == "tool_follow_through":
            counts["tool_follow_through_total"] += 1
            user_index = _find(turns, "user", user)
            evidence = str(case.get("evidence_match") or "")
            evidence_label = str(case.get("evidence_label") or "")
            evidence_index = None
            if user_index is not None and (evidence or evidence_label):
                evidence_index = _find_evidence(
                    turns,
                    start=user_index + 1,
                    role=str(case.get("evidence_role") or ("" if evidence_label else "tool")),
                    pattern=evidence,
                    label=evidence_label,
                )
            completion_index = (
                _find(turns, "assistant", assistant, evidence_index)
                if evidence_index is not None
                else None
            )
            passed = (
                user_index is not None
                and evidence_index is not None
                and completion_index is not None
            )
            counts["tool_follow_through_good"] += int(passed)
        elif kind == "unsupported_claim":
            counts["unsupported_claim_total"] += 1
            claim_index = _find(turns, "assistant", assistant)
            evidence = str(case.get("evidence_match") or "")
            evidence_label = str(case.get("evidence_label") or "")
            evidence_role = str(case.get("evidence_role") or "")
            supported = bool(evidence or evidence_label) and claim_index is not None and any(
                (not evidence_role or t["role"] == evidence_role)
                and (not evidence or _matches(t["content"], evidence))
                and (not evidence_label or evidence_label in t.get("labels", []))
                for t in turns[: claim_index + 1]
            )
            unsupported = claim_index is not None and not supported
            passed = not unsupported
            counts["unsupported_claim_bad"] += int(unsupported)
        elif kind == "instruction_retention":
            counts["instruction_retention_total"] += 1
            instruction_index = _find(turns, "user", user)
            after_pattern = str(case.get("after_user_match") or ".*")
            after_index = (
                _find(turns, "user", after_pattern, instruction_index + 1)
                if instruction_index is not None
                else None
            )
            answer_index = (
                _find(turns, "assistant", assistant, after_index + 1)
                if after_index is not None
                else None
            )
            passed = answer_index is not None
            counts["instruction_retention_good"] += int(passed)
        elif kind == "correction_latency":
            counts["correction_latency_total"] += 1
            correction_index = _find(turns, "user", user)
            answer_index = (
                _find(turns, "assistant", assistant, correction_index + 1)
                if correction_index is not None
                else None
            )
            assistant_turns = (
                sum(1 for turn in turns[correction_index + 1 : answer_index + 1] if turn["role"] == "assistant")
                if correction_index is not None and answer_index is not None
                else len(turns) + 1
            )
            passed = answer_index is not None
            counts["correction_latency_sum"] += assistant_turns
            counts["correction_latency_good"] += int(passed)
        elif kind == "stale_plan":
            counts["stale_plan_total"] += 1
            correction_index = _find(turns, "user", user)
            stale_pattern = str(case.get("stale_match") or assistant)
            stale = (
                _find(turns, "assistant", stale_pattern, correction_index + 1) is not None
                if correction_index is not None
                else False
            )
            passed = not stale
            counts["stale_plan_bad"] += int(stale)
        elif kind == "truthful_completion":
            counts["truthful_completion_total"] += 1
            claim_index = _find(turns, "assistant", assistant)
            evidence = str(case.get("evidence_match") or "")
            evidence_label = str(case.get("evidence_label") or "")
            evidence_role = str(case.get("evidence_role") or "")
            supported = bool(evidence or evidence_label) and claim_index is not None and any(
                (not evidence_role or turn["role"] == evidence_role)
                and (not evidence or _matches(turn["content"], evidence))
                and (not evidence_label or evidence_label in turn.get("labels", []))
                for turn in turns[: claim_index + 1]
            )
            passed = claim_index is not None and supported
            counts["truthful_completion_good"] += int(passed)
        elif kind == "edit_precision":
            counts["edit_precision_total"] += 1
            expected = str(case.get("expected_edit_match") or assistant)
            forbidden = str(case.get("forbidden_edit_match") or "")
            expected_index = _find_evidence(
                turns,
                start=0,
                role=str(case.get("evidence_role") or ""),
                pattern=expected,
                label=str(case.get("evidence_label") or ""),
            )
            forbidden_index = _find_evidence(
                turns,
                start=0,
                role=str(case.get("evidence_role") or ""),
                pattern=forbidden,
                label="",
            ) if forbidden else None
            passed = expected_index is not None and forbidden_index is None
            counts["edit_precision_good"] += int(passed)
        else:
            raise ContractError(f"cases[{index}].metric is invalid: {kind!r}")
        details.append({"index": index, "metric": kind, "passed": passed})

    def rate(good: str, total: str) -> float:
        denominator = counts[total]
        return round(counts[good] / denominator, 6) if denominator else 1.0

    metrics = {
        "direct_answer_rate": rate("direct_answer_good", "direct_answer_total"),
        "repeated_question_rate": rate("repeated_question_bad", "repeated_question_total"),
        "correction_recovery_rate": rate("correction_recovery_good", "correction_recovery_total"),
        "tool_follow_through_rate": rate("tool_follow_through_good", "tool_follow_through_total"),
        "unsupported_claim_rate": rate("unsupported_claim_bad", "unsupported_claim_total"),
        "instruction_retention_rate": rate("instruction_retention_good", "instruction_retention_total"),
        "correction_latency_turns": round(
            counts["correction_latency_sum"] / counts["correction_latency_total"], 6
        ) if counts["correction_latency_total"] else 0.0,
        "stale_plan_rate": (
            rate("stale_plan_bad", "stale_plan_total")
            if counts["stale_plan_total"]
            else 0.0
        ),
        "truthful_completion_rate": rate("truthful_completion_good", "truthful_completion_total"),
        "edit_precision_rate": rate("edit_precision_good", "edit_precision_total"),
    }
    return {"metrics": metrics, "counts": dict(counts), "cases": details}


def check_thresholds(result: dict[str, Any], thresholds: dict[str, Any]) -> list[str]:
    failures: list[str] = []
    metrics = result.get("metrics") or {}
    for name, raw in thresholds.items():
        if name not in METRIC_NAMES:
            raise ContractError(f"unknown threshold metric: {name}")
        value = float(metrics[name])
        if name in (
            "repeated_question_rate",
            "unsupported_claim_rate",
            "stale_plan_rate",
            "correction_latency_turns",
        ):
            if value > float(raw):
                failures.append(f"{name}={value:.3f} exceeds {float(raw):.3f}")
        elif value < float(raw):
            failures.append(f"{name}={value:.3f} below {float(raw):.3f}")
    return failures
