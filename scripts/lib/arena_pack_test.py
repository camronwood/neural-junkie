"""Unit tests for Model Arena pack ensure helper."""

from __future__ import annotations

import unittest
from unittest import mock

from lib.arena_pack import ensure_model_arena_pack


class EnsureModelArenaPackTest(unittest.TestCase):
    def test_already_enabled(self) -> None:
        def fake(base: str, method: str, path: str, body=None, **_kw):
            if method == "GET" and path == "/api/packs":
                return 200, {"packs": [{"id": "model-arena", "enabled": True}]}
            if method == "GET" and path == "/api/arena/challenges":
                return 200, {"challenges": []}
            self.fail(f"unexpected {method} {path}")

        with mock.patch("lib.arena_pack.hub.hub_request", side_effect=fake):
            ok, detail = ensure_model_arena_pack("http://127.0.0.1:18765")
        self.assertTrue(ok)
        self.assertIn("installed and enabled", detail)

    def test_install_then_enable(self) -> None:
        calls: list[tuple[str, str]] = []

        def fake(base: str, method: str, path: str, body=None, **_kw):
            calls.append((method, path))
            if method == "GET" and path == "/api/packs":
                # First list: missing. After install: present disabled.
                if sum(1 for m, p in calls if m == "GET" and p == "/api/packs") == 1:
                    return 200, {"packs": []}
                return 200, {"packs": [{"id": "model-arena", "enabled": False}]}
            if method == "POST" and path == "/api/packs/model-arena/install":
                return 201, {"ok": True}
            if method == "PUT" and path == "/api/packs/model-arena":
                self.assertEqual(body, {"enabled": True})
                return 200, {"ok": True}
            if method == "GET" and path == "/api/arena/challenges":
                return 200, {"challenges": []}
            self.fail(f"unexpected {method} {path} body={body}")

        with mock.patch("lib.arena_pack.hub.hub_request", side_effect=fake):
            ok, detail = ensure_model_arena_pack("http://127.0.0.1:18765")
        self.assertTrue(ok, detail)
        self.assertIn(("POST", "/api/packs/model-arena/install"), calls)
        self.assertIn(("PUT", "/api/packs/model-arena"), calls)

    def test_challenges_still_forbidden(self) -> None:
        def fake(base: str, method: str, path: str, body=None, **_kw):
            if method == "GET" and path == "/api/packs":
                return 200, {"packs": [{"id": "model-arena", "enabled": True}]}
            if method == "GET" and path == "/api/arena/challenges":
                return 403, {"error": "Arena API requires the Model Arena pack"}
            self.fail(f"unexpected {method} {path}")

        with mock.patch("lib.arena_pack.hub.hub_request", side_effect=fake):
            ok, detail = ensure_model_arena_pack("http://127.0.0.1:18765")
        self.assertFalse(ok)
        self.assertIn("403", detail)


if __name__ == "__main__":
    unittest.main()
