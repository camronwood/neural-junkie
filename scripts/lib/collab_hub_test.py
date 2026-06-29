"""Unit tests for collab_hub hub_request retry behavior."""

from __future__ import annotations

import io
import unittest
from unittest import mock

from lib import collab_hub
from lib import hub_auth


class CollabHubRequestTest(unittest.TestCase):
    def setUp(self) -> None:
        hub_auth._session_token = "stale-token"
        self._env = dict(__import__("os").environ)

    def tearDown(self) -> None:
        import os

        os.environ.clear()
        os.environ.update(self._env)
        hub_auth._session_token = None

    def test_hub_request_retries_401_after_clearing_session(self) -> None:
        attempt = {"n": 0}

        def urlopen_side_effect(req, timeout=60):  # noqa: ANN001, ARG001
            attempt["n"] += 1
            if attempt["n"] == 1:
                raise collab_hub.urllib.error.HTTPError(
                    "http://127.0.0.1:18765/api/health",
                    401,
                    "unauthorized",
                    hdrs=None,
                    fp=io.BytesIO(b'{"error":"invalid session"}'),
                )
            return mock.Mock(
                read=lambda: b'{"ok":true}',
                status=200,
                __enter__=lambda s: s,
                __exit__=lambda *a: None,
            )

        with mock.patch("urllib.request.urlopen", urlopen_side_effect):
            with mock.patch("lib.hub_auth.ensure_hub_session", return_value="new-token") as sess:
                code, data = collab_hub.hub_request("http://127.0.0.1:18765", "GET", "/api/health")
        self.assertEqual(code, 200)
        self.assertEqual(data, {"ok": True})
        self.assertEqual(attempt["n"], 2)
        self.assertGreaterEqual(sess.call_count, 1)


if __name__ == "__main__":
    unittest.main()
