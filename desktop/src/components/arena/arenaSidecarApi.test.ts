import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../../config/hubUrl', () => ({
  getHubBaseURL: () => 'http://127.0.0.1:18765',
  hubAuthHeaders: () => ({ Authorization: 'Bearer t' }),
  hubSessionHeaders: () => ({ 'X-Session': 's' }),
}));

describe('arenaSidecarApi', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.resetModules();
  });

  it('arenaListChallenges returns parsed JSON on success', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      statusText: 'OK',
      text: async () => JSON.stringify({ challenges: [{ id: 'chess' }], puzzles: [] }),
    });
    const { arenaListChallenges } = await import('./arenaSidecarApi');
    const data = await arenaListChallenges();
    expect(data.challenges).toHaveLength(1);
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:18765/api/arena/challenges',
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: 'Bearer t' }),
      }),
    );
  });

  it('throws error field from failed JSON response', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 502,
      statusText: 'Bad Gateway',
      text: async () => JSON.stringify({ error: 'sidecar down' }),
    });
    const { arenaGetSession } = await import('./arenaSidecarApi');
    await expect(arenaGetSession('sess-1')).rejects.toThrow('sidecar down');
  });

  it('throws raw body when response is non-JSON', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Error',
      text: async () => 'plain failure',
    });
    const { arenaCreateSession } = await import('./arenaSidecarApi');
    await expect(arenaCreateSession({ challenge: 'chess' })).rejects.toThrow('plain failure');
  });
});
