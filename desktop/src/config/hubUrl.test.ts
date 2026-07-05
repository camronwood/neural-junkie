import { describe, expect, it } from 'vitest';
import {
  DEFAULT_HUB_HTTP,
  getHubBaseURL,
  getHubWebSocketURL,
  normalizeHubBaseURL,
  setHubConnectionOverride,
} from './hubUrl';

describe('hubUrl', () => {
  it('defaults to the hub IPv4 loopback address', () => {
    expect(DEFAULT_HUB_HTTP).toBe('http://127.0.0.1:18765');
    expect(getHubWebSocketURL()).toBe('ws://127.0.0.1:18765/ws');
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
});
