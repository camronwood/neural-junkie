import { describe, expect, it } from 'vitest';
import {
  DEFAULT_HUB_HTTP,
  getHubAccessToken,
  getHubBaseURL,
  getHubWebSocketURL,
  normalizeHubBaseURL,
  setHubConnectionOverride,
  setHubSessionToken,
  subscribeHubAuth,
} from './hubUrl';

describe('hubUrl', () => {
  it('defaults to the hub IPv4 loopback address', () => {
    expect(DEFAULT_HUB_HTTP).toBe('http://127.0.0.1:18765');
    expect(getHubWebSocketURL()).toBe('ws://localhost:18765/ws');
  });

  it('normalizes legacy local port 8080 URLs to the current hub port', () => {
    expect(normalizeHubBaseURL('localhost:8080/api')).toBe('http://localhost:18765');
  });

  it('prefers connection store override over build defaults', () => {
    setHubConnectionOverride('http://192.168.1.50:19999', 'tok');
    expect(getHubBaseURL()).toBe('http://192.168.1.50:19999');
    expect(getHubWebSocketURL()).toBe('ws://192.168.1.50:19999/ws?hub_token=tok');
    setHubConnectionOverride('', '');
    expect(getHubBaseURL()).toBe(DEFAULT_HUB_HTTP);
  });

  it('includes nj_session in WebSocket URL when session is set', () => {
    setHubSessionToken('sess-abc');
    expect(getHubWebSocketURL()).toContain('nj_session=sess-abc');
    setHubSessionToken(null);
    expect(getHubWebSocketURL()).not.toContain('nj_session=');
  });

  it('notifies auth subscribers when session or connection override changes', () => {
    let calls = 0;
    const unsub = subscribeHubAuth(() => {
      calls += 1;
    });
    setHubSessionToken('one');
    setHubConnectionOverride('http://127.0.0.1:18766', 'hub-tok');
    unsub();
    setHubSessionToken(null);
    setHubConnectionOverride('', '');
    expect(calls).toBe(2);
    expect(getHubAccessToken()).toBeUndefined();
  });
});
