#!/usr/bin/env python3
"""Unit tests for hub recovery journaling."""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.hub_regression import (  # noqa: E402
    HubRecoveryEvent,
    HubRecoveryJournal,
    output_suggests_hub_crash,
    read_recovery_log_text,
)


class HubRegressionTest(unittest.TestCase):
    def test_journal_md(self) -> None:
        journal = HubRecoveryJournal()
        journal.record(
            HubRecoveryEvent(
                utc="2026-06-23-120000",
                context="test:stage",
                attempts=2,
                recovered=True,
                detail="hub healthy after restart attempt 2",
            )
        )
        md = journal.format_md()
        self.assertIn("Hub recovery events", md)
        self.assertIn("recovered", md)
        self.assertIn("test:stage", md)

    def test_recovery_log_env(self) -> None:
        import os

        from lib import hub_regression as hr

        with tempfile.TemporaryDirectory() as tmp:
            log_path = Path(tmp) / "hub-recovery.log"
            os.environ[hr.HUB_RECOVERY_LOG_ENV] = str(log_path)
            try:
                hr._append_recovery_log(
                    HubRecoveryEvent(
                        utc="2026-06-23-120001",
                        context="chat:demo",
                        attempts=1,
                        recovered=False,
                        detail="gave up",
                        log_tail="panic: oom",
                    )
                )
                text = read_recovery_log_text()
                self.assertIn("hub recovery", text)
                self.assertIn("chat:demo", text)
                self.assertIn("panic: oom", text)
            finally:
                os.environ.pop(hr.HUB_RECOVERY_LOG_ENV, None)

    def test_output_suggests_hub_crash(self) -> None:
        self.assertTrue(output_suggests_hub_crash("FAIL: hub not healthy"))
        self.assertTrue(output_suggests_hub_crash("RemoteDisconnected: Remote end closed"))
        self.assertFalse(output_suggests_hub_crash("=== FAIL: go-handler ==="))


if __name__ == "__main__":
    raise SystemExit(unittest.main())
