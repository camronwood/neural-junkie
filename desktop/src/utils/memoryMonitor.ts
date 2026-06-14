import type { HardwareTier } from './hardwareRecommendations';

export interface LoadedOllamaModel {
  name: string;
  size_bytes: number;
  vram_bytes: number;
}

export interface SystemMemorySnapshot {
  total_bytes: number;
  available_bytes: number;
  used_bytes: number;
  used_percent: number;
  tier: HardwareTier;
  app_memory_bytes?: number;
  wired_memory_bytes?: number;
  compressed_memory_bytes?: number;
  ollama: {
    running: boolean;
    endpoint: string;
    loaded_models: LoadedOllamaModel[];
    loaded_bytes_total: number;
  };
}

export const MEMORY_MONITOR_POLL_MS = 4000;

export async function fetchSystemMemory(serverAddr: string): Promise<SystemMemorySnapshot | null> {
  try {
    const resp = await fetch(`${serverAddr}/api/system/memory`);
    if (!resp.ok) return null;
    return (await resp.json()) as SystemMemorySnapshot;
  } catch {
    return null;
  }
}

export function formatMemoryBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const gb = bytes / (1024 ** 3);
  if (gb >= 1) return `${gb.toFixed(gb >= 10 ? 0 : 1)} GB`;
  const mb = bytes / (1024 ** 2);
  if (mb >= 1) return `${mb.toFixed(mb >= 10 ? 0 : 1)} MB`;
  return `${Math.round(bytes / 1024)} KB`;
}

export function memoryPressureLevel(usedPercent: number): 'ok' | 'warn' | 'critical' {
  if (usedPercent >= 85) return 'critical';
  if (usedPercent >= 70) return 'warn';
  return 'ok';
}

export function memoryPressureClass(level: 'ok' | 'warn' | 'critical'): string {
  switch (level) {
    case 'critical':
      return 'bg-red-500';
    case 'warn':
      return 'bg-amber-400';
    default:
      return 'bg-green-400';
  }
}

export function memoryPressureTextClass(level: 'ok' | 'warn' | 'critical'): string {
  switch (level) {
    case 'critical':
      return 'text-red-300';
    case 'warn':
      return 'text-amber-300';
    default:
      return 'text-green-300';
  }
}

export function shortModelTag(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) return 'model';
  const slash = trimmed.lastIndexOf('/');
  const base = slash >= 0 ? trimmed.slice(slash + 1) : trimmed;
  return base.length > 18 ? `${base.slice(0, 16)}…` : base;
}
