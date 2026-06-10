#!/usr/bin/env python3
"""Generate settings tab components from SettingsModal.tsx line splices."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "src" / "components"
SETTINGS = SRC / "settings"
MODAL = SRC / "SettingsModal.tsx"

lines = MODAL.read_text().splitlines(keepends=True)


def grab(start: int, end: int) -> str:
    return "".join(lines[start - 1 : end])


def dedent(block: str, spaces: int = 2) -> str:
    prefix = " " * spaces
    out = []
    for line in block.splitlines():
        if line.startswith(prefix):
            out.append(line[spaces:])
        else:
            out.append(line)
    return "\n".join(out).strip() + "\n"


def strip_active_wrapper(block: str, tab_key: str) -> str:
    block = block.strip()
    open_pat = f"{{activeTab === '{tab_key}' && ("
    if block.startswith(open_pat):
        block = block[len(open_pat) :].lstrip("\n")
    if block.endswith(")}"):
        block = block[:-2].rstrip()
    return dedent(block, 10)


def write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content)
    print(f"wrote {path.relative_to(ROOT)} ({len(content.splitlines())} lines)")


# --- shared helpers ---
write(
    SETTINGS / "slackInboxHelpers.ts",
    grab(77, 132).replace("function ", "export function "),
)

write(
    SETTINGS / "settingsShared.ts",
    """import type { ChatAPI } from '../../api/chatAPI';

export async function mergeSettingsPut(
  hubHttp: string,
  patch: (cfg: Record<string, unknown>) => Record<string, unknown>
): Promise<void> {
  const r = await fetch(`${hubHttp}/api/settings`);
  if (!r.ok) {
    throw new Error(await r.text());
  }
  const cfg = (await r.json()) as Record<string, unknown>;
  const put = await fetch(`${hubHttp}/api/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch(cfg)),
  });
  if (!put.ok) {
    throw new Error(await put.text());
  }
}

export function openExternalLink(url: string): void {
  if (typeof window !== 'undefined' && (window as { __TAURI__?: unknown }).__TAURI__) {
    void import('@tauri-apps/api/shell').then(({ open }) => open(url));
  } else {
    window.open(url, '_blank');
  }
}

export interface SettingsTabProps {
  hubHttp: string;
  isActive: boolean;
}
""",
)

# Security tab
security_jsx = strip_active_wrapper(grab(4667, 4729), "security")
write(
    SETTINGS / "SecuritySettingsTab.tsx",
    f"""import {{ useEffect, useState }} from 'react';
import type {{ SettingsTabProps }} from './settingsShared';

type HubSecurity = {{
  hub_token_configured: boolean;
  auth_required: boolean;
  listen_all: boolean;
  loopback_only: boolean;
}};

export function SecuritySettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
  const [hubSecurity, setHubSecurity] = useState<HubSecurity | null>(null);

  useEffect(() => {{
    if (!isActive) return;
    let cancelled = false;
    (async () => {{
      try {{
        const r = await fetch(`${{hubHttp}}/api/system/security`);
        if (!r.ok) return;
        const data = await r.json();
        if (!cancelled) setHubSecurity(data);
      }} catch {{
        if (!cancelled) setHubSecurity(null);
      }}
    }})();
    return () => {{
      cancelled = true;
    }};
  }}, [isActive, hubHttp]);

  if (!isActive) return null;

  return (
{dedent(security_jsx, 0)}
  );
}}
""",
)

# Appearance tab
appearance_jsx = strip_active_wrapper(grab(1991, 2118), "appearance")
write(
    SETTINGS / "AppearanceSettingsTab.tsx",
    f"""import {{ useSettingsStore, type ColorTheme, type FontSizeScope }} from '../../stores/settingsStore';
import type {{ SettingsTabProps }} from './settingsShared';

export function AppearanceSettingsTab({{ isActive }}: SettingsTabProps) {{
  const {{ settings, updateFontSize, updateFontSizeScope, updateColorTheme }} = useSettingsStore();

  if (!isActive) return null;

  const handleFontSizeChange = (e: React.ChangeEvent<HTMLInputElement>) => {{
    updateFontSize(parseInt(e.target.value));
  }};

  const handleScopeChange = (scope: FontSizeScope) => {{
    updateFontSizeScope(scope);
  }};

  const handleColorThemeChange = (theme: ColorTheme) => {{
    updateColorTheme(theme);
  }};

  const activeColorTheme: ColorTheme = settings.colorTheme ?? 'slack';

  return (
{dedent(appearance_jsx, 0)}
  );
}}
""",
)

print("done partial generation — integrations/domain/ai/about require manual splice")
