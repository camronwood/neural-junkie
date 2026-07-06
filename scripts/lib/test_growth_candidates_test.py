"""Unit tests for test_growth_candidates."""

from __future__ import annotations

import unittest

from test_growth_candidates import (
    GrowthCandidate,
    discover_candidates,
    format_candidate_list,
    pick_candidate,
)


class DiscoverCandidatesTest(unittest.TestCase):
    def test_discover_returns_ranked_list(self) -> None:
        candidates = discover_candidates(limit=10)
        self.assertIsInstance(candidates, list)
        if len(candidates) >= 2:
            self.assertGreaterEqual(candidates[0].score, candidates[1].score)

    def test_pick_candidate_skips_ids(self) -> None:
        a = GrowthCandidate(
            id="a",
            kind="unit",
            title="A",
            description="",
            score=10,
        )
        b = GrowthCandidate(
            id="b",
            kind="unit",
            title="B",
            description="",
            score=5,
        )
        picked = pick_candidate([a, b], skip_ids={"a"})
        self.assertIsNotNone(picked)
        assert picked is not None
        self.assertEqual(picked.id, "b")

    def test_format_candidate_list_nonempty(self) -> None:
        c = GrowthCandidate(
            id="x",
            kind="scenario",
            title="Test title",
            description="desc",
            score=50,
        )
        text = format_candidate_list([c])
        self.assertIn("Test title", text)
        self.assertIn("scenario", text)


if __name__ == "__main__":
    unittest.main()
