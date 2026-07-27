import type { ResolvedCapability } from '../api/chatAPI';
import { normalizeToolbarChipLabel, resolvePackToolbarIconUrl } from '../utils/packAssetUrl';

/** NJ built-in viewer ids referenced by pack capability_defs.viewer */
export const NJ_VIEWER = {
  SCAN_SUMMARY: 'nj.scan-summary',
  SCAN_ANALYSIS: 'nj.scan-analysis',
  CAD: 'nj.cad',
  STRUCTURE: 'nj.structure',
  MUSIC: 'nj.music',
  ARENA: 'nj.arena',
  MAP: 'nj.map',
  MARKDOWN: 'nj.markdown',
  MERMAID: 'nj.mermaid',
} as const;

export type PackCapabilityRegistryState = {
  capabilities: string[];
  capabilityRegistry: ResolvedCapability[];
};

export function parseCapabilityRegistry(data: {
  capabilities?: string[];
  capability_registry?: ResolvedCapability[];
}): PackCapabilityRegistryState {
  return {
    capabilities: data.capabilities ?? [],
    capabilityRegistry: data.capability_registry ?? [],
  };
}

export function registryHasCapability(
  capabilities: string[],
  registry: ResolvedCapability[],
  cap: string,
): boolean {
  const c = String(cap).trim();
  if (!c) return false;
  if (capabilities.includes(c)) return true;
  return registry.some((r) => r.id === c || r.qualified_id === c);
}

/** Match hub HasPackCapability legacy fallback: enabled pack manifest capabilities. */
export function enabledPackDeclaresCapability(
  packs: Array<{ id: string; enabled?: boolean; capabilities?: string[] }>,
  cap: string,
): boolean {
  const c = String(cap).trim();
  if (!c) return false;
  return packs.some((p) => {
    if (!p.enabled) return false;
    return (p.capabilities ?? []).some((declared) => declared === c || declared === `${p.id}/${c}`);
  });
}

export function hasPackCapability(
  capabilities: string[],
  registry: ResolvedCapability[],
  packs: Array<{ id: string; enabled?: boolean; capabilities?: string[] }>,
  cap: string,
): boolean {
  if (registryHasCapability(capabilities, registry, cap)) return true;
  return enabledPackDeclaresCapability(packs, cap);
}

/** Match file-viewer capabilities by glob. */
export function matchFileViewer(
  registry: ResolvedCapability[],
  filePath: string,
): ResolvedCapability | undefined {
  const norm = filePath.replace(/\\/g, '/');
  for (const rc of registry) {
    if (rc.kind !== 'file-viewer' || !rc.match_glob) continue;
    if (globMatch(rc.match_glob, norm)) return rc;
  }
  return undefined;
}

export function globMatch(pattern: string, path: string): boolean {
  const normalizedPattern = pattern.replace(/\\/g, '/');
  const normalizedPath = path.replace(/\\/g, '/');
  let source = '';
  for (let i = 0; i < normalizedPattern.length; i += 1) {
    const char = normalizedPattern[i];
    if (char === '*' && normalizedPattern[i + 1] === '*') {
      if (normalizedPattern[i + 2] === '/') {
        source += '(?:.*/)?';
        i += 2;
      } else {
        source += '.*';
        i += 1;
      }
    } else if (char === '*') {
      source += '[^/]*';
    } else if (char === '?') {
      source += '[^/]';
    } else if (char === '{') {
      const close = normalizedPattern.indexOf('}', i + 1);
      if (close > i) {
        source += `(?:${normalizedPattern.slice(i + 1, close).split(',').map(escapeRegex).join('|')})`;
        i = close;
      } else {
        source += '\\{';
      }
    } else {
      source += escapeRegex(char);
    }
  }
  return new RegExp(`^(?:${source}|.*/${source})$`).test(normalizedPath);
}

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

export function matchArtifactRenderer(
  registry: ResolvedCapability[],
  mediaType: string,
  schemaVersion: number,
  path = '',
): ResolvedCapability | undefined {
  return registry.find((rc) => {
    if (rc.kind !== 'artifact-renderer' || !rc.renderer) return false;
    const mediaMatch = (rc.media_types ?? []).includes(mediaType);
    const pathMatch = !!rc.match_glob && !!path && globMatch(rc.match_glob, path);
    if (!mediaMatch && !pathMatch) return false;
    const min = rc.schema_version_min ?? 0;
    const max = rc.schema_version_max ?? Number.MAX_SAFE_INTEGER;
    return schemaVersion >= min && schemaVersion <= max;
  });
}

export type PackToolbarAction = {
  id: string;
  label: string;
  iconUrl?: string;
  modal: string;
  capabilityId: string;
  packId: string;
  packTitle?: string;
  title: string;
};

export function toolbarActionsFromRegistry(
  registry: ResolvedCapability[],
  packs: Array<{ id: string; title: string; custom?: boolean; enabled?: boolean }>,
): PackToolbarAction[] {
  const packTitleById = new Map(packs.map((p) => [p.id, p.title]));
  const out: PackToolbarAction[] = [];
  for (const rc of registry) {
    if (rc.platform) continue;
    const tb = rc.ui?.toolbar;
    if (!tb?.id) continue;
    const packId = rc.pack_id ?? '';
    const packTitle = packTitleById.get(packId);
    const label = normalizeToolbarChipLabel(tb.label, tb.id);
    const iconUrl = resolvePackToolbarIconUrl(packId, tb.icon);
    const modal = rc.ui?.modal ?? rc.id;
    const titleParts = [packTitle ?? packId, rc.id].filter(Boolean);
    if (modal) titleParts.push(modal);
    out.push({
      id: tb.id,
      label,
      iconUrl,
      modal,
      capabilityId: rc.id,
      packId,
      packTitle,
      title: titleParts.join(' · '),
    });
  }
  return out;
}

export function settingsKeysFromRegistry(registry: ResolvedCapability[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const rc of registry) {
    for (const k of rc.settings ?? []) {
      if (seen.has(k)) continue;
      seen.add(k);
      out.push(k);
    }
  }
  return out;
}
