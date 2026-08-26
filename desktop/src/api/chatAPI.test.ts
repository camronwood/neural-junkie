import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ChatAPI } from './chatAPI';
import { PacksApi } from './domains/packsApi';
import { MessagesApi } from './domains/messagesApi';
import { CollabApi } from './domains/collabApi';
import { AgentsApi } from './domains/agentsApi';
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

describe('MessagesApi', () => {
  it('sendMessage posts to /api/send', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify({ status: 'ok' }),
    });
    const api = new MessagesApi(hubFetch);
    const result = await api.sendMessage('general', 'hello', { name: 'User', type: 'human' });
    expect(hubFetch).toHaveBeenCalledWith(
      '/api/send',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          channel: 'general',
          content: 'hello',
          type: 'question',
          from: { name: 'User', type: 'human' },
        }),
      })
    );
    expect(result.status).toBe('ok');
  });
});

describe('CollabApi', () => {
  it('fetchCollaborations calls /api/collaborations', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [],
    });
    const api = new CollabApi(hubFetch);
    const data = await api.fetchCollaborations('general');
    expect(hubFetch).toHaveBeenCalledWith('/api/collaborations?channel=general');
    expect(data).toEqual([]);
  });
});

describe('AgentsApi', () => {
  it('fetchAgents calls /api/agents', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ id: 'a1', name: 'Agent' }],
    });
    const api = new AgentsApi(hubFetch);
    const agents = await api.fetchAgents();
    expect(hubFetch).toHaveBeenCalledWith('/api/agents');
    expect(agents).toHaveLength(1);
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
