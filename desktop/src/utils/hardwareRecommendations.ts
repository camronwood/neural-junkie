export type HardwareTier = 'minimal' | 'light' | 'recommended' | 'heavy';

export interface TrackRecommendation {
  primary_model: string;
  utility_model?: string;
  message: string;
}

export interface HardwareSnapshot {
  total_memory_bytes: number;
  total_memory_gb: number;
  tier: HardwareTier;
  recommendations: Record<string, TrackRecommendation>;
}

export interface ModelLookup {
  name: string;
  title?: string;
  size_hint?: string;
  estimated_disk_gb?: number;
  estimated_ram_gb?: number;
}

export const HARDWARE_DOCS_URL =
  'https://github.com/camronwood/neural-junkie/blob/main/docs/HARDWARE.md';

export function trackKeyForWizard(
  track: 'developer' | 'lifeSciences' | 'cad' | 'general',
): string {
  return track;
}

export async function fetchHardwareSnapshot(serverAddr: string): Promise<HardwareSnapshot | null> {
  try {
    const resp = await fetch(`${serverAddr}/api/system/hardware`);
    if (!resp.ok) return null;
    return (await resp.json()) as HardwareSnapshot;
  } catch {
    return null;
  }
}

export async function fetchModelLookup(
  serverAddr: string,
  name: string,
): Promise<ModelLookup | null> {
  const trimmed = name.trim();
  if (!trimmed) return null;
  try {
    const resp = await fetch(
      `${serverAddr}/api/ollama/library/lookup?name=${encodeURIComponent(trimmed)}`,
    );
    if (!resp.ok) return null;
    const data = await resp.json();
    if (data == null) return null;
    return data as ModelLookup;
  } catch {
    return null;
  }
}

/** Whether the tier should auto-downgrade developer/cad from 14B unless user opts in. */
export function shouldAutoDowngradePrimary(tier: HardwareTier | undefined): boolean {
  return tier === 'minimal' || tier === 'light';
}

export function recommendedPrimaryForTrack(
  snapshot: HardwareSnapshot | null,
  track: 'developer' | 'lifeSciences' | 'cad' | 'general',
  fallback: string,
): string {
  const key = trackKeyForWizard(track);
  return snapshot?.recommendations?.[key]?.primary_model ?? fallback;
}

export function recommendationMessageForTrack(
  snapshot: HardwareSnapshot | null,
  track: 'developer' | 'lifeSciences' | 'cad' | 'general',
): string | null {
  const key = trackKeyForWizard(track);
  return snapshot?.recommendations?.[key]?.message ?? null;
}

export function formatModelResourceHint(lookup: ModelLookup | null): string | null {
  if (!lookup) return null;
  const parts: string[] = [];
  if (lookup.size_hint) {
    parts.push(`Estimated disk: ${lookup.size_hint}`);
  } else if (lookup.estimated_disk_gb) {
    parts.push(`Estimated disk: ~${lookup.estimated_disk_gb} GB`);
  }
  if (lookup.estimated_ram_gb) {
    parts.push(`Suggested RAM: ${lookup.estimated_ram_gb} GB+`);
  }
  return parts.length > 0 ? parts.join(' · ') : null;
}
