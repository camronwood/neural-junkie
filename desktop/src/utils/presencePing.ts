import { isTauri } from '@tauri-apps/api/core';
import { getOrCreateInstallationId } from './appUpdater';

export const PRESENCE_OPT_OUT_KEY = 'nj-presence-opt-out-v1';
export const PRESENCE_LAST_PING_DAY_KEY = 'nj-presence-last-ping-day-v1';
export const PRESENCE_PING_TIMEOUT_MS = 5_000;

export type PresencePingOutcome =
  | { status: 'sent' }
  | { status: 'skipped'; reason: string }
  | { status: 'failed'; reason: string };

export function utcDayString(now = new Date()): string {
  return now.toISOString().slice(0, 10);
}

export function isPresenceOptedOut(storage: Storage = localStorage): boolean {
  return storage.getItem(PRESENCE_OPT_OUT_KEY) === '1';
}

export function setPresenceOptOut(optOut: boolean, storage: Storage = localStorage): void {
  if (optOut) storage.setItem(PRESENCE_OPT_OUT_KEY, '1');
  else storage.removeItem(PRESENCE_OPT_OUT_KEY);
}

/** Public free Worker used by release builds when env is unset. */
export const DEFAULT_PRESENCE_URL = 'https://nj-presence.camronwood.workers.dev';

export function presenceEndpointUrl(
  env: ImportMetaEnv | Record<string, string | undefined> = import.meta.env,
): string | null {
  const raw = (env as { VITE_PRESENCE_URL?: string }).VITE_PRESENCE_URL?.trim();
  if (raw) return raw.replace(/\/$/, '');
  // Production desktop builds: fall back to the public free endpoint.
  if (import.meta.env.PROD) return DEFAULT_PRESENCE_URL;
  return null;
}

export type PresencePingDeps = {
  isDesktop?: () => boolean;
  isDev?: boolean;
  storage?: Storage;
  fetchImpl?: typeof fetch;
  now?: () => Date;
  endpoint?: string | null;
  installationId?: () => string;
  timeoutMs?: number;
};

/**
 * Fire-and-forget anonymous presence: POST {"id":"<uuid>"} once per UTC day.
 * Never throws; never blocks the UI.
 */
export async function maybeSendPresencePing(deps: PresencePingDeps = {}): Promise<PresencePingOutcome> {
  const isDesktop = deps.isDesktop ?? (() => typeof window !== 'undefined' && isTauri());
  const isDev = deps.isDev ?? (import.meta.env.DEV && import.meta.env.MODE !== 'test');
  const storage = deps.storage ?? localStorage;
  const fetchImpl = deps.fetchImpl ?? fetch.bind(globalThis);
  const now = deps.now ?? (() => new Date());
  const timeoutMs = deps.timeoutMs ?? PRESENCE_PING_TIMEOUT_MS;
  const getId = deps.installationId ?? getOrCreateInstallationId;

  if (isDev) return { status: 'skipped', reason: 'dev' };
  if (!isDesktop()) return { status: 'skipped', reason: 'not-desktop' };
  if (isPresenceOptedOut(storage)) return { status: 'skipped', reason: 'opted-out' };

  const endpoint = deps.endpoint !== undefined ? deps.endpoint : presenceEndpointUrl();
  if (!endpoint) return { status: 'skipped', reason: 'no-endpoint' };

  const today = utcDayString(now());
  if (storage.getItem(PRESENCE_LAST_PING_DAY_KEY) === today) {
    return { status: 'skipped', reason: 'already-pinged-today' };
  }

  const id = getId();
  const body = JSON.stringify({ id });
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const res = await fetchImpl(`${endpoint}/v1/ping`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body,
      signal: controller.signal,
    });
    if (!res.ok) {
      return { status: 'failed', reason: `http-${res.status}` };
    }
    storage.setItem(PRESENCE_LAST_PING_DAY_KEY, today);
    return { status: 'sent' };
  } catch (err) {
    const reason = err instanceof Error ? err.name === 'AbortError' ? 'timeout' : err.message : 'network';
    return { status: 'failed', reason };
  } finally {
    clearTimeout(timer);
  }
}
