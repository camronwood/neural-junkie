/** Hub base URL (no trailing slash). Override with VITE_NJ_HUB_URL when 18765 is taken. */
export function getHubBaseURL(): string {
  const raw = import.meta.env.VITE_NJ_HUB_URL as string | undefined;
  if (raw?.trim()) return raw.trim().replace(/\/$/, '');
  return 'http://localhost:18765';
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
