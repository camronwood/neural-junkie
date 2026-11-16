import {
  ACTIVE_WINDOW_DAYS,
  INSTALL_KEY_PREFIX,
  META_STATS_CACHE_KEY,
  META_UNIQUE_KEY,
  authorizeStats,
  countActiveInWindow,
  parsePingBody,
  parseStatsCache,
  parseUniqueCount,
  utcDayString,
  type StatsCache,
  type StatsResult,
} from './presence';

export interface Env {
  PRESENCE_KV: KVNamespace;
  PRESENCE_STATS_TOKEN?: string;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);

    if (url.pathname === '/healthz') {
      return json({ ok: true, service: 'nj-presence' });
    }

    if (url.pathname === '/') {
      return new Response('Neural Junkie presence (anonymous install ping)\n', {
        headers: { 'content-type': 'text/plain; charset=utf-8' },
      });
    }

    if (url.pathname === '/v1/ping') {
      if (request.method !== 'POST') {
        return text('Method not allowed', 405);
      }
      return handlePing(request, env);
    }

    if (url.pathname === '/v1/stats') {
      if (request.method !== 'GET') {
        return text('Method not allowed', 405);
      }
      if (!authorizeStats(request, env.PRESENCE_STATS_TOKEN)) {
        return text('Unauthorized', 401);
      }
      const stats = await computeStats(env);
      return json(stats);
    }

    return text('Not found', 404);
  },
};

async function handlePing(request: Request, env: Env): Promise<Response> {
  let raw: unknown;
  try {
    raw = await request.json();
  } catch {
    return text('invalid JSON', 400);
  }

  const parsed = parsePingBody(raw);
  if (!parsed.ok) {
    return text(parsed.error, 400);
  }

  const today = utcDayString();
  const key = `${INSTALL_KEY_PREFIX}${parsed.id}`;
  const existing = await env.PRESENCE_KV.get(key);

  if (!existing) {
    await env.PRESENCE_KV.put(key, today);
    const current = parseUniqueCount(await env.PRESENCE_KV.get(META_UNIQUE_KEY));
    await env.PRESENCE_KV.put(META_UNIQUE_KEY, String(current + 1));
    await invalidateStatsCache(env);
    return new Response(null, { status: 204 });
  }

  if (existing !== today) {
    await env.PRESENCE_KV.put(key, today);
    await invalidateStatsCache(env);
  }

  return new Response(null, { status: 204 });
}

async function invalidateStatsCache(env: Env): Promise<void> {
  await env.PRESENCE_KV.delete(META_STATS_CACHE_KEY);
}

async function computeStats(env: Env): Promise<StatsResult> {
  const now = Date.now();
  const cached = parseStatsCache(await env.PRESENCE_KV.get(META_STATS_CACHE_KEY), now);
  if (cached) return cached;

  const unique = parseUniqueCount(await env.PRESENCE_KV.get(META_UNIQUE_KEY));
  const today = utcDayString();
  const lastSeenDays: string[] = [];
  let cursor: string | undefined;

  do {
    const page = await env.PRESENCE_KV.list({
      prefix: INSTALL_KEY_PREFIX,
      cursor,
      limit: 1000,
    });
    for (const key of page.keys) {
      const day = await env.PRESENCE_KV.get(key.name);
      if (day) lastSeenDays.push(day);
    }
    cursor = page.list_complete ? undefined : page.cursor;
  } while (cursor);

  const active_30d = countActiveInWindow(lastSeenDays, today, ACTIVE_WINDOW_DAYS);
  const result: StatsResult = { unique_installs: unique, active_30d };
  const cache: StatsCache = {
    ...result,
    computed_at: new Date(now).toISOString(),
  };
  await env.PRESENCE_KV.put(META_STATS_CACHE_KEY, JSON.stringify(cache));
  return result;
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json; charset=utf-8' },
  });
}

function text(message: string, status = 200): Response {
  return new Response(message, {
    status,
    headers: { 'content-type': 'text/plain; charset=utf-8' },
  });
}
