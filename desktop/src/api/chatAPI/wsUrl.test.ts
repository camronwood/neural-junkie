import { afterEach, describe, expect, it, vi } from 'vitest';
import { buildChannelWebSocketURL } from './wsUrl';

describe('buildChannelWebSocketURL', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('includes hub_token when VITE_NJ_HUB_TOKEN is set', () => {
    vi.stubEnv('VITE_NJ_HUB_TOKEN', 'secret-token');
    const url = buildChannelWebSocketURL('http://127.0.0.1:18765', 'general');
    expect(url).toContain('hub_token=secret-token');
    expect(url).toMatch(/^ws:\/\/127\.0\.0\.1:18765\/ws\?/);
    expect(url).toContain('channel=general');
  });

  it('omits hub_token when VITE_NJ_HUB_TOKEN is unset', () => {
    vi.stubEnv('VITE_NJ_HUB_TOKEN', '');
    const url = buildChannelWebSocketURL('http://127.0.0.1:18765', 'general');
    expect(url).not.toContain('hub_token');
  });
});
