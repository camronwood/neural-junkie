#!/usr/bin/env python3
"""Generate docs/sitemap.xml for GitHub Pages / custom domain."""
from __future__ import annotations

import sys
from datetime import datetime, timezone
from pathlib import Path
from xml.etree import ElementTree as ET

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from site_nav import DOCS, SITE_BASE_URL, iter_site_html, page_canonical_url  # noqa: E402

SITEMAP_PATH = DOCS / "sitemap.xml"
ROBOTS_PATH = DOCS / "robots.txt"

# Pages we publish but do not want indexed (none today — keep hook for future use).
NOINDEX_PAGES: set[str] = set()


def page_priority(html_path: Path) -> str:
    rel = html_path.relative_to(DOCS).as_posix()
    if rel == "index.html":
        return "1.0"
    if rel in {"download.html", "start-here.html", "packs.html", "built-for-zero.html", "install-trust.html", "thanks.html"}:
        return "0.9"
    if rel in {"features/index.html", "articles/index.html", "benchmarks/index.html"}:
        return "0.8"
    if rel.startswith("articles/"):
        return "0.7"
    if rel.startswith("features/"):
        return "0.7"
    return "0.6"


def page_changefreq(html_path: Path) -> str:
    rel = html_path.relative_to(DOCS).as_posix()
    if rel in {"index.html", "download.html", "release-notes.html", "known-issues.html"}:
        return "weekly"
    if rel.startswith("articles/"):
        return "monthly"
    return "monthly"


def collect_pages() -> list[Path]:
    pages = [path for path in iter_site_html() if path.relative_to(DOCS).as_posix() not in NOINDEX_PAGES]
    return pages


def write_sitemap(pages: list[Path]) -> None:
    lastmod = datetime.now(timezone.utc).date().isoformat()
    urlset = ET.Element("urlset", xmlns="http://www.sitemaps.org/schemas/sitemap/0.9")

    for path in pages:
        url = ET.SubElement(urlset, "url")
        loc = ET.SubElement(url, "loc")
        loc.text = page_canonical_url(path)
        lm = ET.SubElement(url, "lastmod")
        lm.text = lastmod
        cf = ET.SubElement(url, "changefreq")
        cf.text = page_changefreq(path)
        pr = ET.SubElement(url, "priority")
        pr.text = page_priority(path)

    tree = ET.ElementTree(urlset)
    ET.indent(tree, space="  ")
    tree.write(SITEMAP_PATH, encoding="utf-8", xml_declaration=True)
    # ElementTree does not add the newline after the XML declaration.
    text = SITEMAP_PATH.read_text(encoding="utf-8")
    if not text.endswith("\n"):
        SITEMAP_PATH.write_text(text + "\n", encoding="utf-8")


def write_robots() -> None:
    ROBOTS_PATH.write_text(
        "\n".join(
            [
                "User-agent: *",
                "Allow: /",
                "",
                f"Sitemap: {SITE_BASE_URL}/sitemap.xml",
                "",
            ]
        ),
        encoding="utf-8",
    )


def main() -> int:
    pages = collect_pages()
    write_sitemap(pages)
    write_robots()
    print(f"wrote {SITEMAP_PATH.relative_to(ROOT)} ({len(pages)} URLs)")
    print(f"wrote {ROBOTS_PATH.relative_to(ROOT)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
