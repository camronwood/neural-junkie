"""Tests for docs site navigation and analytics sync helpers."""

from __future__ import annotations

import sys
import textwrap
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPTS_DIR))

from site_nav import (  # noqa: E402
    ANALYTICS_START,
    apply_site_chrome,
    extract_goatcounter_count_url,
)


def _sample_page() -> str:
    return textwrap.dedent(
        """\
        <!DOCTYPE html>
        <html lang="en">
        <head>
          <meta charset="utf-8" />
          <title>Sample</title>
        </head>
        <body>
          <a class="skip-link" href="#main">Skip to content</a>
        <!-- NJ-SITE-CHROME:START -->
          <header class="site-header">
            <div class="wrap"></div>
          </header>
        <!-- NJ-SITE-CHROME:END -->
          <main id="main">Hello</main>
          <footer class="site-footer">
            <div class="wrap">
        <!-- NJ-SITE-FOOTER-NAV:START -->
            <nav class="doc-strip footer-explore" aria-label="Explore site">
              <span class="doc-strip-label">Explore:</span>
            </nav>
        <!-- NJ-SITE-FOOTER-NAV:END -->
            </div>
          </footer>
        </body>
        </html>
        """
    )


class SiteNavAnalyticsTest(unittest.TestCase):
    def test_apply_site_chrome_injects_goatcounter_analytics(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "index.html"
        updated = apply_site_chrome(
            html_path,
            _sample_page(),
            version="1.2.0-beta.4",
            goatcounter_count_url="https://neuraljunkie.goatcounter.com/count",
        )

        self.assertIn(ANALYTICS_START, updated)
        self.assertIn('data-goatcounter="https://neuraljunkie.goatcounter.com/count"', updated)
        self.assertIn("https://gc.zgo.at/count.js", updated)
        self.assertLess(updated.index(ANALYTICS_START), updated.index("</body>"))

    def test_apply_site_chrome_updates_existing_goatcounter_analytics(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "features" / "index.html"
        original = apply_site_chrome(
            html_path,
            _sample_page(),
            version="1.2.0-beta.4",
            goatcounter_count_url="https://old.goatcounter.com/count",
        )

        updated = apply_site_chrome(
            html_path,
            original,
            version="1.2.0-beta.4",
            goatcounter_count_url="https://new.goatcounter.com/count",
        )

        self.assertNotIn("https://old.goatcounter.com/count", updated)
        self.assertEqual(updated.count(ANALYTICS_START), 1)
        self.assertIn('data-goatcounter="https://new.goatcounter.com/count"', updated)

    def test_apply_site_chrome_preserves_existing_goatcounter_analytics_without_url(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "index.html"
        original = apply_site_chrome(
            html_path,
            _sample_page(),
            version="1.2.0-beta.4",
            goatcounter_count_url="https://kept.goatcounter.com/count",
        )

        updated = apply_site_chrome(
            html_path,
            original,
            version="1.2.0-beta.4",
        )

        self.assertIn('data-goatcounter="https://kept.goatcounter.com/count"', updated)
        self.assertEqual(
            extract_goatcounter_count_url(updated),
            "https://kept.goatcounter.com/count",
        )


if __name__ == "__main__":
    raise SystemExit(unittest.main())
