import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  PRESENCE_LAST_PING_DAY_KEY,
  PRESENCE_OPT_OUT_KEY,
  maybeSendPresencePing,
  presenceEndpointUrl,
  setPresenceOptOut,
  utcDayString,
} from './presencePing';

function memoryStorage(initial: Record<string, string> = {}): Storage {
  const map = new Map(Object.entries(initial));
  return {
    get length() {
      return map.size;
    },
    clear: () => map.clear(),
    getItem: (k) => (map.has(k) ? map.get(k)! : null),
    key: (i) => [...map.keys()][i] ?? null,
    removeItem: (k) => {
      map.delete(k);
    },
    setItem: (k, v) => {
      map.set(k, String(v));
    },
  };
}

describe('presencePing', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-11-16T15:00:00.000Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('utcDayString is UTC YYYY-MM-DD', () => {
    expect(utcDayString(new Date('2026-11-16T23:59:59.000Z'))).toBe('2026-11-16');
  });

  it('presenceEndpointUrl strips trailing slash', () => {
    expect(presenceEndpointUrl({ VITE_PRESENCE_URL: 'https://nj-presence.example.workers.dev/' })).toBe(
      'https://nj-presence.example.workers.dev',
    );
    // Without env, non-PROD returns null (PROD falls back to DEFAULT_PRESENCE_URL).
    expect(presenceEndpointUrl({})).toBeNull();
  });

  it('sends id-only body once per day', async () => {
    const storage = memoryStorage();
    const fetchImpl = vi.fn(async () => new Response(null, { status: 204 }));
    const first = await maybeSendPresencePing({
      isDesktop: () => true,
      isDev: false,
      storage,
      fetchImpl,
      endpoint: 'https://nj-presence.example.workers.dev',
      installationId: () => '550e8400-e29b-41d4-a716-446655440000',
    });
    expect(first).toEqual({ status: 'sent' });
    expect(fetchImpl).toHaveBeenCalledTimes(1);
    const init = fetchImpl.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe('POST');
    expect(init.body).toBe(JSON.stringify({ id: '550e8400-e29b-41d4-a716-446655440000' }));
    expect(storage.getItem(PRESENCE_LAST_PING_DAY_KEY)).toBe('2026-11-16');

    const second = await maybeSendPresencePing({
      isDesktop: () => true,
      isDev: false,
      storage,
      fetchImpl,
      endpoint: 'https://nj-presence.example.workers.dev',
      installationId: () => '550e8400-e29b-41d4-a716-446655440000',
    });
    expect(second).toEqual({ status: 'skipped', reason: 'already-pinged-today' });
    expect(fetchImpl).toHaveBeenCalledTimes(1);
  });

  it('skips when opted out, not desktop, or missing endpoint', async () => {
    const storage = memoryStorage();
    setPresenceOptOut(true, storage);
    expect(storage.getItem(PRESENCE_OPT_OUT_KEY)).toBe('1');
    expect(
      await maybeSendPresencePing({
        isDesktop: () => true,
        isDev: false,
        storage,
        endpoint: 'https://x',
        fetchImpl: vi.fn(),
      }),
    ).toEqual({ status: 'skipped', reason: 'opted-out' });

    expect(
      await maybeSendPresencePing({
        isDesktop: () => false,
        isDev: false,
        storage: memoryStorage(),
        endpoint: 'https://x',
        fetchImpl: vi.fn(),
      }),
    ).toEqual({ status: 'skipped', reason: 'not-desktop' });

    expect(
      await maybeSendPresencePing({
        isDesktop: () => true,
        isDev: false,
        storage: memoryStorage(),
        endpoint: null,
        fetchImpl: vi.fn(),
      }),
    ).toEqual({ status: 'skipped', reason: 'no-endpoint' });

    expect(
      await maybeSendPresencePing({
        isDesktop: () => true,
        isDev: true,
        storage: memoryStorage(),
        endpoint: 'https://x',
        fetchImpl: vi.fn(),
      }),
    ).toEqual({ status: 'skipped', reason: 'dev' });
  });

  it('fails open on network/timeout without marking day', async () => {
    const storage = memoryStorage();
    const fetchImpl = vi.fn(
      () =>
        new Promise<Response>((_resolve, reject) => {
          setTimeout(() => reject(Object.assign(new Error('aborted'), { name: 'AbortError' })), 10);
        }),
    );
    const pending = maybeSendPresencePing({
      isDesktop: () => true,
      isDev: false,
      storage,
      fetchImpl,
      endpoint: 'https://nj-presence.example.workers.dev',
      installationId: () => '550e8400-e29b-41d4-a716-446655440000',
      timeoutMs: 5,
    });
    await vi.advanceTimersByTimeAsync(20);
    const result = await pending;
    expect(result.status).toBe('failed');
    expect(storage.getItem(PRESENCE_LAST_PING_DAY_KEY)).toBeNull();
  });

  it('does not mark day on non-2xx', async () => {
    const storage = memoryStorage();
    const result = await maybeSendPresencePing({
      isDesktop: () => true,
      isDev: false,
      storage,
      fetchImpl: vi.fn(async () => new Response('nope', { status: 500 })),
      endpoint: 'https://nj-presence.example.workers.dev',
      installationId: () => '550e8400-e29b-41d4-a716-446655440000',
    });
    expect(result).toEqual({ status: 'failed', reason: 'http-500' });
    expect(storage.getItem(PRESENCE_LAST_PING_DAY_KEY)).toBeNull();
  });
});
