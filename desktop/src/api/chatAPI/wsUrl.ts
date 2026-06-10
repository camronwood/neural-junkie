import { getHubAccessToken } from '../../config/hubUrl';

/** Build hub WebSocket URL for a channel subscription (includes hub_token when configured). */
export function buildChannelWebSocketURL(
  baseURL: string,
  channel: string,
  extraChannels: string[] = []
): string {
  const wsURL = baseURL.replace('http://', 'ws://').replace('https://', 'wss://');
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
  return `${wsURL}/ws?${params.toString()}`;
}

/** Build hub WebSocket URL for a thread subscription (includes hub_token when configured). */
export function buildThreadWebSocketURL(
  baseURL: string,
  channel: string,
  threadId: string
): string {
  const wsURL = baseURL.replace('http://', 'ws://').replace('https://', 'wss://');
  const token = getHubAccessToken();
  const params = new URLSearchParams();
  params.set('channel', channel);
  params.set('thread', threadId);
  if (token) {
    params.set('hub_token', token);
  }
  return `${wsURL}/ws?${params.toString()}`;
}
