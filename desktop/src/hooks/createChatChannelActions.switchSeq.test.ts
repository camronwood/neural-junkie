import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createChatChannelActions, type ChatChannelActionsDeps } from './createChatChannelActions';
import { useChatStore } from '../stores/chatStore';
import type { Message } from '../types/protocol';

function deps(overrides: Partial<ChatChannelActionsDeps> = {}): ChatChannelActionsDeps {
  return {
    api: {
      fetchMessages: vi.fn(),
    } as unknown as ChatChannelActionsDeps['api'],
    channel: 'general',
    username: 'cam',
    agents: [],
    channels: [],
    workspaceGateCollab: null,
    setWorkspaceGateBusy: vi.fn(),
    setWorkspaceGateCollab: vi.fn(),
    setWorkspaceContextMode: vi.fn(),
    dismissedWorkspaceGateIdRef: { current: null },
    workspaceGateToastIdRef: { current: null },
    collaborationsByIDRef: { current: {} },
    loadCollaborations: vi.fn().mockResolvedValue(undefined),
    loadChannels: vi.fn().mockResolvedValue(undefined),
    addToast: vi.fn(),
    setActiveCollab: vi.fn(),
    setRunbookLibraryOpen: vi.fn(),
    setChannelInfoModal: vi.fn(),
    updateSettings: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

function msg(channel: string, id: string, content: string): Message {
  return {
    id,
    type: 'chat',
    channel,
    from: { id: 'u', name: 'User', type: 'human' },
    content,
    timestamp: new Date().toISOString(),
  };
}

describe('createChatChannelActions switch seq', () => {
  beforeEach(() => {
    useChatStore.getState().reset();
    useChatStore.getState().setChannel('general');
  });

  it('ignores stale history when a newer switch completes first', async () => {
    let resolveA: (msgs: Message[]) => void = () => {};
    let resolveB: (msgs: Message[]) => void = () => {};
    const fetchMessages = vi.fn((channel: string) => {
      if (channel === 'alpha') {
        return new Promise<Message[]>((resolve) => {
          resolveA = resolve;
        });
      }
      return new Promise<Message[]>((resolve) => {
        resolveB = resolve;
      });
    });
    const actions = createChatChannelActions(
      deps({
        api: { fetchMessages } as unknown as ChatChannelActionsDeps['api'],
        loadCollaborations: vi.fn().mockResolvedValue(undefined),
      }),
    );

    const switchA = actions.handleSwitchChannel('alpha');
    const switchB = actions.handleSwitchChannel('beta');

    resolveB([msg('beta', 'b1', 'from-beta')]);
    await switchB;
    expect(useChatStore.getState().channel).toBe('beta');
    expect(useChatStore.getState().messages.map((m) => m.id)).toEqual(['b1']);

    resolveA([msg('alpha', 'a1', 'from-alpha-stale')]);
    await switchA;
    expect(useChatStore.getState().channel).toBe('beta');
    expect(useChatStore.getState().messages.map((m) => m.id)).toEqual(['b1']);
    expect(useChatStore.getState().messages.some((m) => m.id === 'a1')).toBe(false);
  });

  it('applies history for the latest switch (happy path)', async () => {
    const fetchMessages = vi.fn(async (channel: string) => [msg(channel, `${channel}-1`, channel)]);
    const actions = createChatChannelActions(
      deps({
        api: { fetchMessages } as unknown as ChatChannelActionsDeps['api'],
        loadCollaborations: vi.fn().mockResolvedValue(undefined),
      }),
    );

    await actions.handleSwitchChannel('reviews');
    expect(useChatStore.getState().channel).toBe('reviews');
    expect(useChatStore.getState().messages[0]?.content).toBe('reviews');
  });
});
