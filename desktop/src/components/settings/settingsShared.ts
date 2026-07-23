import {
  openExternalLink as openExternalLinkImpl,
  openExternalLinkAsync as openExternalLinkAsyncImpl,
} from '../../utils/openExternalLink';

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
  openExternalLinkImpl(url);
}

export async function openExternalLinkAsync(url: string): Promise<boolean> {
  return openExternalLinkAsyncImpl(url);
}

export interface SettingsTabProps {
  hubHttp: string;
  isActive: boolean;
}

export type SaveSettingsResult = {
  status?: string;
  requires_restart?: boolean;
  restart_reasons?: string[];
};

export async function putSystemSecurity(
  hubHttp: string,
  body: Record<string, unknown>
): Promise<SaveSettingsResult> {
  const put = await fetch(`${hubHttp}/api/system/security`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!put.ok) {
    throw new Error(await put.text());
  }
  return (await put.json()) as SaveSettingsResult;
}
