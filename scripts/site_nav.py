"""Canonical site chrome (dev banner + header nav) for docs/ GitHub Pages."""
from __future__ import annotations

import html
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
GITHUB_REPO = "https://github.com/camronwood/neural-junkie"
SITE_BASE_URL = "https://camronwood.github.io/neural-junkie"
SITE_CUSTOM_DOMAIN_URL = "https://www.neuraljunkie.com"
DEFAULT_OG_IMAGE_PATH = "/assets/icon/og-image.png"
DEFAULT_SITE_DESCRIPTION = (
    "Neural Junkie — open-source multi-agent AI workspace. Local-first specialists, "
    "bounded collaboration, routing you can audit, and human approval in one desktop app."
)

CHROME_START = "<!-- NJ-SITE-CHROME:START -->"
CHROME_END = "<!-- NJ-SITE-CHROME:END -->"
FOOTER_NAV_START = "<!-- NJ-SITE-FOOTER-NAV:START -->"
FOOTER_NAV_END = "<!-- NJ-SITE-FOOTER-NAV:END -->"
ANALYTICS_START = "<!-- NJ-SITE-ANALYTICS:START -->"
ANALYTICS_END = "<!-- NJ-SITE-ANALYTICS:END -->"
SEO_START = "<!-- NJ-SITE-SEO:START -->"
SEO_END = "<!-- NJ-SITE-SEO:END -->"

NAV_ITEMS: tuple[dict, ...] = (
    {"id": "start-here", "label": "Start here", "path": "start-here.html"},
    {"id": "product", "label": "Product", "path": "index.html#pillars", "landing_path": "#pillars"},
    {"id": "packs", "label": "Packs", "path": "packs.html"},
    {"id": "guides", "label": "Guides", "path": "features/index.html"},
    {"id": "articles", "label": "Articles", "path": "articles/index.html"},
    {"id": "benchmarks", "label": "Benchmarks", "path": "benchmarks/index.html"},
    {"id": "gallery", "label": "Gallery", "path": "gallery/index.html"},
    {"id": "security", "label": "Security", "path": "security.html"},
    {"id": "releases", "label": "Release notes", "path": "release-notes.html"},
    {"id": "known-issues", "label": "Known issues", "path": "known-issues.html"},
    {"id": "download", "label": "Download", "path": "download.html"},
    {
        "id": "github",
        "label": "Star on GitHub",
        "href": GITHUB_REPO,
        "primary": True,
    },
)

FOOTER_EXPLORE_ITEMS: tuple[dict, ...] = (
    {"id": "start-here", "label": "Start here", "path": "start-here.html"},
    {"id": "built-for-zero", "label": "Built for $0", "path": "built-for-zero.html"},
    {"id": "thanks", "label": "Thank you", "path": "thanks.html"},
    {"id": "install-trust", "label": "Install trust", "path": "install-trust.html"},
    {"id": "packs", "label": "Packs", "path": "packs.html"},
    {"id": "security", "label": "Security", "path": "security.html"},
    {"id": "guides", "label": "Guides", "path": "features/index.html"},
    {"id": "articles", "label": "Articles", "path": "articles/index.html"},
    {"id": "benchmarks", "label": "Benchmarks", "path": "benchmarks/index.html"},
    {"id": "gallery", "label": "Gallery", "path": "gallery/index.html"},
    {"id": "releases", "label": "Release notes", "path": "release-notes.html"},
    {"id": "known-issues", "label": "Known issues", "path": "known-issues.html"},
    {
        "id": "architecture",
        "label": "Architecture",
        "href": f"{GITHUB_REPO}/blob/main/docs/ARCHITECTURE.md",
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
    if rel == "packs.html":
        return "packs"
    if rel == "security.html":
        return "security"
    if rel == "known-issues.html":
        return "known-issues"
    if rel == "download.html":
        return "download"
    if rel == "release-notes.html":
        return "releases"
    if rel == "index.html":
        return "product"
    if rel.startswith("features/"):
        return "guides"
    if rel.startswith("articles/"):
        return "articles"
    if rel == "benchmarks/index.html":
        return "benchmarks"
    if rel == "gallery/index.html":
        return "gallery"
    return None


def detect_active_footer(html_path: Path) -> str | None:
    rel = html_path.relative_to(DOCS).as_posix()
    if rel == "start-here.html":
        return "start-here"
    if rel == "packs.html":
        return "packs"
    if rel == "security.html":
        return "security"
    if rel == "known-issues.html":
        return "known-issues"
    if rel.startswith("features/"):
        return "guides"
    if rel.startswith("articles/"):
        return "articles"
    if rel == "benchmarks/index.html":
        return "benchmarks"
    if rel == "gallery/index.html":
        return "gallery"
    if rel == "release-notes.html":
        return "releases"
    return None


def render_footer_explore(
    *,
    depth: int,
    active: str | None = None,
) -> str:
    prefix = asset_prefix(depth)
    links: list[str] = []
    for item in FOOTER_EXPLORE_ITEMS:
        href = _href(item, prefix=prefix, is_landing=False)
        is_active = active == item["id"]
        classes = ["btn", "btn-ghost", "btn-sm"]
        attrs = f'class="{" ".join(classes)}" href="{href}"'
        if is_active:
            attrs += ' aria-current="page"'
        links.append(f"        <a {attrs}>{item['label']}</a>")

    nav_inner = "\n".join(links)
    return f"""{FOOTER_NAV_START}
    <nav class="doc-strip footer-explore" aria-label="Explore site">
      <span class="doc-strip-label">Explore:</span>
{nav_inner}
    </nav>
{FOOTER_NAV_END}"""


_FEATURE_NAV_RE = re.compile(
    r"\n    <nav class=\"feature-nav\".*?</nav>",
    re.DOTALL,
)

_INLINE_DOC_STRIP_RE = re.compile(
    r"\n        <p class=\"doc-strip\">.*?</p>",
    re.DOTALL,
)

_MARKED_FOOTER_NAV_RE = re.compile(
    re.escape(FOOTER_NAV_START) + r".*?" + re.escape(FOOTER_NAV_END),
    re.DOTALL,
)

_FOOTER_WRAP_RE = re.compile(
    r"(<footer class=\"site-footer\">\s*<div class=\"wrap\">)",
    re.DOTALL,
)


_LEGACY_CHROME_RE = re.compile(
    r"(?:  <div class=\"dev-banner\".*?</div>\s*)?  <header class=\"site-header\">.*?</header>",
    re.DOTALL,
)

_MARKED_CHROME_RE = re.compile(
    re.escape(CHROME_START) + r".*?" + re.escape(CHROME_END),
    re.DOTALL,
)

_MARKED_ANALYTICS_RE = re.compile(
    re.escape(ANALYTICS_START) + r".*?" + re.escape(ANALYTICS_END),
    re.DOTALL,
)

_GOATCOUNTER_ATTR_RE = re.compile(r"""data-goatcounter=(['"])(?P<url>https?://[^'"]+)\1""")
_GOATCOUNTER_SNIPPET = ROOT / ".goatcounter-snippet"


def render_goatcounter_analytics(count_url: str) -> str:
    count_url = count_url.strip()
    return f"""{ANALYTICS_START}
  <script data-goatcounter="{count_url}" async src="https://gc.zgo.at/count.js"></script>
{ANALYTICS_END}"""


def extract_goatcounter_count_url(text: str) -> str | None:
    match = _MARKED_ANALYTICS_RE.search(text)
    snippet = match.group(0) if match else text
    attr_match = _GOATCOUNTER_ATTR_RE.search(snippet)
    if not attr_match:
        return None
    url = str(attr_match.group("url") or "").strip()
    return url or None


def discover_goatcounter_count_url() -> str | None:
    if _GOATCOUNTER_SNIPPET.is_file():
        try:
            url = extract_goatcounter_count_url(_GOATCOUNTER_SNIPPET.read_text(encoding="utf-8"))
        except OSError:
            url = None
        if url:
            return url
    for path in iter_site_html():
        try:
            url = extract_goatcounter_count_url(path.read_text(encoding="utf-8"))
        except OSError:
            continue
        if url:
            return url
    return None


def apply_site_analytics(text: str, goatcounter_count_url: str | None = None) -> str:
    count_url = (goatcounter_count_url or "").strip()
    if not count_url:
        return text
    analytics = render_goatcounter_analytics(count_url)
    if ANALYTICS_START in text and ANALYTICS_END in text:
        return _MARKED_ANALYTICS_RE.sub(analytics, text, count=1)
    body_close = text.rfind("</body>")
    if body_close < 0:
        raise ValueError("no </body> tag found for analytics injection")
    return text[:body_close].rstrip() + "\n\n" + analytics + "\n</body>" + text[body_close + len("</body>") :]


def apply_site_chrome(
    html_path: Path,
    text: str,
    *,
    version: str | None = None,
    goatcounter_count_url: str | None = None,
) -> str:
    active = detect_active_nav(html_path)
    chrome = render_site_chrome(html_path, version=version, active=active)
    if CHROME_START in text and CHROME_END in text:
        text = _MARKED_CHROME_RE.sub(chrome, text, count=1)
    elif _LEGACY_CHROME_RE.search(text):
        text = _LEGACY_CHROME_RE.sub(chrome, text, count=1)
    else:
        raise ValueError(f"no site chrome block found in {html_path}")

    text = _FEATURE_NAV_RE.sub("", text, count=0)
    text = _INLINE_DOC_STRIP_RE.sub("", text, count=1)

    depth = page_depth(html_path)
    footer_active = detect_active_footer(html_path)
    footer_nav = render_footer_explore(depth=depth, active=footer_active)
    if FOOTER_NAV_START in text and FOOTER_NAV_END in text:
        text = _MARKED_FOOTER_NAV_RE.sub(footer_nav, text, count=1)
    elif _FOOTER_WRAP_RE.search(text):
        text = _FOOTER_WRAP_RE.sub(rf"\1\n{footer_nav}", text, count=1)
    else:
        raise ValueError(f"no site footer block found in {html_path}")

    text = apply_site_analytics(text, goatcounter_count_url=goatcounter_count_url)

    return text


def iter_site_html() -> list[Path]:
    return sorted(DOCS.rglob("*.html"))


_TITLE_RE = re.compile(r"<title>(.*?)</title>", re.DOTALL | re.IGNORECASE)
_DESC_RE = re.compile(
    r'<meta\s+name="description"\s+content="([^"]*)"\s*/?>',
    re.IGNORECASE,
)
_MARKED_SEO_RE = re.compile(
    re.escape(SEO_START) + r".*?" + re.escape(SEO_END),
    re.DOTALL,
)
_HEAD_SOCIAL_TAG_RE = re.compile(
    r"\n\s*<(?:meta\s+(?:property=\"og:[^\"]+\"|name=\"twitter:[^\"]+\")|link\s+rel=\"canonical\")[^>]*>",
    re.IGNORECASE,
)


def extract_page_title(text: str) -> str:
    match = _TITLE_RE.search(text)
    if not match:
        return "Neural Junkie"
    return re.sub(r"\s+", " ", match.group(1)).strip()


def extract_meta_description(text: str) -> str:
    match = _DESC_RE.search(text)
    if not match:
        return DEFAULT_SITE_DESCRIPTION
    return match.group(1).strip()


def page_canonical_url(html_path: Path) -> str:
    rel = html_path.relative_to(DOCS).as_posix()
    if rel == "index.html":
        return f"{SITE_BASE_URL}/"
    return f"{SITE_BASE_URL}/{rel}"


def default_og_image_url() -> str:
    return f"{SITE_BASE_URL}{DEFAULT_OG_IMAGE_PATH}"


def page_og_type(html_path: Path) -> str:
    rel = html_path.relative_to(DOCS).as_posix()
    if rel.startswith("articles/") and rel != "articles/index.html":
        return "article"
    return "website"


def render_json_ld(html_path: Path, *, title: str, description: str) -> str | None:
    rel = html_path.relative_to(DOCS).as_posix()
    if rel == "index.html":
        payload = {
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            "name": "Neural Junkie",
            "applicationCategory": "DeveloperApplication",
            "operatingSystem": "macOS, Windows, Linux",
            "description": description,
            "softwareVersion": read_site_version(),
            "url": SITE_BASE_URL,
            "downloadUrl": f"{SITE_BASE_URL}/download.html",
            "license": "https://opensource.org/licenses/MIT",
            "author": {"@type": "Organization", "name": "Neural Junkie"},
        }
    elif rel.startswith("articles/") and rel != "articles/index.html":
        headline = title.removesuffix(" — Neural Junkie").strip() or title
        payload = {
            "@context": "https://schema.org",
            "@type": "Article",
            "headline": headline,
            "description": description,
            "url": page_canonical_url(html_path),
            "publisher": {"@type": "Organization", "name": "Neural Junkie"},
        }
    else:
        return None
    return (
        '  <script type="application/ld+json">\n'
        f"{json.dumps(payload, indent=2)}\n"
        "  </script>"
    )


def render_seo_block(html_path: Path, *, title: str, description: str) -> str:
    url = page_canonical_url(html_path)
    og_image = default_og_image_url()
    title_attr = html.escape(title, quote=True)
    desc_attr = html.escape(description, quote=True)
    og_type = page_og_type(html_path)

    lines = [
        SEO_START,
        f'  <link rel="canonical" href="{url}" />',
        f'  <meta property="og:title" content="{title_attr}" />',
        f'  <meta property="og:description" content="{desc_attr}" />',
        f'  <meta property="og:type" content="{og_type}" />',
        f'  <meta property="og:url" content="{url}" />',
        f'  <meta property="og:image" content="{og_image}" />',
        '  <meta name="twitter:card" content="summary_large_image" />',
        f'  <meta name="twitter:title" content="{title_attr}" />',
        f'  <meta name="twitter:description" content="{desc_attr}" />',
        f'  <meta name="twitter:image" content="{og_image}" />',
    ]
    json_ld = render_json_ld(html_path, title=title, description=description)
    if json_ld:
        lines.append(json_ld)
    lines.append(SEO_END)
    return "\n".join(lines)


def strip_head_social_tags(text: str) -> str:
    head_open = text.lower().find("<head>")
    head_close = text.lower().find("</head>")
    if head_open < 0 or head_close < 0 or head_close <= head_open:
        return text
    head = text[head_open:head_close]
    seo_start = head.find(SEO_START)
    seo_end = head.find(SEO_END)
    if seo_start >= 0 and seo_end > seo_start:
        before = _HEAD_SOCIAL_TAG_RE.sub("", head[:seo_start])
        after = _HEAD_SOCIAL_TAG_RE.sub("", head[seo_end + len(SEO_END) :])
        cleaned_head = before + head[seo_start : seo_end + len(SEO_END)] + after
    else:
        cleaned_head = _HEAD_SOCIAL_TAG_RE.sub("", head)
    return text[:head_open] + cleaned_head + text[head_close:]


def repair_broken_description_seo(text: str) -> str:
    text = re.sub(
        re.escape(SEO_END) + r"\s*/>",
        SEO_END,
        text,
    )
    return re.sub(
        r'(<meta\s+name="description"\s+content="[^"]*")\s*\n\s*' + re.escape(SEO_START),
        r"\1 />\n" + SEO_START,
        text,
        flags=re.IGNORECASE,
    )


def apply_site_seo(html_path: Path, text: str) -> str:
    text = repair_broken_description_seo(text)
    title = extract_page_title(text)
    description = extract_meta_description(text)
    seo = render_seo_block(html_path, title=title, description=description)

    if SEO_START in text and SEO_END in text:
        text = _MARKED_SEO_RE.sub(lambda _match: seo, text, count=1)
        return text

    text = strip_head_social_tags(text)
    desc_match = _DESC_RE.search(text)
    if desc_match:
        insert_at = desc_match.end()
        return text[:insert_at] + "\n" + seo + text[insert_at:]

    head_close = text.lower().find("</head>")
    if head_close < 0:
        raise ValueError(f"no </head> tag found for SEO injection in {html_path}")
    return text[:head_close] + "\n" + seo + "\n" + text[head_close:]
