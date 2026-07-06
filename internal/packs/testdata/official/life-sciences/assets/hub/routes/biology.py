"""Biology pack hub sidecar route dispatcher."""
from __future__ import annotations

from routes import cheminformatics, fold, lookup, sequence

POST_ROUTES = {
    "/api/biology/fold": fold.fold_protein,
    "/api/biology/analyze-sequence": sequence.analyze_sequence,
    "/api/biology/blast": lookup.blast_search,
    "/api/biology/pathway": lookup.pathway_lookup,
    "/api/biology/validate-smiles": cheminformatics.validate_smiles,
    "/api/biology/mol-descriptors": cheminformatics.mol_descriptors,
    "/api/biology/structure-metadata": fold.structure_metadata,
}


def handle_get(handler, path: str, settings: dict, pack_dir: str) -> None:
    if path == "/api/biology/status":
        handler._json(
            200,
            {
                "ok": True,
                "fold_backend": settings.get("biology_fold_backend", "hf"),
                "rdkit_enabled": bool(settings.get("biology_rdkit_enabled")),
            },
        )
        return
    handler._json(404, {"error": "not found"})


def handle_post(handler, path: str, body: dict, settings: dict, pack_dir: str) -> None:
    fn = POST_ROUTES.get(path)
    if fn is None:
        handler._json(404, {"error": "not found"})
        return
    try:
        result = fn(body, settings, pack_dir)
        handler._json(200, result)
    except ValueError as exc:
        handler._json(400, {"error": str(exc)})
    except RuntimeError as exc:
        handler._json(503, {"error": str(exc)})
    except Exception as exc:  # noqa: BLE001
        handler._json(500, {"error": str(exc)})
