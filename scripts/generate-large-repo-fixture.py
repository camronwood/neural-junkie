#!/usr/bin/env python3
"""Generate scenarios/fixtures/large-repo with nested packages and a distant symbol."""
from __future__ import annotations

import argparse
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
FIXTURE = ROOT / "scenarios" / "fixtures" / "large-repo"
TARGET_DEPTH = 12
PACKAGES_PER_LEVEL = 4
SYMBOL = "NjParityDistantHandler"
SYMBOL_FILE = "deep/pkg/alpha/beta/gamma/delta/epsilon/zeta/eta/theta/iota/kappa/lambda/handler.go"


def write_go_mod(base: Path) -> None:
    (base / "go.mod").write_text(
        "module large-repo\n\ngo 1.23\n",
        encoding="utf-8",
    )


def write_handler(base: Path) -> None:
    path = base / SYMBOL_FILE
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        f"""package lambda

// {SYMBOL} is the semantic-search target for parity scenarios.
func {SYMBOL}() string {{
\treturn "parity-ok"
}}
""",
        encoding="utf-8",
    )


def write_noise(base: Path, count: int) -> None:
    for i in range(count):
        pkg = base / "noise" / f"pkg{i:04d}"
        pkg.mkdir(parents=True, exist_ok=True)
        (pkg / "util.go").write_text(
            f"package pkg{i:04d}\n\nfunc Util{i}() int {{ return {i} }}\n",
            encoding="utf-8",
        )


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--noise", type=int, default=200, help="extra noise packages")
    args = parser.parse_args()
    FIXTURE.mkdir(parents=True, exist_ok=True)
    write_go_mod(FIXTURE)
    write_handler(FIXTURE)
    write_noise(FIXTURE, args.noise)
    readme = FIXTURE / "README.md"
    readme.write_text(
        f"# large-repo fixture\n\nGenerated for parity semantic search.\n\n"
        f"Target symbol: `{SYMBOL}` in `{SYMBOL_FILE}`.\n\n"
        f"Regenerate: `python3 scripts/generate-large-repo-fixture.py`\n",
        encoding="utf-8",
    )
    print(f"Wrote {FIXTURE} with symbol {SYMBOL}")


if __name__ == "__main__":
    main()
