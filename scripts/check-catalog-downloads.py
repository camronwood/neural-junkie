#!/usr/bin/env python3
"""Verify each official catalog download_url resolves (HTTP 200).

Opt-in network check for local/CI:

  make check-catalog-downloads

Reads packs/catalog.json (repo SoT). Entries without download_url are skipped.
Uses HEAD first, then GET if HEAD fails. Proxies are disabled by default so
macOS system proxy discovery cannot break GitHub release asset redirects.
"""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_CATALOG = ROOT / "packs" / "catalog.json"


def _opener(*, use_env_proxy: bool) -> urllib.request.OpenerDirector:
    if use_env_proxy:
        return urllib.request.build_opener()
    return urllib.request.build_opener(urllib.request.ProxyHandler({}))


def check_url(
    url: str,
    *,
    timeout: float,
    opener: urllib.request.OpenerDirector,
) -> tuple[bool, str]:
    last_err = ""
    headers = {"User-Agent": "NeuralJunkie-CatalogCheck/1.0", "Accept": "*/*"}
    for method in ("HEAD", "GET"):
        req = urllib.request.Request(url, method=method, headers=headers)
        try:
            with opener.open(req, timeout=timeout) as resp:
                code = getattr(resp, "status", None) or resp.getcode()
                if 200 <= int(code) < 300:
                    return True, f"{method} {code}"
                last_err = f"{method} {code}"
        except urllib.error.HTTPError as exc:
            last_err = f"{method} HTTP {exc.code}"
            if method == "HEAD" and exc.code in (403, 405):
                continue
        except Exception as exc:  # noqa: BLE001 — surface any network failure
            last_err = f"{method} {exc}"
    return False, last_err


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--catalog",
        type=Path,
        default=DEFAULT_CATALOG,
        help="Path to catalog.json (default: packs/catalog.json)",
    )
    parser.add_argument("--timeout", type=float, default=30.0, help="Per-URL timeout seconds")
    parser.add_argument(
        "--use-env-proxy",
        action="store_true",
        help="Honor HTTP(S)_PROXY / system proxies (default: no proxy)",
    )
    args = parser.parse_args()

    data = json.loads(args.catalog.read_text(encoding="utf-8"))
    packs = data.get("packs") or []
    if not packs:
        print(f"FAIL: no packs in {args.catalog}", file=sys.stderr)
        return 1

    opener = _opener(use_env_proxy=args.use_env_proxy)
    failed: list[str] = []
    checked = 0
    for entry in packs:
        pack_id = entry.get("id") or "?"
        url = (entry.get("download_url") or "").strip()
        if not url:
            print(f"SKIP {pack_id}: no download_url")
            continue
        checked += 1
        ok, detail = check_url(url, timeout=args.timeout, opener=opener)
        if ok:
            print(f"OK  {pack_id}: {detail}  {url}")
        else:
            print(f"FAIL {pack_id}: {detail}  {url}", file=sys.stderr)
            failed.append(pack_id)

    if checked == 0:
        print("FAIL: no download_url entries to check", file=sys.stderr)
        return 1
    if failed:
        print(
            f"FAIL: {len(failed)}/{checked} download URLs broken: {', '.join(failed)}",
            file=sys.stderr,
        )
        return 1
    print(f"PASS: {checked} catalog download URLs OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
