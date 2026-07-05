#!/usr/bin/env python3
"""Replace duplicated headers in docs/*.html with canonical site navigation."""
from __future__ import annotations

import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))

from site_nav import (  # noqa: E402
    apply_site_analytics,
    apply_site_chrome,
    discover_goatcounter_count_url,
    iter_site_html,
    read_site_version,
)


def main() -> int:
    version = read_site_version()
    goatcounter_count_url = (
        os.environ.get("GOATCOUNTER_COUNT_URL", "").strip() or discover_goatcounter_count_url()
    )
    updated = 0
    failed: list[str] = []

    for path in iter_site_html():
        text = path.read_text(encoding="utf-8")
        try:
            new_text = apply_site_chrome(
                path,
                text,
                version=version,
                goatcounter_count_url=goatcounter_count_url,
            )
        except ValueError as exc:
            if "no site chrome block found" not in str(exc):
                failed.append(str(exc))
                continue
            try:
                new_text = apply_site_analytics(text, goatcounter_count_url=goatcounter_count_url)
            except ValueError as analytics_exc:
                failed.append(str(analytics_exc))
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
