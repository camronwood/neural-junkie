"""Sequence analysis for biology sidecar."""
from __future__ import annotations

import re
from typing import Any

DNA = set("ACGTN")
RNA = set("ACGUN")
PROTEIN = set("ACDEFGHIKLMNPQRSTVWY")


def _normalize(raw: str) -> str:
    lines = []
    for line in raw.splitlines():
        line = line.strip()
        if not line or line.startswith(">") or line.startswith(";"):
            continue
        lines.append(line)
    return "".join(lines).upper()


def _classify(seq: str) -> str:
    if not seq:
        return "unknown"
    dna = rna = protein = invalid = 0
    for ch in seq:
        if ch in "ACGT":
            dna += 1
        elif ch == "U":
            rna += 1
        elif ch == "N":
            dna += 1
            rna += 1
        elif ch == "*":
            protein += 1
        elif ch in PROTEIN:
            protein += 1
        else:
            invalid += 1
    if invalid > max(1, len(seq) // 10):
        return "unknown"
    if protein and protein >= dna and protein >= rna:
        return "protein"
    if rna > dna and rna > 0:
        return "rna"
    if dna > 0:
        return "dna"
    return "unknown"


def _reverse_complement_dna(seq: str) -> str:
    comp = {"A": "T", "T": "A", "C": "G", "G": "C", "N": "N"}
    return "".join(comp.get(c, c) for c in reversed(seq))


def analyze_sequence(body: dict, settings: dict, pack_dir: str) -> dict[str, Any]:
    raw = str(body.get("sequence", "")).strip()
    seq = _normalize(raw)
    if not seq:
        raise ValueError("no sequence found (paste FASTA or raw sequence)")
    max_len = int(body.get("max_length") or 10000)
    truncated = False
    if len(seq) > max_len:
        seq = seq[:max_len]
        truncated = True
    kind = _classify(seq)
    out: dict[str, Any] = {
        "type": kind,
        "length": len(seq),
        "truncated": truncated,
        "summary": f"Sequence analysis (in silico)\nType: {kind}\nLength: {len(seq)}",
    }
    if kind == "dna":
        rc = _reverse_complement_dna(seq)
        out["reverse_complement"] = rc[:120] + ("..." if len(rc) > 120 else "")
    if kind == "protein":
        max_fold = int(body.get("max_fold_length") or 400)
        out["fold_eligible"] = len(seq) <= max_fold
    out["disclaimer"] = "research/education only; not clinical advice"
    return out
