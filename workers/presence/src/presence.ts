/** Presence-only unique install counting (opaque UUID + last-seen day). */

export const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export const INSTALL_KEY_PREFIX = 'i:';
export const META_UNIQUE_KEY = 'meta:unique';
export const META_STATS_CACHE_KEY = 'meta:stats_cache';
export const STATS_CACHE_TTL_MS = 60 * 60 * 1000;
export const ACTIVE_WINDOW_DAYS = 30;

export type PingResult =
  | { ok: true; created: boolean; updated: boolean }
  | { ok: false; status: number; error: string };

export type StatsResult = {
  unique_installs: number;
  active_30d: number;
};

export type StatsCache = StatsResult & { computed_at: string };

export function utcDayString(now = new Date()): string {
  return now.toISOString().slice(0, 10);
}

export function dayOffsetUtc(day: string, daysAgo: number): string {
  const d = new Date(`${day}T00:00:00.000Z`);
  d.setUTCDate(d.getUTCDate() - daysAgo);
  return d.toISOString().slice(0, 10);
}

export function isValidInstallId(id: unknown): id is string {
  return typeof id === 'string' && UUID_RE.test(id);
}

/** Accept only `{ id: string }` with no extra keys. */
export function parsePingBody(raw: unknown): { ok: true; id: string } | { ok: false; error: string } {
  if (raw === null || typeof raw !== 'object' || Array.isArray(raw)) {
    return { ok: false, error: 'body must be a JSON object' };
  }
  const keys = Object.keys(raw as Record<string, unknown>);
  if (keys.length !== 1 || keys[0] !== 'id') {
    return { ok: false, error: 'body must be {"id":"<uuid>"} only' };
  }
  const id = (raw as { id: unknown }).id;
  if (!isValidInstallId(id)) {
    return { ok: false, error: 'id must be a UUID' };
  }
  return { ok: true, id: id.toLowerCase() };
}

export function countActiveInWindow(
  lastSeenById: Iterable<string>,
  today: string,
  windowDays = ACTIVE_WINDOW_DAYS,
): number {
  const cutoff = dayOffsetUtc(today, windowDays - 1);
  let n = 0;
  for (const day of lastSeenById) {
    if (day >= cutoff && day <= today) n += 1;
  }
  return n;
}

export function parseUniqueCount(raw: string | null): number {
  if (!raw) return 0;
  const n = Number.parseInt(raw, 10);
  return Number.isFinite(n) && n >= 0 ? n : 0;
}

export function parseStatsCache(
  raw: string | null,
  nowMs: number,
  ttlMs = STATS_CACHE_TTL_MS,
): StatsResult | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<StatsCache>;
    if (
      typeof parsed.unique_installs !== 'number' ||
      typeof parsed.active_30d !== 'number' ||
      typeof parsed.computed_at !== 'string'
    ) {
      return null;
    }
    const at = Date.parse(parsed.computed_at);
    if (!Number.isFinite(at) || nowMs - at > ttlMs) return null;
    return {
      unique_installs: parsed.unique_installs,
      active_30d: parsed.active_30d,
    };
  } catch {
    return null;
  }
}

export function authorizeStats(request: Request, expectedToken: string | undefined): boolean {
  if (!expectedToken) return false;
  const header = request.headers.get('authorization')?.trim() ?? '';
  if (header.toLowerCase().startsWith('bearer ')) {
    return header.slice(7).trim() === expectedToken;
  }
  const url = new URL(request.url);
  const q = url.searchParams.get('token')?.trim() ?? '';
  return q.length > 0 && q === expectedToken;
}
