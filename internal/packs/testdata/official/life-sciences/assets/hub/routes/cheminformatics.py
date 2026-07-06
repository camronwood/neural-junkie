"""Optional RDKit cheminformatics tools."""
from __future__ import annotations

from typing import Any


def _require_rdkit(settings: dict):
    if not settings.get("biology_rdkit_enabled"):
        raise RuntimeError("RDKit not enabled; set biology_rdkit_enabled and run setup-biology-sidecar.sh")
    try:
        from rdkit import Chem  # type: ignore
        from rdkit.Chem import Descriptors  # type: ignore
    except ImportError as exc:
        raise RuntimeError("RDKit not installed in biology venv") from exc
    return Chem, Descriptors


def validate_smiles(body: dict, settings: dict, pack_dir: str) -> dict[str, Any]:
    Chem, _ = _require_rdkit(settings)
    smiles = str(body.get("smiles", "")).strip()
    if not smiles:
        raise ValueError("smiles required")
    mol = Chem.MolFromSmiles(smiles)
    if mol is None:
        raise ValueError("invalid SMILES")
    return {"ok": True, "canonical": Chem.MolToSmiles(mol)}


def mol_descriptors(body: dict, settings: dict, pack_dir: str) -> dict[str, Any]:
    Chem, Descriptors = _require_rdkit(settings)
    smiles = str(body.get("smiles", "")).strip()
    if not smiles:
        raise ValueError("smiles required")
    mol = Chem.MolFromSmiles(smiles)
    if mol is None:
        raise ValueError("invalid SMILES")
    return {
        "ok": True,
        "mw": round(Descriptors.MolWt(mol), 2),
        "logp": round(Descriptors.MolLogP(mol), 2),
        "hbd": Descriptors.NumHDonors(mol),
        "hba": Descriptors.NumHAcceptors(mol),
        "tpsa": round(Descriptors.TPSA(mol), 2),
    }
