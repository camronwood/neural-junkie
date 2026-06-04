import { useState, useRef, useCallback, useEffect, useMemo, startTransition } from 'react';
import { shallow } from 'zustand/shallow';
import { useChatStore } from '../stores/chatStore';
import { useTerminalStore, createNewTab } from '../stores/terminalStore';
import { useSettingsStore } from '../stores/settingsStore';
import { usePacksStore } from '../stores/packsStore';
import { PACK_CAP } from '../stores/packCapabilities';
import { GitModal } from './GitPanel';
import { QuickOpenModal } from './QuickOpenModal';
import { SymbolModal } from './SymbolModal';
import { ProblemsPanel } from './ProblemsPanel';
import { FastEditModal } from './FastEditModal';
import { useEditorStore } from '../stores/editorStore';
import { getLanguageFromPath } from '../utils/editorLanguage';
import { useToastStore } from '../stores/toastStore';
import { useComposerPrefillStore } from '../stores/composerPrefillStore';
import { ChatAPI } from '../api/chatAPI';
import { clearCredentials } from '../utils/secureStorage';
import {
  buildHumanOutboundMetadata,
  cycleWorkspaceContextMode,
  loadWorkspaceContextMode,
  workspaceContextModeLabel,
  WORKSPACE_CONTEXT_MODE_KEY,
} from '../utils/outboundChatMetadata';
import {
  cycleConversationModeSetting,
  formatContextIndicator,
  loadConversationModeSetting,
  resolveConversationMode,
  CONVERSATION_MODE_STORAGE_KEY,
} from '../utils/conversationMode';
import { channelNameToKind, resolveContextScope } from '../utils/inferContextScope';
import type { ConversationModeSetting, WorkspaceContextMode } from '../constants/promptMetadata';
import { METADATA_CHANNEL_HOLD } from '../types/protocol';
import { GRANTED_HUB_DATA_ACCESS_KEY, IMPLEMENTATION_FILES_CHANGED_KEY, IMPLEMENTATION_SESSION_COMPLETE_KEY, CAD_FILES_WRITTEN_KEY } from '../constants/promptMetadata';
import {
  detectHubDataAccessNeeds,
  hasGrantedHubDataAccess,
  type HubDataAccessOption,
} from '../utils/hubDataAccess';
import { HubDataAccessModal } from './HubDataAccessModal';
import { shouldSendChannelJoinMessage } from '../utils/joinMessage';
import { devLog } from '../utils/devLog';
import {
  registeredFileChangeId,
  shouldPromptFileChangeApproval,
} from '../utils/fileChangeApprovalPrompt';
import {
  fileChangeProposalPaths,
  refreshFileExplorerForPaths,
} from '../utils/refreshFileExplorer';
import { useWebSocket } from '../hooks/useWebSocket';
import { useSidebarAutoUnhide } from '../hooks/useSidebarAutoUnhide';
import { agentSidebarHideKey } from '../utils/dmChannelDisplay';
import {
  patchRevealForChannel,
  patchRevealSidebarItems,
} from '../utils/sidebarVisibility';
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
import {
  isNonTerminalCollaborationPhase,
  resolvePanelCollaboration,
} from '../utils/collaborationPanelState';
import { RunbookBuilderPanel } from './RunbookBuilderPanel';
import { CollaborationWorkspaceGate } from './CollaborationWorkspaceGate';
import { TaskManagementPanel } from './TaskManagementPanel';
import { SecondaryAnalysisPanel } from './SecondaryAnalysisPanel';
import { useSecondaryAnalysisStore } from '../stores/secondaryAnalysisStore';
import { ModelLibraryModal } from './ModelLibraryModal';
import { PhoenixBrowserModal } from './PhoenixBrowserModal';
import { LearningProposalModal } from './LearningProposalModal';
import type { LearningProposalAction } from '../api/chatAPI';
import type { LoraTrainPrefill } from './LoraTrainingPanel';
import { LeftSidebarIcon, RightSidebarIcon } from './Icons';
import { ChatToolbarActions } from './ChatToolbarActions';
import { ChatToolbarSidebar } from './ChatToolbarSidebar';
import { ChatFindBar } from './ChatFindBar';
import type {
  AssistantReminder,
  AssistantTask,
  AgentInfo,
  Channel,
  Collaboration,
  CollaborationAgent,
  CommandDefinition,
  Message,
  ThinkingAgent,
  ThinkingStatusMetadata,
} from '../types/protocol';
import { isCollaborationMessage, getCollaborationId, showThreadReplyInMainTimeline } from '../types/protocol';
import { findThreadParentMessage } from '../utils/slackThread';
import { isSlackMirrorChannelName, slackChannelDisplayName } from '../utils/slackChannelDisplay';
import { confirmStartCollaborationWhileExecuting } from '../utils/collaborationConfirm';
import { ensureCollaborationExecutionWorkspace } from '../utils/collaborationExecutionWorkspace';
import { syncCollabTurnThinking } from '../utils/collabThinking';
import { resolveTerminalCwd } from '../utils/terminalCwd';
import { runAgentTerminalCommand } from '../utils/runTerminalCommand';
import type { CommandSuggestion } from '../stores/terminalStore';
import {
  ensureRepoAgentWorkspace,
  isRepoAgentWorkspaceAction,
  parseCreateRepoAgentCommand,
} from '../utils/repoAgentWorkspace';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { getHubBaseURL } from '../config/hubUrl';
import { isIdeLayout, layoutPresetLabel, panelsForPreset } from '../utils/layoutPresets';
import { shrinkablePanelStyle } from '../utils/panelLayout';
import { useHorizontalPanelResize } from '../hooks/useHorizontalPanelResize';
import { MAX_COLLAB_AGENTS } from '../utils/collaborationLimits';
import type { LayoutPreset } from '../stores/settingsStore';
import {
  buildIdeDispatchPayload,
  buildImplementationSessionMetadata,
  ideRoutingChipLabel,
  mergeCodebaseAttachments,
} from '../utils/ideComposer';
import { hasCodeTaskSignals } from '../utils/conversationMode';

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
  const { serverAddr, channel, username, agents, channels, switchAllAgentProviders, switchAgentProvider } = useChatStore(
    (s) => ({
      serverAddr: s.serverAddr,
      channel: s.channel,
      username: s.username,
      agents: s.agents,
      channels: s.channels,
      switchAllAgentProviders: s.switchAllAgentProviders,
      switchAgentProvider: s.switchAgentProvider,
    }),
    shallow
  );

  const { openThreadId, parentMessage } = useChatStore(
    (s) => ({
      openThreadId: s.openThreadId,
      parentMessage: s.openThreadId ? findThreadParentMessage(s.messages, s.openThreadId) : null,
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

  const channelHeld = useChatStore((s) => s.channelHeld.get(s.channel) === true, shallow);

  const hasStreamingOnChannel = useChatStore(
    (s) => {
      const ch = s.channel;
      return Object.values(s.streamingMessages).some((m) => (m.channel || ch) === ch);
    },
    shallow
  );

  const showAgentStop = thinkingAgentsForChannel.length > 0 || hasStreamingOnChannel;

  const myAgentsPanelOpen = useChatStore((s) => s.myAgentsPanelOpen);
  const setMyAgentsPanelOpen = useChatStore((s) => s.setMyAgentsPanelOpen);

  const { isPanelOpen, panelHeight, addSuggestedCommand, setPanelOpen } = useTerminalStore();
  const { layoutSettings, loadLayoutSettings, updateSettings, updateLayoutSettings } =
    useSettingsStore();
  const addToast = useToastStore(s => s.addToast);

  useSidebarAutoUnhide(agents, channels);

  // State for tracking counts
  const [totalAgentsCount, setTotalAgentsCount] = useState(0);

  // State for file explorer and code editor panels
  const [fileExplorerOpen, setFileExplorerOpen] = useState(false);
  const [codeEditorOpen, setCodeEditorOpen] = useState(false);
  const [quickOpenOpen, setQuickOpenOpen] = useState(false);
  const [symbolModalOpen, setSymbolModalOpen] = useState(false);
  const [fastEditOpen, setFastEditOpen] = useState(false);
  const [problemsOpen, setProblemsOpen] = useState(false);
  const [gitModalOpen, setGitModalOpen] = useState(false);
  const [phoenixModalOpen, setPhoenixModalOpen] = useState(false);
  const layoutProfile = usePacksStore((s) => s.layoutProfile);
  const hasIdeV2 = usePacksStore((s) => s.hasCapability('ide-v2'));
  const hasIdeComposer = usePacksStore((s) => s.hasCapability('ide-v3-composer'));
  const ideLayout = layoutProfile === 'ide' && isIdeLayout(layoutSettings);
  const devPackEnabled = hasIdeV2;
  const phoenixPackInstalled = usePacksStore((s) =>
    s.packs.some(
      (p) =>
        p.installed &&
        (p.capabilities?.includes(PACK_CAP.PHOENIX_IMPORT) || (p.custom && p.id.includes('brightest-bio'))),
    ),
  );
  const chatPanelVisible = layoutSettings.chatPanelVisible !== false;
  const toolbarChipsPlacement = layoutSettings.toolbarChipsPlacement ?? 'top';
  const mainContentRef = useRef<HTMLDivElement>(null);
  const mainChatResize = useHorizontalPanelResize({
    storageKey: 'main-chat-panel-width',
    defaultWidth: 420,
    minWidth: 260,
    maxWidthRatio: 0.9,
    getMaxWidth: () => {
      const el = mainContentRef.current;
      if (!el) return window.innerWidth * 0.9;
      return Math.max(260, el.clientWidth - 320);
    },
    edge: 'left',
  });
  const fetchPacks = usePacksStore((s) => s.fetchPacks);
  const { activeWorkspaceId, workspaces: explorerWorkspaces } = useFileExplorerStore(
    (s) => ({ activeWorkspaceId: s.activeWorkspaceId, workspaces: s.workspaces }),
    shallow
  );
  const openFileInEditor = useEditorStore((s) => s.openFile);
  const revealLineInEditor = useEditorStore((s) => s.revealLine);
  const activeEditorTab = useEditorStore((s) => {
    const id = s.activeTabId;
    return id ? (s.tabs.find((t) => t.id === id) ?? null) : null;
  });
  const inputRef = useRef<HTMLTextAreaElement | null>(null);

  useEffect(() => {
    void fetchPacks();
  }, [fetchPacks]);

  const handleQuickOpenPath = useCallback(
    async (path: string) => {
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      try {
        const api = new ChatAPI(getHubBaseURL());
        const content = await api.fetchFileContent(ws.id, path);
        openFileInEditor(ws.id, path, content, getLanguageFromPath(path));
        setCodeEditorOpen(true);
        setFileExplorerOpen(true);
      } catch (e) {
        addToast({
          type: 'error',
          title: 'Open file',
          message: e instanceof Error ? e.message : String(e),
        });
      }
    },
    [activeWorkspaceId, explorerWorkspaces, openFileInEditor, addToast]
  );

  const handleImplementationSessionComplete = useCallback(
    async (metadata?: Record<string, unknown>) => {
      const raw = metadata?.[IMPLEMENTATION_FILES_CHANGED_KEY];
      const paths = Array.isArray(raw) ? raw.filter((p): p is string => typeof p === 'string' && p.trim() !== '') : [];
      if (paths.length === 0) return;
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      const chatApi = new ChatAPI(getHubBaseURL());
      for (const relPath of paths) {
        try {
          const existing = useEditorStore.getState().getTabByPath(ws.id, relPath);
          if (existing) {
            await useEditorStore.getState().refreshTabFromDisk(ws.id, relPath);
            revealLineInEditor(ws.id, relPath, 1);
          } else {
            const content = await chatApi.fetchFileContent(ws.id, relPath);
            openFileInEditor(ws.id, relPath, content, getLanguageFromPath(relPath));
          }
        } catch (e) {
          console.error('[impl-session] open changed file:', relPath, e);
        }
      }
      await refreshFileExplorerForPaths(ws.id, paths);
      setCodeEditorOpen(true);
      setFileExplorerOpen(true);
      void useFileChangeStore.getState().fetchPendingChanges(username || 'default');
    },
    [
      activeWorkspaceId,
      explorerWorkspaces,
      openFileInEditor,
      revealLineInEditor,
      username,
    ]
  );

  const handleCADFilesWritten = useCallback(
    async (metadata?: Record<string, unknown>) => {
      const raw = metadata?.[CAD_FILES_WRITTEN_KEY];
      const paths = Array.isArray(raw)
        ? raw.filter((p): p is string => typeof p === 'string' && p.trim() !== '')
        : [];
      if (paths.length === 0) return;
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      await refreshFileExplorerForPaths(ws.id, paths);
      setFileExplorerOpen(true);
    },
    [activeWorkspaceId, explorerWorkspaces]
  );

  const handleOpenAtLine = useCallback(
    async (path: string, line: number) => {
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      try {
        const api = new ChatAPI(getHubBaseURL());
        const content = await api.fetchFileContent(ws.id, path);
        openFileInEditor(ws.id, path, content, getLanguageFromPath(path));
        revealLineInEditor(ws.id, path, line);
        setCodeEditorOpen(true);
        setFileExplorerOpen(true);
      } catch (e) {
        addToast({
          type: 'error',
          title: 'Open symbol',
          message: e instanceof Error ? e.message : String(e),
        });
      }
    },
    [
      activeWorkspaceId,
      explorerWorkspaces,
      openFileInEditor,
      revealLineInEditor,
      addToast,
    ]
  );

  useEffect(() => {
    if (!devPackEnabled) return;
    const onKey = (e: KeyboardEvent) => {
      const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
      const cmd = isMac ? e.metaKey : e.ctrlKey;
      const target = e.target as HTMLElement;
      const inInput =
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.contentEditable === 'true';
      if (cmd && e.key === 'p' && !e.shiftKey && !inInput) {
        e.preventDefault();
        setQuickOpenOpen(true);
      }
      if (cmd && e.shiftKey && (e.key === 'o' || e.key === 'O') && !inInput) {
        e.preventDefault();
        setSymbolModalOpen(true);
      }
      if (cmd && (e.key === 'k' || e.key === 'K') && codeEditorOpen) {
        e.preventDefault();
        setFastEditOpen(true);
      }
      if (cmd && (e.key === 'l' || e.key === 'L') && ideLayout) {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [devPackEnabled, codeEditorOpen, ideLayout]);

  // State for pending changes panel
  const [pendingChangesOpen, setPendingChangesOpen] = useState(false);
  const [pendingChangePreviewId, setPendingChangePreviewId] = useState<string | null>(null);
  const { pendingChanges, fetchPendingChanges } = useFileChangeStore(
    (s) => ({
      pendingChanges: s.pendingChanges,
      fetchPendingChanges: s.fetchPendingChanges,
    }),
    shallow
  );

  // Sidebar visibility
  const [channelSidebarOpen, setChannelSidebarOpen] = useState<boolean>(() => {
    return localStorage.getItem('channel-sidebar-open') !== 'false';
  });
  const [toolbarSidebarOpen, setToolbarSidebarOpen] = useState<boolean>(() => {
    return localStorage.getItem('toolbar-sidebar-open') === 'true';
  });
  const [isWideViewport, setIsWideViewport] = useState(() =>
    typeof window.matchMedia === 'function'
      ? window.matchMedia('(min-width: 1024px)').matches
      : true
  );

  // State for create channel modal
  const [createChannelOpen, setCreateChannelOpen] = useState(false);
  const [createNewDmOpen, setCreateNewDmOpen] = useState(false);
  const [channelInfoModal, setChannelInfoModal] = useState<Channel | null>(null);

  // State for command palette
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [commandPaletteFilter, setCommandPaletteFilter] = useState('');
  const [commandDefs, setCommandDefs] = useState<CommandDefinition[]>([]);
  const [modelLibraryOpen, setModelLibraryOpen] = useState(false);
  const [modelLibraryInitialTab, setModelLibraryInitialTab] = useState<'ollama' | 'huggingface' | 'train' | undefined>();
  const [loraTrainPrefill, setLoraTrainPrefill] = useState<LoraTrainPrefill | null>(null);
  const [learningProposal, setLearningProposal] = useState<LearningProposalAction | null>(null);
  const [learningProposalOpen, setLearningProposalOpen] = useState(false);

  // State for active collaboration panel
  const [activeCollab, setActiveCollab] = useState<Collaboration | null>(null);
  const activeCollabRef = useRef<Collaboration | null>(null);
  const collaborationsByIDRef = useRef<Record<string, Collaboration>>({});
  const [taskManagementOpen, setTaskManagementOpen] = useState(false);
  const secondaryAnalysisOpen = useSecondaryAnalysisStore((s) => s.panelOpen);
  const setSecondaryAnalysisOpen = useSecondaryAnalysisStore((s) => s.setPanelOpen);
  const [collaborationsByID, setCollaborationsByID] = useState<Record<string, Collaboration>>({});
  const [assistantTasks, setAssistantTasks] = useState<AssistantTask[]>([]);
  const [assistantReminders, setAssistantReminders] = useState<AssistantReminder[]>([]);
  const [messageSearchQuery, setMessageSearchQuery] = useState('');
  const [chatFindOpen, setChatFindOpen] = useState(false);
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

  useEffect(() => {
    void usePacksStore.getState().fetchPacks();
  }, []);

  const [workspaceContextMode, setWorkspaceContextMode] = useState<WorkspaceContextMode>(() =>
    loadWorkspaceContextMode()
  );
  const [conversationModeSetting, setConversationModeSetting] = useState<ConversationModeSetting>(() =>
    loadConversationModeSetting()
  );
  const [composerDraft, setComposerDraft] = useState('');
  const composerPrefillPending = useComposerPrefillStore((s) => s.pendingText);
  const consumeComposerPrefill = useComposerPrefillStore((s) => s.consumePrefill);

  useEffect(() => {
    if (!composerPrefillPending) return;
    const text = consumeComposerPrefill();
    if (!text) return;
    setComposerDraft(text);
    const input = inputRef.current as (HTMLTextAreaElement & { setDraftText?: (t: string) => void }) | null;
    input?.setDraftText?.(text);
  }, [composerPrefillPending, consumeComposerPrefill]);

  const activeChannelMeta = useMemo(
    () => channels.find((c) => c.name === channel),
    [channels, channel]
  );

  const contextScopePreview = useMemo(() => {
    const activeTabPath = activeEditorTab?.path;
    const channelKind = channelNameToKind(channel, activeChannelMeta?.type);
    const scopeResult = resolveContextScope({
      message: composerDraft,
      mode: workspaceContextMode,
      channelKind,
      activeTabPath,
      ideCoding: ideLayout && hasIdeComposer,
    });
    const resolvedMode = resolveConversationMode(conversationModeSetting, composerDraft, {
      ideCoding: ideLayout && hasIdeComposer,
      channelKind,
      hasOpenTab: Boolean(activeTabPath),
    });
    const scope =
      resolvedMode === 'chat' ? ('none' as const) : scopeResult.scope;
    return { ...scopeResult, scope, resolvedMode };
  }, [
    composerDraft,
    workspaceContextMode,
    conversationModeSetting,
    channel,
    activeChannelMeta?.type,
    activeEditorTab?.path,
    ideLayout,
    hasIdeComposer,
  ]);

  const ideRoutingLabel = useMemo(() => {
    if (!ideLayout || !hasIdeComposer) return '';
    return ideRoutingChipLabel(activeEditorTab, agents);
  }, [ideLayout, hasIdeComposer, activeEditorTab, agents]);

  const contextIndicatorLabel = useMemo(
    () =>
      formatContextIndicator({
        modeSetting: conversationModeSetting,
        resolvedMode: contextScopePreview.resolvedMode,
        scope: contextScopePreview.scope,
        scopeReason: contextScopePreview.reason,
        activeTabPath: activeEditorTab?.path,
      }),
    [
      conversationModeSetting,
      contextScopePreview.resolvedMode,
      contextScopePreview.scope,
      contextScopePreview.reason,
      activeEditorTab?.path,
    ]
  );

  const [workspaceGateCollab, setWorkspaceGateCollab] = useState<Collaboration | null>(null);
  const [workspaceGateBusy, setWorkspaceGateBusy] = useState(false);
  const dismissedWorkspaceGateIdRef = useRef<string | null>(null);
  const handledRepoWorkspaceActionsRef = useRef<Set<string>>(new Set());
  const handledLearningProposalsRef = useRef<Set<string>>(new Set());
  const handledFileChangeApprovalsRef = useRef<Set<string>>(new Set());
  const handledParticipantRequestPromptsRef = useRef<Set<string>>(new Set());

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

  const activeCollabForChannel = useMemo(
    () => Object.values(collaborationsByID).find((c) => c.channel === channel),
    [collaborationsByID, channel],
  );

  // Clear stale chat width from localStorage
  useEffect(() => {
    localStorage.removeItem('main-chat-area-width');
  }, []);

  const preferSidebarChips = toolbarChipsPlacement === 'sidebar';
  const useSidebarChips = !isWideViewport || preferSidebarChips;
  const showTopToolbarChips = isWideViewport && !preferSidebarChips;

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return;
    const mq = window.matchMedia('(min-width: 1024px)');
    const onChange = () => setIsWideViewport(mq.matches);
    onChange();
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);

  useEffect(() => {
    if (!useSidebarChips) {
      setToolbarSidebarOpen(false);
      localStorage.setItem('toolbar-sidebar-open', 'false');
    }
  }, [useSidebarChips]);

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

  // ⌘F / Ctrl+F — in-chat find (Monaco keeps its own find when editor focused)
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'f') {
        const target = e.target as HTMLElement | null;
        if (target?.closest('.monaco-editor')) return;
        e.preventDefault();
        setChatFindOpen(true);
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
  
  // Load layout settings on mount
  useEffect(() => {
    loadLayoutSettings();
  }, [loadLayoutSettings]);

  // Apply layout settings when specific panel keys change (not whole object — avoids chat toggle resetting files/editor).
  useEffect(() => {
    if (layoutSettings) {
      setFileExplorerOpen(layoutSettings.filesPanelVisible);
    }
  }, [layoutSettings?.filesPanelVisible]);

  useEffect(() => {
    if (layoutSettings) {
      setCodeEditorOpen(layoutSettings.editorPanelVisible);
    }
  }, [layoutSettings?.editorPanelVisible]);

  useEffect(() => {
    if (layoutSettings) {
      setPanelOpen(layoutSettings.terminalPanelVisible);
    }
  }, [layoutSettings?.terminalPanelVisible, setPanelOpen]);

  useEffect(() => {
    if (layoutSettings) {
      setMyAgentsPanelOpen(layoutSettings.myAgentsPanelVisible);
    }
  }, [layoutSettings?.myAgentsPanelVisible, setMyAgentsPanelOpen]);

  useEffect(() => {
    if (layoutSettings) {
      setPendingChangesOpen(layoutSettings.pendingChangesPanelVisible);
    }
  }, [layoutSettings?.pendingChangesPanelVisible, setPendingChangesOpen]);

  // Load agents function
  const loadAgents = useCallback(async () => {
    try {
      const agentList = await api.fetchAgents({ includeToolCounts: true });
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

    setActiveCollab(current => (current?.id === snapshot.id ? snapshot : current));

    const ch = snapshot.channel || useChatStore.getState().channel;
    if (ch && !isTerminal) {
      syncCollabTurnThinking(snapshot, ch);
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

  const collaborationForChannel = useMemo(
    () => trackedCollaborations.find(c => c.channel === channel) ?? null,
    [trackedCollaborations, channel]
  );

  const panelCollaboration = useMemo(
    () => resolvePanelCollaboration(activeCollab, collaborationsByID),
    [activeCollab, collaborationsByID]
  );

  const isClosedCollaborationChannel = Boolean(
    collaborationForChannel && isTerminalCollaborationPhase(collaborationForChannel.phase)
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

  const revealSidebarForChannel = useCallback(
    (channelName: string) => {
      const { settings, isLoaded } = useSettingsStore.getState();
      if (!isLoaded) return;
      const patch = patchRevealForChannel(
        settings,
        channelName,
        useChatStore.getState().channels,
        useChatStore.getState().agents
      );
      if (patch) {
        void updateSettings(patch);
      }
    },
    [updateSettings]
  );

  // Handle switching channel: switch store state, then load fresh messages
  const handleSwitchChannel = useCallback(
    async (channelName: string) => {
      const prevChannel = useChatStore.getState().channel;
      if (channelName === prevChannel) return;
      revealSidebarForChannel(channelName);
      // Collaboration side panel is channel-scoped; clear when navigating.
      setActiveCollab(null);
      useChatStore.getState().switchChannel(channelName);
      localStorage.setItem('last-channel', channelName);
      if (prevChannel && prevChannel !== channelName) {
        useChatStore.getState().clearThinkingAgents(prevChannel);
      }
      try {
        const msgs = await api.fetchMessages(channelName, 50);
        useChatStore.getState().setMessages(msgs);
        useChatStore.getState().cleanupStaleThinking(channelName, msgs);
        await loadCollaborations(channelName);
        const collab = Object.values(collaborationsByIDRef.current).find(
          (c) => c.channel === channelName && !isTerminalCollaborationPhase(c.phase)
        );
        if (collab) {
          syncCollabTurnThinking(collab, channelName);
        }
        const cwd = resolveTerminalCwd({ collaboration: collab ?? null });
        useTerminalStore.getState().alignActiveTabCwd(cwd);
      } catch (error) {
        console.error('Failed to load messages for channel:', error);
      }
    },
    [api, loadCollaborations, revealSidebarForChannel]
  );

  const handleNewRunbook = useCallback(async () => {
    const pool = agents.filter((a) => a.status === 'active' || a.status === 'idle');
    if (pool.length < 1) {
      addToast({ type: 'error', title: 'No agents', message: 'Add at least one active agent before creating a runbook.' });
      return;
    }
    const currentChannel = channels.find((c) => c.name === channel);
    const channelAgentIds = new Set(
      currentChannel?.agents?.map((a) => a.id) ?? currentChannel?.members ?? []
    );
    const channelPool = pool.filter((a) => channelAgentIds.has(a.id));
    const pickFrom = channelPool.length > 0 ? channelPool : pool;
    const picked = pickFrom.slice(0, Math.min(MAX_COLLAB_AGENTS, pickFrom.length));
    try {
      const result = await api.createRunbook({
        description: 'New runbook',
        agent_ids: picked.map((a) => a.id),
        channel,
        created_by: username || 'User',
      });
      if (result.collaboration_channel && result.collaboration_channel !== channel) {
        await handleSwitchChannel(result.collaboration_channel);
      }
      setActiveCollab(result.collaboration);
      addToast({
        type: 'success',
        title: 'Runbook created',
        message: 'Define tasks in the runbook builder panel.',
      });
      void loadCollaborations(channel);
    } catch (e) {
      addToast({
        type: 'error',
        title: 'Runbook failed',
        message: e instanceof Error ? e.message : String(e),
      });
    }
  }, [agents, api, channel, channels, username, addToast, loadCollaborations, handleSwitchChannel]);

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
      const { settings, isLoaded } = useSettingsStore.getState();
      const agent = useChatStore.getState().agents.find((a) => a.id === agentId);
      if (isLoaded) {
        const patch = patchRevealSidebarItems(settings, {
          agentIds: [agentId],
          agentSidebarKeys: agent ? [agentSidebarHideKey(agent)] : undefined,
          dmChannelNames: [ch.name],
        });
        if (patch) {
          void updateSettings(patch);
        }
      }
      await loadChannels();
      await handleSwitchChannel(ch.name);
    } catch (error) {
      console.error('Failed to create DM channel:', error);
    }
  }, [api, username, loadChannels, handleSwitchChannel, updateSettings]);

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
        const { settings, isLoaded } = useSettingsStore.getState();
        if (isLoaded) {
          const patch = patchRevealForChannel(
            settings,
            ch.name,
            merged,
            useChatStore.getState().agents
          );
          if (patch) {
            void updateSettings(patch);
          }
        }
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
    [addToast, api, loadAgents, handleSwitchChannel, updateSettings]
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

  const refreshExplorerForFileChange = useCallback(
    (message: Message) => {
      const paths = fileChangeProposalPaths(message);
      if (paths.length === 0) return;
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      void refreshFileExplorerForPaths(ws.id, paths);
      for (const relPath of paths) {
        void useEditorStore.getState().refreshTabFromDisk(ws.id, relPath);
      }
    },
    [activeWorkspaceId, explorerWorkspaces]
  );

  const promptFileChangeApproval = useCallback(
    async (message: Message) => {
      if (
        !shouldPromptFileChangeApproval(
          message,
          useChatStore.getState().channel,
          collaborationsByIDRef.current,
        )
      ) {
        return;
      }
      const changeId = registeredFileChangeId(message);
      if (!changeId || handledFileChangeApprovalsRef.current.has(message.id)) {
        return;
      }
      handledFileChangeApprovalsRef.current.add(message.id);
      try {
        await fetchPendingChanges(username || 'default');
      } catch (error) {
        console.error('[ChatWindow] fetch pending file changes failed:', error);
      }
      setPendingChangePreviewId(changeId);
      setPendingChangesOpen(true);
      addToast({
        type: 'info',
        title: 'File change needs approval',
        message: `${message.from.name} proposed a file change. Review and approve to apply it.`,
      });
    },
    [addToast, fetchPendingChanges, username],
  );

  // WebSocket connection
  const { status } = useWebSocket({
    url: wsURL,
    onMessage: async (message: Message) => {
      try {
        const st = useChatStore.getState();
        const activeChannel = st.channel;

      // Handle all agent_status messages - never add them to chat
      if (message.type === 'agent_status') {
        const msgChannel = message.channel || activeChannel;
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
          if (thinkingStatus === 'started') {
            st.addThinkingAgent(msgChannel, message.from.id, message.from.name, message.from.type);
            if (
              msgChannel !== activeChannel &&
              msgChannel.startsWith('collab-')
            ) {
              st.addThinkingAgent(activeChannel, message.from.id, message.from.name, message.from.type);
            }
          } else if (
            thinkingStatus === 'completed' ||
            thinkingStatus === 'error' ||
            thinkingStatus === 'aborted'
          ) {
            st.removeThinkingAgent(msgChannel, message.from.id);
            if (msgChannel !== activeChannel) {
              st.removeThinkingAgent(activeChannel, message.from.id);
            }
          }
        }

        if (message.metadata && METADATA_CHANNEL_HOLD in message.metadata) {
          const held = message.metadata[METADATA_CHANNEL_HOLD] === true;
          st.setChannelHold(msgChannel, held);
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
      const streamChannel = message.channel || activeChannel;
      const streamOnMainTimeline =
        (!message.channel || message.channel === activeChannel) &&
        (!message.is_thread_reply || showThreadReplyInMainTimeline(streamChannel));
      if (message.type === 'stream_delta') {
        if (streamOnMainTimeline) {
          st.appendStreamDelta(message);
        }
        st.removeThinkingAgent(message.channel || activeChannel, message.from.id);
        return;
      }
      if (message.type === 'stream_end') {
        if (streamOnMainTimeline) {
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

      if (message.metadata?.event === 'collab-participant-add-request') {
        const collabID = getCollaborationId(message) || collabData?.id || '';
        const agentID = typeof message.metadata.requested_agent_id === 'string'
          ? message.metadata.requested_agent_id
          : '';
        if (collabID && agentID) {
          const key = `${collabID}:${agentID}:${message.id}`;
          if (!handledParticipantRequestPromptsRef.current.has(key)) {
            handledParticipantRequestPromptsRef.current.add(key);
            const agentName = typeof message.metadata.requested_agent_name === 'string'
              ? message.metadata.requested_agent_name
              : 'the agent';
            const requestedBy = typeof message.metadata.requested_by_name === 'string'
              ? message.metadata.requested_by_name
              : 'An agent';
            void (async () => {
              const approved = window.confirm(
                `${requestedBy} wants to add ${agentName} to this collaboration. Allow?`
              );
              try {
                const updated = approved
                  ? await api.approveCollabParticipantRequest(collabID, agentID)
                  : await api.denyCollabParticipantRequest(collabID, agentID);
                mergeCollaborationSnapshot(updated);
                if (activeCollabRef.current?.id === updated.id) {
                  setActiveCollab(updated);
                }
                addToast({
                  type: approved ? 'success' : 'info',
                  title: approved ? 'Agent added' : 'Agent add denied',
                  message: approved
                    ? `@${agentName} joined "${updated.title}".`
                    : `@${agentName} was not added to "${updated.title}".`,
                });
              } catch (error) {
                addToast({
                  type: 'error',
                  title: 'Participant request failed',
                  message: error instanceof Error ? error.message : 'Could not update collaboration participants.',
                });
              }
            })();
          }
        }
      }

      // Thread replies: NJ channels use ThreadPanel only; Slack mirrors also show in main timeline.
      if (message.is_thread_reply && message.thread_id) {
        const threadChannel = message.channel || activeChannel;
        void api
          .fetchThreadMetadata(message.thread_id)
          .then(metadata => useChatStore.getState().updateThreadMetadata(message.thread_id!, metadata))
          .catch(error => console.error('Failed to fetch thread metadata:', error));
        if (showThreadReplyInMainTimeline(threadChannel)) {
          if (threadChannel === activeChannel) {
            st.addMessage(message);
          } else if (message.channel) {
            st.addMessageToCache(message.channel, message);
            st.markChannelUnread(message.channel);
          }
        }
        return;
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
        if (message.type === 'file_change') {
          refreshExplorerForFileChange(message);
          await promptFileChangeApproval(message);
        }
      } else {
        // Message belongs to the active channel (never wrap addMessage in startTransition —
        // high-frequency agent_status updates can starve transitions and leave the chat empty).
        st.addMessage(message);

        if (message.metadata?.[IMPLEMENTATION_SESSION_COMPLETE_KEY] === true) {
          void handleImplementationSessionComplete(
            message.metadata as Record<string, unknown> | undefined
          );
        }

        if (message.metadata?.[CAD_FILES_WRITTEN_KEY]) {
          void handleCADFilesWritten(message.metadata as Record<string, unknown> | undefined);
        }

        if (message.metadata?.suggested_commands) {
          const suggestions = message.metadata.suggested_commands as CommandSuggestion[];
          const msgCh = message.channel || activeChannel;
          const collabCtx = Object.values(collaborationsByIDRef.current).find(
            (c) => c.channel === msgCh
          );
          for (const suggestion of suggestions) {
            const enriched = {
              ...suggestion,
              cwd:
                suggestion.cwd?.trim() ||
                resolveTerminalCwd({ collaboration: collabCtx ?? null }),
            };
            if (enriched.is_safe) {
              useTerminalStore.getState().setPanelOpen(true);
              void runAgentTerminalCommand(enriched, {
                collaboration: collabCtx ?? null,
                channel: msgCh,
                api,
              });
            } else {
              addSuggestedCommand(enriched);
              useTerminalStore.getState().setPanelOpen(true);
            }
          }
        }

        if (message.metadata?.event === 'agent-open-terminal') {
          const agentName = message.metadata.agent_name as string || 'Agent';
          const msgCh = message.channel || activeChannel;
          const collabCtx = Object.values(collaborationsByIDRef.current).find(
            (c) => c.channel === msgCh
          );
          const cwd =
            (message.metadata.cwd as string | undefined)?.trim() ||
            resolveTerminalCwd({ collaboration: collabCtx ?? null });
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

        if (
          clientAction &&
          typeof clientAction === 'object' &&
          clientAction.type === 'learning_proposal' &&
          !handledLearningProposalsRef.current.has(message.id)
        ) {
          handledLearningProposalsRef.current.add(message.id);
          setLearningProposal(clientAction as LearningProposalAction);
          setLearningProposalOpen(true);
        }

        if (message.type === 'file_change') {
          refreshExplorerForFileChange(message);
          await promptFileChangeApproval(message);
        }
      }
      
      // Clear thinking indicator when agent sends actual message
      if (
        message.type === 'chat' ||
        message.type === 'answer' ||
        message.type === 'collaboration_discussion'
      ) {
        const ch = message.channel || activeChannel;
        st.removeThinkingAgent(ch, message.from.id);
        if (ch !== activeChannel) {
          st.removeThinkingAgent(activeChannel, message.from.id);
        }
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
      devLog('Connected to chat');
      useChatStore.getState().setConnectionStatus('connected');
      loadInitialData();
    },
    onDisconnect: () => {
      devLog('Disconnected from chat');
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
        conversationMode: conversationModeSetting,
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
    [api, channel, username, workspaceContextMode, conversationModeSetting, activeChannelMeta?.type]
  );

  const handleChannelInterject = useCallback(async () => {
    try {
      await api.channelInterject(channel, username);
      const st = useChatStore.getState();
      st.setChannelHold(channel, true);
      st.clearThinkingAgents(channel);
      st.stopAllStreamsForChannel(channel);
    } catch (error) {
      console.error('Channel interject failed:', error);
      addToast({
        type: 'error',
        title: 'Stop failed',
        message: error instanceof Error ? error.message : 'Could not stop agents.',
      });
    }
  }, [api, channel, username, addToast]);

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape' || !showAgentStop) return;
      const target = e.target as HTMLElement | null;
      if (target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA' || target?.isContentEditable) {
        return;
      }
      e.preventDefault();
      void handleChannelInterject();
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [showAgentStop, handleChannelInterject]);

  const dispatchMessage = useCallback(
    async (content: string, metadata?: Record<string, unknown>) => {
      useChatStore.getState().setChannelHold(channel, false);

      let sendContent = content;
      let composerMeta = metadata ?? {};
      if (ideLayout && devPackEnabled) {
        const ws =
          explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
          explorerWorkspaces[0];
        const idePayload = buildIdeDispatchPayload({
          content,
          agents,
          activeTab: activeEditorTab,
          editorAgentMode: layoutSettings.editorAgentMode ?? 'agent',
          editorAgentTrust: layoutSettings.editorAgentTrust ?? 'interactive',
          composerMetadata: composerMeta,
        });
        sendContent = idePayload.content;
        composerMeta = await mergeCodebaseAttachments(
          api,
          sendContent,
          ws?.path,
          idePayload.metadata
        );
      } else if (
        devPackEnabled &&
        (layoutSettings.editorAgentMode ?? 'agent') === 'agent' &&
        hasCodeTaskSignals(content)
      ) {
        const ws =
          explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
          explorerWorkspaces[0];
        composerMeta = buildImplementationSessionMetadata({
          content,
          agents,
          activeTab: activeEditorTab,
          editorAgentMode: layoutSettings.editorAgentMode ?? 'agent',
          editorAgentTrust: layoutSettings.editorAgentTrust ?? 'interactive',
          composerMetadata: composerMeta,
        });
        if (ws?.path) {
          composerMeta = await mergeCodebaseAttachments(api, content, ws.path, composerMeta);
        }
      }

      const mergedMetadata = buildHumanOutboundMetadata({
        contextMode: workspaceContextMode,
        conversationMode: conversationModeSetting,
        message: sendContent,
        channel,
        channelType: activeChannelMeta?.type,
        composerMetadata: composerMeta,
        ideCoding: ideLayout && hasIdeComposer,
      });

      useChatStore.getState().setIsTyping(true);
      try {
        const trimmed = sendContent.trimStart();
        if (trimmed.startsWith('/collaborate')) {
          if (!confirmStartCollaborationWhileExecuting(executingCollaborationForChannel)) {
            return;
          }
        }
        const sendResult = await api.sendMessage(
          channel,
          sendContent,
          { name: username, type: 'human' },
          'question',
          mergedMetadata
        );
        let timelineChannel = channel;
        if (sendResult.collaboration_channel) {
          await loadChannels();
          await handleSwitchChannel(sendResult.collaboration_channel);
          timelineChannel = sendResult.collaboration_channel;
          const collab = Object.values(collaborationsByIDRef.current).find(
            (c) => c.channel === sendResult.collaboration_channel
          );
          if (collab) {
            syncCollabTurnThinking(collab, sendResult.collaboration_channel);
          }
        }
        if (sendContent.trimStart().startsWith('/')) {
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
      conversationModeSetting,
      activeChannelMeta?.type,
      loadChannels,
      handleSwitchChannel,
      loadCollaborations,
      executingCollaborationForChannel,
      addToast,
      ideLayout,
      devPackEnabled,
      agents,
      activeEditorTab,
      layoutSettings.editorAgentMode,
      layoutSettings.editorAgentTrust,
      explorerWorkspaces,
      activeWorkspaceId,
    ]
  );

  const handleTrainLoRAForAgent = useCallback(
    async (agentId: string) => {
      try {
        const ctx = await api.fetchLoraExpertContext(agentId);
        setLoraTrainPrefill({
          source: ctx.source,
          sourceId: ctx.source_id,
          agentName: ctx.agent_name,
          baseTag: ctx.suggested_base_ollama_tag,
          ollamaTag: ctx.suggested_ollama_tag,
          expertName: ctx.agent_name,
          agentId: ctx.agent_id,
          previewRows: ctx.preview_rows,
          ready: ctx.ready,
        });
        setModelLibraryInitialTab('train');
        setModelLibraryOpen(true);
      } catch (e) {
        addToast({
          type: 'error',
          title: 'Train LoRA',
          message: e instanceof Error ? e.message : 'Failed to load expert training context',
        });
      }
    },
    [api, addToast],
  );

  const handleSendMessage = async (content: string, metadata?: Record<string, unknown>) => {
    if (content.trim() === '/nj-open-model-library') {
      setModelLibraryOpen(true);
      return;
    }

    const trimmed = content.trimStart();
    if (
      isClosedCollaborationChannel &&
      trimmed.length > 0 &&
      trimmed[0] !== '/'
    ) {
      addToast({
        type: 'info',
        title: 'Collaboration closed',
        message:
          collaborationForChannel?.phase === 'cancelled'
            ? 'This session was cancelled. Chat is read-only; use /commands or start a new /collaborate or /runbook.'
            : 'This session is complete. Chat is read-only; use /commands or start a new /collaborate or /runbook.',
      });
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
      const trimmed = content.trimStart();
      if (isClosedCollaborationChannel && trimmed.length > 0 && trimmed[0] !== '/') {
        addToast({
          type: 'info',
          title: 'Collaboration closed',
          message: 'Threads are read-only on closed collaboration channels.',
        });
        return;
      }
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
    [openThreadId, dispatchThreadReply, isClosedCollaborationChannel, addToast]
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
  const ensureCommandDefs = useCallback(async (forceRefresh: boolean = false) => {
    if (!forceRefresh && commandDefs.length > 0) return;
    try {
      const defs = await api.fetchCommands(forceRefresh);
      setCommandDefs(withClientPaletteCommands(defs));
    } catch (err) {
      console.error('Failed to load command definitions:', err);
      setCommandDefs(withClientPaletteCommands([]));
    }
  }, [api, commandDefs.length]);

  // Handle command executed from command palette
  const handleCommandExecute = async (
    commandString: string,
    metadata?: Record<string, unknown>
  ) => {
    if (inputRef.current && (inputRef.current as any).clearInput) {
      (inputRef.current as any).clearInput();
    }
    const trimmed = commandString.trim();
    if (trimmed === '/nj-open-model-library') {
      setModelLibraryOpen(true);
      return;
    }
    const repoAgentCmd = parseCreateRepoAgentCommand(trimmed);
    await handleSendMessage(commandString, metadata);
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

  // Open command palette from toolbar button
  const openCommandPalette = useCallback(async () => {
    await ensureCommandDefs(true);
    void loadCollaborations(channel);
    void api
      .fetchAssistantState(channel)
      .then((state) => {
        setAssistantTasks(state.tasks || []);
        setAssistantReminders(state.reminders || []);
      })
      .catch((error) => console.error('Failed to load assistant state:', error));
    void fetchPendingChanges(username || 'default').catch((error) =>
      console.error('Failed to load pending file changes:', error)
    );
    setCommandPaletteFilter('');
    setCommandPaletteOpen(true);
  }, [api, channel, ensureCommandDefs, fetchPendingChanges, loadCollaborations, username]);

  useEffect(() => {
    const handleCommandPaletteShortcut = (e: KeyboardEvent) => {
      const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
      const cmd = isMac ? e.metaKey : e.ctrlKey;
      if (!cmd || !e.shiftKey || e.key.toLowerCase() !== 'p') return;
      e.preventDefault();
      e.stopPropagation();
      void openCommandPalette();
    };

    window.addEventListener('keydown', handleCommandPaletteShortcut, true);
    return () => window.removeEventListener('keydown', handleCommandPaletteShortcut, true);
  }, [openCommandPalette]);

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

  // Hub phase changes are not always re-broadcast on every message; poll while panel is open.
  useEffect(() => {
    if (!activeCollab?.id) return;
    const targetChannel = activeCollab.channel?.trim() || channel;
    const tick = () => {
      const latest = collaborationsByIDRef.current[activeCollab.id];
      if (!latest || !isNonTerminalCollaborationPhase(latest.phase)) {
        return;
      }
      void loadCollaborations(targetChannel);
    };
    tick();
    const id = window.setInterval(tick, 10_000);
    return () => window.clearInterval(id);
  }, [activeCollab?.id, activeCollab?.channel, channel, loadCollaborations]);

  const toggleToolbarSidebar = useCallback(() => {
    setToolbarSidebarOpen((prev) => {
      const next = !prev;
      localStorage.setItem('toolbar-sidebar-open', String(next));
      return next;
    });
  }, []);

  const toggleChannelSidebar = useCallback(() => {
    setChannelSidebarOpen((prev) => {
      const next = !prev;
      localStorage.setItem('channel-sidebar-open', String(next));
      return next;
    });
  }, []);

  const toolbarActionsProps = useMemo(
    () => ({
      onOpenCommandPalette: openCommandPalette,
      chatPanelVisible,
      onToggleChatPanel: () => void updateLayoutSettings({ chatPanelVisible: !chatPanelVisible }),
      conversationModeSetting,
      onCycleConversationMode: () => {
        const next = cycleConversationModeSetting(conversationModeSetting);
        setConversationModeSetting(next);
        localStorage.setItem(CONVERSATION_MODE_STORAGE_KEY, next);
      },
      workspaceContextMode,
      onCycleWorkspaceContext: () => {
        const next = cycleWorkspaceContextMode(workspaceContextMode);
        setWorkspaceContextMode(next);
        localStorage.setItem(WORKSPACE_CONTEXT_MODE_KEY, next);
      },
      workspaceContextButtonTitle: `Workspace context: ${workspaceContextModeLabel(workspaceContextMode)} (click to cycle). Next send: ${contextScopePreview.scope}`,
      onOpenPendingChanges: () => setPendingChangesOpen(true),
      onOpenFileExplorer: () => {
        setFileExplorerOpen(true);
        void updateLayoutSettings({ filesPanelVisible: true });
      },
      onOpenCodeEditor: () => {
        setCodeEditorOpen(true);
        void updateLayoutSettings({ editorPanelVisible: true });
      },
      phoenixPackInstalled,
      onOpenPhoenix: phoenixPackInstalled ? () => setPhoenixModalOpen(true) : undefined,
      taskManagementOpen,
      onToggleTaskManagement: () => setTaskManagementOpen((o) => !o),
      onNewRunbook: () => void handleNewRunbook(),
      onOpenMyAgents: () => setMyAgentsPanelOpen(true),
      totalAgentsCount,
      devPackEnabled,
      onOpenProblems: () => setProblemsOpen(true),
      gitModalOpen,
      onToggleGitModal: () => setGitModalOpen((open) => !open),
      ideLayout,
      onToggleIdeLayout: () => {
        const next: LayoutPreset = ideLayout ? 'team' : 'ide';
        void updateLayoutSettings(panelsForPreset(next));
      },
      ideLayoutButtonTitle: `Layout: ${layoutPresetLabel(ideLayout ? 'ide' : 'team')} (click to switch)`,
      onOpenModelLibrary: () => setModelLibraryOpen(true),
      onOpenSettings,
      onLogout: onLogout ? handleLogout : undefined,
      username,
      serverAddr,
    }),
    [
      openCommandPalette,
      chatPanelVisible,
      updateLayoutSettings,
      conversationModeSetting,
      workspaceContextMode,
      contextScopePreview.scope,
      taskManagementOpen,
      handleNewRunbook,
      totalAgentsCount,
      devPackEnabled,
      phoenixPackInstalled,
      gitModalOpen,
      ideLayout,
      onOpenSettings,
      onLogout,
      handleLogout,
      username,
      serverAddr,
    ]
  );

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
                  {isDM
                    ? `@ ${ch?.agents?.[0]?.name ?? channel}`
                    : ch && isSlackMirrorChannelName(ch.name)
                      ? slackChannelDisplayName(ch)
                      : `# ${channel}`}
                </h1>
                {ch && isSlackMirrorChannelName(ch.name) && (
                  <span
                    className="text-xs text-slack-textMuted hidden sm:inline truncate max-w-[200px] font-mono"
                    title="Hub channel id"
                  >
                    {ch.name}
                  </span>
                )}
                {ch?.description && !isSlackMirrorChannelName(ch.name) && (
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
        </div>
        
        <div className="flex items-center gap-1.5 shrink-0" aria-label="Sidebar toggles">
          <button
            type="button"
            onClick={toggleChannelSidebar}
            className={`w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 ${
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
          {useSidebarChips && (
            <button
              type="button"
              onClick={toggleToolbarSidebar}
              className={`w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 ${
                toolbarSidebarOpen
                  ? 'bg-slack-accent text-white'
                  : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text hover:bg-slack-border'
              }`}
              title={toolbarSidebarOpen ? 'Close toolbar panel' : 'Open toolbar panel'}
              aria-label={toolbarSidebarOpen ? 'Close toolbar panel' : 'Open toolbar panel'}
              aria-pressed={toolbarSidebarOpen}
            >
              <RightSidebarIcon className="w-3.5 h-3.5" />
            </button>
          )}
          {showTopToolbarChips && (
            <>
              <div className="w-px h-5 bg-slack-border mx-0.5 shrink-0" />
              <ChatToolbarActions layout="horizontal" {...toolbarActionsProps} />
            </>
          )}
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex flex-1 min-w-0 overflow-hidden" data-testid="chat-main-content-row">
        <div ref={mainContentRef} className="flex flex-1 min-w-0 overflow-hidden" data-testid="chat-main-inner-column">
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

        {/* File Explorer */}
        {fileExplorerOpen && (
          <FileExplorerPanel
            variant={ideLayout ? 'embedded' : 'overlay'}
            onClose={() => {
              setFileExplorerOpen(false);
              if (ideLayout) {
                void updateLayoutSettings({ filesPanelVisible: false });
              }
            }}
            onFileOpen={() => setCodeEditorOpen(true)}
          />
        )}

        {/* Code Editor */}
        {codeEditorOpen && (
          <CodeEditorPanel
            variant={ideLayout ? 'embedded' : 'overlay'}
            onClose={() => {
              setCodeEditorOpen(false);
              if (ideLayout) {
                void updateLayoutSettings({ editorPanelVisible: false });
              }
            }}
          />
        )}

        {/* Keep chat pinned to the right when the editor is hidden (editor flex-1 normally fills this gap). */}
        {ideLayout && !codeEditorOpen && chatPanelVisible && (
          <div className="flex-1 min-w-0" aria-hidden="true" />
        )}

        {/* Main Chat Area */}
        {chatPanelVisible && (
        <div
          className={
            ideLayout
              ? 'flex flex-col h-full min-h-0 relative border-l border-slack-border'
              : 'flex flex-col flex-1 min-h-0 min-w-[220px] sm:min-w-[260px] transition-all duration-300 ease-in-out relative overflow-hidden'
          }
          style={ideLayout ? shrinkablePanelStyle(mainChatResize.width, 220) : undefined}
        >
        {ideLayout && (
          <div
            className="absolute left-0 top-0 bottom-0 cursor-col-resize z-[100] group"
            onMouseDown={mainChatResize.onResizeStart}
            aria-label="Resize chat panel"
            style={{
              width: '6px',
              marginLeft: '-3px',
              pointerEvents: 'auto',
            }}
          >
            <div className="absolute inset-0 bg-transparent group-hover:bg-blue-500/30 transition-colors" />
            <div className="absolute left-1/2 top-1/2 -translate-y-1/2 -translate-x-1/2 w-1 h-8 bg-gray-400 group-hover:bg-blue-500 rounded-full opacity-0 group-hover:opacity-100 transition-opacity" />
          </div>
        )}

        <div className="flex flex-col flex-1 min-h-0 min-w-0 overflow-hidden">

        {isClosedCollaborationChannel && collaborationForChannel && (
            <div
              className={`mx-3 mt-2 px-3 py-2 rounded-md text-sm border ${
                collaborationForChannel.phase === 'cancelled'
                  ? 'border-red-700/50 bg-red-950/40 text-red-100'
                  : 'border-emerald-700/50 bg-emerald-950/40 text-emerald-100'
              }`}
              role="status"
            >
              {collaborationForChannel.phase === 'cancelled' ? (
                <>Collaboration cancelled — this channel is read-only.</>
              ) : (
                <>
                  Collaboration complete —{' '}
                  {collaborationForChannel.tasks?.filter(t => t.status === 'completed').length ?? 0}/
                  {collaborationForChannel.tasks?.length ?? 0} tasks done.
                  {collaborationForChannel.session_recap?.trim() ? (
                    <>
                      {' '}
                      {collaborationForChannel.session_recap.trim().split('\n')[0].slice(0, 120)}
                      {collaborationForChannel.session_recap.trim().length > 120 ? '…' : ''}
                    </>
                  ) : null}{' '}
                  This channel is read-only.
                </>
              )}
            </div>
          )}

        {/* Messages */}
        {chatFindOpen && (
          <ChatFindBar
            query={messageSearchQuery}
            onQueryChange={setMessageSearchQuery}
            onClose={() => {
              setChatFindOpen(false);
              setMessageSearchQuery('');
            }}
          />
        )}
        <MessageList key={channel} searchQuery={messageSearchQuery} />

        <div className="flex-shrink-0">
          <TypingIndicator
            agents={thinkingAgentsForChannel}
            showStop={showAgentStop}
            onStop={() => void handleChannelInterject()}
          />
        </div>

        {channelHeld && (
          <div
            className="mx-3 mb-1 px-3 py-2 rounded-md text-sm border border-amber-700/50 bg-amber-950/40 text-amber-100"
            role="status"
          >
            Agents paused — send a message to continue.
          </div>
        )}

        {ideLayout && devPackEnabled && (
          <div className="flex items-center gap-2 px-3 py-1.5 border-t border-slack-border bg-slack-bg/80 text-xs">
            <span className="text-slack-textMuted">IDE</span>
            <div className="inline-flex rounded border border-slack-border overflow-hidden">
              {(['ask', 'agent'] as const).map((m) => (
                <button
                  key={m}
                  type="button"
                  onClick={() => void updateLayoutSettings({ editorAgentMode: m })}
                  className={`px-2.5 py-0.5 capitalize ${
                    (layoutSettings.editorAgentMode ?? 'agent') === m
                      ? 'bg-teal-600 text-white'
                      : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                  }`}
                  title={m === 'ask' ? 'Read-only — no file edits' : 'May propose file changes'}
                >
                  {m}
                </button>
              ))}
            </div>
            <span className="text-slack-textMuted truncate" title="Routes to specialist by open file">
              @codebase · ⌘L focus · open file + selection in context
            </span>
          </div>
        )}

        {/* Input */}
        <RichTextInput
          onSend={handleSendMessage}
          disabled={status !== 'connected' || isClosedCollaborationChannel}
          placeholder={
            isClosedCollaborationChannel
              ? 'Collaboration closed — read-only (slash commands still work)'
              : status === 'connected'
                ? activeChannelMeta?.type === 'dm'
                  ? 'Message this agent directly — no @mention needed…'
                  : ideLayout && devPackEnabled
                    ? 'Ask about the project — routes by open file; @mention to pick an agent…'
                    : 'Type your message here...'
                : 'Connecting...'
          }
          agents={agents}
          ref={inputRef}
          onDraftChange={setComposerDraft}
        />

        {(workspaceContextMode === 'auto' || conversationModeSetting === 'auto') && composerDraft.trim() && (
          <div
            className="px-3 py-1 text-xs text-slack-textMuted border-t border-slack-border"
            title={contextScopePreview.reason}
          >
            Context: <span className="text-slack-text">{contextIndicatorLabel}</span>
            {ideRoutingLabel ? (
              <span className="ml-2 text-slack-accent">{ideRoutingLabel}</span>
            ) : null}
          </div>
        )}
        </div>
        </div>
        )}

        {/* Thread Panel - slides in when thread is open */}
        {openThreadId && parentMessage && (
          <ThreadPanel
            threadId={openThreadId}
            parentMessage={parentMessage}
            onClose={closeThread}
            onSendReply={handleThreadSend}
          />
        )}

        {/* Collaboration / Runbook Panel */}
        {panelCollaboration && showRunbookBuilderForCollab(panelCollaboration) ? (
          <RunbookBuilderPanel
            collaboration={panelCollaboration}
            hubAgents={agentsToCollaborationAgents(agents)}
            onClose={() => setActiveCollab(null)}
            onSaved={(snap) => {
              setActiveCollab(snap);
              void loadCollaborations(channel);
            }}
            onStarted={(snap) => {
              setActiveCollab(snap);
              void loadCollaborations(channel);
            }}
          />
        ) : panelCollaboration ? (
          <CollaborationPanel
            collaboration={panelCollaboration}
            extendableCollaborations={extendableCollaborations}
            executingCollaboration={executingCollaborationForChannel}
            onClose={() => setActiveCollab(null)}
            onAfterCollaborationCommand={async () => {
              await loadCollaborations(panelCollaboration.channel || channel);
            }}
          />
        ) : null}

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

        {secondaryAnalysisOpen && (
          <SecondaryAnalysisPanel onClose={() => setSecondaryAnalysisOpen(false)} />
        )}

        {/* My Agents Panel - slides in from right */}
        {myAgentsPanelOpen && (
          <MyAgentsPanel
            onClose={() => setMyAgentsPanelOpen(false)}
            onTrainLoRA={handleTrainLoRAForAgent}
          />
        )}

        {/* Pending Changes Panel */}
        {pendingChangesOpen && (
          <PendingChangesPanel
            initialChangeId={pendingChangePreviewId}
            onClose={() => {
              setPendingChangesOpen(false);
              setPendingChangePreviewId(null);
            }}
          />
        )}

        </div>

        {useSidebarChips && (
          <ChatToolbarSidebar open={toolbarSidebarOpen} className="self-stretch h-full">
            <ChatToolbarActions layout="vertical" {...toolbarActionsProps} />
          </ChatToolbarSidebar>
        )}
      </div>

      {/* Terminal Panel - slides up from bottom */}
      <div 
        className="transition-all duration-300 ease-in-out overflow-hidden"
        style={{ height: isPanelOpen ? `${panelHeight}px` : '0px' }}
      >
        <TerminalPanel
          height={panelHeight}
          channel={channel}
          api={api}
          collaboration={collaborationForChannel}
        />
      </div>
      
      <GitModal isOpen={gitModalOpen && devPackEnabled} onClose={() => setGitModalOpen(false)} />

      <QuickOpenModal
        isOpen={quickOpenOpen && devPackEnabled}
        workspaceId={
          explorerWorkspaces.find((w) => w.id === activeWorkspaceId)?.id ??
          explorerWorkspaces[0]?.id
        }
        onClose={() => setQuickOpenOpen(false)}
        onOpenPath={handleQuickOpenPath}
      />

      <SymbolModal
        isOpen={symbolModalOpen && devPackEnabled}
        workspaceId={
          explorerWorkspaces.find((w) => w.id === activeWorkspaceId)?.id ??
          explorerWorkspaces[0]?.id
        }
        onClose={() => setSymbolModalOpen(false)}
        onOpenSymbol={handleOpenAtLine}
      />

      <ProblemsPanel
        isOpen={problemsOpen && devPackEnabled}
        onClose={() => setProblemsOpen(false)}
        onOpenAt={handleOpenAtLine}
      />

      <FastEditModal
        isOpen={fastEditOpen && devPackEnabled}
        workspaceId={
          explorerWorkspaces.find((w) => w.id === activeWorkspaceId)?.id ??
          explorerWorkspaces[0]?.id
        }
        onClose={() => setFastEditOpen(false)}
      />

      {/* Command Palette */}
      <CommandPalette
        commands={commandDefs}
        agents={agents}
        channels={channels}
        collaborations={trackedCollaborations}
        assistantTasks={assistantTasks}
        pendingChanges={pendingChanges}
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
          onClearHistory={async (name) => {
            await api.clearChannelHistory(name);
            const msgs = await api.fetchMessages(name, 50);
            useChatStore.getState().setMessages(msgs);
            addToast({ type: 'success', title: 'Channel history cleared' });
          }}
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
        onClose={() => {
          setModelLibraryOpen(false);
          setModelLibraryInitialTab(undefined);
          setLoraTrainPrefill(null);
        }}
        serverAddr={hubHttp}
        switchAllAgentProviders={switchAllAgentProviders}
        switchAgentProvider={switchAgentProvider}
        runtimeAgents={agents.map((a) => ({ id: a.id, name: a.name, type: a.type }))}
        defaultChannel={channel}
        initialTab={modelLibraryInitialTab}
        loraTrainPrefill={loraTrainPrefill}
      />

      <PhoenixBrowserModal isOpen={phoenixModalOpen} onClose={() => setPhoenixModalOpen(false)} />

      <LearningProposalModal
        isOpen={learningProposalOpen}
        proposal={learningProposal}
        serverAddr={hubHttp}
        collaborationId={activeCollabForChannel?.id}
        onClose={() => {
          setLearningProposalOpen(false);
          setLearningProposal(null);
        }}
        onSaved={async (agentId) => {
          addToast({
            type: 'success',
            title: 'Learning saved',
            message: `Saved for ${learningProposal?.agent_name ?? 'expert'}.`,
          });
          try {
            const stats = await api.fetchLearningStats(agentId);
            if (stats.ready_for_lora) {
              addToast({
                type: 'info',
                title: 'Train LoRA',
                message: `10+ turns with ${learningProposal?.agent_name ?? 'this expert'} — open agent info to train LoRA.`,
              });
            }
          } catch {
            /* stats optional */
          }
        }}
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

function showRunbookBuilderForCollab(collab: Collaboration): boolean {
  return (
    collab.source === 'runbook' &&
    (collab.phase === 'draft' || collab.phase === 'reviewing')
  );
}

function agentsToCollaborationAgents(agents: AgentInfo[]): CollaborationAgent[] {
  return agents.map((a) => ({
    agent_id: a.id,
    agent_name: a.name,
    agent_type: a.type,
    expertise: a.expertise ?? [],
    role: '',
  }));
}
