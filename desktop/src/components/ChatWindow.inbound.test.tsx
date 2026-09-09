import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createChatInboundSurfaces } from '../hooks/createChatInboundSurfaces';
import { useChatStore } from '../stores/chatStore';
import type { ChatAPI } from '../api/chatAPI';
import type { Message } from '../types/protocol';

vi.mock('../stores/editorStore', () => {
  const state = {
    refreshTabFromDisk: vi.fn(),
  };
  return {
    useEditorStore: Object.assign(() => state, { getState: () => state }),
  };
});

vi.mock('../utils/refreshFileExplorer', () => ({
  fileChangeProposalPaths: () => [],
  refreshFileExplorerForPaths: vi.fn(),
}));

vi.mock('../components/MessageList', () => ({
  chatScrollerElRef: { current: null },
}));

vi.mock('../stores/approvalStore', () => {
  const state = {
    upsertPendingTool: vi.fn(),
    removePendingTool: vi.fn(),
  };
  return {
    useApprovalStore: Object.assign(() => state, { getState: () => state }),
  };
});

function baseDeps(overrides: Partial<Parameters<typeof createChatInboundSurfaces>[0]> = {}) {
  return {
    api: {} as ChatAPI,
    username: 'camron',
    handleSwitchChannel: vi.fn().mockResolvedValue(undefined),
    addToast: vi.fn(),
    loadAgents: vi.fn(),
    loadCounts: vi.fn(),
    activeWorkspaceId: null,
    explorerWorkspaces: [],
    fetchPendingChanges: vi.fn().mockResolvedValue(undefined),
    fetchPendingGitChanges: vi.fn().mockResolvedValue(undefined),
    pendingChangeCount: 0,
    pendingChanges: [],
    pendingGitChanges: [],
    agentRefreshTimeoutRef: { current: null as number | null },
    explorerRefreshTimeoutRef: { current: null as number | null },
    handledChangeProposalNoticesRef: { current: new Set<string>() },
    ...overrides,
  };
}

function makeMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: 'msg-1',
    type: 'chat',
    content: 'hello',
    from: { id: 'agent-1', name: 'Assistant', type: 'agent' },
    timestamp: '2026-01-01T00:00:00Z',
    channel: 'general',
    ...overrides,
  } as Message;
}

describe('createChatInboundSurfaces', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useChatStore.setState({
      channel: 'general',
      messages: [],
      pendingScrollToMessageId: null,
      highlightMessageId: null,
    });
  });

  it('navigateToMessage sets pending scroll then switches channel', async () => {
    const handleSwitchChannel = vi.fn().mockResolvedValue(undefined);
    const { navigateToMessage } = createChatInboundSurfaces(
      baseDeps({ handleSwitchChannel })
    );

    await navigateToMessage('reviews', 'msg-42');

    expect(useChatStore.getState().pendingScrollToMessageId).toBe('msg-42');
    expect(handleSwitchChannel).toHaveBeenCalledWith('reviews');
  });

  it('surfaceChangeProposal toasts once for pending proposal outside active channel', () => {
    const addToast = vi.fn();
    const fetchPendingChanges = vi.fn().mockResolvedValue(undefined);
    const handledChangeProposalNoticesRef = { current: new Set<string>() };
    const { surfaceChangeProposal } = createChatInboundSurfaces(
      baseDeps({ addToast, fetchPendingChanges, handledChangeProposalNoticesRef })
    );

    const message = makeMessage({
      id: 'proposal-msg',
      channel: 'reviews',
      metadata: {
        change_proposal: {
          version: 1,
          kind: 'file_change',
          id: 'change-1',
          status: 'pending',
          operation: 'edit',
          file_path: 'src/a.ts',
        },
      },
    });

    surfaceChangeProposal(message, false);
    surfaceChangeProposal(message, false);

    expect(fetchPendingChanges).toHaveBeenCalledWith('camron');
    expect(addToast).toHaveBeenCalledTimes(1);
    expect(addToast).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'info',
        title: 'Change needs review',
        message: expect.stringContaining('file change'),
      })
    );
    expect(handledChangeProposalNoticesRef.current.has('proposal-msg')).toBe(true);
  });

  it('surfaceChangeProposal skips toast when already on the channel', () => {
    const addToast = vi.fn();
    const { surfaceChangeProposal } = createChatInboundSurfaces(baseDeps({ addToast }));

    surfaceChangeProposal(
      makeMessage({
        metadata: {
          change_proposal: {
            version: 1,
            kind: 'git_change',
            id: 'git-1',
            status: 'pending',
            operation: 'commit',
          },
        },
      }),
      true
    );

    expect(addToast).not.toHaveBeenCalled();
  });
});
