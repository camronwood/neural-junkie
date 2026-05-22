import React from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ChatWindow } from './ChatWindow';
import { useChatStore } from '../stores/chatStore';
import { METADATA_CHANNEL_HOLD } from '../types/protocol';
import type { Message } from '../types/protocol';

const { apiHarness, wsHarness, addToastMock } = vi.hoisted(() => {
  const apiHarness = {
    fetchMessages: vi.fn().mockResolvedValue([]),
    fetchCollaborations: vi.fn().mockResolvedValue([]),
    sendMessage: vi.fn().mockResolvedValue({}),
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
    layoutSettings: null,
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
    sel({ addToast: addToastMock }),
}));

vi.mock('../stores/packsStore', () => ({
  usePacksStore: Object.assign(
    () => ({ packs: [], fetchPacks: vi.fn() }),
    { getState: () => ({ packs: [], fetchPacks: vi.fn() }) }
  ),
}));

vi.mock('../hooks/useSidebarAutoUnhide', () => ({ useSidebarAutoUnhide: vi.fn() }));

vi.mock('./MessageList', () => ({ MessageList: () => <div data-testid="message-list" /> }));
vi.mock('./ThreadPanel', () => ({ ThreadPanel: () => null }));
vi.mock('./MyAgentsPanel', () => ({ MyAgentsPanel: () => null }));
vi.mock('./PendingChangesPanel', () => ({ PendingChangesPanel: () => null }));
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
vi.mock('./RichTextInput', () => ({
  RichTextInput: ({ onSend }: { onSend: (c: string) => void }) => (
    <button type="button" onClick={() => onSend('user resume')}>
      send-test
    </button>
  ),
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

describe('ChatWindow channel interject', () => {
  beforeEach(() => {
    seedStore();
    apiHarness.channelInterject.mockClear();
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

  it('clears channel hold banner on WS channel_hold false', async () => {
    render(<ChatWindow />);
    useChatStore.getState().setChannelHold('general', true);

    await waitFor(() => expect(screen.getByText(/Agents paused/)).toBeTruthy());

    const statusMsg: Message = {
      id: 'status-1',
      type: 'agent_status',
      channel: 'general',
      from: { id: 'system', name: 'System', type: 'general', expertise: [], status: 'active', model: '', is_paused: false },
      content: '',
      timestamp: new Date().toISOString(),
      metadata: { [METADATA_CHANNEL_HOLD]: false },
    };
    await wsHarness.lastOpts?.onMessage(statusMsg);

    await waitFor(() => {
      expect(screen.queryByText(/Agents paused/)).toBeNull();
    });
    expect(useChatStore.getState().isChannelHeld('general')).toBe(false);
  });

  it('removes thinking indicator on aborted status', async () => {
    render(<ChatWindow />);
    useChatStore.getState().addThinkingAgent('general', 'a1', 'Agent', 'backend');

    const statusMsg: Message = {
      id: 'status-abort',
      type: 'agent_status',
      channel: 'general',
      from: { id: 'a1', name: 'Agent', type: 'backend', expertise: [], status: 'active', model: '', is_paused: false },
      content: '',
      timestamp: new Date().toISOString(),
      metadata: { thinking_status: 'aborted', question_id: 'q1' },
    };
    await wsHarness.lastOpts?.onMessage(statusMsg);

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Stop agents' })).toBeNull();
    });
  });
});
