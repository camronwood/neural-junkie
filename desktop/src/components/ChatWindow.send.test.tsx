import React from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ChatWindow } from './ChatWindow';
import { useChatStore } from '../stores/chatStore';
import type { Message } from '../types/protocol';

const { apiHarness, wsHarness, addToastMock } = vi.hoisted(() => {
  const apiHarness = {
    fetchMessages: vi.fn().mockResolvedValue([]),
    fetchCollaborations: vi.fn().mockResolvedValue([]),
    sendMessage: vi.fn().mockResolvedValue({}),
    answerUserQuestion: vi.fn().mockResolvedValue({}),
    channelInterject: vi.fn().mockResolvedValue({ channel: 'general', held: true }),
    fetchCommands: vi.fn().mockResolvedValue([]),
    fetchAgents: vi.fn().mockResolvedValue([]),
    fetchChannels: vi.fn().mockResolvedValue([
      {
        id: 'c-gen',
        name: 'general',
        description: '',
        type: 'public' as const,
        created: new Date().toISOString(),
        agents: [],
      },
    ]),
    fetchMyAgents: vi.fn().mockResolvedValue([]),
    fetchRemovedAgents: vi.fn().mockResolvedValue([]),
    getWebSocketURL: vi.fn(() => 'ws://127.0.0.1:9/ws'),
    fetchAssistantState: vi.fn().mockResolvedValue({ tasks: [], reminders: [] }),
    createChannel: vi.fn(),
    deleteChannel: vi.fn(),
    markAssistantTaskDone: vi.fn(),
    dismissAssistantReminder: vi.fn(),
    fetchWorkspaces: vi.fn().mockResolvedValue([]),
    fetchFileContent: vi.fn().mockResolvedValue(''),
    fetchFiles: vi.fn().mockResolvedValue([]),
    listPendingFileChanges: vi.fn().mockResolvedValue([]),
    fetchGitChanges: vi.fn().mockResolvedValue([]),
  };
  const wsHarness = {
    lastOpts: null as null | { onMessage: (m: Message) => void | Promise<void>; onConnect?: () => void },
  };
  const addToastMock = vi.fn();
  return { apiHarness, wsHarness, addToastMock };
});

vi.mock('../api/chatAPI', () => ({
  ChatAPI: class {
    fetchMessages = apiHarness.fetchMessages;
    fetchCollaborations = apiHarness.fetchCollaborations;
    sendMessage = apiHarness.sendMessage;
    answerUserQuestion = apiHarness.answerUserQuestion;
    channelInterject = apiHarness.channelInterject;
    fetchCommands = apiHarness.fetchCommands;
    fetchAgents = apiHarness.fetchAgents;
    fetchChannels = apiHarness.fetchChannels;
    fetchMyAgents = apiHarness.fetchMyAgents;
    fetchRemovedAgents = apiHarness.fetchRemovedAgents;
    getWebSocketURL = apiHarness.getWebSocketURL;
    fetchAssistantState = apiHarness.fetchAssistantState;
    createChannel = apiHarness.createChannel;
    deleteChannel = apiHarness.deleteChannel;
    markAssistantTaskDone = apiHarness.markAssistantTaskDone;
    dismissAssistantReminder = apiHarness.dismissAssistantReminder;
    fetchWorkspaces = apiHarness.fetchWorkspaces;
    fetchFileContent = apiHarness.fetchFileContent;
    fetchFiles = apiHarness.fetchFiles;
    listPendingFileChanges = apiHarness.listPendingFileChanges;
    fetchGitChanges = apiHarness.fetchGitChanges;
  },
}));

vi.mock('../hooks/useWebSocket', () => ({
  useWebSocket: (opts: {
    onMessage: (m: Message) => void | Promise<void>;
    onConnect?: () => void;
  }) => {
    wsHarness.lastOpts = opts;
    React.useEffect(() => {
      void opts.onConnect?.();
    }, []);
    return { status: 'connected' as const };
  },
}));

vi.mock('../utils/outboundChatMetadata', () => ({
  buildHumanOutboundMetadata: () => ({}),
  loadWorkspaceContextMode: () => 'off' as const,
  cycleWorkspaceContextMode: (m: string) => m,
  workspaceContextModeLabel: (m: string) => m,
  WORKSPACE_CONTEXT_MODE_KEY: 'workspace-context-mode',
}));

vi.mock('../stores/settingsStore', () => {
  const state = {
    layoutSettings: {
      layoutPreset: 'team',
      ideChatDock: 'right',
      filesPanelVisible: false,
      editorPanelVisible: false,
      chatPanelVisible: true,
      terminalPanelVisible: false,
      myAgentsPanelVisible: false,
      editorAgentMode: 'agent',
      sidebarAgentsVisible: true,
      toolbarChipsPlacement: 'top',
    },
    updateLayoutSettings: vi.fn(),
    loadLayoutSettings: vi.fn(),
    settings: {},
    isLoaded: true,
    saveSettings: vi.fn(),
    updateSettings: vi.fn(),
  };
  const useSettingsStore = Object.assign(() => state, { getState: () => state });
  return { useSettingsStore };
});

vi.mock('../stores/toastStore', () => ({
  useToastStore: (sel: (s: { addToast: (...a: unknown[]) => void }) => unknown) =>
    sel({ addToast: addToastMock }),
}));

vi.mock('../stores/packsStore', () => {
  const packState = {
    packs: [],
    fetchPacks: vi.fn().mockResolvedValue(undefined),
    softwareDevelopmentEnabled: () => false,
    softwareDevelopmentPackActive: () => false,
    ideEnabled: () => false,
    idePackActive: () => false,
    lifeSciencesEnabled: () => false,
    hasCapability: () => false,
    layoutProfile: 'team' as const,
    layoutOwner: '',
    capabilities: [],
    catalog: [],
    applyPacksResponse: vi.fn(),
    fetchPackCatalog: vi.fn(),
    installPack: vi.fn(),
    uninstallPack: vi.fn(),
    getToolbarActions: () => [],
  };
  return {
    usePacksStore: Object.assign(
      (sel: (s: typeof packState) => unknown) => sel(packState),
      { getState: () => packState }
    ),
  };
});

const editorMock = {
  tabs: [] as unknown[],
  activeTabId: null as string | null,
  activeSelection: null,
  openFile: vi.fn(),
  revealLine: vi.fn(),
};
vi.mock('../stores/editorStore', () => ({
  useEditorStore: (sel: (s: typeof editorMock) => unknown) => sel(editorMock),
}));

vi.mock('../utils/prepareOutboundPayload', () => ({
  prepareOutboundPayload: vi.fn(async ({ content, composerMetadata }: { content: string; composerMetadata?: Record<string, unknown> }) => ({
    content,
    metadata: composerMetadata ?? {},
  })),
}));

vi.mock('./chat/ChatInputArea', () => ({
  ChatInputArea: ({
    onSend,
    showAgentStop,
    onChannelInterject,
    channelHeld,
    hasPendingUserQuestion,
  }: {
    onSend: (c: string) => void | Promise<unknown>;
    showAgentStop?: boolean;
    onChannelInterject?: () => void;
    channelHeld?: boolean;
    hasPendingUserQuestion?: boolean;
  }) => (
    <>
      {showAgentStop ? (
        <button type="button" aria-label="Stop agents" onClick={() => onChannelInterject?.()}>
          Stop agents
        </button>
      ) : null}
      {channelHeld && !hasPendingUserQuestion ? (
        <div>Agents paused — send a message to continue.</div>
      ) : null}
      <button type="button" onClick={() => void onSend('hello world')}>
        send-test
      </button>
    </>
  ),
}));

vi.mock('./GitPanel', () => ({ GitModal: () => null, GitPanel: () => null }));
vi.mock('./QuickOpenModal', () => ({ QuickOpenModal: () => null }));
vi.mock('./SymbolModal', () => ({ SymbolModal: () => null }));
vi.mock('./ProblemsPanel', () => ({ ProblemsPanel: () => null }));
vi.mock('./FastEditModal', () => ({ FastEditModal: () => null }));

vi.mock('../hooks/useSidebarAutoUnhide', () => ({ useSidebarAutoUnhide: vi.fn() }));

vi.mock('./MessageList', () => ({ MessageList: () => <div data-testid="message-list" /> }));
vi.mock('./ThreadPanel', () => ({ ThreadPanel: () => null }));
vi.mock('./MyAgentsPanel', () => ({ MyAgentsPanel: () => null }));
vi.mock('./TerminalPanel', () => ({ TerminalPanel: () => null }));
vi.mock('./FileExplorerPanel', () => ({ FileExplorerPanel: () => null }));
vi.mock('./CodeEditorPanel', () => ({ CodeEditorPanel: () => null }));
vi.mock('./Toast', () => ({ ToastContainer: () => null }));
vi.mock('./CommandPalette', () => ({ CommandPalette: () => null }));
vi.mock('./ChannelSidebar', () => ({ ChannelSidebar: () => null }));
vi.mock('./CreateChannelModal', () => ({ CreateChannelModal: () => null }));
vi.mock('./ChannelInfoModal', () => ({ ChannelInfoModal: () => null }));
vi.mock('./CreateNewDMModal', () => ({ CreateNewDMModal: () => null }));
vi.mock('./CollaborationPanel', () => ({ CollaborationPanel: () => null }));
vi.mock('./RunbookBuilderPanel', () => ({ RunbookBuilderPanel: () => null }));
vi.mock('./TaskManagementPanel', () => ({ TaskManagementPanel: () => null }));
vi.mock('./CollaborationWorkspaceGate', () => ({ CollaborationWorkspaceGate: () => null }));
vi.mock('./HubDataAccessModal', () => ({ HubDataAccessModal: () => null }));
vi.mock('./PhoenixBrowserModal', () => ({ PhoenixBrowserModal: () => null }));
vi.mock('./LearningProposalModal', () => ({ LearningProposalModal: () => null }));
vi.mock('./ModelLibraryModal', () => ({ ModelLibraryModal: () => null }));
vi.mock('./DomainPacksModal', () => ({ DomainPacksModal: () => null }));
vi.mock('./SecondaryAnalysisPanel', () => ({ SecondaryAnalysisPanel: () => null }));
vi.mock('./PendingApprovalsBar', () => ({ PendingApprovalsBar: () => null }));
vi.mock('../stores/approvalStore', () => {
  const state = {
    pendingTools: [],
    upsertPendingTool: vi.fn(),
    removePendingTool: vi.fn(),
    syncPendingFromHub: vi.fn().mockResolvedValue(undefined),
  };
  return {
    useApprovalStore: Object.assign(
      (sel: (s: typeof state) => unknown) => sel(state),
      { getState: () => state }
    ),
  };
});
vi.mock('./RichTextInput', () => ({
  RichTextInput: () => null,
}));

function seedStore() {
  useChatStore.getState().reset();
  useChatStore.setState({
    serverAddr: 'http://127.0.0.1:9',
    channel: 'general',
    username: 'Tester',
    connectionStatus: 'connected',
    agents: [],
    messages: [],
  });
}

afterEach(() => cleanup());

describe('ChatWindow send', () => {
  beforeEach(() => {
    seedStore();
    apiHarness.fetchMessages.mockReset().mockResolvedValue([]);
    apiHarness.channelInterject.mockClear().mockResolvedValue({ channel: 'general', held: true });
    apiHarness.sendMessage.mockClear().mockResolvedValue({});
    apiHarness.answerUserQuestion.mockClear();
    apiHarness.fetchCommands.mockClear();
    addToastMock.mockClear();
  });

  it('shows error toast when sendMessage rejects', async () => {
    apiHarness.sendMessage.mockImplementation(() => Promise.reject(new Error('network down')));
    render(<ChatWindow />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'send-test' })).toBeTruthy();
    });

    fireEvent.click(screen.getByRole('button', { name: 'send-test' }));

    await waitFor(() => {
      expect(apiHarness.sendMessage).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(addToastMock).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'error',
          title: 'Message not sent',
          message: 'network down',
        })
      );
    });
  });
});

describe('ChatWindow channel interject (send hook)', () => {
  beforeEach(() => {
    seedStore();
    apiHarness.fetchMessages.mockReset().mockResolvedValue([]);
    apiHarness.channelInterject.mockClear().mockResolvedValue({ channel: 'general', held: true });
    apiHarness.sendMessage.mockClear();
    addToastMock.mockClear();
  });

  it('shows Stop and calls channelInterject when an agent is thinking', async () => {
    render(<ChatWindow />);
    useChatStore.getState().addThinkingAgent('general', 'cli-1', 'Cursor', 'cli');

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Stop agents' })).toBeTruthy();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Stop agents' }));

    await waitFor(() => {
      expect(apiHarness.channelInterject).toHaveBeenCalledWith('general', 'Tester');
    });
    expect(useChatStore.getState().isChannelHeld('general')).toBe(true);
    expect(screen.getByText(/Agents paused/)).toBeTruthy();
  });

  it('shows error toast when channelInterject rejects', async () => {
    apiHarness.channelInterject.mockRejectedValueOnce(new Error('interject failed'));
    render(<ChatWindow />);
    useChatStore.getState().addThinkingAgent('general', 'cli-1', 'Cursor', 'cli');

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Stop agents' })).toBeTruthy();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Stop agents' }));

    await waitFor(() => {
      expect(addToastMock).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'error',
          title: 'Stop failed',
          message: 'interject failed',
        })
      );
    });
  });
});
