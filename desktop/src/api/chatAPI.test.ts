import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ChatAPI } from './chatAPI';
import { PacksApi } from './domains/packsApi';
import { getHubBaseURL, hubAuthHeaders, hubSessionHeaders, normalizeHubBaseURL, setHubSessionToken, getHubSessionToken } from '../config/hubUrl';

describe('PacksApi', () => {
  it('fetchPacks calls /api/packs', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ packs: [], layout_owner: 'ide' }),
    });
    const api = new PacksApi(hubFetch);
    const data = await api.fetchPacks();
    expect(hubFetch).toHaveBeenCalledWith('/api/packs');
    expect(data.layout_owner).toBe('ide');
  });

  it('setLayoutOwner sends pack_id', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ packs: [], layout_owner: 'software-development' }),
    });
    const api = new PacksApi(hubFetch);
    await api.setLayoutOwner('software-development');
    expect(hubFetch).toHaveBeenCalledWith(
      '/api/packs/layout-owner',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ layout_owner: 'software-development' }),
      })
    );
  });
});

describe('ChatAPI hubFetch 401', () => {
  beforeEach(() => {
    setHubSessionToken('test-token');
  });

  afterEach(() => {
    setHubSessionToken(null);
  });

  it('clears session token on 401', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: false,
      status: 401,
      statusText: 'Unauthorized',
    } as Response);

    const api = new ChatAPI('http://127.0.0.1:18765');
    await expect(api.fetchPacks()).rejects.toThrow();
    expect(getHubSessionToken()).toBeNull();

    fetchMock.mockRestore();
  });
});
