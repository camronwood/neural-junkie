import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  getHubSessionToken: vi.fn(),
  setHubSessionToken: vi.fn(),
  hubAuthHeaders: vi.fn(() => ({})),
  hubSessionHeaders: vi.fn(() => ({})),
}));

vi.mock('../config/hubUrl', () => ({
  getHubSessionToken: mocks.getHubSessionToken,
  setHubSessionToken: mocks.setHubSessionToken,
  hubAuthHeaders: mocks.hubAuthHeaders,
  hubSessionHeaders: mocks.hubSessionHeaders,
}));

import {
  ensureHubMutationSession,
  hubMutationHeaders,
  hubMutationPut,
} from './hubMutation';

describe('hubMutation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getHubSessionToken.mockReturnValue(null);
    mocks.hubAuthHeaders.mockReturnValue({});
    mocks.hubSessionHeaders.mockReturnValue({ 'X-NJ-Session': 'sess-1' });
  });

  it('hubMutationHeaders merges content-type, auth, and session', () => {
    mocks.hubAuthHeaders.mockReturnValue({ 'X-NJ-Hub-Token': 'hub' });
    expect(hubMutationHeaders({ 'X-Extra': '1' })).toEqual({
      'Content-Type': 'application/json',
      'X-NJ-Hub-Token': 'hub',
      'X-NJ-Session': 'sess-1',
      'X-Extra': '1',
    });
  });

  it('ensureHubMutationSession reuses an existing token', async () => {
    mocks.getHubSessionToken.mockReturnValue('existing');
    await expect(ensureHubMutationSession('http://127.0.0.1:18765')).resolves.toBe('existing');
  });

  it('ensureHubMutationSession mints and stores a session when missing', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ token: 'new-token' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(ensureHubMutationSession('http://127.0.0.1:18765', 'setup')).resolves.toBe(
      'new-token',
    );
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:18765/api/auth/session',
      expect.objectContaining({ method: 'POST' }),
    );
    expect(mocks.setHubSessionToken).toHaveBeenCalledWith('new-token');
  });

  it('hubMutationPut sends authenticated PUT and surfaces non-ok', async () => {
    mocks.getHubSessionToken.mockReturnValue('sess-1');
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => 'ok',
    });
    vi.stubGlobal('fetch', fetchMock);

    const resp = await hubMutationPut('http://127.0.0.1:18765', '/api/settings', {
      setup_completed: true,
    });
    expect(resp.ok).toBe(true);
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:18765/api/settings',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ setup_completed: true }),
      }),
    );
  });
});
