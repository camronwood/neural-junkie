#!/usr/bin/env python3
"""LoRA fine-tuning via Unsloth (optional). Emits JSON progress lines on stdout."""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path


def emit(status: str, **extra) -> None:
    payload = {"status": status, **extra}
    print(json.dumps(payload), flush=True)


def load_rows(dataset_path: Path) -> list[dict]:
    rows: list[dict] = []
    with dataset_path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rows.append(json.loads(line))
    return rows


def format_example(row: dict) -> str:
    instruction = (row.get("instruction") or "").strip()
    inp = (row.get("input") or "").strip()
    output = (row.get("output") or "").strip()
    user = instruction if not inp else f"{instruction}\n\n{inp}"
    return f"### Instruction:\n{user}\n\n### Response:\n{output}"


def train_unsloth(args: argparse.Namespace) -> None:
    try:
        from unsloth import FastLanguageModel  # type: ignore
        import torch  # type: ignore
        from trl import SFTTrainer  # type: ignore
        from transformers import TrainingArguments  # type: ignore
        from datasets import Dataset  # type: ignore
    except ImportError as exc:
        raise RuntimeError(
            "Unsloth not installed. Run: make deps  (or ./scripts/setup-lora-deps.sh)"
        ) from exc

    emit("loading_model", base=args.base_model)
    model, tokenizer = FastLanguageModel.from_pretrained(
        model_name=args.base_model,
        max_seq_length=args.max_seq_len,
        dtype=None,
        load_in_4bit=True,
    )
    model = FastLanguageModel.get_peft_model(
        model,
        r=args.rank,
        target_modules=[
            "q_proj",
            "k_proj",
            "v_proj",
            "o_proj",
            "gate_proj",
            "up_proj",
            "down_proj",
        ],
        lora_alpha=args.rank * 2,
        lora_dropout=0,
        bias="none",
        use_gradient_checkpointing="unsloth",
    )

    rows = load_rows(Path(args.dataset))
    texts = [format_example(r) for r in rows]
    ds = Dataset.from_dict({"text": texts})

    out_dir = Path(args.output_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    emit("training", rows=len(texts), epochs=args.epochs)
    trainer = SFTTrainer(
        model=model,
        tokenizer=tokenizer,
        train_dataset=ds,
        dataset_text_field="text",
        max_seq_length=args.max_seq_len,
        args=TrainingArguments(
            output_dir=str(out_dir / "checkpoints"),
            num_train_epochs=args.epochs,
            per_device_train_batch_size=1,
            gradient_accumulation_steps=4,
            learning_rate=args.learning_rate,
            logging_steps=1,
            save_strategy="no",
            fp16=not torch.cuda.is_bf16_supported(),
            bf16=torch.cuda.is_bf16_supported(),
            report_to="none",
        ),
    )
    trainer.train()

    emit("saving_adapter")
    model.save_pretrained(str(out_dir))
    tokenizer.save_pretrained(str(out_dir))

    adapter = out_dir / "adapter_model.safetensors"
    if not adapter.exists():
        # Unsloth may write under adapter subdir
        for candidate in out_dir.rglob("adapter_model.safetensors"):
            candidate.replace(adapter)
            break
    if not adapter.exists():
        raise RuntimeError("adapter_model.safetensors not found after training")

    emit("done", adapter=str(adapter))


def main() -> int:
    parser = argparse.ArgumentParser(description="Neural Junkie LoRA trainer")
    parser.add_argument("--dataset", required=True)
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--base-model", required=True)
    parser.add_argument("--rank", type=int, default=16)
    parser.add_argument("--epochs", type=int, default=1)
    parser.add_argument("--learning-rate", type=float, default=2e-4)
    parser.add_argument("--max-seq-len", type=int, default=2048)
    args = parser.parse_args()

    if not Path(args.dataset).is_file():
        emit("error", message=f"dataset not found: {args.dataset}")
        return 1

    try:
        if os.environ.get("NJ_LORA_DRY_RUN") == "1":
            out = Path(args.output_dir)
            out.mkdir(parents=True, exist_ok=True)
            (out / "adapter_model.safetensors").write_bytes(b"dry-run")
            emit("done", dry_run=True)
            return 0
        train_unsloth(args)
        return 0
    except Exception as exc:  # noqa: BLE001
        emit("error", message=str(exc))
        print(str(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
