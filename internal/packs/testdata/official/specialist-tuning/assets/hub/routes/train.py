"""LoRA training routes for specialist-tuning pack sidecar."""
from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path
from typing import Any


def _expand(path: str) -> str:
    return os.path.expanduser(path.strip())


def _python(settings: dict) -> str:
    venv = _expand(settings.get("lora_venv", ""))
    if venv:
        candidate = Path(venv) / "bin" / "python"
        if candidate.is_file():
            return str(candidate)
    py = settings.get("python_executable", "python3")
    return str(py).strip() or "python3"


def _train_script(pack_dir: str) -> Path:
    return Path(pack_dir) / "scripts" / "lora_train.py"


def _status(settings: dict, pack_dir: str) -> dict[str, Any]:
    script = _train_script(pack_dir)
    venv = _expand(settings.get("lora_venv", ""))
    return {
        "ready": script.is_file(),
        "script": str(script),
        "venv": venv,
        "backend": settings.get("lora_backend", "auto"),
    }


def handle_get(handler, path: str, settings: dict, pack_dir: str) -> None:
    if path.rstrip("/") == "/api/lora/sidecar/status":
        handler._json(200, _status(settings, pack_dir))
        return
    handler._json(404, {"error": "not found"})


def handle_post(handler, path: str, body: dict, settings: dict, pack_dir: str) -> None:
    if path.rstrip("/") != "/api/lora/sidecar/run":
        handler._json(404, {"error": "not found"})
        return

    dataset = str(body.get("dataset") or "").strip()
    output_dir = str(body.get("output_dir") or "").strip()
    base_model = str(body.get("base_model") or "").strip()
    if not dataset or not output_dir or not base_model:
        handler._json(400, {"error": "dataset, output_dir, and base_model required"})
        return

    script = _train_script(pack_dir)
    if not script.is_file():
        handler._json(503, {"error": f"training script missing: {script}"})
        return

    rank = int(body.get("rank") or 16)
    epochs = int(body.get("epochs") or 1)
    learning_rate = float(body.get("learning_rate") or 2e-4)
    max_seq_len = int(body.get("max_seq_len") or 2048)
    backend = str(body.get("backend") or settings.get("lora_backend") or "auto")
    resume = str(body.get("resume_adapter") or "").strip()

    args = [
        _python(settings),
        str(script),
        "--dataset",
        dataset,
        "--output-dir",
        output_dir,
        "--base-model",
        base_model,
        "--rank",
        str(rank),
        "--epochs",
        str(epochs),
        "--learning-rate",
        str(learning_rate),
        "--max-seq-len",
        str(max_seq_len),
        "--backend",
        backend,
    ]
    if resume:
        args.extend(["--resume-adapter", resume])

    os.makedirs(output_dir, exist_ok=True)
    proc = subprocess.run(args, capture_output=True, text=True)
    if proc.returncode != 0:
        handler._json(
            500,
            {
                "error": "training failed",
                "stderr": proc.stderr[-4000:],
                "stdout": proc.stdout[-4000:],
            },
        )
        return
    handler._json(200, {"ok": True, "stdout": proc.stdout[-8000:]})


# Allow hub to import this module directly when sidecar.module points here.
if __name__ == "__main__" and len(sys.argv) > 1:
    raise SystemExit("use assets/hub/server.py as sidecar entrypoint")
