"""Protein folding and structure metadata for biology sidecar."""
from __future__ import annotations

import json
import os
import re
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

from routes.sequence import _classify, _normalize


def _artifacts_dir(settings: dict) -> Path:
    raw = str(settings.get("biology_artifacts_dir") or "~/.neural-junkie/bio")
    path = Path(os.path.expanduser(raw))
    path.mkdir(parents=True, exist_ok=True)
    return path


def _hf_token(settings: dict) -> str:
    return str(settings.get("hf_token") or os.environ.get("HF_TOKEN") or "").strip()


def _fold_hf(seq: str, settings: dict) -> bytes:
    token = _hf_token(settings)
    if not token:
        raise ValueError("Hugging Face token required for hf fold backend")
    model = str(settings.get("biology_esmfold_model") or "facebook/esmfold_v1")
    url = f"https://api-inference.huggingface.co/models/{model}"
    req = urllib.request.Request(
        url,
        data=json.dumps({"inputs": seq}).encode("utf-8"),
        headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=600) as resp:
            data = resp.read()
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")[:500]
        raise RuntimeError(f"ESMFold API error {exc.code}: {body}") from exc
    if b"ATOM" not in data:
        raise RuntimeError("unexpected ESMFold response (not PDB)")
    return data


def fold_protein(body: dict, settings: dict, pack_dir: str) -> dict[str, Any]:
    raw = str(body.get("sequence", "")).strip()
    seq = _normalize(raw)
    if not seq:
        raise ValueError("no protein sequence provided")
    if _classify(seq) != "protein":
        raise ValueError("sequence does not look like protein (use analyze_sequence first)")
    max_len = int(body.get("max_length") or 400)
    if len(seq) > max_len:
        raise ValueError(f"sequence length {len(seq)} exceeds max {max_len} for folding")

    backend = str(settings.get("biology_fold_backend") or "hf").strip().lower()
    if backend == "hf":
        pdb_bytes = _fold_hf(seq, settings)
    elif backend in ("local-esmfold", "colabfold"):
        local_url = str(settings.get("biology_fold_local_url") or "http://127.0.0.1:8765/fold").strip()
        payload = json.dumps({"sequence": seq, "backend": backend}).encode("utf-8")
        req = urllib.request.Request(
            local_url,
            data=payload,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=900) as resp:
                result = json.loads(resp.read().decode("utf-8"))
        except urllib.error.URLError as exc:
            raise RuntimeError(
                f"local fold backend unavailable ({backend}); run setup-biology-sidecar.sh"
            ) from exc
        if "pdb" in result:
            pdb_bytes = result["pdb"].encode("utf-8") if isinstance(result["pdb"], str) else result["pdb"]
        elif "pdb_path" in result:
            pdb_bytes = Path(result["pdb_path"]).read_bytes()
        else:
            raise RuntimeError("local fold returned no pdb")
    else:
        raise ValueError(f"unknown biology_fold_backend: {backend}")

    out_dir = _artifacts_dir(settings)
    name = f"fold_{int(time.time() * 1000)}.pdb"
    out_path = out_dir / name
    out_path.write_bytes(pdb_bytes)
    return {
        "ok": True,
        "model": str(settings.get("biology_esmfold_model") or "facebook/esmfold_v1"),
        "sequence_length": len(seq),
        "pdb_path": str(out_path),
        "summary": (
            f"Structure prediction complete (in silico)\n"
            f"Sequence length: {len(seq)} aa\n"
            f"PDB file: {out_path}\n"
            f"This is a computational model, not experimental structure data."
        ),
    }


def structure_metadata(body: dict, settings: dict, pack_dir: str) -> dict[str, Any]:
    path = str(body.get("path", "")).strip()
    if not path:
        raise ValueError("path required")
    text = Path(os.path.expanduser(path)).read_text(encoding="utf-8", errors="replace")
    atoms = 0
    chains: set[str] = set()
    b_factors: list[float] = []
    for line in text.splitlines():
        if not line.startswith("ATOM") and not line.startswith("HETATM"):
            continue
        atoms += 1
        if len(line) >= 22:
            chains.add(line[21].strip() or "?")
        if len(line) >= 66:
            try:
                b_factors.append(float(line[60:66]))
            except ValueError:
                pass
    meta: dict[str, Any] = {
        "atom_count": atoms,
        "chain_count": len(chains),
        "chains": sorted(chains),
    }
    if b_factors:
        meta["mean_b_factor"] = round(sum(b_factors) / len(b_factors), 2)
        meta["plddt_note"] = "B-factors may reflect pLDDT in AlphaFold/ESMFold exports"
    return meta
