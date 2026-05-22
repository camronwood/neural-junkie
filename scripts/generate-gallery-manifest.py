#!/usr/bin/env python3
"""Build docs/gallery/manifest.json from docs/media/gallery/."""
from __future__ import annotations

import json
import re
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MEDIA = ROOT / "docs" / "media" / "gallery"
OUT = ROOT / "docs" / "gallery" / "manifest.json"
EXT = {".png", ".jpg", ".jpeg", ".webp", ".gif"}


def title_from_filename(name: str) -> str:
    base = Path(name).stem
    base = re.sub(r"^neural-junkie-", "", base, flags=re.I)
    base = re.sub(r"-ad-1080$", "", base, flags=re.I)
    base = base.replace("-", " ").replace("_", " ")
    return base.strip().title() or name


def load_sidecar(path: Path) -> dict:
    side = path.with_suffix(path.suffix + ".json")
    if not side.is_file():
        return {}
    try:
        return json.loads(side.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {}


def tags_for(rel: Path, extra: list[str]) -> list[str]:
    tags = list(extra)
    name = rel.as_posix().lower()
    if "slack" in name:
        tags.append("slack")
    if "nondev" in name:
        tags.append("non-dev")
    if "cli" in name:
        tags.append("cli")
    if "security" in name:
        tags.append("security")
    if "collab" in name:
        tags.append("collaboration")
    return sorted(set(tags))


def main() -> None:
    items = []
    if not MEDIA.is_dir():
        MEDIA.mkdir(parents=True, exist_ok=True)

    for path in sorted(MEDIA.rglob("*")):
        if not path.is_file() or path.suffix.lower() not in EXT:
            continue
        rel = path.relative_to(MEDIA)
        category = rel.parts[0] if len(rel.parts) > 1 else "misc"
        meta = load_sidecar(path)
        src = "../media/gallery/" + rel.as_posix()
        items.append(
            {
                "id": path.stem.replace(" ", "-"),
                "src": src,
                "title": meta.get("title") or title_from_filename(path.name),
                "caption": meta.get("caption", ""),
                "category": meta.get("category", category),
                "tags": tags_for(rel, meta.get("tags", [])),
                "filename": path.name,
            }
        )

    payload = {
        "updated": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "count": len(items),
        "items": items,
    }
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    print(f"Wrote {OUT} ({len(items)} images)")


if __name__ == "__main__":
    main()
