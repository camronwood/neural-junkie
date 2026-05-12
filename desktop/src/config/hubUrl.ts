/** Default hub HTTP origin (Neural Junkie chat hub; avoids common :8080 collisions). */
export const DEFAULT_HUB_HTTP = 'http://localhost:18765';

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

/** Hub base URL (no trailing slash). Override with VITE_NJ_HUB_URL when 18765 is taken. */
export function getHubBaseURL(): string {
  const raw = import.meta.env.VITE_NJ_HUB_URL as string | undefined;
  if (raw?.trim()) {
    return normalizeLegacyHubServerAddr(raw.trim().replace(/\/$/, ''));
  }
  return DEFAULT_HUB_HTTP;
}

/** WebSocket URL for the hub (matches getHubBaseURL host/port). */
export function getHubWebSocketURL(): string {
  try {
    const u = new URL(getHubBaseURL());
    const wsProto = u.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${wsProto}//${u.host}/ws`;
  } catch {
    return 'ws://localhost:18765/ws';
  }
}
