import React from 'react';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ChatWindow } from './ChatWindow';
import { useChatStore } from '../stores/chatStore';
import type { AgentInfo, Collaboration, Message } from '../types/protocol';

const { apiHarness, wsHarness, confirmStartMock, confirmReplaceMock, addToastMock } = vi.hoisted(() => {
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
      {
        id: 'c-alpha',
        name: 'alpha',
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
  };
  const wsHarness = {
    lastOpts: null as null | {
      onMessage: (m: Message) => void | Promise<void>;
      onConnect?: () => void;
    },
  };
  const confirmStartMock = vi.fn((_e: unknown) => true);
  const confirmReplaceMock = vi.fn((_a: unknown, _b: unknown) => true);
  const addToastMock = vi.fn();
  return { apiHarness, wsHarness, confirmStartMock, confirmReplaceMock, addToastMock };
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
    markAssistantTaskDone = apiHarness.markAssistantTaskDone;
    dismissAssistantReminder = apiHarness.dismissAssistantReminder;
  },
}));

vi.mock('../hooks/useWebSocket', () => ({
  useWebSocket: (opts: {
    onMessage: (m: Message) => void | Promise<void>;
    onConnect?: () => void;
  }) => {
    wsHarness.lastOpts = opts;
    // ChatWindow passes a new opts object each render; depend only on mount so onConnect runs once.
    React.useEffect(() => {
      void opts.onConnect?.();
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);
    return { status: 'connected' as const };
  },
}));

vi.mock('../utils/outboundChatMetadata', () => ({
  buildHumanOutboundMetadata: () => ({}),
  loadWorkspaceContextMode: () => 'auto' as const,
  cycleWorkspaceContextMode: (m: string) => (m === 'auto' ? 'always' : m === 'always' ? 'off' : 'auto'),
  workspaceContextModeLabel: (m: string) => m,
  WORKSPACE_CONTEXT_MODE_KEY: 'workspace-context-mode',
}));

vi.mock('../stores/settingsStore', () => ({
  useSettingsStore: () => ({
    layoutSettings: null,
    loadLayoutSettings: vi.fn(),
  }),
}));

vi.mock('../stores/toastStore', () => ({
  useToastStore: (sel: (s: { addToast: (...a: unknown[]) => void }) => unknown) =>
    sel({ addToast: addToastMock }),
}));

vi.mock('../utils/collaborationConfirm', () => ({
  confirmStartCollaborationWhileExecuting: (executing: unknown) => confirmStartMock(executing) as boolean,
  confirmReplaceCollaborationExecution: (executing: unknown, incoming: unknown) =>
    confirmReplaceMock(executing, incoming) as boolean,
}));

vi.mock('./MessageList', () => ({
  MessageList: () => <div data-testid="message-list" />,
}));
vi.mock('./TypingIndicator', () => ({
  TypingIndicator: () => null,
}));
vi.mock('./ThreadPanel', () => ({
  ThreadPanel: () => null,
}));
vi.mock('./MyAgentsPanel', () => ({
  MyAgentsPanel: () => null,
}));
vi.mock('./PendingChangesPanel', () => ({
  PendingChangesPanel: () => null,
}));
vi.mock('./TerminalPanel', () => ({
  TerminalPanel: () => null,
}));
vi.mock('./FileExplorerPanel', () => ({
  FileExplorerPanel: () => null,
}));
vi.mock('./CodeEditorPanel', () => ({
  CodeEditorPanel: () => null,
}));
vi.mock('./Toast', () => ({
  ToastContainer: () => null,
}));
vi.mock('./CommandPalette', () => ({
  CommandPalette: () => null,
}));
vi.mock('./CreateChannelModal', () => ({
  CreateChannelModal: () => null,
}));
vi.mock('./ChannelInfoModal', () => ({
  ChannelInfoModal: () => null,
}));
vi.mock('./CreateNewDMModal', () => ({
  CreateNewDMModal: () => null,
}));

vi.mock('./ChannelSidebar', () => ({
  ChannelSidebar: ({ onSwitchChannel }: { onSwitchChannel: (name: string) => void }) => (
    <button type="button" data-testid="switch-to-alpha" onClick={() => void onSwitchChannel('alpha')}>
      switch-to-alpha
    </button>
  ),
}));

vi.mock('./RichTextInput', () => ({
  RichTextInput: React.forwardRef(function MockRichTextInput(
    props: { onSend: (c: string, meta?: Record<string, unknown>) => void },
    _ref: unknown
  ) {
    return (
      <div>
        <button
          type="button"
          data-testid="send-collaborate"
          onClick={() => void props.onSend('/collaborate @A @B do the thing')}
        >
          send-collaborate
        </button>
      </div>
    );
  }),
}));

function makeCollaboration(overrides: Partial<Collaboration> = {}): Collaboration {
  const id = overrides.id ?? '11111111-2222-3333-4444-555555555555';
  return {
    id,
    title: 'Wire-test collaboration',
    description: 'From ChatWindow integration harness',
    phase: 'reviewing',
    agents: [],
    tasks: [],
    channel: 'general',
    created_by: 'tester',
    created_at: '2026-01-01T00:00:00.000Z',
    updated_at: '2026-01-02T00:00:00.000Z',
    ...overrides,
  };
}

function minimalAgent(overrides: Partial<AgentInfo> = {}): AgentInfo {
  return {
    id: 'agent-1',
    name: 'AgentOne',
    type: 'rust',
    expertise: [],
    status: 'active',
    model: 'mock',
    is_paused: false,
    ...overrides,
  };
}

async function flushWsConnect() {
  await waitFor(() => expect(apiHarness.fetchMessages).toHaveBeenCalled());
}

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.setItem('channel-sidebar-open', 'true');
  localStorage.setItem('last-channel', 'general');
  useChatStore.getState().reset();
  useChatStore.setState({
    channel: 'general',
    username: 'Tester',
    serverAddr: 'http://127.0.0.1:9',
    connectionStatus: 'connected',
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
        id: 'c-alpha',
        name: 'alpha',
        description: '',
        type: 'public',
        created: new Date().toISOString(),
        agents: [],
      },
    ],
  });
  apiHarness.fetchMessages.mockResolvedValue([]);
  apiHarness.fetchCollaborations.mockResolvedValue([]);
  apiHarness.sendMessage.mockResolvedValue({});
  confirmStartMock.mockReturnValue(true);
  confirmReplaceMock.mockReturnValue(true);
});

describe('ChatWindow collaboration wiring', () => {
  it('opens CollaborationPanel from TaskManagement and closes the task drawer', async () => {
    const collab = makeCollaboration();
    apiHarness.fetchCollaborations.mockResolvedValue([collab]);

    render(<ChatWindow />);
    await flushWsConnect();

    fireEvent.click(screen.getByRole('button', { name: 'Open task management' }));

    await waitFor(() => {
      expect(screen.getByText('Wire-test collaboration')).toBeInTheDocument();
    });

    fireEvent.click(screen.getAllByRole('button', { name: 'Open' })[0]);

    await waitFor(() => {
      expect(screen.getByText('Reviewing Plan')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: 'Open task management' })).toHaveAttribute('aria-pressed', 'false');
  });

  it('runs task-panel approve via ChatWindow (sends /resume-plan short id)', async () => {
    const collab = makeCollaboration();
    apiHarness.fetchCollaborations.mockResolvedValue([collab]);

    render(<ChatWindow />);
    await flushWsConnect();

    fireEvent.click(screen.getByRole('button', { name: 'Open task management' }));
    await waitFor(() => expect(screen.getByText('Wire-test collaboration')).toBeInTheDocument());

    const resumeButtons = screen.getAllByRole('button', { name: 'Resume plan' });
    fireEvent.click(resumeButtons[0]);

    await waitFor(() => {
      expect(apiHarness.sendMessage).toHaveBeenCalledWith(
        'general',
        '/resume-plan 11111111',
        { name: 'Tester', type: 'human' }
      );
    });
  });

  it('clears the collaboration side panel when switching channels', async () => {
    const collab = makeCollaboration();
    apiHarness.fetchCollaborations.mockResolvedValue([collab]);

    render(<ChatWindow />);
    await flushWsConnect();

    fireEvent.click(screen.getByRole('button', { name: 'Open task management' }));
    await waitFor(() => expect(screen.getByText('Wire-test collaboration')).toBeInTheDocument());
    fireEvent.click(screen.getAllByRole('button', { name: 'Open' })[0]);

    await waitFor(() => expect(screen.getByText('Reviewing Plan')).toBeInTheDocument());

    fireEvent.click(screen.getByTestId('switch-to-alpha'));

    await waitFor(() => {
      expect(useChatStore.getState().channel).toBe('alpha');
    });
    expect(screen.queryByText('Reviewing Plan')).not.toBeInTheDocument();
  });

  it('auto-opens the collaboration panel on first collaboration_discussion with metadata', async () => {
    apiHarness.fetchCollaborations.mockResolvedValue([]);

    render(<ChatWindow />);
    await flushWsConnect();

    const opts = wsHarness.lastOpts;
    expect(opts).toBeTruthy();
    const collab = makeCollaboration({ id: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee', phase: 'planning' });
    const msg: Message = {
      id: 'ws-1',
      type: 'collaboration_discussion',
      channel: 'general',
      from: minimalAgent(),
      content: 'Kickoff',
      timestamp: new Date().toISOString(),
      metadata: { collaboration_data: collab },
    };
    await opts!.onMessage(msg);

    await waitFor(() => {
      expect(screen.getByText('Planning')).toBeInTheDocument();
    });
    expect(screen.getByText('Wire-test collaboration')).toBeInTheDocument();
  });

  it('keeps the panel open read-only and toasts when collaboration completes over WS', async () => {
    apiHarness.fetchCollaborations.mockResolvedValue([]);
    addToastMock.mockClear();

    render(<ChatWindow />);
    await flushWsConnect();

    const opts = wsHarness.lastOpts!;
    const openCollab = makeCollaboration({
      id: 'bbbbbbbb-2222-3333-4444-555555555555',
      phase: 'planning',
      tasks: [
        {
          id: 't1',
          title: 'Ship UI',
          description: '',
          assigned_to: 'a1',
          assigned_name: 'Agent',
          status: 'completed',
          created_at: '2026-01-01T00:00:00.000Z',
          updated_at: '2026-01-01T00:00:00.000Z',
        },
      ],
    });
    await opts.onMessage({
      id: 'ws-open',
      type: 'collaboration_discussion',
      channel: 'general',
      from: minimalAgent(),
      content: 'Kickoff',
      timestamp: new Date().toISOString(),
      metadata: { collaboration_data: openCollab },
    });
    await waitFor(() => expect(screen.getByText('Planning')).toBeInTheDocument());

    const completed = {
      ...openCollab,
      phase: 'completed' as const,
      updated_at: '2026-01-05T00:00:00.000Z',
    };
    await opts.onMessage({
      id: 'ws-done',
      type: 'collaboration_status',
      channel: 'general',
      from: minimalAgent(),
      content: 'done',
      timestamp: new Date().toISOString(),
      metadata: { collaboration_data: completed },
    });

    await waitFor(() => {
      expect(screen.getByText('Completed')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Dismiss' })).toBeInTheDocument();
    });
    expect(screen.queryByRole('button', { name: 'Resume plan' })).not.toBeInTheDocument();
    expect(addToastMock).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'success',
        title: 'Collaboration completed',
      })
    );
  });

  it('shows a completion banner on the collab channel after the panel was open', async () => {
    const collabChannel = 'collab-bbbb2222';
    const base = makeCollaboration({
      id: 'bbbbbbbb-2222-3333-4444-555555555555',
      channel: collabChannel,
      phase: 'planning',
      tasks: [
        {
          id: 't1',
          title: 'Done task',
          description: '',
          assigned_to: 'a1',
          assigned_name: 'Agent',
          status: 'pending',
          created_at: '2026-01-01T00:00:00.000Z',
          updated_at: '2026-01-01T00:00:00.000Z',
        },
      ],
    });
    apiHarness.fetchCollaborations.mockResolvedValue([]);

    useChatStore.setState({ channel: collabChannel });
    render(<ChatWindow />);
    await flushWsConnect();

    const opts = wsHarness.lastOpts!;
    await opts.onMessage({
      id: 'ws-collab-open',
      type: 'collaboration_discussion',
      channel: collabChannel,
      from: minimalAgent(),
      content: 'Kickoff',
      timestamp: new Date().toISOString(),
      metadata: { collaboration_data: base },
    });
    await waitFor(() => expect(screen.getByText('Planning')).toBeInTheDocument());

    const completed = {
      ...base,
      phase: 'completed' as const,
      tasks: [{ ...base.tasks![0], status: 'completed' as const }],
    };
    await opts.onMessage({
      id: 'ws-collab-done',
      type: 'collaboration_status',
      channel: collabChannel,
      from: minimalAgent(),
      content: 'done',
      timestamp: new Date().toISOString(),
      metadata: { collaboration_data: completed },
    });

    await waitFor(() => {
      expect(screen.getByRole('status')).toHaveTextContent(/Collaboration complete/i);
      expect(screen.getByText(/1\/1 tasks done/)).toBeInTheDocument();
    });
  });

  it('blocks /collaborate when confirmStartCollaborationWhileExecuting returns false', async () => {
    confirmStartMock.mockReturnValue(false);
    apiHarness.fetchCollaborations.mockResolvedValue([
      makeCollaboration({ id: 'exec-1111-2222-3333-4444-555555555555', phase: 'executing' }),
    ]);

    render(<ChatWindow />);
    await flushWsConnect();

    fireEvent.click(screen.getByTestId('send-collaborate'));

    await waitFor(() => expect(confirmStartMock).toHaveBeenCalled());
    const collaborateSends = apiHarness.sendMessage.mock.calls.filter(
      (c) => typeof c[1] === 'string' && (c[1] as string).trimStart().startsWith('/collaborate')
    );
    expect(collaborateSends).toHaveLength(0);
  });

  it('follows collaboration_channel redirect after /collaborate', async () => {
    apiHarness.sendMessage.mockImplementation(async (_channel, content: string) => {
      if (content.trimStart().startsWith('/collaborate')) {
        return { collaboration_channel: 'collab-redirect' };
      }
      return {};
    });

    render(<ChatWindow />);
    await flushWsConnect();

    fireEvent.click(screen.getByTestId('send-collaborate'));

    await waitFor(() => {
      expect(useChatStore.getState().channel).toBe('collab-redirect');
    });
    expect(apiHarness.fetchChannels).toHaveBeenCalled();
  });
});
