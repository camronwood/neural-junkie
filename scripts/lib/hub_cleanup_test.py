"""Unit tests for hub_cleanup helpers."""

from __future__ import annotations

import unittest
from unittest import mock

from lib import hub_cleanup


class HubCleanupTest(unittest.TestCase):
    def test_clear_pending_file_changes(self) -> None:
        with mock.patch("lib.hub_cleanup.hub.list_pending_file_changes", return_value=[{"id": "abc123"}]):
            with mock.patch("lib.hub_cleanup.hub.hub_request", return_value=(200, {})) as req:
                n = hub_cleanup.clear_pending_file_changes("http://127.0.0.1:18765")
        self.assertEqual(n, 1)
        req.assert_called_once()

    def test_wait_for_agent_roster_ok(self) -> None:
        with mock.patch("lib.hub_cleanup.hub.verify_agents_online", return_value=(True, [])):
            ok, missing = hub_cleanup.wait_for_agent_roster(
                "http://127.0.0.1:18765",
                ["BackendEngineer"],
                timeout_s=1.0,
                poll_s=0.01,
            )
        self.assertTrue(ok)
        self.assertEqual(missing, [])


if __name__ == "__main__":
    unittest.main()
