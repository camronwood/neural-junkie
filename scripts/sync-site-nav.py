#!/usr/bin/env python3
"""Replace duplicated headers in docs/*.html with canonical site navigation."""
from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from site_nav import apply_site_chrome, iter_site_html, read_site_version  # noqa: E402


def main() -> int:
    version = read_site_version()
    updated = 0
    failed: list[str] = []

    for path in iter_site_html():
        text = path.read_text(encoding="utf-8")
        try:
            new_text = apply_site_chrome(path, text, version=version)
        except ValueError as exc:
            failed.append(str(exc))
            continue
        if new_text != text:
            path.write_text(new_text, encoding="utf-8")
            updated += 1
            print(f"  {path.relative_to(ROOT)}")

    print(f"synced {updated} page(s) (v{version})")
    if failed:
        for msg in failed:
            print(f"warn: {msg}", file=sys.stderr)
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
