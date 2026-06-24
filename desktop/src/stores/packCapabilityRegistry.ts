import type { ResolvedCapability } from '../api/chatAPI';

/** NJ built-in viewer ids referenced by pack capability_defs.viewer */
export const NJ_VIEWER = {
  SCAN_SUMMARY: 'nj.scan-summary',
  SCAN_ANALYSIS: 'nj.scan-analysis',
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

/** Match file-viewer capabilities by glob (simple ** suffix/prefix rules). */
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

function globMatch(pattern: string, path: string): boolean {
  const p = pattern.replace(/\\/g, '/');
  if (p.startsWith('**/')) {
    const suffix = p.slice(3);
    return path.endsWith(suffix) || path.includes('/' + suffix);
  }
  if (p.includes('*')) {
    const parts = p.split('*');
    if (parts.length === 2) {
      return path.startsWith(parts[0]) && path.endsWith(parts[1]);
    }
  }
  return path === p || path.endsWith('/' + p);
}

export function toolbarActionsFromRegistry(
  registry: ResolvedCapability[],
): Array<{ id: string; label: string; modal: string; capabilityId: string }> {
  const out: Array<{ id: string; label: string; modal: string; capabilityId: string }> = [];
  for (const rc of registry) {
    const tb = rc.ui?.toolbar;
    if (!tb?.id) continue;
    out.push({
      id: tb.id,
      label: tb.label ?? tb.id,
      modal: rc.ui?.modal ?? rc.id,
      capabilityId: rc.id,
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
