import {
  getHubSessionToken,
  hubAuthHeaders,
  hubSessionHeaders,
  setHubSessionToken,
} from '../config/hubUrl';

/** Headers for hub mutations (session + optional hub token). */
export function hubMutationHeaders(extra?: Record<string, string>): Record<string, string> {
  return {
    'Content-Type': 'application/json',
    ...hubAuthHeaders(),
    ...hubSessionHeaders(),
    ...extra,
  };
}

/**
 * Ensure a hub session exists so mutations succeed when auth_required is on
 * (or when relaxed_local is off). Safe to call before login / during first-run setup.
 */
export async function ensureHubMutationSession(
  serverAddr: string,
  username = 'setup',
): Promise<string> {
  const existing = getHubSessionToken();
  if (existing) return existing;

  const resp = await fetch(`${serverAddr}/api/auth/session`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...hubAuthHeaders(),
    },
    body: JSON.stringify({ username }),
  });
  if (!resp.ok) {
    throw new Error(`Failed to create hub session: ${resp.status} ${await resp.text()}`);
  }
  const data = (await resp.json()) as { token?: string };
  if (!data.token?.trim()) {
    throw new Error('Hub session response missing token');
  }
  setHubSessionToken(data.token);
  return data.token;
}

/** PUT JSON to a hub path with mutation auth, minting a session if needed. */
export async function hubMutationPut(
  serverAddr: string,
  path: string,
  body: unknown,
): Promise<Response> {
  await ensureHubMutationSession(serverAddr);
  const url = path.startsWith('http') ? path : `${serverAddr}${path}`;
  const resp = await fetch(url, {
    method: 'PUT',
    headers: hubMutationHeaders(),
    body: JSON.stringify(body),
  });
  if (resp.status === 401) {
    // Stale session — clear, mint once, retry.
    setHubSessionToken(null);
    await ensureHubMutationSession(serverAddr);
    return fetch(url, {
      method: 'PUT',
      headers: hubMutationHeaders(),
      body: JSON.stringify(body),
    });
  }
  return resp;
}
