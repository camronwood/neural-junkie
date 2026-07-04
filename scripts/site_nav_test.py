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
    extract_cloudflare_analytics_token,
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
    def test_apply_site_chrome_injects_cloudflare_analytics(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "index.html"
        updated = apply_site_chrome(
            html_path,
            _sample_page(),
            version="1.2.0-beta.4",
            cloudflare_token="cf-test-token",
        )

        self.assertIn(ANALYTICS_START, updated)
        self.assertIn("static.cloudflareinsights.com/beacon.min.js?token=cf-test-token", updated)
        self.assertLess(updated.index(ANALYTICS_START), updated.index("</body>"))

    def test_apply_site_chrome_updates_existing_cloudflare_analytics(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "features" / "index.html"
        original = apply_site_chrome(
            html_path,
            _sample_page(),
            version="1.2.0-beta.4",
            cloudflare_token="old-token",
        )

        updated = apply_site_chrome(
            html_path,
            original,
            version="1.2.0-beta.4",
            cloudflare_token="new-token",
        )

        self.assertNotIn("token=old-token", updated)
        self.assertEqual(updated.count(ANALYTICS_START), 1)
        self.assertIn("token=new-token", updated)

    def test_apply_site_chrome_preserves_existing_cloudflare_analytics_without_token(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "index.html"
        original = apply_site_chrome(
            html_path,
            _sample_page(),
            version="1.2.0-beta.4",
            cloudflare_token="kept-token",
        )

        updated = apply_site_chrome(
            html_path,
            original,
            version="1.2.0-beta.4",
        )

        self.assertIn("token=kept-token", updated)
        self.assertEqual(extract_cloudflare_analytics_token(updated), "kept-token")


if __name__ == "__main__":
    raise SystemExit(unittest.main())
