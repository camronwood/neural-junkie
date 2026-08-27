import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createChatChannelActions } from '../hooks/createChatChannelActions';
import { useChatStore } from '../stores/chatStore';
import type { Collaboration, Message } from '../types/protocol';
import type { ChatAPI } from '../api/chatAPI';
import type { ChatChannelActionsDeps } from '../hooks/createChatChannelActions';

vi.mock('../utils/collaborationExecutionWorkspace', () => ({
  ensureCollaborationExecutionWorkspace: vi.fn().mockResolvedValue(undefined),
}));

vi.mock('../utils/collabThinking', () => ({
  syncCollabTurnThinking: vi.fn(),
}));

vi.mock('../utils/terminalCwd', () => ({
  resolveTerminalCwd: () => '',
}));

vi.mock('../utils/sidebarVisibility', () => ({
  patchRevealForChannel: () => null,
}));

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

vi.mock('../stores/terminalStore', () => {
  const state = {
    alignActiveTabCwd: vi.fn(),
  };
  return {
    useTerminalStore: Object.assign(() => state, { getState: () => state }),
  };
});

vi.mock('../stores/activityLogStore', () => ({
  logActivity: vi.fn(),
}));

function sampleMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: 'm1',
    type: 'question',
    channel: 'ops',
    from: {
      id: 'u1',
      name: 'Tester',
      type: 'human',
      expertise: [],
      status: 'active',
      model: '',
      is_paused: false,
    },
    content: 'hello from ops',
    timestamp: new Date().toISOString(),
    ...overrides,
  };
}

function sampleCollab(overrides: Partial<Collaboration> = {}): Collaboration {
  return {
    id: 'collab-1',
    title: 'Gate test',
    description: '',
    phase: 'executing',
    agents: [],
    channel: 'collab-gate',
    created_by: 'Tester',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    execution_mode: 'sandbox',
    working_directory: '/tmp/collab-gate',
    workspace_acknowledged: false,
    ...overrides,
  };
}

function buildDeps(overrides: Partial<ChatChannelActionsDeps> = {}): ChatChannelActionsDeps {
  const api = {
    fetchMessages: vi.fn().mockResolvedValue([]),
    acknowledgeCollaborationWorkspace: vi.fn().mockResolvedValue({}),
    createChannel: vi.fn(),
    deleteChannel: vi.fn(),
    createRunbook: vi.fn(),
  } as unknown as ChatAPI;

  return {
    api,
    channel: 'general',
    username: 'Tester',
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

describe('createChatChannelActions', () => {
  beforeEach(() => {
    useChatStore.getState().reset();
    useChatStore.setState({
      channel: 'general',
      username: 'Tester',
      messages: [],
      channels: [
        {
          id: 'c-gen',
          name: 'general',
          description: '',
          type: 'public',
          created: new Date().toISOString(),
          agents: [],
        },
        {
          id: 'c-ops',
          name: 'ops',
          description: '',
          type: 'custom',
          created: new Date().toISOString(),
          agents: [],
        },
      ],
    });
  });

  it('handleSwitchChannel fetches messages and setMessages', async () => {
    const msgs = [sampleMessage()];
    const deps = buildDeps();
    (deps.api.fetchMessages as ReturnType<typeof vi.fn>).mockResolvedValue(msgs);

    await createChatChannelActions(deps).handleSwitchChannel('ops');

    expect(deps.api.fetchMessages).toHaveBeenCalledWith('ops', 50);
    expect(useChatStore.getState().channel).toBe('ops');
    expect(useChatStore.getState().messages).toEqual(msgs);
    expect(deps.loadCollaborations).toHaveBeenCalledWith('ops');
    expect(deps.setActiveCollab).toHaveBeenCalledWith(null);
  });

  it('handleWorkspaceGateContinue acknowledges collaboration workspace when gate collab is set', async () => {
    const collab = sampleCollab();
    useChatStore.setState({ channel: collab.channel });
    const deps = buildDeps({
      workspaceGateCollab: collab,
      channel: collab.channel,
    });

    await createChatChannelActions(deps).handleWorkspaceGateContinue();

    expect(deps.api.acknowledgeCollaborationWorkspace).toHaveBeenCalledWith(collab.id, undefined);
    expect(deps.setWorkspaceGateCollab).toHaveBeenCalledWith(null);
    expect(deps.setWorkspaceGateBusy).toHaveBeenCalledWith(true);
    expect(deps.setWorkspaceGateBusy).toHaveBeenCalledWith(false);
    expect(deps.loadCollaborations).toHaveBeenCalledWith(collab.channel);
    expect(deps.setWorkspaceContextMode).toHaveBeenCalledWith('always');
  });
});
