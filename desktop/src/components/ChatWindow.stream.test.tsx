import React from 'react';
import { cleanup, render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ChatWindow } from './ChatWindow';
import { useChatStore } from '../stores/chatStore';
import type { Message } from '../types/protocol';

const { apiHarness, wsHarness } = vi.hoisted(() => {
  const apiHarness = {
    fetchMessages: vi.fn().mockResolvedValue([]),
    fetchCollaborations: vi.fn().mockResolvedValue([]),
    sendMessage: vi.fn().mockResolvedValue({}),
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
    createSession: vi.fn().mockResolvedValue({ token: "t", username: "Tester" }),
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
  return { apiHarness, wsHarness };
});

vi.mock('../api/chatAPI', () => ({
  ChatAPI: class {
    fetchMessages = apiHarness.fetchMessages;
    fetchCollaborations = apiHarness.fetchCollaborations;
    sendMessage = apiHarness.sendMessage;
    fetchCommands = apiHarness.fetchCommands;
    fetchAgents = apiHarness.fetchAgents;
    fetchChannels = apiHarness.fetchChannels;
    fetchMyAgents = apiHarness.fetchMyAgents;
    fetchRemovedAgents = apiHarness.fetchRemovedAgents;
    getWebSocketURL = apiHarness.getWebSocketURL;
    fetchAssistantState = apiHarness.fetchAssistantState;
    createChannel = apiHarness.createChannel;
    deleteChannel = apiHarness.deleteChannel;
    createSession = apiHarness.createSession;
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
  loadWorkspaceContextMode: () => 'auto' as const,
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
  };
  const useSettingsStore = Object.assign(() => state, { getState: () => state });
  return { useSettingsStore };
});

vi.mock('../stores/toastStore', () => ({
  useToastStore: (sel: (s: { addToast: (...a: unknown[]) => void }) => unknown) =>
    sel({ addToast: vi.fn() }),
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

vi.mock('../hooks/useSidebarAutoUnhide', () => ({ useSidebarAutoUnhide: vi.fn() }));
vi.mock('./MessageList', () => ({ MessageList: () => <div data-testid="message-list" /> }));
vi.mock('./ThreadPanel', () => ({ ThreadPanel: () => null }));
vi.mock('./MyAgentsPanel', () => ({ MyAgentsPanel: () => null }));
vi.mock('./TerminalPanel', () => ({ TerminalPanel: () => null }));
vi.mock('./lazyPanels', () => ({
  LazyPanelShell: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
  LazyFileExplorerPanel: () => null,
  LazyCodeEditorPanel: () => null,
  LazyCollaborationPanel: () => null,
  LazyRunbookBuilderPanel: () => null,
  LazyRunbookLibraryModal: () => null,
  LazyTaskManagementPanel: () => null,
  LazySecondaryAnalysisPanel: () => null,
  LazyDomainPacksModal: () => null,
  LazyModelArenaModal: () => null,
  LazyModelLibraryModal: () => null,
  LazyPhoenixBrowserModal: () => null,
  LazyRoomChatModal: () => null,
  LazyAIInterviewPrepModal: () => null,
}));

describe('ChatWindow stream handling', () => {
  beforeEach(() => {
    useChatStore.setState({
      channel: 'general',
      messages: [],
      streamingMessages: {},
      username: 'Test',
      serverAddr: 'http://127.0.0.1:9',
    });
    wsHarness.lastOpts = null;
  });

  afterEach(() => {
    cleanup();
  });

  it('accumulates stream_delta on the active channel timeline', async () => {
    render(<ChatWindow onLogout={() => {}} />);
    await waitFor(() => expect(wsHarness.lastOpts?.onMessage).toBeTruthy());

    const streamId = 'stream-1';
    await wsHarness.lastOpts!.onMessage({
      id: streamId,
      type: 'stream_delta',
      channel: 'general',
      content: 'Hello',
      timestamp: new Date().toISOString(),
      from: { id: 'a1', name: 'Agent', type: 'assistant' },
    });

    await waitFor(() => {
      expect(useChatStore.getState().streamingMessages[streamId]?.content).toContain('Hello');
    });

    await wsHarness.lastOpts!.onMessage({
      id: streamId,
      type: 'stream_end',
      channel: 'general',
      content: '',
      timestamp: new Date().toISOString(),
      from: { id: 'a1', name: 'Agent', type: 'assistant' },
    });

    await waitFor(() => {
      expect(useChatStore.getState().messages.some((m) => m.id === streamId)).toBe(true);
    });
  });
});
