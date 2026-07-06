"""BLAST and pathway lookups for biology sidecar."""
from __future__ import annotations

import json
import urllib.parse
import urllib.request
from typing import Any


def blast_search(body: dict, settings: dict, pack_dir: str) -> dict[str, Any]:
    query = str(body.get("query", "")).strip()
    if not query:
        raise ValueError("query required")
    program = str(body.get("program") or "blastp").strip()
    database = str(body.get("database") or "nr").strip()
    # NCBI BLAST API (rate-limited; research use)
    params = urllib.parse.urlencode(
        {
            "CMD": "Put",
            "PROGRAM": program,
            "DATABASE": database,
            "QUERY": query,
            "FORMAT_TYPE": "JSON2",
        }
    )
    url = f"https://blast.ncbi.nlm.nih.gov/Blast.cgi?{params}"
    req = urllib.request.Request(url, method="POST")
    with urllib.request.urlopen(req, timeout=60) as resp:
        raw = resp.read().decode("utf-8", errors="replace")
    rid_match = None
    for line in raw.splitlines():
        if line.startswith("RID ="):
            rid_match = line.split("=", 1)[1].strip()
            break
    if not rid_match:
        return {"ok": False, "note": "BLAST job submission did not return RID; try again or use local blast+"}
    return {
        "ok": True,
        "rid": rid_match,
        "status_url": f"https://blast.ncbi.nlm.nih.gov/Blast.cgi?CMD=Get&FORMAT_OBJECT=SearchInfo&RID={rid_match}",
        "note": "Poll status URL for results; NCBI rate limits apply",
    }


def pathway_lookup(body: dict, settings: dict, pack_dir: str) -> dict[str, Any]:
    gene = str(body.get("gene") or body.get("query") or "").strip()
    if not gene:
        raise ValueError("gene or query required")
    # Reactome search (read-only REST)
    q = urllib.parse.quote(gene)
    url = f"https://reactome.org/ContentService/search/query?query={q}&species=Homo+sapiens&types=Pathway"
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=30) as resp:
        data = json.loads(resp.read().decode("utf-8"))
    entries = data.get("results") or data.get("entries") or []
    pathways = []
    for item in entries[:10]:
        if isinstance(item, dict):
            pathways.append(
                {
                    "name": item.get("name") or item.get("displayName"),
                    "stId": item.get("stId"),
                    "species": item.get("species"),
                }
            )
    return {"ok": True, "gene": gene, "pathways": pathways, "source": "reactome"}
