import { useState, useRef, useCallback, useEffect, useMemo, startTransition } from 'react';
import { shallow } from 'zustand/shallow';
import { useChatStore } from '../stores/chatStore';
import { useTerminalStore, createNewTab } from '../stores/terminalStore';
import { useSettingsStore } from '../stores/settingsStore';
import { useToastStore } from '../stores/toastStore';
import { ChatAPI } from '../api/chatAPI';
import { clearCredentials } from '../utils/secureStorage';
import {
  buildHumanOutboundMetadata,
  cycleWorkspaceContextMode,
  loadWorkspaceContextMode,
  workspaceContextModeLabel,
  WORKSPACE_CONTEXT_MODE_KEY,
} from '../utils/outboundChatMetadata';
import { channelNameToKind, resolveContextScope } from '../utils/inferContextScope';
import type { WorkspaceContextMode } from '../constants/promptMetadata';
import { GRANTED_HUB_DATA_ACCESS_KEY } from '../constants/promptMetadata';
import {
  detectHubDataAccessNeeds,
  hasGrantedHubDataAccess,
  type HubDataAccessOption,
} from '../utils/hubDataAccess';
import { HubDataAccessModal } from './HubDataAccessModal';
import { shouldSendChannelJoinMessage } from '../utils/joinMessage';
import { useWebSocket } from '../hooks/useWebSocket';
import { MessageList } from './MessageList';
import { TypingIndicator } from './TypingIndicator';
import { RichTextInput } from './RichTextInput';
import { ThreadPanel } from './ThreadPanel';
import { MyAgentsPanel } from './MyAgentsPanel';
import { PendingChangesPanel } from './PendingChangesPanel';
import { TerminalPanel } from './TerminalPanel';
import { FileExplorerPanel } from './FileExplorerPanel';
import { CodeEditorPanel } from './CodeEditorPanel';
import { ToastContainer } from './Toast';
import { ErrorBoundary } from './ErrorBoundary';
import { CommandPalette } from './CommandPalette';
import { ChannelSidebar } from './ChannelSidebar';
import { CreateChannelModal } from './CreateChannelModal';
import { ChannelInfoModal } from './ChannelInfoModal';
import { CreateNewDMModal } from './CreateNewDMModal';
import { CollaborationPanel } from './CollaborationPanel';
import { CollaborationWorkspaceGate } from './CollaborationWorkspaceGate';
import { TaskManagementPanel } from './TaskManagementPanel';
import { ModelLibraryModal } from './ModelLibraryModal';
import {
  PendingChangesIcon,
  MyAgentsIcon,
  FilesIcon,
  EditorIcon,
  TerminalIcon,
  ModelLibraryIcon,
  SettingsIcon,
  LogoutIcon,
  LeftSidebarIcon,
  TaskManagementIcon,
} from './Icons';
import type {
  AssistantReminder,
  AssistantTask,
  Channel,
  Collaboration,
  CommandDefinition,
  Message,
  ThinkingAgent,
  ThinkingStatusMetadata,
} from '../types/protocol';
import { isCollaborationMessage, getCollaborationId } from '../types/protocol';
import { confirmStartCollaborationWhileExecuting } from '../utils/collaborationConfirm';
import { ensureCollaborationExecutionWorkspace } from '../utils/collaborationExecutionWorkspace';
import {
  ensureRepoAgentWorkspace,
  isRepoAgentWorkspaceAction,
  parseCreateRepoAgentCommand,
} from '../utils/repoAgentWorkspace';
import { useFileExplorerStore } from '../stores/fileExplorerStore';

const CLIENT_PALETTE_COMMANDS: CommandDefinition[] = [
  {
    name: '/nj-open-model-library',
    description: 'Open model library (Ollama & Hugging Face — download, install, assign to agents)',
    category: 'Neural Junkie',
    arguments: [],
  },
];

function withClientPaletteCommands(defs: CommandDefinition[]): CommandDefinition[] {
  const names = new Set(CLIENT_PALETTE_COMMANDS.map((c) => c.name));
  return [...CLIENT_PALETTE_COMMANDS, ...defs.filter((d) => !names.has(d.name))];
}

const EMPTY_THINKING_AGENTS: ThinkingAgent[] = [];

interface ChatWindowProps {
  onOpenSettings?: () => void;
  onLogout?: () => void;
}

export function ChatWindow({ onOpenSettings, onLogout }: ChatWindowProps = {}) {
  const { serverAddr, channel, username, agents, channels, switchAllAgentProviders } = useChatStore(
    (s) => ({
      serverAddr: s.serverAddr,
      channel: s.channel,
      username: s.username,
      agents: s.agents,
      channels: s.channels,
      switchAllAgentProviders: s.switchAllAgentProviders,
    }),
    shallow
  );

  const { openThreadId, parentMessage } = useChatStore(
    (s) => ({
      openThreadId: s.openThreadId,
      parentMessage: s.openThreadId ? s.messages.find((m) => m.id === s.openThreadId) ?? null : null,
    }),
    shallow
  );

  const thinkingAgentsForChannel = useChatStore(
    (s) => {
      const inner = s.channelThinkingAgents.get(s.channel);
      if (!inner || inner.size === 0) return EMPTY_THINKING_AGENTS;
      return Array.from(inner.values());
    },
    shallow
  );

  const myAgentsPanelOpen = useChatStore((s) => s.myAgentsPanelOpen);
  const setMyAgentsPanelOpen = useChatStore((s) => s.setMyAgentsPanelOpen);

  const { isPanelOpen, panelHeight, addSuggestedCommand, setPanelOpen } = useTerminalStore();
  const { layoutSettings, loadLayoutSettings } = useSettingsStore();
  const addToast = useToastStore(s => s.addToast);

  // State for tracking counts
  const [totalAgentsCount, setTotalAgentsCount] = useState(0);

  // State for file explorer and code editor panels
  const [fileExplorerOpen, setFileExplorerOpen] = useState(false);
  const [codeEditorOpen, setCodeEditorOpen] = useState(false);
  
  // State for pending changes panel
  const [pendingChangesOpen, setPendingChangesOpen] = useState(false);

  // Sidebar visibility
  const [channelSidebarOpen, setChannelSidebarOpen] = useState<boolean>(() => {
    return localStorage.getItem('channel-sidebar-open') !== 'false';
  });

  // State for create channel modal
  const [createChannelOpen, setCreateChannelOpen] = useState(false);
  const [createNewDmOpen, setCreateNewDmOpen] = useState(false);
  const [channelInfoModal, setChannelInfoModal] = useState<Channel | null>(null);

  // State for command palette
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [commandPaletteFilter, setCommandPaletteFilter] = useState('');
  const [commandDefs, setCommandDefs] = useState<CommandDefinition[]>([]);
  const [modelLibraryOpen, setModelLibraryOpen] = useState(false);

  // State for active collaboration panel
  const [activeCollab, setActiveCollab] = useState<Collaboration | null>(null);
  const activeCollabRef = useRef<Collaboration | null>(null);
  const collaborationsByIDRef = useRef<Record<string, Collaboration>>({});
  const [taskManagementOpen, setTaskManagementOpen] = useState(false);
  const [collaborationsByID, setCollaborationsByID] = useState<Record<string, Collaboration>>({});
  const [assistantTasks, setAssistantTasks] = useState<AssistantTask[]>([]);
  const [assistantReminders, setAssistantReminders] = useState<AssistantReminder[]>([]);
  const [messageSearchQuery, setMessageSearchQuery] = useState('');
  const [hubAccessPending, setHubAccessPending] = useState<{
    mode: 'main' | 'thread';
    threadId?: string;
    content: string;
    metadata?: Record<string, unknown>;
    options: HubDataAccessOption[];
  } | null>(null);
  const [hubAccessLoading, setHubAccessLoading] = useState(false);
  const [hubAccessError, setHubAccessError] = useState<string | null>(null);

  const isTerminalCollaborationPhase = (phase?: Collaboration['phase']) =>
    phase === 'completed' || phase === 'cancelled';

  useEffect(() => {
    activeCollabRef.current = activeCollab;
  }, [activeCollab]);

  useEffect(() => {
    collaborationsByIDRef.current = collaborationsByID;
  }, [collaborationsByID]);

  const [workspaceContextMode, setWorkspaceContextMode] = useState<WorkspaceContextMode>(() =>
    loadWorkspaceContextMode()
  );
  const [composerDraft, setComposerDraft] = useState('');

  const activeChannelMeta = useMemo(
    () => channels.find((c) => c.name === channel),
    [channels, channel]
  );

  const contextScopePreview = useMemo(() => {
    return resolveContextScope({
      message: composerDraft,
      mode: workspaceContextMode,
      channelKind: channelNameToKind(channel, activeChannelMeta?.type),
    });
  }, [composerDraft, workspaceContextMode, channel, activeChannelMeta?.type]);

  const [workspaceGateCollab, setWorkspaceGateCollab] = useState<Collaboration | null>(null);
  const [workspaceGateBusy, setWorkspaceGateBusy] = useState(false);
  const dismissedWorkspaceGateIdRef = useRef<string | null>(null);
  const handledRepoWorkspaceActionsRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    const activeCh = useChatStore.getState().channel;
    let next: Collaboration | null = null;
    for (const c of Object.values(collaborationsByID)) {
      if (c.phase !== 'executing' || c.workspace_acknowledged) continue;
      const isWorktree = c.execution_mode === 'worktree';
      if (!isWorktree && !c.working_directory?.trim()) continue;
      if (c.channel !== activeCh) continue;
      if (dismissedWorkspaceGateIdRef.current === c.id) continue;
      next = c;
      break;
    }
    setWorkspaceGateCollab(next);
  }, [collaborationsByID, channel]);

  // Clear stale chat width from localStorage (no longer used - chat area always flex-grows)
  useEffect(() => {
    localStorage.removeItem('main-chat-area-width');
  }, []);

  // Keyboard shortcuts for sidebar toggles
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.metaKey && !e.shiftKey && e.key === 'b') {
        e.preventDefault();
        setChannelSidebarOpen((prev) => {
          const next = !prev;
          localStorage.setItem('channel-sidebar-open', String(next));
          return next;
        });
      } else if (e.metaKey && e.shiftKey && e.key.toLowerCase() === 't') {
        e.preventDefault();
        setTaskManagementOpen(prev => !prev);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);
  
  const api = useMemo(() => new ChatAPI(serverAddr), [serverAddr]);
  const hubHttp = useMemo(
    () => (serverAddr.startsWith('http') ? serverAddr : `http://${serverAddr}`),
    [serverAddr]
  );
  const wsURL = useMemo(() => api.getWebSocketURL(channel), [api, channel]);
  
  // Debounce timeout ref for agent list refresh
  const agentRefreshTimeoutRef = useRef<number | null>(null);
  
  // Ref to access RichTextInput methods
  const inputRef = useRef<HTMLTextAreaElement | null>(null);

  // Load layout settings on mount
  useEffect(() => {
    loadLayoutSettings();
  }, [loadLayoutSettings]);

  // Apply layout settings when they change (including initial mount and real-time updates)
  useEffect(() => {
    if (layoutSettings) {
      // Apply panel visibility from settings
      setFileExplorerOpen(layoutSettings.filesPanelVisible);
      setCodeEditorOpen(layoutSettings.editorPanelVisible);
      setPanelOpen(layoutSettings.terminalPanelVisible);
      setMyAgentsPanelOpen(layoutSettings.myAgentsPanelVisible);
      setPendingChangesOpen(layoutSettings.pendingChangesPanelVisible);
    }
  }, [layoutSettings, setPanelOpen, setMyAgentsPanelOpen, setPendingChangesOpen]);

  // Load agents function
  const loadAgents = useCallback(async () => {
    try {
      const agentList = await api.fetchAgents();
      useChatStore.getState().setAgents(agentList);

      // Remove agents from loading state if they're now active
      const { loadingAgents, removeLoadingAgent } = useChatStore.getState();
      const activeAgentNames = new Set(agentList.map((agent) => agent.name));

      loadingAgents.forEach((agentName) => {
        if (activeAgentNames.has(agentName)) {
          removeLoadingAgent(agentName);
        }
      });
    } catch (error) {
      console.error('Failed to load agents:', error);
    }
  }, [api]);

  // Load counts for badges
  const loadCounts = useCallback(async () => {
    try {
      const [myAgents, removedAgents] = await Promise.all([
        api.fetchMyAgents(),
        api.fetchRemovedAgents()
      ]);
      setTotalAgentsCount(myAgents.length + removedAgents.length);
    } catch (error) {
      console.error('Failed to load counts:', error);
    }
  }, [api]);

  // Load channels
  const loadChannels = useCallback(async () => {
    try {
      const channelList = await api.fetchChannels();
      useChatStore.getState().setChannels(channelList);
    } catch (error) {
      console.error('Failed to load channels:', error);
    }
  }, [api]);

  const pruneTerminalCollaborations = (
    next: Record<string, Collaboration>,
    channelName: string
  ): Record<string, Collaboration> => {
    const terminals = Object.values(next)
      .filter(
        c =>
          isTerminalCollaborationPhase(c.phase) &&
          (c.channel || '') === channelName
      )
      .sort(
        (a, b) => Date.parse(b.updated_at || '') - Date.parse(a.updated_at || '')
      );
    if (terminals.length <= 3) return next;
    const drop = new Set(terminals.slice(3).map(c => c.id));
    const pruned = { ...next };
    for (const id of drop) {
      delete pruned[id];
    }
    return pruned;
  };

  const mergeCollaborationSnapshot = useCallback((snapshot: Collaboration) => {
    if (!snapshot?.id) return;
    const isTerminal = isTerminalCollaborationPhase(snapshot.phase);
    const wasTerminal =
      collaborationsByIDRef.current[snapshot.id] &&
      isTerminalCollaborationPhase(collaborationsByIDRef.current[snapshot.id].phase);

    setCollaborationsByID(prev => {
      const existing = prev[snapshot.id];
      if (
        existing &&
        existing.updated_at === snapshot.updated_at &&
        existing.phase === snapshot.phase &&
        existing.workspace_acknowledged === snapshot.workspace_acknowledged
      ) {
        return prev;
      }
      if (existing) {
        const nextTime = Date.parse(snapshot.updated_at || '');
        const existingTime = Date.parse(existing.updated_at || '');
        if (!Number.isNaN(nextTime) && !Number.isNaN(existingTime) && nextTime < existingTime) {
          return prev;
        }
      }
      let next = { ...prev, [snapshot.id]: snapshot };
      if (isTerminal && snapshot.channel) {
        next = pruneTerminalCollaborations(next, snapshot.channel);
      }
      return next;
    });

    if (isTerminal && !wasTerminal) {
      const completed =
        snapshot.tasks?.filter(t => t.status === 'completed').length ?? 0;
      const total = snapshot.tasks?.length ?? 0;
      addToast({
        type: 'success',
        title: 'Collaboration completed',
        message:
          total > 0
            ? `${snapshot.title || 'Collaboration'} — ${completed}/${total} tasks done.`
            : `${snapshot.title || 'Collaboration'} is closed.`,
      });
    }

    if (isTerminal) {
      setActiveCollab(current => (current?.id === snapshot.id ? snapshot : current));
    }
  }, []);

  const clearActiveCollabIf = useCallback((collaborationID: string) => {
    setActiveCollab(current => (current?.id === collaborationID ? null : current));
  }, []);

  const loadCollaborations = useCallback(async (targetChannel: string) => {
    try {
      const includeTerminal = targetChannel.startsWith('collab-');
      const snapshots = await api.fetchCollaborations(undefined, includeTerminal);
      setCollaborationsByID(() => {
        const next: Record<string, Collaboration> = {};
        for (const snapshot of snapshots) {
          if (!snapshot?.id) continue;
          if (isTerminalCollaborationPhase(snapshot.phase)) {
            if (includeTerminal && snapshot.channel === targetChannel) {
              next[snapshot.id] = snapshot;
            }
            continue;
          }
          next[snapshot.id] = snapshot;
        }
        if (includeTerminal) {
          return pruneTerminalCollaborations(next, targetChannel);
        }
        return next;
      });
      setActiveCollab(current => {
        if (!current || current.channel !== targetChannel) return current;
        const refreshed = snapshots.find(snapshot => snapshot.id === current.id);
        if (!refreshed) return null;
        return refreshed;
      });
    } catch (error) {
      console.error('Failed to load collaborations:', error);
    }
  }, [api]);

  const handleWorkspaceGateContinue = useCallback(async () => {
    const c = workspaceGateCollab;
    if (!c) return;
    setWorkspaceGateBusy(true);
    try {
      let sourceRepoPath: string | undefined;
      if (c.execution_mode === 'worktree' && !c.source_repo_path?.trim()) {
        const active = useFileExplorerStore.getState().getActiveWorkspace();
        if (!active?.path?.trim()) {
          throw new Error('Select a git workspace in the file explorer before continuing.');
        }
        if (!active.is_git_repo) {
          throw new Error('Active workspace is not a git repository.');
        }
        sourceRepoPath = active.path;
      }
      const deferWorktree = c.execution_mode === 'worktree' && !c.working_directory?.trim();
      if (!deferWorktree) {
        await ensureCollaborationExecutionWorkspace(c);
      }
      await api.acknowledgeCollaborationWorkspace(c.id, sourceRepoPath);
      dismissedWorkspaceGateIdRef.current = null;
      if (useChatStore.getState().channel === c.channel) {
        setWorkspaceContextMode('always');
        localStorage.setItem(WORKSPACE_CONTEXT_MODE_KEY, 'always');
      }
      await loadCollaborations(channel);
      if (deferWorktree) {
        const refreshed = collaborationsByIDRef.current[c.id];
        if (refreshed?.working_directory?.trim()) {
          await ensureCollaborationExecutionWorkspace(refreshed);
        }
      }
      setWorkspaceGateCollab(null);
    } catch (e) {
      console.error('[workspace gate]', e);
    } finally {
      setWorkspaceGateBusy(false);
    }
  }, [workspaceGateCollab, api, channel, loadCollaborations]);

  const handleWorkspaceGateDismiss = useCallback(() => {
    if (workspaceGateCollab) {
      dismissedWorkspaceGateIdRef.current = workspaceGateCollab.id;
    }
    setWorkspaceGateCollab(null);
  }, [workspaceGateCollab]);

  const trackedCollaborations = useMemo(
    () =>
      Object.values(collaborationsByID).sort(
        (a, b) => Date.parse(b.updated_at || '') - Date.parse(a.updated_at || '')
      ),
    [collaborationsByID]
  );

  const executingCollaborationForChannel = useMemo(
    () =>
      trackedCollaborations.find(c => c.channel === channel && c.phase === 'executing') ?? null,
    [trackedCollaborations, channel]
  );

  const extendableCollaborations = useMemo(
    () =>
      trackedCollaborations.filter(
        (c) =>
          (c.phase === 'planning' || c.phase === 'reviewing') &&
          (c.discussion?.status === 'budget_exhausted' ||
            c.discussion?.status === 'timed_out' ||
            c.discussion?.status === 'active')
      ),
    [trackedCollaborations]
  );

  // Handle switching channel: switch store state, then load fresh messages
  const handleSwitchChannel = useCallback(
    async (channelName: string) => {
      if (channelName === useChatStore.getState().channel) return;
      // Collaboration side panel is channel-scoped; clear when navigating.
      setActiveCollab(null);
      useChatStore.getState().switchChannel(channelName);
      localStorage.setItem('last-channel', channelName);
      try {
        const msgs = await api.fetchMessages(channelName, 50);
        useChatStore.getState().setMessages(msgs);
        useChatStore.getState().cleanupStaleThinking(channelName, msgs);
        await loadCollaborations(channelName);
      } catch (error) {
        console.error('Failed to load messages for channel:', error);
      }
    },
    [api, loadCollaborations]
  );

  // Create a custom channel
  const handleCreateChannel = useCallback(async (name: string, description: string, agentIds: string[]) => {
    try {
      await api.createChannel(name, description, 'custom', agentIds, username);
      await loadChannels();
      await handleSwitchChannel(name);
    } catch (error) {
      console.error('Failed to create channel:', error);
    }
  }, [api, username, loadChannels, handleSwitchChannel]);

  const handleDeleteChannel = useCallback(
    async (name: string) => {
      if (!window.confirm(`Delete channel #${name}? This cannot be undone.`)) return;
      try {
        await api.deleteChannel(name);
        const wasActive = useChatStore.getState().channel === name;
        await loadChannels();
        if (wasActive) {
          await handleSwitchChannel('general');
        }
        setChannelInfoModal((cur) => (cur?.name === name ? null : cur));
        addToast({
          type: 'success',
          title: 'Channel deleted',
          message: `#${name} was removed.`,
        });
      } catch (error) {
        console.error('Failed to delete channel:', error);
        addToast({
          type: 'error',
          title: 'Could not delete channel',
          message: error instanceof Error ? error.message : 'Unknown error',
        });
      }
    },
    [api, loadChannels, handleSwitchChannel, addToast]
  );

  const handleOpenChannelInfo = useCallback(
    async (ch: Channel) => {
      try {
        await loadChannels();
        const list = useChatStore.getState().channels;
        const fresh = list.find((c) => c.name === ch.name) ?? ch;
        setChannelInfoModal(fresh);
      } catch {
        setChannelInfoModal(ch);
      }
    },
    [loadChannels]
  );

  // Create a DM channel with an agent
  const handleCreateDM = useCallback(async (agentId: string) => {
    try {
      const ch = await api.createChannel('', '', 'dm', [agentId], username);
      await loadChannels();
      await handleSwitchChannel(ch.name);
    } catch (error) {
      console.error('Failed to create DM channel:', error);
    }
  }, [api, username, loadChannels, handleSwitchChannel]);

  const handleNewDmCreated = useCallback(
    async (ch: Channel) => {
      try {
        addToast({
          type: 'success',
          title: 'Direct message ready',
          message: `Opened ${ch.description || ch.name}`,
        });
        const channelList = await api.fetchChannels();
        const merged = channelList.some((c) => c.name === ch.name) ? channelList : [...channelList, ch];
        useChatStore.getState().setChannels(merged);
        await loadAgents();
        await handleSwitchChannel(ch.name);
      } catch (e) {
        console.error('Failed after creating DM agent:', e);
        addToast({
          type: 'error',
          title: 'Could not open DM',
          message: e instanceof Error ? e.message : 'Unknown error',
        });
      }
    },
    [addToast, api, loadAgents, handleSwitchChannel]
  );

  // Debounced agent refresh (prevents excessive API calls).
  // Channel list is only refreshed on agent_join/agent_leave, not on every status tick.
  const debouncedRefreshAgents = useCallback(() => {
    if (agentRefreshTimeoutRef.current) {
      clearTimeout(agentRefreshTimeoutRef.current);
    }
    agentRefreshTimeoutRef.current = window.setTimeout(() => {
      loadAgents();
      loadCounts();
    }, 300);
  }, [loadAgents, loadCounts]);

  // WebSocket connection
  const { status } = useWebSocket({
    url: wsURL,
    onMessage: async (message: Message) => {
      try {
        const st = useChatStore.getState();
        const activeChannel = st.channel;

      // Handle all agent_status messages - never add them to chat
      if (message.type === 'agent_status') {
        if (message.metadata?.history_resync === true) {
          const ch = message.channel || channel;
          try {
            const msgs = await api.fetchMessages(ch, 50);
            const st = useChatStore.getState();
            st.replaceChannelMessagesCache(ch, msgs);
            if (ch === st.channel) {
              st.setMessages(msgs);
              st.cleanupStaleThinking(ch, msgs);
            }
          } catch (e) {
            console.error('[ChatWindow] history_resync refetch failed:', e);
          }
          return;
        }
        // Handle thinking status -> typing indicator
        if (message.metadata?.thinking_status) {
          const thinkingStatus = message.metadata.thinking_status as ThinkingStatusMetadata['thinking_status'];
          const msgChannel = message.channel || activeChannel;
          
          if (thinkingStatus === 'started') {
            st.addThinkingAgent(msgChannel, message.from.id, message.from.name, message.from.type);
          } else if (thinkingStatus === 'completed' || thinkingStatus === 'error') {
            st.removeThinkingAgent(msgChannel, message.from.id);
          }
        }
        
        // Handle status updates - update agent info immediately
        if (message.metadata?.indexing_status !== undefined || 
            message.metadata?.index_progress !== undefined ||
            message.metadata?.status !== undefined ||
            message.from.is_paused !== undefined) {
          const statusUpdates: Partial<typeof message.from> = {};
          
          if (message.metadata?.indexing_status !== undefined) {
            statusUpdates.indexing_status = message.metadata.indexing_status as string;
          }
          if (message.metadata?.index_progress !== undefined) {
            statusUpdates.index_progress = message.metadata.index_progress as number;
          }
          if (message.metadata?.status !== undefined) {
            statusUpdates.status = message.metadata.status as string;
          }
          if (message.from.is_paused !== undefined) {
            statusUpdates.is_paused = message.from.is_paused;
          }
          
          st.updateAgentStatus(message.from.id, statusUpdates);
        }
        
        return; // Never add agent_status to message list
      }
      
      // Handle streaming tokens -- accumulate deltas, finalize on stream_end
      if (message.type === 'stream_delta') {
        if (!message.is_thread_reply) {
          if (!message.channel || message.channel === activeChannel) {
            st.appendStreamDelta(message);
          }
        }
        st.removeThinkingAgent(message.channel || activeChannel, message.from.id);
        return;
      }
      if (message.type === 'stream_end') {
        if (!message.is_thread_reply && (!message.channel || message.channel === activeChannel)) {
          st.finalizeStream(message.id);
        }
        return;
      }

      // Track collaboration snapshots from message metadata (transition: keeps typing/input responsive during agent bursts).
      const collabData = message.metadata?.collaboration_data as Collaboration | undefined;
      if (collabData?.id) {
        startTransition(() => {
          const collabChannel = collabData.channel || message.channel;
          const isActiveChannelCollab = !collabChannel || collabChannel === activeChannel;
          const previousSnapshot = collaborationsByIDRef.current[collabData.id];
          if (
            previousSnapshot &&
            isActiveChannelCollab &&
            (collabData.phase === 'planning' || collabData.phase === 'reviewing')
          ) {
            const existingIDs = new Set((previousSnapshot.agents || []).map(a => a.agent_id));
            const addedAgents = (collabData.agents || []).filter(a => !existingIDs.has(a.agent_id));
            if (addedAgents.length > 0) {
              const names = addedAgents.map(a => `@${a.agent_name}`).join(', ');
              addToast({
                type: 'info',
                title: 'Collaborator added',
                message: `${names} joined "${collabData.title}".`,
              });
            }
          }
          mergeCollaborationSnapshot(collabData);
          const currentlyOpen = activeCollabRef.current;
          if (currentlyOpen?.id === collabData.id) {
            if (isActiveChannelCollab || isTerminalCollaborationPhase(collabData.phase)) {
              setActiveCollab(collabData);
            }
          } else if (
            !currentlyOpen &&
            isActiveChannelCollab &&
            isCollaborationMessage(message) &&
            !isTerminalCollaborationPhase(collabData.phase)
          ) {
            setActiveCollab(collabData);
          }
        });
      }

      // Handle thread messages - only update metadata, ThreadPanel's WebSocket will add the actual message
      if (message.is_thread_reply && message.thread_id) {
        void api
          .fetchThreadMetadata(message.thread_id)
          .then(metadata => useChatStore.getState().updateThreadMetadata(message.thread_id!, metadata))
          .catch(error => console.error('Failed to fetch thread metadata:', error));
      } else if (message.channel && message.channel !== activeChannel) {
        // Message belongs to a different channel -- cache it and mark unread
        st.addMessageToCache(message.channel, message);
        st.markChannelUnread(message.channel);
        if (isCollaborationMessage(message) || getCollaborationId(message)) {
          addToast({
            type: 'info',
            title: 'Collaboration update',
            message: `Activity in #${message.channel} — switch there to see messages.`,
          });
        }
      } else {
        // Message belongs to the active channel (never wrap addMessage in startTransition —
        // high-frequency agent_status updates can starve transitions and leave the chat empty).
        st.addMessage(message);

        if (message.metadata?.suggested_commands) {
          const suggestions = message.metadata.suggested_commands as any[];
          suggestions.forEach((suggestion) => {
            addSuggestedCommand(suggestion);
          });
        }

        if (message.metadata?.event === 'agent-open-terminal') {
          const agentName = message.metadata.agent_name as string || 'Agent';
          const cwd = message.metadata.cwd as string || undefined;
          const tab = createNewTab('agent', agentName, cwd);
          useTerminalStore.getState().addTab(tab);
          useTerminalStore.getState().setPanelOpen(true);
        }

        const clientAction = message.metadata?.client_action;
        if (
          clientAction &&
          isRepoAgentWorkspaceAction(clientAction) &&
          !handledRepoWorkspaceActionsRef.current.has(message.id)
        ) {
          handledRepoWorkspaceActionsRef.current.add(message.id);
          void ensureRepoAgentWorkspace(clientAction.path, {
            preferredName: clientAction.name,
          }).then((workspaceId) => {
            if (workspaceId) {
              setFileExplorerOpen(true);
            }
          });
        }
      }
      
      // Clear thinking indicator when agent sends actual message
      if (message.type === 'chat' || message.type === 'answer') {
        st.removeThinkingAgent(message.channel || activeChannel, message.from.id);
      }
      
      // Auto-refresh agents and channels for join/leave events
      if (message.type === 'agent_join' || message.type === 'agent_leave') {
        debouncedRefreshAgents();
        loadChannels();
      }
      } catch (err) {
        console.error('[ChatWindow] WebSocket message handler error:', err);
      }
    },
    onConnect: () => {
      console.log('Connected to chat');
      useChatStore.getState().setConnectionStatus('connected');
      loadInitialData();
    },
    onDisconnect: () => {
      console.log('Disconnected from chat');
      useChatStore.getState().setConnectionStatus('disconnected');
    },
    onError: (error) => {
      console.error('WebSocket error:', error);
      useChatStore.getState().setConnectionStatus('error');
    },
  });

  // Load initial data when connected (parallelize; never skip channels because another request failed)
  const loadInitialData = async () => {
    const activeCh = useChatStore.getState().channel;
    const results = await Promise.allSettled([
      api.fetchMessages(activeCh, 50).then((msgs) => useChatStore.getState().setMessages(msgs)),
      loadCollaborations(activeCh),
      loadAgents(),
      loadCounts(),
      loadChannels(),
      useFileExplorerStore.getState().loadWorkspaces(),
    ]);

    results.forEach((r, i) => {
      if (r.status === 'rejected') {
        const label = ['messages', 'collaborations', 'agents', 'counts', 'channels', 'workspaces'][i];
        console.error(`[loadInitialData] ${label} failed:`, r.reason);
      }
    });

    try {
      const defs = await api.fetchCommands();
      setCommandDefs(withClientPaletteCommands(defs));
    } catch (err) {
      console.error('Failed to load command definitions:', err);
      setCommandDefs(withClientPaletteCommands([]));
    }

    const { channel: joinCh, username: joinUser } = useChatStore.getState();
    const joinName = joinUser?.trim() || 'User';
    if (shouldSendChannelJoinMessage(joinCh, joinName)) {
      void api
        .sendMessage(
          joinCh,
          `${joinName} has joined the chat`,
          { name: joinName, type: 'human' },
          'system_info'
        )
        .catch((e) => console.error('[loadInitialData] join message failed:', e));
    }
  };

  const dispatchThreadReply = useCallback(
    async (threadId: string, content: string, metadata?: Record<string, unknown>) => {
      const mergedMetadata = buildHumanOutboundMetadata({
        contextMode: workspaceContextMode,
        message: content,
        channel,
        channelType: activeChannelMeta?.type,
        composerMetadata: metadata,
      });
      await api.sendThreadReply(
        threadId,
        channel,
        content,
        { name: username, type: 'human' },
        mergedMetadata
      );
    },
    [api, channel, username, workspaceContextMode, activeChannelMeta?.type]
  );

  const dispatchMessage = useCallback(
    async (content: string, metadata?: Record<string, unknown>) => {
      const mergedMetadata = buildHumanOutboundMetadata({
        contextMode: workspaceContextMode,
        message: content,
        channel,
        channelType: activeChannelMeta?.type,
        composerMetadata: metadata,
      });

      useChatStore.getState().setIsTyping(true);
      try {
        const trimmed = content.trimStart();
        if (trimmed.startsWith('/collaborate')) {
          if (!confirmStartCollaborationWhileExecuting(executingCollaborationForChannel)) {
            return;
          }
        }
        const sendResult = await api.sendMessage(
          channel,
          content,
          { name: username, type: 'human' },
          'question',
          mergedMetadata
        );
        let timelineChannel = channel;
        if (sendResult.collaboration_channel) {
          await loadChannels();
          await handleSwitchChannel(sendResult.collaboration_channel);
          timelineChannel = sendResult.collaboration_channel;
        }
        if (content.trimStart().startsWith('/')) {
          try {
            const msgs = await api.fetchMessages(timelineChannel, 50);
            useChatStore.getState().setMessages(msgs);
            await loadCollaborations(timelineChannel);
          } catch (e) {
            console.error('[dispatchMessage] post-command refresh failed:', e);
          }
        }
      } catch (error) {
        console.error('Failed to send message:', error);
        addToast({
          type: 'error',
          title: 'Message not sent',
          message: error instanceof Error ? error.message : 'Failed to send message.',
        });
      } finally {
        useChatStore.getState().setIsTyping(false);
      }
    },
    [
      api,
      channel,
      username,
      workspaceContextMode,
      activeChannelMeta?.type,
      loadChannels,
      handleSwitchChannel,
      loadCollaborations,
      executingCollaborationForChannel,
      addToast,
    ]
  );

  const handleSendMessage = async (content: string, metadata?: Record<string, unknown>) => {
    if (content.trim() === '/nj-open-model-library') {
      setModelLibraryOpen(true);
      return;
    }

    const needs = detectHubDataAccessNeeds(content);
    const composerMeta = metadata ?? {};
    if (needs.length > 0 && !hasGrantedHubDataAccess(composerMeta)) {
      setHubAccessError(null);
      setHubAccessPending({ mode: 'main', content, metadata: composerMeta, options: needs });
      return;
    }

    await dispatchMessage(content, composerMeta);
  };

  const handleThreadSend = useCallback(
    async (content: string, composerMeta?: Record<string, unknown>) => {
      if (!openThreadId) return;
      const needs = detectHubDataAccessNeeds(content);
      const meta = composerMeta ?? {};
      if (needs.length > 0 && !hasGrantedHubDataAccess(meta)) {
        setHubAccessError(null);
        setHubAccessPending({
          mode: 'thread',
          threadId: openThreadId,
          content,
          metadata: meta,
          options: needs,
        });
        return;
      }
      await dispatchThreadReply(openThreadId, content, meta);
    },
    [openThreadId, dispatchThreadReply]
  );

  const handleHubAccessConfirm = async (selected: HubDataAccessOption[]) => {
    if (!hubAccessPending) return;
    setHubAccessLoading(true);
    setHubAccessError(null);
    try {
      const result = await api.readHubDataAccess(
        selected.map((s) => ({ kind: s.kind, relative_path: s.relativePath }))
      );
      const merged = {
        ...(hubAccessPending.metadata ?? {}),
        [GRANTED_HUB_DATA_ACCESS_KEY]: result,
      };
      if (hubAccessPending.mode === 'thread' && hubAccessPending.threadId) {
        await dispatchThreadReply(hubAccessPending.threadId, hubAccessPending.content, merged);
      } else {
        await dispatchMessage(hubAccessPending.content, merged);
      }
      setHubAccessPending(null);
    } catch (err) {
      setHubAccessError(err instanceof Error ? err.message : 'Failed to read hub data');
    } finally {
      setHubAccessLoading(false);
    }
  };

  // Ensure command definitions are loaded, fetching them if needed
  const ensureCommandDefs = async (forceRefresh: boolean = false) => {
    if (!forceRefresh && commandDefs.length > 0) return;
    try {
      const defs = await api.fetchCommands(forceRefresh);
      setCommandDefs(withClientPaletteCommands(defs));
    } catch (err) {
      console.error('Failed to load command definitions:', err);
      setCommandDefs(withClientPaletteCommands([]));
    }
  };

  // Handle command executed from command palette
  const handleCommandExecute = async (commandString: string) => {
    if (inputRef.current && (inputRef.current as any).clearInput) {
      (inputRef.current as any).clearInput();
    }
    const trimmed = commandString.trim();
    if (trimmed === '/nj-open-model-library') {
      setModelLibraryOpen(true);
      return;
    }
    const repoAgentCmd = parseCreateRepoAgentCommand(trimmed);
    await handleSendMessage(commandString);
    if (repoAgentCmd) {
      window.setTimeout(() => {
        void ensureRepoAgentWorkspace(repoAgentCmd.repoPath, {
          preferredName: repoAgentCmd.agentName,
        }).then((workspaceId) => {
          if (workspaceId) {
            setFileExplorerOpen(true);
          }
        });
      }, 400);
    }
  };

  // Handle slash trigger from input
  const handleSlashTrigger = async (query: string) => {
    await ensureCommandDefs(true);
    setCommandPaletteFilter(query);
    setCommandPaletteOpen(true);
  };

  // Open command palette from toolbar button
  const openCommandPalette = async () => {
    await ensureCommandDefs(true);
    setCommandPaletteFilter('');
    setCommandPaletteOpen(true);
  };

  const handleLogout = async () => {
    try {
      // Clear saved credentials
      await clearCredentials();
      
      // Reset chat store state
      useChatStore.getState().logout();
      
      // Notify parent to switch to login view
      if (onLogout) {
        onLogout();
      }
    } catch (error) {
      console.error('[ChatWindow] Failed to logout:', error);
    }
  };

  const closeThread = useChatStore((s) => s.closeThread);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || !e.shiftKey) return;
      if (e.key.toLowerCase() !== 'm') return;
      e.preventDefault();
      setModelLibraryOpen(true);
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, []);

  const getStatusColor = () => {
    switch (status) {
      case 'connected':
        return 'bg-green-500';
      case 'connecting':
        return 'bg-yellow-500';
      case 'error':
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  const getStatusText = () => {
    switch (status) {
      case 'connected':
        return 'Connected';
      case 'connecting':
        return 'Connecting...';
      case 'error':
        return 'Connection Error';
      default:
        return 'Disconnected';
    }
  };

  const loadAssistantState = useCallback(async () => {
    try {
      const state = await api.fetchAssistantState(channel);
      setAssistantTasks(state.tasks || []);
      setAssistantReminders(state.reminders || []);
    } catch (error) {
      console.error('Failed to load assistant state:', error);
    }
  }, [api, channel]);

  useEffect(() => {
    if (!taskManagementOpen) return;
    void loadCollaborations(channel);
    loadAssistantState();
    const id = window.setInterval(loadAssistantState, 30000);
    return () => window.clearInterval(id);
  }, [taskManagementOpen, loadAssistantState, loadCollaborations, channel]);

  return (
    <ErrorBoundary>
      <div className="flex flex-col h-screen bg-slack-bg">
      <CollaborationWorkspaceGate
        collaboration={workspaceGateCollab}
        busy={workspaceGateBusy}
        onContinue={handleWorkspaceGateContinue}
        onNotNow={handleWorkspaceGateDismiss}
      />
      {/* Top Toolbar - always visible, spans full width */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-slack-border bg-slack-bgHover flex-shrink-0">
        <div className="flex items-center gap-2">
          {(() => {
            const ch = channels.find(c => c.name === channel);
            const isDM = ch?.type === 'dm';
            const agentCount = ch?.agents?.length ?? 0;
            return (
              <>
                <h1 className="text-sm font-bold text-slack-text">
                  {isDM ? '@ ' : '# '}{isDM && ch?.agents?.[0] ? ch.agents[0].name : channel}
                </h1>
                {ch?.description && (
                  <span className="text-xs text-slack-textMuted hidden sm:inline truncate max-w-[200px]" title={ch.description}>
                    {ch.description}
                  </span>
                )}
                {agentCount > 0 && !isDM && (
                  <span className="text-xs text-slack-textMuted bg-slack-bgHover px-1.5 py-0.5 rounded">
                    {agentCount} agent{agentCount !== 1 ? 's' : ''}
                  </span>
                )}
              </>
            );
          })()}
          <div className="flex items-center gap-1.5 text-xs">
            <div className={`w-1.5 h-1.5 rounded-full ${getStatusColor()}`} />
            <span className="text-slack-textMuted">{getStatusText()}</span>
          </div>
          <input
            type="search"
            value={messageSearchQuery}
            onChange={(e) => setMessageSearchQuery(e.target.value)}
            placeholder="Search chat…"
            aria-label="Search messages in this channel"
            className="ml-2 w-44 sm:w-56 rounded-md border border-slack-border bg-slack-bg px-2 py-1 text-xs text-slack-text placeholder:text-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
          />
        </div>
        
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => {
              const next = !channelSidebarOpen;
              setChannelSidebarOpen(next);
              localStorage.setItem('channel-sidebar-open', String(next));
            }}
            className={`w-7 h-7 rounded transition-colors flex items-center justify-center ${
              channelSidebarOpen
                ? 'bg-slack-accent text-white'
                : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text hover:bg-slack-border'
            }`}
            title="Toggle channels sidebar (⌘B)"
            aria-label="Toggle channels sidebar"
            aria-pressed={channelSidebarOpen}
          >
            <LeftSidebarIcon className="w-3.5 h-3.5" />
          </button>

          <div className="w-px h-5 bg-slack-border mx-0.5" />

          <button
            type="button"
            onClick={openCommandPalette}
            className="w-7 h-7 bg-indigo-600 hover:bg-indigo-700 text-white rounded transition-colors flex items-center justify-center font-mono text-xs font-bold focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-400"
            title="Command palette"
            aria-label="Open command palette"
          >
            /
          </button>

          <button
            type="button"
            onClick={() => {
              const next = cycleWorkspaceContextMode(workspaceContextMode);
              setWorkspaceContextMode(next);
              localStorage.setItem(WORKSPACE_CONTEXT_MODE_KEY, next);
            }}
            className={`w-7 h-7 rounded transition-colors flex items-center justify-center relative ${
              workspaceContextMode !== 'off'
                ? 'bg-purple-600 hover:bg-purple-700 text-white ring-1 ring-purple-400 ring-offset-1 ring-offset-slack-bg'
                : 'bg-slack-bgHover hover:bg-slack-border text-slack-textMuted'
            }`}
            title={`Workspace context: ${workspaceContextModeLabel(workspaceContextMode)} (click to cycle). Next send: ${contextScopePreview.scope}`}
            aria-label={`Workspace context mode ${workspaceContextModeLabel(workspaceContextMode)}`}
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
            </svg>
            {workspaceContextMode === 'auto' && (
              <span className="absolute -bottom-0.5 -right-0.5 bg-green-500 rounded-full h-2 w-2 border border-slack-bg" />
            )}
          </button>

          <button
            type="button"
            onClick={() => setPendingChangesOpen(true)}
            className="w-7 h-7 bg-orange-600 hover:bg-orange-700 text-white rounded transition-colors flex items-center justify-center relative focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-orange-400"
            title="Pending changes"
            aria-label="Open pending file changes"
          >
            <PendingChangesIcon className="w-3.5 h-3.5" />
          </button>

          <button
            type="button"
            onClick={() => setTaskManagementOpen(true)}
            className={`w-7 h-7 rounded transition-colors flex items-center justify-center ${
              taskManagementOpen ? 'bg-violet-600 hover:bg-violet-700' : 'bg-violet-700/80 hover:bg-violet-700'
            } text-white`}
            title="Task management (⌘⇧T)"
            aria-label="Open task management"
            aria-pressed={taskManagementOpen}
          >
            <TaskManagementIcon className="w-3.5 h-3.5" />
          </button>
          
          <button
            type="button"
            onClick={() => setMyAgentsPanelOpen(true)}
            className="w-7 h-7 bg-slack-accent hover:bg-slack-accentHover text-white rounded transition-colors flex items-center justify-center relative focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
            title="My agents"
            aria-label="Open my agents"
          >
            <MyAgentsIcon className="w-3.5 h-3.5" />
            {totalAgentsCount > 0 && (
              <span className="absolute -bottom-0.5 -right-0.5 bg-white text-slack-accent text-[10px] font-bold rounded-full h-4 w-4 flex items-center justify-center leading-none">
                {totalAgentsCount}
              </span>
            )}
          </button>
          
          <button
            type="button"
            onClick={() => setFileExplorerOpen(true)}
            className="w-7 h-7 bg-green-600 hover:bg-green-700 text-white rounded transition-colors flex items-center justify-center focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-green-400"
            title="File explorer"
            aria-label="Open file explorer"
          >
            <FilesIcon className="w-3.5 h-3.5" />
          </button>
          
          <button
            type="button"
            onClick={() => setCodeEditorOpen(true)}
            className="w-7 h-7 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors flex items-center justify-center focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-400"
            title="Code editor"
            aria-label="Open code editor"
          >
            <EditorIcon className="w-3.5 h-3.5" />
          </button>
          
          <button
            type="button"
            onClick={() => useTerminalStore.getState().togglePanel()}
            className="w-7 h-7 bg-gray-600 hover:bg-gray-700 text-white rounded transition-colors flex items-center justify-center relative focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-400"
            title="Terminal (⌘J)"
            aria-label="Toggle terminal panel"
          >
            <TerminalIcon className="w-3.5 h-3.5" />
            {useTerminalStore.getState().suggestedCommands.length > 0 && (
              <span className="absolute -bottom-0.5 -right-0.5 bg-yellow-500 text-black text-[10px] font-bold rounded-full h-4 w-4 flex items-center justify-center leading-none">
                {useTerminalStore.getState().suggestedCommands.length}
              </span>
            )}
          </button>
          
          <div className="w-px h-5 bg-slack-border mx-0.5" />

          <button
            type="button"
            onClick={() => setModelLibraryOpen(true)}
            className="w-7 h-7 bg-amber-600 hover:bg-amber-500 text-white rounded transition-colors flex items-center justify-center focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-amber-300"
            title="Model library (Ctrl+Shift+M or ⌘⇧M)"
            aria-label="Open model library"
          >
            <ModelLibraryIcon className="w-3.5 h-3.5" />
          </button>

          {onOpenSettings && (
            <button
              type="button"
              onClick={onOpenSettings}
              className="w-7 h-7 text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover rounded transition-colors flex items-center justify-center focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
              title="Settings (⌘,)"
              aria-label="Open settings"
            >
              <SettingsIcon className="w-3.5 h-3.5" />
            </button>
          )}
          
          {onLogout && (
            <button
              type="button"
              onClick={handleLogout}
              className="w-7 h-7 text-slack-textMuted hover:text-red-500 hover:bg-red-500/10 rounded transition-colors flex items-center justify-center focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-400"
              title="Logout"
              aria-label="Log out"
            >
              <LogoutIcon className="w-3.5 h-3.5" />
            </button>
          )}
          
          <div className="w-px h-5 bg-slack-border mx-0.5" />
          
          <div className="text-xs text-slack-textMuted">
            <span className="font-medium text-slack-text">{username}</span>
            <span className="mx-1">•</span>
            <span>{serverAddr}</span>
          </div>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Channel Sidebar */}
        {channelSidebarOpen && (
          <ChannelSidebar
            channels={channels}
            agents={agents}
            onSwitchChannel={handleSwitchChannel}
            onCreateChannel={() => setCreateChannelOpen(true)}
            onCreateDM={handleCreateDM}
            onOpenNewDM={() => setCreateNewDmOpen(true)}
            onDeleteChannel={handleDeleteChannel}
            onOpenChannelInfo={handleOpenChannelInfo}
          />
        )}

        {/* File Explorer Panel - slides in from left */}
        {fileExplorerOpen && (
          <FileExplorerPanel
            onClose={() => setFileExplorerOpen(false)}
            onFileOpen={() => setCodeEditorOpen(true)}
          />
        )}

        {/* Code Editor Panel - slides in from left */}
        {codeEditorOpen && (
          <CodeEditorPanel
            onClose={() => setCodeEditorOpen(false)}
          />
        )}

        {/* Main Chat Area - always flex-grow to fill remaining space */}
        <div 
          className="flex flex-col flex-1 min-h-0 min-w-[300px] transition-all duration-300 ease-in-out relative overflow-hidden"
        >

        {activeCollab &&
          activeCollab.channel === channel &&
          isTerminalCollaborationPhase(activeCollab.phase) && (
            <div
              className="mx-3 mt-2 px-3 py-2 rounded-md text-sm border border-emerald-700/50 bg-emerald-950/40 text-emerald-100"
              role="status"
            >
              Collaboration completed —{' '}
              {activeCollab.tasks?.filter(t => t.status === 'completed').length ?? 0}/
              {activeCollab.tasks?.length ?? 0} tasks done. Review the panel or dismiss when finished.
            </div>
          )}

        {/* Messages */}
        <MessageList key={channel} searchQuery={messageSearchQuery} />

        <div className="flex-shrink-0">
          <TypingIndicator agents={thinkingAgentsForChannel} />
        </div>

        {/* Input */}
        <RichTextInput
          onSend={handleSendMessage}
          disabled={status !== 'connected'}
          placeholder={
            status === 'connected'
              ? 'Type your message here...'
              : 'Connecting...'
          }
          agents={agents}
          ref={inputRef}
          onSlashTrigger={handleSlashTrigger}
          onDraftChange={setComposerDraft}
        />

        {workspaceContextMode === 'auto' && composerDraft.trim() && (
          <div
            className="px-3 py-1 text-xs text-slack-textMuted border-t border-slack-border"
            title={contextScopePreview.reason}
          >
            Context: Auto → <span className="text-slack-text">{contextScopePreview.scope}</span>
          </div>
        )}
        </div>

        {/* Thread Panel - slides in when thread is open */}
        {openThreadId && parentMessage && (
          <ThreadPanel
            threadId={openThreadId}
            parentMessage={parentMessage}
            onClose={closeThread}
            onSendReply={handleThreadSend}
          />
        )}

        {/* Collaboration Panel */}
        {activeCollab && (
          <CollaborationPanel
            collaboration={activeCollab}
            extendableCollaborations={extendableCollaborations}
            executingCollaboration={executingCollaborationForChannel}
            onClose={() => setActiveCollab(null)}
            onAfterCollaborationCommand={async () => {
              await loadCollaborations(channel);
            }}
          />
        )}

        {/* Task Management Panel */}
        {taskManagementOpen && (
          <TaskManagementPanel
            collaborations={trackedCollaborations}
            assistantTasks={assistantTasks}
            assistantReminders={assistantReminders}
            onClose={() => setTaskManagementOpen(false)}
            onOpenCollaboration={async (collab) => {
              if (collab.channel && collab.channel !== channel) {
                try {
                  await handleSwitchChannel(collab.channel);
                } catch (e) {
                  console.error('[TaskPanel] failed to switch to collaboration channel:', e);
                }
              }
              setActiveCollab(collab);
              setTaskManagementOpen(false);
            }}
            onAssistantTaskDone={async (taskID) => {
              const previousTasks = assistantTasks;
              const targetTask = previousTasks.find(task => task.id === taskID);
              setAssistantTasks(prev =>
                prev.map(task => (task.id === taskID ? { ...task, status: 'done' } : task))
              );
              try {
                await api.markAssistantTaskDone(taskID);
                addToast({
                  type: 'success',
                  title: 'Task marked done',
                  message: targetTask ? `"${targetTask.title}" moved to done.` : 'Assistant task moved to done.',
                });
                void loadAssistantState();
              } catch (error) {
                console.error('Failed to mark assistant task done:', error);
                setAssistantTasks(previousTasks);
                addToast({
                  type: 'error',
                  title: 'Task update failed',
                  message: error instanceof Error ? error.message : 'Failed to mark assistant task done.',
                });
              }
            }}
            onAssistantReminderDismiss={async (reminderID) => {
              const previousReminders = assistantReminders;
              const targetReminder = previousReminders.find(reminder => reminder.id === reminderID);
              setAssistantReminders(prev => prev.filter(reminder => reminder.id !== reminderID));
              try {
                await api.dismissAssistantReminder(reminderID);
                addToast({
                  type: 'success',
                  title: 'Reminder dismissed',
                  message: targetReminder ? `"${targetReminder.content}" dismissed.` : 'Assistant reminder dismissed.',
                });
                void loadAssistantState();
              } catch (error) {
                console.error('Failed to dismiss assistant reminder:', error);
                setAssistantReminders(previousReminders);
                addToast({
                  type: 'error',
                  title: 'Reminder dismiss failed',
                  message: error instanceof Error ? error.message : 'Failed to dismiss assistant reminder.',
                });
              }
            }}
            onCollaborationCommand={async (command, collaborationID, feedbackText, taskIndex) => {
              const from = { name: username || 'User', type: 'human' };
              const shortID = collaborationID.slice(0, 8);
              const collab =
                collaborationsByIDRef.current[collaborationID] ??
                trackedCollaborations.find(c => c.id === collaborationID);
              const targetChannel = collab?.channel?.trim() || channel;
              let content = '';
              if (command === 'approve') {
                content = `/resume-plan ${shortID}`;
              } else if (command === 'revise') {
                const trimmed = (feedbackText || '').trim();
                if (!trimmed) {
                  throw new Error('Revision feedback is required.');
                }
                content = `/revise-plan ${shortID} ${trimmed}`;
              } else if (command === 'complete') {
                const open = collab?.tasks?.filter(t => t.status !== 'completed') ?? [];
                content = `/complete-collab ${shortID}${open.length > 0 ? ' --force' : ''}`;
              } else if (command === 'task-done') {
                if (taskIndex == null || taskIndex < 0) {
                  throw new Error('Task index is required.');
                }
                content = `/collab-task-done ${shortID} ${taskIndex + 1}`;
              } else {
                content = `/cancel-plan ${shortID}`;
              }
              try {
                if (targetChannel !== channel) {
                  await handleSwitchChannel(targetChannel);
                }
                await api.sendMessage(targetChannel, content, from);
                await loadCollaborations(targetChannel);
                if (command === 'cancel') {
                  clearActiveCollabIf(collaborationID);
                  setTaskManagementOpen(false);
                }
              } catch (e) {
                addToast({
                  type: 'error',
                  title: 'Collaboration command failed',
                  message: e instanceof Error ? e.message : 'Request failed.',
                });
                throw e;
              }
            }}
          />
        )}

        {/* My Agents Panel - slides in from right */}
        {myAgentsPanelOpen && (
          <MyAgentsPanel
            onClose={() => setMyAgentsPanelOpen(false)}
          />
        )}

        {/* Pending Changes Panel */}
        {pendingChangesOpen && (
          <PendingChangesPanel
            onClose={() => setPendingChangesOpen(false)}
          />
        )}
      </div>

      {/* Terminal Panel - slides up from bottom */}
      <div 
        className="transition-all duration-300 ease-in-out overflow-hidden"
        style={{ height: isPanelOpen ? `${panelHeight}px` : '0px' }}
      >
        <TerminalPanel height={panelHeight} />
      </div>
      
      {/* Command Palette */}
      <CommandPalette
        commands={commandDefs}
        agents={agents}
        api={api}
        isOpen={commandPaletteOpen}
        initialFilter={commandPaletteFilter}
        onClose={() => {
          setCommandPaletteOpen(false);
          if (inputRef.current && (inputRef.current as any).clearInput) {
            (inputRef.current as any).clearInput();
          }
        }}
        onExecute={handleCommandExecute}
      />

      {/* Create Channel Modal */}
      <CreateChannelModal
        agents={agents}
        isOpen={createChannelOpen}
        onClose={() => setCreateChannelOpen(false)}
        onCreate={handleCreateChannel}
      />

      {channelInfoModal && (
        <ChannelInfoModal
          channel={channelInfoModal}
          agents={agents}
          onClose={() => setChannelInfoModal(null)}
        />
      )}

      <CreateNewDMModal
        api={api}
        username={username}
        isOpen={createNewDmOpen}
        onClose={() => setCreateNewDmOpen(false)}
        onCreated={handleNewDmCreated}
      />

      <ModelLibraryModal
        isOpen={modelLibraryOpen}
        onClose={() => setModelLibraryOpen(false)}
        serverAddr={hubHttp}
        switchAllAgentProviders={switchAllAgentProviders}
      />

      {hubAccessPending && (
        <HubDataAccessModal
          options={hubAccessPending.options}
          isLoading={hubAccessLoading}
          error={hubAccessError}
          onCancel={() => {
            setHubAccessPending(null);
            setHubAccessError(null);
          }}
          onConfirm={handleHubAccessConfirm}
        />
      )}

      {/* Toast Notifications */}
      <ToastContainer />
      </div>
    </ErrorBoundary>
  );
}

