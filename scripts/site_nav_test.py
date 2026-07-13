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
    SEO_END,
    SEO_START,
    apply_site_chrome,
    apply_site_seo,
    extract_goatcounter_count_url,
    page_canonical_url,
    render_site_header,
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


class SiteNavMobileTest(unittest.TestCase):
    def test_render_site_header_includes_mobile_toggle_and_panel(self) -> None:
        header = render_site_header(depth=0, is_landing=True, active="product")

        self.assertIn('class="nav-toggle"', header)
        self.assertIn('aria-controls="site-nav-mobile"', header)
        self.assertIn('id="site-nav-mobile"', header)
        self.assertIn('aria-label="Primary mobile"', header)
        self.assertIn("site-header-bar", header)
        self.assertIn("nav-open", header)
        self.assertEqual(header.count('href="#pillars"'), 2)
        self.assertEqual(header.count('aria-current="page"'), 2)

    def test_apply_site_chrome_keeps_mobile_nav_markup(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "index.html"
        updated = apply_site_chrome(
            html_path,
            _sample_page(),
            version="1.2.0-beta.4",
        )

        self.assertIn('class="nav-toggle"', updated)
        self.assertIn('id="site-nav-mobile"', updated)
        self.assertIn("is-nav-open", updated)


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


class SiteNavSeoTest(unittest.TestCase):
    def test_apply_site_seo_injects_canonical_and_social_tags(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "articles" / "beta-5.html"
        page = _sample_page().replace(
            "<title>Sample</title>",
            "<title>Beta 5 — Neural Junkie</title>\n"
            '  <meta name="description" content="Release notes for beta 5." />',
        )
        updated = apply_site_seo(html_path, page)

        self.assertIn(SEO_START, updated)
        self.assertIn(SEO_END, updated)
        self.assertIn(f'<link rel="canonical" href="{page_canonical_url(html_path)}" />', updated)
        self.assertIn('property="og:title"', updated)
        self.assertIn('name="twitter:card" content="summary_large_image"', updated)
        self.assertIn('"@type": "Article"', updated)

    def test_apply_site_seo_preserves_description_tag_closure(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "index.html"
        page = _sample_page().replace(
            "<title>Sample</title>",
            "<title>Neural Junkie</title>\n"
            '  <meta name="description" content="Multi-agent workspace." />',
        )
        updated = apply_site_seo(html_path, page)

        self.assertIn('content="Multi-agent workspace." />', updated)
        self.assertNotIn(f"{SEO_END} />", updated)

    def test_apply_site_seo_replaces_legacy_social_tags(self) -> None:
        html_path = SCRIPTS_DIR.parent / "docs" / "index.html"
        page = _sample_page().replace(
            "<title>Sample</title>",
            "<title>Neural Junkie</title>\n"
            '  <meta name="description" content="Multi-agent workspace." />\n'
            '  <meta property="og:title" content="Old title" />\n'
            '  <meta name="twitter:image" content="https://example.com/old.png" />',
        )
        updated = apply_site_seo(html_path, page)

        self.assertNotIn("Old title", updated)
        self.assertNotIn("example.com/old.png", updated)
        self.assertIn('"@type": "SoftwareApplication"', updated)


if __name__ == "__main__":
    raise SystemExit(unittest.main())
