"""Unit tests for hub_auth helpers (no live hub required)."""

from __future__ import annotations

import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from lib import hub_auth


class HubAuthTest(unittest.TestCase):
    def setUp(self) -> None:
        hub_auth._session_token = None
        self._env = dict(os.environ)

    def tearDown(self) -> None:
        os.environ.clear()
        os.environ.update(self._env)
        hub_auth._session_token = None

    def test_load_automation_api_key_from_env(self) -> None:
        os.environ["NEURAL_JUNKIE_API_KEY"] = "nj_test_key_abc"
        self.assertEqual(hub_auth.load_automation_api_key(), "nj_test_key_abc")

    def test_hub_auth_headers_prefers_api_key(self) -> None:
        os.environ["NEURAL_JUNKIE_API_KEY"] = "nj_test_key_abc"
        headers = hub_auth.hub_auth_headers()
        self.assertEqual(headers.get("Authorization"), "Bearer nj_test_key_abc")
        self.assertNotIn("X-NJ-Session", headers)

    def test_bootstrap_token_from_file(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "bootstrap.token"
            path.write_text("secret-bootstrap\n", encoding="utf-8")
            os.environ["NEURAL_JUNKIE_BOOTSTRAP_TOKEN_FILE"] = str(path)
            self.assertEqual(hub_auth.bootstrap_token(), "secret-bootstrap")

    def test_ensure_hub_session_caches_token(self) -> None:
        payload = b'{"token":"sess-123","username":"automation"}'

        def fake_urlopen(req, timeout=30):  # noqa: ANN001, ARG001
            self.assertEqual(req.get_method(), "POST")
            self.assertIn("/api/auth/session", req.full_url)
            return mock.Mock(
                read=lambda: payload,
                __enter__=lambda s: s,
                __exit__=lambda *a: None,
            )

        with mock.patch("urllib.request.urlopen", fake_urlopen):
            tok = hub_auth.ensure_hub_session("http://127.0.0.1:18765")
            self.assertEqual(tok, "sess-123")
            self.assertEqual(hub_auth.ensure_hub_session("http://127.0.0.1:18765"), "sess-123")


if __name__ == "__main__":
    unittest.main()
