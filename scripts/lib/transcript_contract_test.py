from __future__ import annotations

import unittest

from transcript_contract import (
    ContractError,
    check_thresholds,
    evaluate_contract,
    extract_transcript,
    sanitize_text,
    validate_contract,
)


class TranscriptContractTest(unittest.TestCase):
    def test_extractor_redacts_sensitive_values_and_drops_metadata(self) -> None:
        contract = extract_transcript(
            [
                {
                    "type": "question",
                    "from": {"name": "person@example.com", "type": "human", "id": "private"},
                    "content": "Read /Users/alice/private/a.md with sk-abcdefghijklmnop",
                    "metadata": {"token": "do-not-copy"},
                }
            ],
            source="/Users/alice/session.json",
        )
        encoded = str(contract)
        self.assertNotIn("person@example.com", encoded)
        self.assertNotIn("/Users/alice", encoded)
        self.assertNotIn("sk-abcdefghijklmnop", encoded)
        self.assertNotIn("do-not-copy", encoded)
        self.assertEqual(contract["turns"][0]["role"], "user")

    def test_extractor_preserves_only_safe_durable_metadata_evidence(self) -> None:
        contract = extract_transcript(
            [
                {
                    "type": "chat",
                    "from": {"name": "Agent", "type": "assistant"},
                    "content": "Here is the generated cover.",
                    "metadata": {
                        "generated_image": {
                            "mime": "image/png",
                            "data": "private-base64-payload",
                            "path": "/Users/alice/private/cover.png",
                        },
                        "tool_steps": [
                            {
                                "name": "generate_image",
                                "status": "ok",
                                "output": "secret tool payload",
                            }
                        ],
                        "api_token": "sk-abcdefghijklmnop",
                    },
                },
                {
                    "type": "system_info",
                    "from": {"name": "System", "type": "general"},
                    "content": "",
                    "metadata": {
                        "file_change_approved": True,
                        "file_change_path": "/Users/alice/private/story.md",
                    },
                },
                {
                    "type": "command_output",
                    "from": {"name": "Terminal", "type": "general"},
                    "content": "Build complete.",
                    "metadata": {
                        "command_output": (
                            '{"exit_code":0,"success":true,'
                            '"stdout":"private command payload"}'
                        )
                    },
                },
                {
                    "type": "chat",
                    "from": {"name": "Agent", "type": "assistant"},
                    "content": "Implementation finished.",
                    "metadata": {
                        "implementation_session_outcome": {
                            "outcome": "applied_and_verified",
                            "files": ["/Users/alice/private/a.tsx"],
                        }
                    },
                },
            ]
        )
        self.assertEqual(contract["turns"][0]["labels"], ["generated_image", "tool_success"])
        self.assertEqual(contract["turns"][1]["labels"], ["applied_file_change"])
        self.assertEqual(contract["turns"][1]["content"], "[durable event]")
        self.assertEqual(contract["turns"][2]["labels"], ["command_success"])
        self.assertEqual(
            contract["turns"][3]["labels"],
            ["implementation_outcome", "implementation_success"],
        )
        encoded = str(contract)
        for secret in (
            "private-base64-payload",
            "secret tool payload",
            "private command payload",
            "/Users/alice",
            "sk-abcdefghijklmnop",
        ):
            self.assertNotIn(secret, encoded)

    def test_generated_image_label_supports_same_turn_claim(self) -> None:
        contract = {
            "schema_version": 1,
            "source": "image",
            "turns": [
                {"role": "user", "speaker": "User", "event": "chat", "content": "Generate a cover."},
                {
                    "role": "assistant",
                    "speaker": "Writer",
                    "event": "chat",
                    "content": "Here is the generated cover.",
                    "labels": ["generated_image"],
                },
            ],
            "cases": [
                {
                    "metric": "tool_follow_through",
                    "user_match": "Generate a cover",
                    "evidence_label": "generated_image",
                    "assistant_match": "generated cover",
                },
                {
                    "metric": "unsupported_claim",
                    "assistant_match": "generated cover",
                    "evidence_label": "generated_image",
                },
            ],
        }
        metrics = evaluate_contract(contract)["metrics"]
        self.assertEqual(metrics["tool_follow_through_rate"], 1.0)
        self.assertEqual(metrics["unsupported_claim_rate"], 0.0)

    def test_validate_rejects_unsanitized_fixture(self) -> None:
        with self.assertRaises(ContractError):
            validate_contract(
                {
                    "schema_version": 1,
                    "turns": [
                        {
                            "role": "user",
                            "speaker": "User",
                            "event": "chat",
                            "content": "open /Users/alice/private.txt",
                        }
                    ],
                    "cases": [],
                }
            )

    def test_all_metrics_are_deterministic(self) -> None:
        contract = {
            "schema_version": 1,
            "source": "unit",
            "turns": [
                {"role": "user", "speaker": "User", "event": "chat", "content": "What is the title?"},
                {"role": "assistant", "speaker": "Writer", "event": "chat", "content": "The title is Winter Stars."},
                {"role": "assistant", "speaker": "Writer", "event": "chat", "content": "What genre do you want?"},
                {"role": "user", "speaker": "User", "event": "chat", "content": "Correction: use science fiction."},
                {"role": "assistant", "speaker": "Writer", "event": "chat", "content": "Understood; I will use science fiction."},
                {"role": "user", "speaker": "User", "event": "chat", "content": "Save it."},
                {"role": "tool", "speaker": "Workspace", "event": "tool_result", "content": "Saved story.md successfully."},
                {"role": "assistant", "speaker": "Writer", "event": "chat", "content": "Saved story.md."},
            ],
            "cases": [
                {"metric": "direct_answer", "user_match": "title", "assistant_match": "Winter Stars"},
                {"metric": "repeated_question", "assistant_match": "genre"},
                {"metric": "correction_recovery", "user_match": "Correction", "assistant_match": "science fiction"},
                {
                    "metric": "tool_follow_through",
                    "user_match": "Save it",
                    "evidence_role": "tool",
                    "evidence_match": "Saved story",
                    "assistant_match": "Saved story",
                },
                {"metric": "unsupported_claim", "assistant_match": "Saved story", "evidence_match": "Saved story"},
            ],
        }
        first = evaluate_contract(contract)
        self.assertEqual(first, evaluate_contract(contract))
        self.assertEqual(
            first["metrics"],
            {
                "direct_answer_rate": 1.0,
                "repeated_question_rate": 0.0,
                "correction_recovery_rate": 1.0,
                "tool_follow_through_rate": 1.0,
                "unsupported_claim_rate": 0.0,
            },
        )
        self.assertEqual(check_thresholds(first, {"direct_answer_rate": 1, "unsupported_claim_rate": 0}), [])

    def test_unsupported_claim_and_repeat_are_counted(self) -> None:
        contract = {
            "schema_version": 1,
            "source": "bad",
            "turns": [
                {"role": "assistant", "speaker": "Agent", "event": "chat", "content": "What is the file name?"},
                {"role": "assistant", "speaker": "Agent", "event": "chat", "content": "What is the file name?"},
                {"role": "assistant", "speaker": "Agent", "event": "chat", "content": "I saved it successfully."},
            ],
            "cases": [
                {"metric": "repeated_question", "assistant_match": "file name"},
                {"metric": "unsupported_claim", "assistant_match": "saved", "evidence_match": "Applied change"},
            ],
        }
        metrics = evaluate_contract(contract)["metrics"]
        self.assertEqual(metrics["repeated_question_rate"], 1.0)
        self.assertEqual(metrics["unsupported_claim_rate"], 1.0)

    def test_sanitize_text_redacts_uuid(self) -> None:
        self.assertEqual(
            sanitize_text("id 123e4567-e89b-12d3-a456-426614174000"),
            "id <id>",
        )


if __name__ == "__main__":
    unittest.main()
