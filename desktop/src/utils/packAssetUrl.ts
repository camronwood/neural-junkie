import { getHubBaseURL } from '../config/hubUrl';

/** Hub URL for a pack-relative asset (toolbar chip icon, etc.). */
export function packAssetUrl(packId: string, relativePath: string): string {
  const base = getHubBaseURL().replace(/\/$/, '');
  const params = new URLSearchParams({ path: relativePath.replace(/\\/g, '/') });
  return `${base}/api/packs/${encodeURIComponent(packId)}/asset?${params.toString()}`;
}

/** Resolve toolbar icon to a fetchable URL (pack asset or absolute http(s)). */
export function resolvePackToolbarIconUrl(packId: string, icon: string | undefined): string | undefined {
  const raw = icon?.trim();
  if (!raw) return undefined;
  if (/^https?:\/\//i.test(raw)) return raw;
  return packAssetUrl(packId, raw);
}

/** Toolbar chip text: max 3 characters, uppercase. */
export function normalizeToolbarChipLabel(label: string | undefined, fallbackId: string): string {
  const fromLabel = (label ?? '').trim().slice(0, 3).toUpperCase();
  if (fromLabel) return fromLabel;
  return fallbackId.slice(0, 3).toUpperCase() || '?';
}
