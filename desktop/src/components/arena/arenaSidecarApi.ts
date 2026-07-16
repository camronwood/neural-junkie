import { getHubBaseURL, hubAuthHeaders, hubSessionHeaders } from '../../config/hubUrl';

export type ArenaStepMeta = {
  model?: string;
  seat?: string;
  reply?: string;
  parsed_move?: Record<string, unknown>;
  parsed_answer?: string;
  skipped?: boolean;
  reason?: string;
};

export type ArenaSession = {
  id: string;
  challenge: string;
  status?: string;
  result?: string;
  state?: Record<string, unknown>;
  puzzle?: { id?: string; prompt?: string; title?: string; difficulty?: string };
  players?: { white?: string; black?: string };
  moves?: Array<Record<string, unknown>>;
  _arena_step?: ArenaStepMeta;
};

async function arenaFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${getHubBaseURL()}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...hubAuthHeaders(),
      ...hubSessionHeaders(),
      ...(init?.headers as Record<string, string> | undefined),
    },
  });
  const raw = await res.text();
  let data: T & { error?: string };
  if (raw.trim()) {
    try {
      data = JSON.parse(raw) as T & { error?: string };
    } catch {
      throw new Error(raw.trim() || res.statusText || `HTTP ${res.status}`);
    }
  } else {
    data = {} as T & { error?: string };
  }
  if (!res.ok) {
    const msg =
      (typeof data === 'object' && data !== null && typeof data.error === 'string' && data.error) ||
      raw.trim() ||
      res.statusText ||
      `HTTP ${res.status}`;
    throw new Error(msg);
  }
  if (typeof data === 'object' && data !== null && 'error' in data && data.error) {
    throw new Error(data.error);
  }
  return data;
}

export async function arenaListChallenges() {
  return arenaFetch<{ challenges: Array<Record<string, unknown>>; puzzles: Array<Record<string, unknown>> }>(
    '/api/arena/challenges',
  );
}

export async function arenaCreateSession(body: Record<string, unknown>) {
  return arenaFetch<ArenaSession>('/api/arena/sessions', { method: 'POST', body: JSON.stringify(body) });
}

export async function arenaGetSession(sessionId: string) {
  return arenaFetch<ArenaSession>(`/api/arena/sessions/${sessionId}`);
}

export async function arenaMakeMove(sessionId: string, body: Record<string, unknown>) {
  return arenaFetch<ArenaSession>(`/api/arena/sessions/${sessionId}/move`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export async function arenaSubmitAnswer(sessionId: string, answer: string, by?: string) {
  return arenaFetch<ArenaSession>(`/api/arena/sessions/${sessionId}/answer`, {
    method: 'POST',
    body: JSON.stringify({ answer, by }),
  });
}

export async function arenaLeaderboard() {
  return arenaFetch<{ models: Record<string, Record<string, number>>; sessions: string[] }>(
    '/api/arena/leaderboard',
  );
}

export async function arenaMatchStep(body: {
  session_id: string;
  provider_id?: string;
  model?: string;
}) {
  return arenaFetch<ArenaSession>('/api/arena/match/step', { method: 'POST', body: JSON.stringify(body) });
}

export async function arenaMatchRun(body: {
  session_id: string;
  provider_id?: string;
  model?: string;
  max_steps?: number;
}) {
  return arenaFetch<ArenaSession>('/api/arena/match/run', { method: 'POST', body: JSON.stringify(body) });
}

export async function fetchProviders() {
  return arenaFetch<Array<{ id: string; name: string; model?: string; type?: string }>>('/api/providers');
}

/** Tags installed in local Ollama (via hub). Empty when Ollama is stopped. */
export async function fetchInstalledOllamaModels(): Promise<string[]> {
  const res = await fetch(`${getHubBaseURL()}/api/ollama/models`, {
    headers: {
      ...hubAuthHeaders(),
      ...hubSessionHeaders(),
    },
  });
  if (!res.ok) {
    return [];
  }
  const data = (await res.json()) as { models?: string[] };
  return Array.isArray(data.models) ? data.models.filter((m) => typeof m === 'string' && m.trim()) : [];
}
