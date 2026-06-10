/** Mirrors internal/integrations/slack/oauth_state.go and oauth_relay.go */

export const OAUTH_STATE_PREFIX = 'nj1.';
export const BOT_CALLBACK_PATH = '/api/slack/oauth/callback';
export const USER_DM_CALLBACK_PATH = '/api/slack/oauth/user-dm/callback';

const ALLOWED_CALLBACK_PATHS = new Set([BOT_CALLBACK_PATH, USER_DM_CALLBACK_PATH]);

export type ParseStateResult =
  | { ok: true; nonce: string; localReturn: string }
  | { ok: true; nonce: string; localReturn: ''; legacy: true }
  | { ok: false };

/** Parse nj1.{nonce}.{base64url(localCallback)} or legacy plain hex state. */
export function parseOAuthState(state: string): ParseStateResult {
  const trimmed = state.trim();
  if (!trimmed) {
    return { ok: false };
  }
  if (!trimmed.startsWith(OAUTH_STATE_PREFIX)) {
    return { ok: true, nonce: trimmed, localReturn: '', legacy: true };
  }
  const rest = trimmed.slice(OAUTH_STATE_PREFIX.length);
  const dot = rest.indexOf('.');
  if (dot <= 0 || dot >= rest.length - 1) {
    return { ok: false };
  }
  const nonce = rest.slice(0, dot);
  const enc = rest.slice(dot + 1);
  let localReturn: string;
  try {
    localReturn = new TextDecoder().decode(base64UrlDecode(enc)).trim();
  } catch {
    return { ok: false };
  }
  if (!nonce || !localReturn) {
    return { ok: false };
  }
  return { ok: true, nonce, localReturn };
}

/** Only allow http loopback callbacks on known NJ hub paths. */
export function isAllowedLocalOAuthCallback(raw: string): boolean {
  let u: URL;
  try {
    u = new URL(raw.trim());
  } catch {
    return false;
  }
  if (u.protocol !== 'http:') {
    return false;
  }
  const host = u.hostname.toLowerCase();
  if (host !== '127.0.0.1' && host !== 'localhost') {
    return false;
  }
  const path = u.pathname.startsWith('/') ? u.pathname : `/${u.pathname}`;
  return ALLOWED_CALLBACK_PATHS.has(path);
}

/** Build loopback redirect preserving Slack query params. */
export function buildRelayRedirectURL(localCallback: string, searchParams: URLSearchParams): string | null {
  if (!isAllowedLocalOAuthCallback(localCallback)) {
    return null;
  }
  const u = new URL(localCallback.trim());
  u.search = searchParams.toString();
  return u.toString();
}

function base64UrlDecode(input: string): Uint8Array {
  const padded = input.replace(/-/g, '+').replace(/_/g, '/');
  const pad = padded.length % 4 === 0 ? padded : padded + '='.repeat(4 - (padded.length % 4));
  const binary = atob(pad);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}
