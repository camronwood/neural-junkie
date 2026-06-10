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
