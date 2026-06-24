"""Canonical site chrome (dev banner + header nav) for docs/ GitHub Pages."""
from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
GITHUB_REPO = "https://github.com/camronwood/neural-junkie"

CHROME_START = "<!-- NJ-SITE-CHROME:START -->"
CHROME_END = "<!-- NJ-SITE-CHROME:END -->"

NAV_ITEMS: tuple[dict, ...] = (
    {"id": "start-here", "label": "Start here", "path": "start-here.html"},
    {"id": "product", "label": "Product", "path": "index.html#pillars", "landing_path": "#pillars"},
    {"id": "guides", "label": "Guides", "path": "features/index.html"},
    {"id": "known-issues", "label": "Known issues", "path": "known-issues.html"},
    {"id": "download", "label": "Download", "path": "download.html"},
    {
        "id": "github",
        "label": "Star on GitHub",
        "href": GITHUB_REPO,
        "primary": True,
    },
)


def read_site_version(default: str = "1.2.0-beta.2") -> str:
    pkg = ROOT / "desktop" / "package.json"
    if not pkg.is_file():
        return default
    try:
        data = json.loads(pkg.read_text(encoding="utf-8"))
        version = str(data.get("version", "")).strip()
        return version or default
    except (json.JSONDecodeError, OSError):
        return default


def asset_prefix(depth: int) -> str:
    return "../" * depth if depth > 0 else ""


def page_depth(html_path: Path) -> int:
    rel = html_path.relative_to(DOCS)
    return max(0, len(rel.parts) - 1)


def is_landing_page(html_path: Path) -> bool:
    return html_path.resolve() == (DOCS / "index.html").resolve()


def _href(item: dict, *, prefix: str, is_landing: bool) -> str:
    if "href" in item:
        return item["href"]
    if is_landing and item.get("landing_path"):
        return item["landing_path"]
    return f"{prefix}{item['path']}"


def render_dev_banner(*, prefix: str, version: str) -> str:
    return f"""  <div class="dev-banner" role="status">
    <div class="wrap dev-banner-inner">
      <strong>v{version} open beta</strong>
      <span><a href="{prefix}download.html">Download</a> · <a href="{prefix}start-here.html">Start here</a> — multi-agent workspace with IDE v4 (full LSP, remote SSH). macOS bundles Ollama; Win/Linux wizard installs it. <a href="{prefix}features/ide-v4.html">IDE v4</a> · <a href="{prefix}known-issues.html">Known issues</a> · <a href="{GITHUB_REPO}/issues">Issues</a> welcome.</span>
    </div>
  </div>"""


def render_site_header(
    *,
    depth: int,
    is_landing: bool = False,
    active: str | None = None,
) -> str:
    prefix = asset_prefix(depth)
    logo_href = f"{prefix}index.html" if not is_landing else "index.html"
    icon_src = f"{prefix}assets/icon/favicon-32.png"

    links: list[str] = []
    for item in NAV_ITEMS:
        href = _href(item, prefix=prefix, is_landing=is_landing)
        is_primary = bool(item.get("primary"))
        is_active = active == item["id"]
        classes = ["btn"]
        if is_primary and not is_active:
            classes.append("btn-primary")
        elif is_active:
            classes.append("btn-primary")
        else:
            classes.append("btn-ghost")
        attrs = f'class="{" ".join(classes)}" href="{href}"'
        if is_active:
            attrs += ' aria-current="page"'
        links.append(f"        <a {attrs}>{item['label']}</a>")

    nav_inner = "\n".join(links)
    return f"""  <header class="site-header">
    <div class="wrap">
      <a class="logo" href="{logo_href}" aria-label="Neural Junkie home">
        <span class="logo-mark" aria-hidden="true"><img src="{icon_src}" alt="" width="32" height="32" decoding="async" /></span>
        Neural Junkie
      </a>
      <nav class="nav-actions" aria-label="Primary">
{nav_inner}
      </nav>
    </div>
  </header>"""


def render_site_chrome(
    html_path: Path,
    *,
    version: str | None = None,
    active: str | None = None,
    include_banner: bool | None = None,
) -> str:
    depth = page_depth(html_path)
    landing = is_landing_page(html_path)
    ver = version or read_site_version()
    prefix = asset_prefix(depth)
    show_banner = landing if include_banner is None else include_banner

    parts: list[str] = [CHROME_START]
    if show_banner:
        parts.append(render_dev_banner(prefix=prefix, version=ver))
    parts.append(render_site_header(depth=depth, is_landing=landing, active=active))
    parts.append(CHROME_END)
    return "\n\n".join(parts)


def detect_active_nav(html_path: Path) -> str | None:
    rel = html_path.relative_to(DOCS).as_posix()
    if rel == "start-here.html":
        return "start-here"
    if rel == "known-issues.html":
        return "known-issues"
    if rel == "download.html":
        return "download"
    if rel == "index.html":
        return "product"
    if rel.startswith("features/"):
        return "guides"
    return None


_LEGACY_CHROME_RE = re.compile(
    r"(?:  <div class=\"dev-banner\".*?</div>\s*)?  <header class=\"site-header\">.*?</header>",
    re.DOTALL,
)

_MARKED_CHROME_RE = re.compile(
    re.escape(CHROME_START) + r".*?" + re.escape(CHROME_END),
    re.DOTALL,
)


def apply_site_chrome(html_path: Path, text: str, *, version: str | None = None) -> str:
    active = detect_active_nav(html_path)
    chrome = render_site_chrome(html_path, version=version, active=active)
    if CHROME_START in text and CHROME_END in text:
        return _MARKED_CHROME_RE.sub(chrome, text, count=1)
    if _LEGACY_CHROME_RE.search(text):
        return _LEGACY_CHROME_RE.sub(chrome, text, count=1)
    raise ValueError(f"no site chrome block found in {html_path}")


def iter_site_html() -> list[Path]:
    return sorted(DOCS.rglob("*.html"))
