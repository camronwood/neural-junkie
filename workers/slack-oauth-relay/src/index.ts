import {
  BOT_CALLBACK_PATH,
  USER_DM_CALLBACK_PATH,
  buildRelayRedirectURL,
  parseOAuthState,
} from './relay';

const CALLBACK_PATHS = new Set([BOT_CALLBACK_PATH, USER_DM_CALLBACK_PATH]);

export default {
  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);

    if (url.pathname === '/healthz') {
      return json({ ok: true, service: 'nj-slack-oauth-relay' });
    }

    if (url.pathname === '/') {
      return new Response('Neural Junkie Slack OAuth relay\n', {
        headers: { 'content-type': 'text/plain; charset=utf-8' },
      });
    }

    if (request.method !== 'GET') {
      return text('Method not allowed', 405);
    }

    const path = normalizeCallbackPath(url.pathname);
    if (!path || !CALLBACK_PATHS.has(path)) {
      return text('Not found', 404);
    }

    const state = url.searchParams.get('state')?.trim() ?? '';
    if (!state) {
      return text('missing state', 400);
    }

    const parsed = parseOAuthState(state);
    if (!parsed.ok || ('legacy' in parsed && parsed.legacy) || !parsed.localReturn) {
      return text('invalid state — start Connect Slack from Neural Junkie again', 400);
    }

    const target = buildRelayRedirectURL(parsed.localReturn, url.searchParams);
    if (!target) {
      return text('slack oauth relay: callback must be loopback http', 400);
    }

    return Response.redirect(target, 302);
  },
};

function normalizeCallbackPath(pathname: string): string | null {
  if (CALLBACK_PATHS.has(pathname)) {
    return pathname;
  }
  if (pathname.endsWith(BOT_CALLBACK_PATH)) {
    return BOT_CALLBACK_PATH;
  }
  if (pathname.endsWith(USER_DM_CALLBACK_PATH)) {
    return USER_DM_CALLBACK_PATH;
  }
  return null;
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
