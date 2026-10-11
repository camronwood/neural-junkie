import { beforeEach, describe, expect, it, vi } from 'vitest';
import { loadChannelsFromHub } from './useChatWindowData';
import { useChatStore } from '../stores/chatStore';
import type { Channel, Message } from '../types/protocol';
import type { ChatAPI } from '../api/chatAPI';

describe('loadChannelsFromHub missing-channel fallback', () => {
  beforeEach(() => {
    useChatStore.getState().reset();
    useChatStore.getState().setServerAddr('http://127.0.0.1:18765');
    useChatStore.getState().setChannel('stale-dm');
  });

  it('switches to general and loads history when active channel is missing', async () => {
    const channels: Channel[] = [
      {
        id: 'ch-general',
        name: 'general',
        type: 'public',
        description: '',
        created: '',
        agents: [],
      },
    ];
    const history: Message[] = [
      {
        id: 'g1',
        type: 'chat',
        channel: 'general',
        from: { id: 'u', name: 'User', type: 'human' },
        content: 'welcome',
        timestamp: new Date().toISOString(),
      },
    ];
    const api = {
      fetchChannels: vi.fn().mockResolvedValue(channels),
      fetchMessages: vi.fn().mockResolvedValue(history),
    } as unknown as ChatAPI;

    await loadChannelsFromHub(api);
    expect(useChatStore.getState().channel).toBe('general');
    expect(api.fetchMessages).toHaveBeenCalledWith('general', 50);
    expect(useChatStore.getState().messages.map((m) => m.id)).toEqual(['g1']);
  });

  it('uses onMissingActiveChannel when provided', async () => {
    const onMissingActiveChannel = vi.fn().mockResolvedValue(undefined);
    const api = {
      fetchChannels: vi.fn().mockResolvedValue([
        {
          id: 'ch-general',
          name: 'general',
          type: 'public',
          description: '',
          created: '',
          agents: [],
        },
      ]),
      fetchMessages: vi.fn(),
    } as unknown as ChatAPI;

    await loadChannelsFromHub(api, { onMissingActiveChannel });
    expect(onMissingActiveChannel).toHaveBeenCalledWith('general');
    expect(api.fetchMessages).not.toHaveBeenCalled();
  });

  it('does not switch when active channel still exists', async () => {
    useChatStore.getState().setChannel('general');
    const api = {
      fetchChannels: vi.fn().mockResolvedValue([
        {
          id: 'ch-general',
          name: 'general',
          type: 'public',
          description: '',
          created: '',
          agents: [],
        },
      ]),
      fetchMessages: vi.fn(),
    } as unknown as ChatAPI;

    await loadChannelsFromHub(api);
    expect(useChatStore.getState().channel).toBe('general');
    expect(api.fetchMessages).not.toHaveBeenCalled();
  });
});
