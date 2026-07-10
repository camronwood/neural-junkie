#!/usr/bin/env python3
"""Build docs/articles/ from docs/marketing/* LinkedIn article sources."""
from __future__ import annotations

import html
import json
import re
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MARKETING = ROOT / "docs" / "marketing"
OUT_DIR = ROOT / "docs" / "articles"
COVERS_DIR = ROOT / "docs" / "media" / "articles" / "covers"
MANIFEST = OUT_DIR / "manifest.json"

sys.path.insert(0, str(ROOT / "scripts"))
from site_nav import render_footer_explore, render_site_chrome  # noqa: E402

# Explicit order + optional overrides (cover when not in source metadata).
ARTICLE_ORDER = [
    "beta-5",
    "hardware",
    "hub",
    "model-layering",
    "modular-ai-composition",
    "inference-layer",
    "loop-stack",
    "react-tools",
    "fix-loop",
    "fix-growth-loops",
    "ide-v4",
    "conversation-memory",
    "personal-learning",
    "lora",
    "lora-v2",
    "two-tier-lora",
    "mcp-lora",
    "collaboration",
    "solo-vs-collab-parity",
    "conversational-test-harness",
]

SOURCE_BY_SLUG = {
    "beta-5": "BETA25-LINKEDIN.md",
    "hardware": "HARDWARE-LINKEDIN.md",
    "hub": "HUB-LINKEDIN.md",
    "model-layering": "MODEL-LAYERING-LINKEDIN.md",
    "modular-ai-composition": "MODULAR-AI-COMPOSITION-LINKEDIN.md",
    "inference-layer": "INFERENCE-LAYER-LINKEDIN.md",
    "loop-stack": "LOOP-STACK-LINKEDIN.md",
    "react-tools": "REACT-TOOLS-LINKEDIN.md",
    "fix-loop": "FIX-LOOP-LINKEDIN.md",
    "fix-growth-loops": "FIX-GROWTH-LOOPS-LINKEDIN.md",
    "ide-v4": "IDE-V4-LINKEDIN.md",
    "conversation-memory": "CONVERSATION-MEMORY-LINKEDIN.md",
    "personal-learning": "PERSONAL-LEARNING-LINKEDIN.md",
    "lora": "LORA-LINKEDIN.md",
    "lora-v2": "LORA-V2-LINKEDIN.md",
    "two-tier-lora": "TWO-TIER-LORA-LINKEDIN.md",
    "mcp-lora": "MCP-LORA-LINKEDIN.md",
    "collaboration": "COLLABORATION-LINKEDIN.md",
    "solo-vs-collab-parity": "SOLO-VS-COLLAB-PARITY-LINKEDIN.md",
    "conversational-test-harness": "CONVERSATIONAL-TEST-HARNESS.md",
}

COVER_OVERRIDES = {
    "beta-5": "assets/neural-junkie-beta5-1200.png",
    "collaboration": "assets/neural-junkie-collaboration-ad-1080.png",
    "fix-loop": "docs/media/articles/covers/neural-junkie-fix-loop-1200.png",
    "ide-v4": "assets/marketing/ide-v4-hero-banner.png",
}

META_OVERRIDES: dict[str, dict[str, object]] = {
    "beta-5": {
        "title": "v1.2.0-beta.5: The Release Where the Loops Close",
        "teaser": (
            "Runbooks you can replay, routing you can audit, collab hardened by live scenario gates, "
            "ReAct tools on Gemma, multi-repo workspace scope, LoRA v2 specialists, and the release "
            "engineering that keeps betas honest — everything shipping in Neural Junkie beta.5 this week."
        ),
        "tags": ["ai", "localai", "multiagent", "opensource", "developertools", "release"],
    },
    "react-tools": {
        "title": "Gemma Can't Call Tools. We Taught It Anyway.",
        "teaser": (
            "Strong local models like Gemma 3 12B reason well but lack native function calling. "
            "Neural Junkie's ReAct wrapper runs MCP tools on the same model — with Qwen swap as a safety net when parsing fails."
        ),
        "tags": ["ai", "localai", "ollama", "developertools", "opensource", "multiagent"],
    },
    "ide-v4": {
        "title": "Build the IDE You Actually Own",
        "teaser": (
            "IDE v4 adds Monaco LSP, remote SSH via nj-remote, dev containers, and tree-sitter "
            "symbols — local-first and open source, for when the IDE you loved has a new owner."
        ),
        "tags": ["ai", "developertools", "opensource", "localfirst", "ide"],
    },
}

TOPIC_BY_SLUG = {
    "beta-5": "release",
    "hardware": "hardware",
    "hub": "architecture",
    "model-layering": "architecture",
    "modular-ai-composition": "architecture",
    "inference-layer": "architecture",
    "loop-stack": "architecture",
    "react-tools": "architecture",
    "fix-loop": "architecture",
    "fix-growth-loops": "testing",
    "ide-v4": "architecture",
    "conversation-memory": "chat",
    "personal-learning": "learning",
    "lora": "lora",
    "lora-v2": "lora",
    "two-tier-lora": "lora",
    "mcp-lora": "lora",
    "collaboration": "collaboration",
    "solo-vs-collab-parity": "collaboration",
    "conversational-test-harness": "testing",
}


def ensure_markdown():
    venv_py = ROOT / ".venv-icon" / "bin" / "python"
    if not venv_py.is_file():
        subprocess.check_call([sys.executable, "-m", "venv", str(ROOT / ".venv-icon")], cwd=ROOT)
        subprocess.check_call(
            [str(venv_py), "-m", "pip", "install", "-q", "markdown"],
            cwd=ROOT,
        )
    else:
        try:
            subprocess.check_call(
                [str(venv_py), "-c", "import markdown"],
                cwd=ROOT,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        except subprocess.CalledProcessError:
            subprocess.check_call(
                [str(venv_py), "-m", "pip", "install", "-q", "markdown"],
                cwd=ROOT,
            )
    if str(venv_py) not in sys.path:
        site = ROOT / ".venv-icon" / "lib"
        for lib in site.glob("python*/site-packages"):
            sys.path.insert(0, str(lib))
            break


def slug_from_filename(name: str) -> str:
    base = name.replace("-LINKEDIN.md", "").replace(".md", "")
    return base.lower().replace("_", "-")


def parse_title(text: str) -> str:
    m = re.search(r"\*\*Suggested title \(pick one\):\*\*\s*\n- (.+)", text)
    if m:
        return m.group(1).strip()
    m = re.search(r"^#\s+(.+)$", text, re.MULTILINE)
    if m:
        return re.sub(r"^LinkedIn article —\s*", "", m.group(1)).strip()
    return "Neural Junkie article"


def parse_teaser(text: str) -> str:
    for pattern in (
        r"\*\*Feed post teaser:\*\*\s*\n>\s*(.+)",
        r"\*\*Feed teaser:\*\*\s*\n\n>\s*(.+)",
        r"\*\*Feed teaser\*\*[^\n]*\n\n>\s*(.+)",
        r"\*\*Optional feed post\*\*[^\n]*\n\n>\s*(.+)",
    ):
        m = re.search(pattern, text, re.MULTILINE)
        if m:
            return m.group(1).strip()
    return ""


def parse_cover(text: str, slug: str) -> str:
    if slug in COVER_OVERRIDES:
        return COVER_OVERRIDES[slug]
    m = re.search(r"\*\*Cover image:\*\*\s*`([^`]+)`", text)
    if m:
        return m.group(1).strip()
    return ""


def parse_tags(text: str) -> list[str]:
    m = re.search(r"`(#\w+(?:\s+#\w+)*)`", text)
    if not m:
        return []
    return [t.lstrip("#").lower() for t in m.group(1).split()]


def extract_body(text: str) -> str:
    if "## PASTE START" in text:
        start = text.index("## PASTE START") + len("## PASTE START")
        end = text.index("## PASTE END")
        return text[start:end].strip()

    if "## Article body" in text:
        chunk = text.split("## Article body", 1)[1]
        # Skip divider lines until first markdown heading or paragraph.
        lines = chunk.splitlines()
        body_lines: list[str] = []
        started = False
        for line in lines:
            if not started:
                if line.strip() in ("", "---"):
                    continue
                started = True
            body_lines.append(line)
        return "\n".join(body_lines).strip()

    return text


def md_to_html(body: str) -> str:
    import markdown

    return prepare_article_body_html(
        markdown.markdown(
            body,
            extensions=["tables", "fenced_code", "sane_lists", "nl2br"],
        )
    )


MERMAID_BLOCK_RE = re.compile(
    r'<pre><code class="language-mermaid">(.*?)</code></pre>',
    re.DOTALL | re.IGNORECASE,
)


def prepare_article_body_html(body_html: str) -> str:
    """Turn fenced mermaid code blocks into renderable diagrams on the static site."""

    def repl(match: re.Match[str]) -> str:
        src = html.unescape(match.group(1)).strip()
        return f'<pre class="mermaid">{src}</pre>'

    return MERMAID_BLOCK_RE.sub(repl, body_html)


def publish_cover(cover: str) -> tuple[str, str]:
    """Copy cover art into docs/media so GitHub Pages can serve it."""
    if not cover:
        return "", ""
    src = ROOT / cover
    if not src.is_file():
        print(f"warn: cover missing: {cover}", file=sys.stderr)
        return "", ""
    COVERS_DIR.mkdir(parents=True, exist_ok=True)
    dest = COVERS_DIR / src.name
    if src.resolve() != dest.resolve():
        shutil.copy2(src, dest)
    return cover, f"../media/articles/covers/{src.name}"


def article_html_page(meta: dict, body_html: str) -> str:
    title = html.escape(meta["title"])
    teaser = html.escape(meta.get("teaser", ""))
    cover = meta.get("coverWeb", "")
    slug = html.escape(meta["slug"])
    tags = meta.get("tags", [])
    tag_html = "".join(
        f'<span class="article-tag">{html.escape(t)}</span>' for t in tags[:6]
    )
    cover_block = ""
    if cover:
        cover_block = f"""
    <figure class="article-cover">
      <img src="{html.escape(cover)}" alt="" loading="eager" decoding="async" width="1200" height="627" />
    </figure>"""

    site_chrome = render_site_chrome(OUT_DIR / f"{meta['slug']}.html")
    footer_nav = render_footer_explore(depth=2, active="articles")

    return f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{title} — Neural Junkie</title>
  <meta name="description" content="{teaser or title}" />
  <link rel="icon" type="image/png" sizes="32x32" href="../assets/icon/favicon-32.png" />
  <link rel="apple-touch-icon" href="../assets/icon/apple-touch-icon.png" />
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link href="https://fonts.googleapis.com/css2?family=DM+Sans:ital,opsz,wght@0,9..40,400;0,9..40,600;0,9..40,800&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet" />
  <link rel="stylesheet" href="../css/landing.css" />
  <link rel="stylesheet" href="../css/articles.css" />
</head>
<body>
  <a class="skip-link" href="#main">Skip to content</a>
{site_chrome}

  <main id="main" class="wrap article-page">
    <nav class="breadcrumb" aria-label="Breadcrumb">
      <a href="../index.html">Home</a> · <a href="index.html">Articles</a> · {title}
    </nav>
    <header class="article-header">
      <p class="article-kicker">Long-form · Neural Junkie</p>
      <h1>{title}</h1>
      {"<p class=\"article-lead\">" + teaser + "</p>" if teaser else ""}
      {"<div class=\"article-tags\">" + tag_html + "</div>" if tag_html else ""}
    </header>{cover_block}
    <article class="feature-prose article-body">
{body_html}
    </article>
    <nav class="article-footer-nav">
      <a href="index.html">← All articles</a>
      <a href="https://github.com/camronwood/neural-junkie/blob/main/docs/marketing/{html.escape(meta.get('sourceFile', ''))}">Source markdown</a>
    </nav>
  </main>

  <footer class="site-footer">
    <div class="wrap">
{footer_nav}
      <p><a href="../index.html">Neural Junkie</a> — articles · <a href="{slug}.html">{title}</a></p>
    </div>
  </footer>
  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
    mermaid.initialize({{ startOnLoad: true, theme: 'dark', securityLevel: 'loose' }});
  </script>
</body>
</html>
"""


def article_card_html(item: dict) -> str:
    tags = item.get("tags", [])
    tag_data = ",".join(tags)
    tags_html = "".join(
        f'<span class="article-tag">{html.escape(t)}</span>' for t in tags[:4]
    )
    cover = item.get("coverWeb", "")
    media = ""
    if cover:
        media = (
            f'<div class="articles-card-media">'
            f'<img src="{html.escape(cover)}" alt="" loading="lazy" decoding="async" />'
            f"</div>"
        )
    teaser = item.get("teaser", "")
    teaser_html = (
        f'<p class="articles-card-teaser">{html.escape(teaser)}</p>' if teaser else ""
    )
    tags_block = (
        f'<div class="articles-card-tags">{tags_html}</div>' if tags_html else ""
    )
    return (
        f'<a class="articles-card" href="{html.escape(item["href"])}" '
        f'data-topic="{html.escape(item.get("topic", ""))}" '
        f'data-tags="{html.escape(tag_data)}">'
        f"{media}"
        f'<div class="articles-card-body">'
        f"<strong>{html.escape(item['title'])}</strong>"
        f"{teaser_html}"
        f"{tags_block}"
        f"</div>"
        f"</a>"
    )


def write_index_page(items: list[dict], updated: str) -> None:
    index_path = OUT_DIR / "index.html"
    if not index_path.is_file():
        print(f"warn: missing {index_path}", file=sys.stderr)
        return

    grid_html = "\n      ".join(article_card_html(item) for item in items)
    text = index_path.read_text(encoding="utf-8")

    def replace_count(m: re.Match[str]) -> str:
        return f"{m.group(1)}{len(items)} articles{m.group(2)}"

    if "<!-- NJ-ARTICLES-GRID:START -->" in text:
        text = re.sub(
            r"(<!-- NJ-ARTICLES-GRID:START -->).*?(<!-- NJ-ARTICLES-GRID:END -->)",
            rf"\1\n      {grid_html}\n      \2",
            text,
            count=1,
            flags=re.DOTALL,
        )
    else:
        text = re.sub(
            r'(<div class="articles-grid" id="articles-grid" aria-live="polite">).*?(</div>\n\n    <aside class="articles-how")',
            rf"\1\n      {grid_html}\n    \2",
            text,
            count=1,
            flags=re.DOTALL,
        )
    text = re.sub(
        r'(<span class="articles-count" id="articles-count">).*?(</span>)',
        replace_count,
        text,
        count=1,
    )
    updated_label = f"updated {updated[:10]}" if updated else ""
    text = re.sub(
        r'(<span id="articles-updated">).*?(</span>)',
        rf"\1{html.escape(updated_label)}\2",
        text,
        count=1,
    )
    index_path.write_text(text, encoding="utf-8")
    print(f"  index.html ← {len(items)} article cards")


def load_article(slug: str) -> dict | None:
    source_name = SOURCE_BY_SLUG.get(slug)
    if not source_name:
        return None
    path = MARKETING / source_name
    if not path.is_file():
        print(f"warn: missing source {path}", file=sys.stderr)
        return None

    text = path.read_text(encoding="utf-8")
    body_md = extract_body(text)
    cover_src = parse_cover(text, slug)
    cover, cover_web = publish_cover(cover_src)

    overrides = META_OVERRIDES.get(slug, {})
    return {
        "slug": slug,
        "title": str(overrides.get("title") or parse_title(text)),
        "teaser": str(overrides.get("teaser") or parse_teaser(text)),
        "cover": cover,
        "coverWeb": cover_web,
        "topic": TOPIC_BY_SLUG.get(slug, "general"),
        "tags": list(overrides.get("tags") or parse_tags(text)),
        "sourceFile": source_name,
        "sourcePath": f"docs/marketing/{source_name}",
        "href": f"{slug}.html",
        "bodyMd": body_md,
        "bodyHtml": md_to_html(body_md),
    }


def main() -> None:
    ensure_markdown()
    OUT_DIR.mkdir(parents=True, exist_ok=True)

    items: list[dict] = []
    for slug in ARTICLE_ORDER:
        article = load_article(slug)
        if not article:
            continue

        page_path = OUT_DIR / f"{slug}.html"
        page_path.write_text(
            article_html_page(article, article["bodyHtml"]),
            encoding="utf-8",
        )

        public = {k: v for k, v in article.items() if k not in ("bodyMd", "bodyHtml")}
        items.append(public)
        print(f"  {slug}.html ← {article['sourceFile']}")

    payload = {
        "updated": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "count": len(items),
        "items": items,
    }
    MANIFEST.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    print(f"Wrote {MANIFEST} ({len(items)} articles)")
    write_index_page(items, payload["updated"])


if __name__ == "__main__":
    main()
