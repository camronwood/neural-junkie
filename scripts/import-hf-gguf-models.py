#!/usr/bin/env python3
"""Download curated HF GGUFs and `ollama create` brand tags (for models that are not ollama-pullable)."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ROOT = SCRIPTS_DIR.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from lib.model_benchmark import (  # noqa: E402
    MODELS_CONFIG,
    load_models,
    model_is_installed,
    model_requires_hf_import,
)

OLLAMA_TAGS = "http://127.0.0.1:11434/api/tags"


def installed_tags() -> set[str]:
    try:
        with urllib.request.urlopen(OLLAMA_TAGS, timeout=10) as resp:
            data = json.loads(resp.read().decode())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, OSError):
        return set()
    out: set[str] = set()
    for item in data.get("models") or []:
        if isinstance(item, dict):
            name = (item.get("name") or "").strip()
            if name:
                out.add(name)
    return out


def default_cache_dir() -> Path:
    hf_home = (os.environ.get("HF_HOME") or "").strip()
    if hf_home:
        return Path(hf_home) / "hub"
    return Path.home() / ".cache" / "huggingface" / "hub"


def snapshot_path(cache_dir: Path, repo_id: str, filename: str) -> Path:
    safe = repo_id.replace("/", "--")
    return cache_dir / f"models--{safe}" / "snapshots" / "main" / filename


def download_file(url: str, dest: Path, *, token: str = "") -> None:
    dest.parent.mkdir(parents=True, exist_ok=True)
    partial = dest.with_suffix(dest.suffix + ".partial")
    headers = {"User-Agent": "neural-junkie-import-hf-gguf/1.0"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    existing = partial.stat().st_size if partial.is_file() else 0
    if existing:
        headers["Range"] = f"bytes={existing}-"
    req = urllib.request.Request(url, headers=headers, method="GET")
    with urllib.request.urlopen(req, timeout=120) as resp:
        total = resp.headers.get("Content-Length")
        total_i = int(total) + existing if total and existing else (int(total) if total else 0)
        mode = "ab" if existing and resp.status == 206 else "wb"
        if mode == "wb":
            existing = 0
        written = existing
        last_pct = -1
        with open(partial, mode) as out:
            while True:
                chunk = resp.read(1024 * 1024)
                if not chunk:
                    break
                out.write(chunk)
                written += len(chunk)
                if total_i > 0:
                    pct = int(100 * written / total_i)
                    if pct != last_pct and pct % 5 == 0:
                        print(f"    {dest.name}: {pct}% ({written / 1e9:.2f} GB)", flush=True)
                        last_pct = pct
    partial.replace(dest)


def ensure_file(repo_id: str, filename: str, *, cache_dir: Path, token: str) -> Path:
    dest = snapshot_path(cache_dir, repo_id, filename)
    if dest.is_file() and dest.stat().st_size > 0:
        print(f"  cached: {dest}")
        return dest
    url = f"https://huggingface.co/{repo_id}/resolve/main/{filename}"
    print(f"  download {url}")
    download_file(url, dest, token=token)
    print(f"  saved: {dest} ({dest.stat().st_size / 1e9:.2f} GB)")
    return dest


def ollama_create(tag: str, gguf: Path, mmproj: Path | None = None) -> bool:
    lines = [f'FROM "{gguf}"\n']
    if mmproj is not None:
        lines.append(f'FROM "{mmproj}"\n')
    with tempfile.NamedTemporaryFile("w", suffix=".Modelfile", delete=False) as tmp:
        tmp.write("".join(lines))
        path = tmp.name
    try:
        print(f"  ollama create {tag}")
        proc = subprocess.run(["ollama", "create", tag, "-f", path], cwd=ROOT, check=False)
        return proc.returncode == 0
    finally:
        Path(path).unlink(missing_ok=True)


def models_to_import(want_tags: list[str] | None) -> list[dict]:
    models = [m for m in load_models(MODELS_CONFIG) if model_requires_hf_import(m)]
    if want_tags:
        want = {t.strip() for t in want_tags if t.strip()}
        models = [m for m in models if str(m.get("tag") or "").strip() in want]
    return models


def import_model(model: dict, *, cache_dir: Path, token: str, with_mmproj: bool) -> tuple[bool, str]:
    tag = str(model.get("tag") or "").strip()
    repo = str(model.get("hf_repo_id") or "").strip()
    filename = str(model.get("hf_filename") or "").strip()
    if not tag or not repo or not filename:
        return False, f"{tag or '?'}: missing hf_repo_id/hf_filename"
    installed = installed_tags()
    if model_is_installed(installed, tag):
        return True, f"{tag}: already installed"
    try:
        gguf = ensure_file(repo, filename, cache_dir=cache_dir, token=token)
    except Exception as exc:  # noqa: BLE001
        return False, f"{tag}: download failed: {exc}"
    mmproj_path: Path | None = None
    mmproj = str(model.get("hf_mmproj") or "").strip()
    if with_mmproj and mmproj:
        try:
            mmproj_path = ensure_file(repo, mmproj, cache_dir=cache_dir, token=token)
        except Exception as exc:  # noqa: BLE001
            print(f"  WARN: mmproj download failed ({exc}); importing text-only", file=sys.stderr)
    if not ollama_create(tag, gguf, mmproj_path):
        return False, f"{tag}: ollama create failed"
    return True, f"{tag}: imported"


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--models", help="Comma-separated brand tags (default: all requires_hf_import)")
    p.add_argument("--cache-dir", type=Path, default=None)
    p.add_argument("--token", default=os.environ.get("HF_TOKEN", "").strip())
    p.add_argument("--no-mmproj", action="store_true", help="Skip vision projector GGUF")
    args = p.parse_args()

    want = [t.strip() for t in (args.models or "").split(",") if t.strip()] or None
    models = models_to_import(want)
    if not models:
        print("No HF-import models selected.")
        return 0

    cache_dir = args.cache_dir or default_cache_dir()
    print(f"Importing {len(models)} HF GGUF model(s) → Ollama")
    print(f"  cache: {cache_dir}")
    failed: list[str] = []
    for model in models:
        tag = str(model.get("tag") or "")
        print(f"\n>>> {tag}")
        ok, detail = import_model(
            model,
            cache_dir=cache_dir,
            token=args.token,
            with_mmproj=not args.no_mmproj,
        )
        print(f"  {detail}")
        if not ok:
            failed.append(detail)
    if failed:
        print(f"\nFAIL: {len(failed)} import(s) failed", file=sys.stderr)
        return 1
    print("\nOK: HF GGUF imports complete")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
