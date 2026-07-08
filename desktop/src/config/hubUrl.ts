/** Default hub HTTP origin (Neural Junkie chat hub; server binds IPv4 loopback by default). */
export const DEFAULT_HUB_HTTP = 'http://127.0.0.1:18765';

/**
 * Map legacy local hub URLs that used port 8080 to the current default (18765).
 * Idempotent for already-correct URLs. Does not change non-local hosts or other ports.
 */
export function normalizeLegacyHubServerAddr(addr: string): string {
  const t = addr.trim();
  if (!t) return DEFAULT_HUB_HTTP;
  try {
    const withScheme = t.includes('://') ? t : `http://${t}`;
    const u = new URL(withScheme);
    const port = u.port ? parseInt(u.port, 10) : u.protocol === 'https:' ? 443 : 80;
    const h = u.hostname.toLowerCase();
    const isLocal =
      h === 'localhost' || h === '127.0.0.1' || h === '[::1]' || h === '::1';
    if (isLocal && port === 8080) {
      u.port = '18765';
      return u.toString().replace(/\/$/, '');
    }
  } catch {
    /* keep original string */
  }
  return t;
}

/** Canonical hub origin: scheme, no trailing slash, no `/api` suffix. */
export function normalizeHubBaseURL(addr: string): string {
  let base = normalizeLegacyHubServerAddr(addr.trim());
  if (!base) return DEFAULT_HUB_HTTP;
  if (!base.includes('://')) {
    base = `http://${base}`;
  }
  base = base.replace(/\/+$/, '');
  if (base.toLowerCase().endsWith('/api')) {
    base = base.slice(0, -4);
  }
  return base;
}

/** Hub base URL (no trailing slash). Override with VITE_NJ_HUB_URL when 18765 is taken. */
let connectionHubUrl: string | undefined;
let connectionHubToken: string | undefined;

export function setHubConnectionOverride(url: string, token: string): void {
  connectionHubUrl = url?.trim() || undefined;
  connectionHubToken = token?.trim() || undefined;
}

export function clearHubConnectionOverride(): void {
  connectionHubUrl = undefined;
  connectionHubToken = undefined;
}

export function getHubBaseURL(): string {
  if (connectionHubUrl) {
    return normalizeHubBaseURL(connectionHubUrl);
  }
  const raw = import.meta.env.VITE_NJ_HUB_URL as string | undefined;
  if (raw?.trim()) {
    return normalizeHubBaseURL(raw);
  }
  return DEFAULT_HUB_HTTP;
}

/** Optional hub API token (must match server NEURAL_JUNKIE_HUB_TOKEN). */
export function getHubAccessToken(): string | undefined {
  if (connectionHubToken) {
    return connectionHubToken;
  }
  const raw = import.meta.env.VITE_NJ_HUB_TOKEN as string | undefined;
  const t = raw?.trim();
  return t || undefined;
}

/** Headers for authenticated hub mutations when VITE_NJ_HUB_TOKEN is set. */
export function hubAuthHeaders(): Record<string, string> {
  const token = getHubAccessToken();
  if (!token) return {};
  return { 'X-NJ-Hub-Token': token };
}

let hubSessionToken: string | null = null;

export function setHubSessionToken(token: string | null): void {
  hubSessionToken = token?.trim() || null;
}

export function getHubSessionToken(): string | null {
  return hubSessionToken;
}

/** Session from POST /api/auth/session (channel ACL + per-user auth). */
export function hubSessionHeaders(): Record<string, string> {
  if (!hubSessionToken) return {};
  return { 'X-NJ-Session': hubSessionToken };
}

/** WebSocket URL for the hub (matches getHubBaseURL host/port). */
export function getHubWebSocketURL(): string {
  try {
    const u = new URL(getHubBaseURL());
    const wsProto = u.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = getHubAccessToken();
    const sess = getHubSessionToken();
    const q = new URLSearchParams();
    if (token) q.set('hub_token', token);
    if (sess) q.set('nj_session', sess);
    const qs = q.toString();
    return `${wsProto}//${u.host}/ws${qs ? `?${qs}` : ''}`;
  } catch {
    return 'ws://127.0.0.1:18765/ws';
  }
}
