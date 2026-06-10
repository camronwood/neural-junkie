#!/usr/bin/env python3
"""Extract SettingsModal tab panels into desktop/src/components/settings/."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "src" / "components"
SETTINGS_DIR = SRC / "settings"
MODAL = SRC / "SettingsModal.tsx"

TAB_JSX: dict[str, tuple[int, int]] = {
    "AppearanceSettingsTab": (1991, 2118),
    "IntegrationsSettingsTab": (2506, 3621),
    "DomainPacksSettingsTab": (3623, 3920),
    "AIProvidersSettingsTab": (3922, 4665),
    "SecuritySettingsTab": (4667, 4729),
    "AboutSettingsTab": (4731, 4859),
}

KEEP_TABS: dict[str, tuple[int, int]] = {
    "layout": (2120, 2322),
    "keyboard": (2324, 2355),
    "chat": (2357, 2504),
}


def line_range(lines: list[str], start: int, end: int) -> str:
    return "".join(lines[start - 1 : end])


def strip_tab_wrapper(block: str, tab_id: str) -> str:
    """Remove `{activeTab === 'x' && (` wrapper and closing `)}`."""
    block = block.strip()
    prefix = f"{{activeTab === '{tab_id}' && ("
    alt = {
        "AppearanceSettingsTab": "appearance",
        "IntegrationsSettingsTab": "integrations",
        "DomainPacksSettingsTab": "domain-packs",
        "AIProvidersSettingsTab": "ai-providers",
        "SecuritySettingsTab": "security",
        "AboutSettingsTab": "about",
    }[tab_id]
    prefix = f"{{activeTab === '{alt}' && ("
    if block.startswith(prefix):
        block = block[len(prefix) :].lstrip("\n")
    if block.endswith(")}"):
        block = block[: -2].rstrip()
    return block.strip() + "\n"


def main() -> None:
    text = MODAL.read_text()
    lines = text.splitlines(keepends=True)
    SETTINGS_DIR.mkdir(parents=True, exist_ok=True)

    for name, (start, end) in TAB_JSX.items():
        block = line_range(lines, start, end)
        tab_id = name.replace("SettingsTab", "")
        inner = strip_tab_wrapper(block, name)
        out = SETTINGS_DIR / f"{name}.tsx"
        print(f"write {out.name} ({start}-{end}, {len(inner.splitlines())} lines inner)")

    # Emit shell content regions for verification
    keep_blocks = []
    for tab, (start, end) in KEEP_TABS.items():
        keep_blocks.append((tab, line_range(lines, start, end)))
        print(f"keep in shell: {tab} {start}-{end}")


if __name__ == "__main__":
    main()
