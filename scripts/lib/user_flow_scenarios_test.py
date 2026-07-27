"""Unit tests for user-flow suite registry."""

from __future__ import annotations

import unittest

from lib.user_flow_scenarios import USER_FLOW_SCENARIOS, user_flow_names, user_flow_scenarios


class UserFlowScenariosTest(unittest.TestCase):
    def test_suite_non_empty(self) -> None:
        self.assertGreaterEqual(len(USER_FLOW_SCENARIOS), 5)

    def test_names_unique(self) -> None:
        names = [e.name for e in USER_FLOW_SCENARIOS]
        self.assertEqual(len(names), len(set(names)))

    def test_kinds_valid(self) -> None:
        for entry in user_flow_scenarios():
            self.assertIn(entry.kind, ("implement", "collab"))
            self.assertIn(entry.source, ("user-flows", "core"))

    def test_filter_by_kind(self) -> None:
        impl = user_flow_names(kind="implement")
        collab = user_flow_names(kind="collab")
        self.assertTrue(impl)
        self.assertTrue(collab)
        self.assertTrue(set(impl).isdisjoint(set(collab)))

    def test_includes_camron_flows(self) -> None:
        names = set(user_flow_names())
        for want in (
            "trip-research-vacation",
            "rust-blackjack-2d",
            "nodejs-user-crud",
            "ios-trivia-swift",
            "collaboration-station-branded",
            "admin-cms-website",
            "vite-boot-fix-corrupt-appjs",
            "journey-crud-clarify-correct",
            "journey-blackjack-cli-correction",
            "journey-boot-fix-then-feature",
            "journey-notes-rename-to-memos",
            "journey-landing-brand-correction",
        ):
            self.assertIn(want, names)

    def test_journey_json_files_exist(self) -> None:
        from pathlib import Path

        root = Path(__file__).resolve().parents[2]
        for name in (
            "journey-crud-clarify-correct",
            "journey-blackjack-cli-correction",
            "journey-boot-fix-then-feature",
            "journey-notes-rename-to-memos",
            "journey-landing-brand-correction",
        ):
            path = root / "scenarios" / "user-flows" / "implement" / f"{name}.json"
            self.assertTrue(path.is_file(), msg=f"missing {path}")

    def test_trip_research_parked_until_mapping(self) -> None:
        active = set(user_flow_names(include_skipped=False))
        self.assertNotIn("trip-research-vacation", active)
        trip = next(e for e in USER_FLOW_SCENARIOS if e.name == "trip-research-vacation")
        self.assertTrue(trip.skip_reason)

    def test_only_trip_research_skipped_from_default_suite(self) -> None:
        active = set(user_flow_names(include_skipped=False))
        self.assertNotIn("trip-research-vacation", active)
        for want in (
            "rust-blackjack-2d",
            "nodejs-user-crud",
            "ios-trivia-swift",
            "collaboration-station-branded",
            "admin-cms-website",
            "vite-boot-fix-corrupt-appjs",
            "journey-crud-clarify-correct",
            "journey-blackjack-cli-correction",
            "journey-boot-fix-then-feature",
            "journey-notes-rename-to-memos",
            "journey-landing-brand-correction",
        ):
            self.assertIn(want, active)


if __name__ == "__main__":
    unittest.main()
