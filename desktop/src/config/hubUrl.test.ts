import { describe, expect, it } from 'vitest';
import { DEFAULT_HUB_HTTP, getHubWebSocketURL, normalizeHubBaseURL } from './hubUrl';

describe('hubUrl', () => {
  it('defaults to the hub IPv4 loopback address', () => {
    expect(DEFAULT_HUB_HTTP).toBe('http://127.0.0.1:18765');
    expect(getHubWebSocketURL()).toBe('ws://127.0.0.1:18765/ws');
  });

  it('normalizes legacy local port 8080 URLs to the current hub port', () => {
    expect(normalizeHubBaseURL('localhost:8080/api')).toBe('http://localhost:18765');
  });
});
