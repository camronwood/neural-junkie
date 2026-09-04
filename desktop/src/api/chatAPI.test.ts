import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ChatAPI } from './chatAPI';
import { PacksApi } from './domains/packsApi';
import { MessagesApi } from './domains/messagesApi';
import { CollabApi } from './domains/collabApi';
import { AgentsApi } from './domains/agentsApi';
import { ArtifactsApi } from './domains/artifactsApi';
import { RunbooksApi } from './domains/runbooksApi';
import { RoomsApi } from './domains/roomsApi';
import { ConnectorsApi } from './domains/connectorsApi';
import { StreamsApi } from './domains/streamsApi';
import { GitChangesApi } from './domains/gitChangesApi';
import { AssistantApi } from './domains/assistantApi';
import { SlackApi } from './domains/slackApi';
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

  it('collabTaskComplete posts to task complete endpoint', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 'c1', phase: 'executing' }),
      text: async () => '',
    });
    const api = new CollabApi(hubFetch);
    await api.collabTaskComplete('c1', 't1');
    expect(hubFetch).toHaveBeenCalledWith(
      '/api/collaborations/c1/tasks/t1/complete',
      expect.objectContaining({ method: 'POST' })
    );
  });
});

describe('ConnectorsApi', () => {
  it('listConnectors calls /api/connectors', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ id: 'conn-1' }],
      text: async () => '',
    });
    const api = new ConnectorsApi(hubFetch);
    const data = await api.listConnectors();
    expect(hubFetch).toHaveBeenCalledWith('/api/connectors');
    expect(data).toHaveLength(1);
  });
});

describe('StreamsApi', () => {
  it('getStreamStatus calls /api/stream/status', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ running: true }),
      text: async () => '',
    });
    const api = new StreamsApi(hubFetch);
    const data = await api.getStreamStatus();
    expect(hubFetch).toHaveBeenCalledWith('/api/stream/status');
    expect(data.running).toBe(true);
  });
});

describe('GitChangesApi', () => {
  it('fetchGitChanges calls /api/git-changes', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ id: 'gc-1' }],
    });
    const api = new GitChangesApi(hubFetch);
    const data = await api.fetchGitChanges('user-1');
    expect(hubFetch).toHaveBeenCalledWith('/api/git-changes?user_id=user-1');
    expect(data).toHaveLength(1);
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

describe('ArtifactsApi', () => {
  it('fetchArtifacts calls /api/artifacts', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ id: 'art-1' }],
      text: async () => '',
    });
    const api = new ArtifactsApi(hubFetch);
    const data = await api.fetchArtifacts({ workspace_id: 'ws-1' });
    expect(hubFetch).toHaveBeenCalledWith('/api/artifacts?workspace_id=ws-1');
    expect(data).toHaveLength(1);
  });
});

describe('RunbooksApi', () => {
  it('listRunbookDefinitions calls /api/runbook-definitions', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ id: 'rb-1', title: 'Demo' }],
      text: async () => '',
    });
    const api = new RunbooksApi(hubFetch);
    const data = await api.listRunbookDefinitions();
    expect(hubFetch).toHaveBeenCalledWith('/api/runbook-definitions');
    expect(data).toHaveLength(1);
  });
});

describe('RoomsApi', () => {
  it('createSession posts to /api/auth/session and stores token', async () => {
    setHubSessionToken(null);
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ token: 'sess-1', username: 'camron', role: 'user' }),
      statusText: 'OK',
    });
    const api = new RoomsApi(hubFetch, 'http://127.0.0.1:18765');
    const data = await api.createSession('camron');
    expect(hubFetch).toHaveBeenCalledWith(
      '/api/auth/session',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ username: 'camron' }),
      })
    );
    expect(data.token).toBe('sess-1');
    expect(getHubSessionToken()).toBe('sess-1');
    setHubSessionToken(null);
  });
});

describe('AssistantApi', () => {
  it('fetchAssistantState calls /api/assistant/state', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ tasks: [], reminders: [] }),
    });
    const api = new AssistantApi(hubFetch);
    const data = await api.fetchAssistantState('general');
    expect(hubFetch).toHaveBeenCalledWith('/api/assistant/state?channel=general');
    expect(data.tasks).toEqual([]);
  });
});

describe('SlackApi', () => {
  it('getSlackConfig calls /api/slack/config', async () => {
    const hubFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ enabled: false }),
    });
    const api = new SlackApi(hubFetch);
    const data = await api.getSlackConfig();
    expect(hubFetch).toHaveBeenCalledWith('/api/slack/config');
    expect(data.enabled).toBe(false);
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
