import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createChatDmActions } from '../hooks/createChatDmActions';
import { useChatStore } from '../stores/chatStore';
import type { ChatAPI } from '../api/chatAPI';
import type { Channel } from '../types/protocol';

vi.mock('../stores/settingsStore', () => {
  const state = {
    settings: {},
    isLoaded: true,
    updateSettings: vi.fn(),
  };
  return {
    useSettingsStore: Object.assign(() => state, { getState: () => state }),
  };
});

describe('createChatDmActions', () => {
  beforeEach(() => {
    useChatStore.setState({
      agents: [{ id: 'agent-1', name: 'Assistant', type: 'assistant', expertise: [], status: 'active', model: '', is_paused: false }],
      channels: [],
    });
  });

  it('handleCreateDM opens existing DM by membership without calling openDM', async () => {
    const existing: Channel = {
      name: 'dm-existing',
      description: 'DM',
      type: 'dm',
      agents: [{ id: 'agent-1', name: 'Assistant', type: 'assistant', expertise: [], status: 'active', model: '', is_paused: false }],
    };
    useChatStore.setState({ channels: [existing] });

    const handleSwitchChannel = vi.fn().mockResolvedValue(undefined);
    const openDM = vi.fn();
    const dmCreateInFlightRef = { current: new Map<string, Promise<void>>() };
    const dmOpenChainRef = { current: Promise.resolve() };

    const { handleCreateDM } = createChatDmActions({
      api: { openDM } as unknown as ChatAPI,
      username: 'camron',
      loadChannels: vi.fn(),
      loadAgents: vi.fn(),
      handleSwitchChannel,
      updateSettings: vi.fn(),
      addToast: vi.fn(),
      dmCreateInFlightRef,
      dmOpenChainRef,
    });

    await handleCreateDM('agent-1');

    expect(openDM).not.toHaveBeenCalled();
    expect(handleSwitchChannel).toHaveBeenCalledWith('dm-existing');
  });

  it('handleCreateDM calls openDM and switches when no existing channel', async () => {
    const created: Channel = {
      name: 'dm-new',
      description: 'New DM',
      type: 'dm',
    };
    const openDM = vi.fn().mockResolvedValue(created);
    const handleSwitchChannel = vi.fn().mockResolvedValue(undefined);
    const dmCreateInFlightRef = { current: new Map<string, Promise<void>>() };
    const dmOpenChainRef = { current: Promise.resolve() };

    const { handleCreateDM } = createChatDmActions({
      api: { openDM } as unknown as ChatAPI,
      username: 'camron',
      loadChannels: vi.fn(),
      loadAgents: vi.fn(),
      handleSwitchChannel,
      updateSettings: vi.fn(),
      addToast: vi.fn(),
      dmCreateInFlightRef,
      dmOpenChainRef,
    });

    await handleCreateDM('agent-1');

    expect(openDM).toHaveBeenCalledWith('agent-1', 'camron');
    expect(handleSwitchChannel).toHaveBeenCalledWith('dm-new');
    expect(useChatStore.getState().channels.some((c) => c.name === 'dm-new')).toBe(true);
  });
});
