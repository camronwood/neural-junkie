import { getHubAccessToken, getHubSessionToken, getHubWebSocketHost } from '../../config/hubUrl';

function httpBaseToWsBase(baseURL: string): string {
  const trimmed = baseURL.replace(/\/+$/, '');
  const wsURL = trimmed.replace(/^http:\/\//i, 'ws://').replace(/^https:\/\//i, 'wss://');
  try {
    const u = new URL(wsURL);
    if (u.hostname === '127.0.0.1') {
      u.hostname = 'localhost';
    }
    return `${u.protocol}//${u.host}`;
  } catch {
    return `ws://${getHubWebSocketHost()}`;
  }
}

/**
 * Build hub WebSocket URL for a channel subscription.
 * Browser WebSocket cannot set X-NJ-Hub-Token; use hub_token query when LAN-exposed.
 * Non-browser clients should prefer the X-NJ-Hub-Token header on upgrade.
 */
export function buildChannelWebSocketURL(
  baseURL: string,
  channel: string,
  extraChannels: string[] = []
): string {
  const wsURL = httpBaseToWsBase(baseURL);
  const token = getHubAccessToken();
  const params = new URLSearchParams();
  params.set('channel', channel);
  const extra = extraChannels.filter((c) => c && c !== channel);
  if (extra.length > 0) {
    params.set('extra', extra.join(','));
  }
  if (token) {
    params.set('hub_token', token);
  }
  const session = getHubSessionToken();
  if (session) {
    params.set('nj_session', session);
  }
  return `${wsURL}/ws?${params.toString()}`;
}

/** Build hub WebSocket URL for a thread subscription (includes hub_token when configured). */
export function buildThreadWebSocketURL(
  baseURL: string,
  channel: string,
  threadId: string
): string {
  const wsURL = httpBaseToWsBase(baseURL);
  const token = getHubAccessToken();
  const params = new URLSearchParams();
  params.set('channel', channel);
  params.set('thread', threadId);
  if (token) {
    params.set('hub_token', token);
  }
  const session = getHubSessionToken();
  if (session) {
    params.set('nj_session', session);
  }
  return `${wsURL}/ws?${params.toString()}`;
}
