import { useState, useRef, useCallback, useEffect, useMemo } from 'react';
import { shallow } from 'zustand/shallow';
import { useChatStore } from '../../stores/chatStore';
import { useTerminalStore } from '../../stores/terminalStore';
import { useSettingsStore } from '../../stores/settingsStore';
import { usePacksStore } from '../../stores/packsStore';
import { useEditorStore } from '../../stores/editorStore';
import { useToastStore } from '../../stores/toastStore';
import { useComposerPrefillStore } from '../../stores/composerPrefillStore';
import { clearCredentials } from '../../utils/secureStorage';
import {
  cycleWorkspaceContextMode,
  loadWorkspaceContextMode,
  workspaceContextModeLabel,
  WORKSPACE_CONTEXT_MODE_KEY,
} from '../../utils/outboundChatMetadata';
import {
  formatContextIndicator,
  loadConversationModeSetting,
  resolveConversationMode,
} from '../../utils/conversationMode';
import { channelNameToKind, resolveContextScope } from '../../utils/inferContextScope';
import type { ConversationModeSetting, WorkspaceContextMode } from '../../constants/promptMetadata';
import {
  detectHubDataAccessNeeds,
  hasGrantedHubDataAccess,
  type HubDataAccessOption,
} from '../../utils/hubDataAccess';
import { onOpenDomainPacksModal } from '../../utils/domainPacksModal';
import { loadChatWindowInitialData } from '../../hooks/useChatWindowData';
import { devLog } from '../../utils/devLog';
import { useChatInboundMessages } from '../../hooks/useChatInboundMessages';
import { useChatInboundSurfaces } from '../../hooks/useChatInboundSurfaces';
import { useWebSocket } from '../../hooks/useWebSocket';
import { useSidebarAutoUnhide } from '../../hooks/useSidebarAutoUnhide';
import {
  LazyCodeEditorPanel,
  LazyCollaborationPanel,
  LazyFileExplorerPanel,
  LazyPanelShell,
  LazyRunbookBuilderPanel,
  LazySecondaryAnalysisPanel,
  LazyTaskManagementPanel,
} from '../lazyPanels';
import { useSecondaryAnalysisStore } from '../../stores/secondaryAnalysisStore';
import type { LearningProposalAction } from '../../api/chatAPI';
import type { LoraTrainPrefill } from '../LoraTrainingPanel';
import type {
  Channel,
  CommandDefinition,
  Message,
  ThinkingAgent,
} from '../../types/protocol';
import {
  CONTINUATION_OF_METADATA_KEY,
  CONTINUATION_REASON_METADATA_KEY,
  OUTPUT_LENGTH_CONTINUATION_PROMPT,
} from '../../types/protocol';
import { findThreadParentMessage } from '../../utils/slackThread';
import { useSuggestedCommands } from '../../hooks/useSuggestedCommands';
import { useFileChangeStore } from '../../stores/fileChangeStore';
import { useGitChangeStore } from '../../stores/gitChangeStore';
import { layoutPresetLabel, panelsForPreset } from '../../utils/layoutPresets';
import {
  mainChatMaxWidth,
  measureMainChatMaxWidth,
  type MainChatPanelVisibility,
} from '../../utils/mainChatMaxWidth';
import { useHorizontalPanelResize } from '../../hooks/useHorizontalPanelResize';
import { useChatShortcutHandlers } from '../../hooks/useChatShortcutHandlers';
import { useChatShortcutOverlays } from '../../hooks/useChatShortcutOverlays';
import { useShortcutDispatcher } from '../../shortcuts/useShortcutDispatcher';
import type { LayoutPreset } from '../../stores/settingsStore';
import {
  loadComposerMode,
  COMPOSER_MODE_STORAGE_KEY,
  composerModePlaceholder,
  type ComposerMode,
} from '../../constants/composerMode';
import { NJ_BUILD_PLAN_EVENT, buildPlanBuildMessage } from '../../utils/planCard';
import { ideRoutingChipLabel } from '../../utils/ideComposer';
import { useChatChannelActions } from '../../hooks/useChatChannelActions';
import { useChatDmActions } from '../../hooks/useChatDmActions';
import { useChatCommandActions } from '../../hooks/useChatCommandActions';
import { useChatOutboundDispatch } from '../../hooks/useChatOutboundDispatch';
import { registerRestartBlocker } from '../../utils/restartSafety';
import { pendingProposalCount } from '../../utils/pendingChangeNavigation';
import { useChatWindowRoomChrome } from '../../hooks/useChatWindowRoomChrome';
import { useChatWindowIdeChrome } from '../../hooks/useChatWindowIdeChrome';
import { useChatWindowWorkspaceActions } from '../../hooks/useChatWindowWorkspaceActions';
import { useChatWindowCollaborations } from '../../hooks/useChatWindowCollaborations';
import { useChatWindowHubBootstrap } from '../../hooks/useChatWindowHubBootstrap';
import { useChatWindowTaskPanel } from '../../hooks/useChatWindowTaskPanel';
import {
  withClientPaletteCommands,
  agentsToCollaborationAgents,
  showRunbookBuilderForCollab,
} from './chatWindowHelpers';
import {
  pendingUserQuestionIds,
  pendingUserQuestionMessages,
} from '../../utils/pendingUserQuestions';

import type { ChatWindowProps } from './chatWindowTypes';

import { PendingApprovalsBar } from '../PendingApprovalsBar';
import { LocationRequestModal } from '../LocationRequestModal';
import { ChatMessageList } from './ChatMessageList';
import { ChatInputArea } from './ChatInputArea';
import { ChatChannelToolbar } from './ChatChannelToolbar';
import { ChatModalHost } from './ChatModalHost';
import { ThreadPanel } from '../ThreadPanel';
import { MyAgentsPanel } from '../MyAgentsPanel';
import { TerminalPanel } from '../TerminalPanel';
import { ToastContainer } from '../Toast';
import { ErrorBoundary } from '../ErrorBoundary';
import { ChannelSidebar } from '../ChannelSidebar';
import { CollaborationWorkspaceGate } from '../CollaborationWorkspaceGate';
import { ChatToolbarActions } from '../ChatToolbarActions';
import { ChatToolbarSidebar } from '../ChatToolbarSidebar';
import { shrinkablePanelStyle } from '../../utils/panelLayout';

const EMPTY_THINKING_AGENTS: ThinkingAgent[] = [];


export function ChatWindow({
  onOpenSettings,
  onLogout,
}: ChatWindowProps) {
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
      // While ask_user cards are pending, hide the "responding" bar entirely.
      if (pendingUserQuestionMessages(s.messages).length > 0) {
        return EMPTY_THINKING_AGENTS;
      }
      const inner = s.channelThinkingAgents.get(s.channel);
      if (!inner || inner.size === 0) return EMPTY_THINKING_AGENTS;
      return Array.from(inner.values());
    },
    shallow
  );

  const channelHeld = useChatStore((s) => s.channelHeld.get(s.channel) === true, shallow);
  const hasPendingUserQuestion = useChatStore(
    (s) => pendingUserQuestionMessages(s.messages).length > 0,
    shallow
  );

  const {
    activeRoomId,
    visibleChannels,
    activeRoomSidebarMeta,
    currentChannelType,
    connectedRoomMembers,
  } = useChatWindowRoomChrome(channels, channel);

  const hasStreamingOnChannel = useChatStore(
    (s) => {
      if (pendingUserQuestionMessages(s.messages).length > 0) return false;
      const ch = s.channel;
      return Object.values(s.streamingMessages).some((m) => (m.channel || ch) === ch);
    },
    shallow
  );

  const showAgentStop = thinkingAgentsForChannel.length > 0 || hasStreamingOnChannel;

  const myAgentsPanelOpen = useChatStore((s) => s.myAgentsPanelOpen);
  const setMyAgentsPanelOpen = useChatStore((s) => s.setMyAgentsPanelOpen);
  const closeThread = useChatStore((s) => s.closeThread);

  const { isPanelOpen, panelHeight, setPanelOpen } = useTerminalStore();
  const { layoutSettings, loadLayoutSettings, updateSettings, updateLayoutSettings } =
    useSettingsStore();
  const addToast = useToastStore(s => s.addToast);

  useSidebarAutoUnhide(agents, channels);

  useEffect(() => onOpenDomainPacksModal(() => setDomainPacksOpen(true)), []);

  const {
    fileExplorerOpen,
    setFileExplorerOpen,
    codeEditorOpen,
    setCodeEditorOpen,
    quickOpenOpen,
    symbolModalOpen,
    fastEditOpen,
    problemsOpen,
    gitModalOpen,
    setQuickOpenOpen,
    setSymbolModalOpen,
    setFastEditOpen,
    setProblemsOpen,
    setGitModalOpen,
    setActivePackModal,
    phoenixModalOpen,
    roomChatModalOpen,
    modelArenaModalOpen,
    aiInterviewModalOpen,
    hasIdeComposer,
    ideLayout,
    ideEnabled,
    ideLayoutAvailable,
    enabledPackCount,
    customPackToolbarActions,
  } = useChatWindowIdeChrome();
  const chatPanelVisible = layoutSettings.chatPanelVisible !== false;
  const toolbarChipsPlacement = layoutSettings.toolbarChipsPlacement ?? 'top';
  const mainContentRef = useRef<HTMLDivElement>(null);
  const chatPanelRef = useRef<HTMLDivElement>(null);
  const [mainChatReclampKey, setMainChatReclampKey] = useState('');
  const mainChatVisibilityRef = useRef<MainChatPanelVisibility>({
    channelSidebarOpen: false,
    fileExplorerOpen: false,
    fileExplorerEmbedded: false,
    threadOpen: false,
    collaborationOpen: false,
    taskManagementOpen: false,
  });
  const mainChatResize = useHorizontalPanelResize({
    storageKey: 'main-chat-panel-width',
    defaultWidth: 420,
    minWidth: 260,
    maxWidthRatio: 0.9,
    containerRef: mainContentRef,
    reclampKey: mainChatReclampKey,
    getMaxWidth: () => {
      const container = mainContentRef.current;
      const chatEl = chatPanelRef.current;
      if (container && chatEl) {
        return measureMainChatMaxWidth(container, chatEl, mainChatVisibilityRef.current);
      }
      const containerWidth = container?.clientWidth ?? window.innerWidth * 0.9;
      return mainChatMaxWidth(containerWidth, mainChatVisibilityRef.current);
    },
    edge: 'left',
  });
  const fetchPacks = usePacksStore((s) => s.fetchPacks);
  const openArenaWorkbench = useEditorStore((s) => s.openArenaWorkbench);
  const openKnowledgeGraphWorkbench = useEditorStore((s) => s.openKnowledgeGraphWorkbench);
  const activeEditorTab = useEditorStore((s) => {
    const id = s.activeTabId;
    return id ? (s.tabs.find((t) => t.id === id) ?? null) : null;
  });
  const inputRef = useRef<HTMLTextAreaElement | null>(null);
  const channelSearchRef = useRef<HTMLInputElement | null>(null);
  const approveFirstPendingRef = useRef<(() => void | Promise<void>) | null>(null);
  const rejectFirstPendingRef = useRef<(() => void | Promise<void>) | null>(null);

  useEffect(() => {
    void fetchPacks();
  }, [fetchPacks]);

  // Keep the main composer focused while agents wait on ask_user.
  useEffect(() => {
    if (!hasPendingUserQuestion) return;
    const t = window.setTimeout(() => {
      inputRef.current?.focus();
    }, 0);
    return () => window.clearTimeout(t);
  }, [hasPendingUserQuestion, channel]);

  const { pendingChanges, fetchPendingChanges } = useFileChangeStore(
    (s) => ({
      pendingChanges: s.pendingChanges,
      fetchPendingChanges: s.fetchPendingChanges,
    }),
    shallow
  );
  const { pendingGitChanges, fetchPendingGitChanges } = useGitChangeStore(
    (s) => ({
      pendingGitChanges: s.pendingGitChanges,
      fetchPendingGitChanges: s.fetchPendingGitChanges,
    }),
    shallow,
  );
  const pendingChangeCount = pendingProposalCount(
    pendingChanges.map((change) => change.id),
    pendingGitChanges.map((change) => change.id),
  );

  useEffect(() => {
    const refresh = () => {
      void fetchPendingChanges(username || 'default').catch((error) =>
        console.error('Failed to load pending file changes:', error),
      );
      void fetchPendingGitChanges(username || 'default').catch((error) =>
        console.error('Failed to load pending Git changes:', error),
      );
    };
    refresh();
    const timer = window.setInterval(refresh, 30_000);
    return () => window.clearInterval(timer);
  }, [fetchPendingChanges, fetchPendingGitChanges, username]);

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
  const [startCollaborationOpen, setStartCollaborationOpen] = useState(false);
  const [commandDefs, setCommandDefs] = useState<CommandDefinition[]>([]);
  const [modelLibraryOpen, setModelLibraryOpen] = useState(false);
  const [modelLibraryInitialTab, setModelLibraryInitialTab] = useState<'installed' | 'ollama' | 'huggingface' | 'train' | undefined>();
  const [domainPacksOpen, setDomainPacksOpen] = useState(false);
  const [loraTrainPrefill, setLoraTrainPrefill] = useState<LoraTrainPrefill | null>(null);
  const [learningProposal, setLearningProposal] = useState<LearningProposalAction | null>(null);
  const [learningProposalOpen, setLearningProposalOpen] = useState(false);

  const [runbookLibraryOpen, setRunbookLibraryOpen] = useState(false);
  const [taskManagementOpen, setTaskManagementOpen] = useState(false);
  const secondaryAnalysisOpen = useSecondaryAnalysisStore((s) => s.panelOpen);
  const setSecondaryAnalysisOpen = useSecondaryAnalysisStore((s) => s.setPanelOpen);
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
  const [composerHasAttachments, setComposerHasAttachments] = useState(false);
  const [composerMode, setComposerMode] = useState<ComposerMode>(() => loadComposerMode());
  const composerPrefillPending = useComposerPrefillStore((s) => s.pendingText);
  const consumeComposerPrefill = useComposerPrefillStore((s) => s.consumePrefill);

  useEffect(() => {
    const syncMode = () => setConversationModeSetting(loadConversationModeSetting());
    window.addEventListener('nj-conversation-mode-changed', syncMode);
    return () => window.removeEventListener('nj-conversation-mode-changed', syncMode);
  }, []);

  useEffect(() => {
    if (!composerPrefillPending) return;
    const text = consumeComposerPrefill();
    if (!text) return;
    setComposerDraft(text);
    const input = inputRef.current as (HTMLTextAreaElement & { setDraftText?: (t: string) => void }) | null;
    input?.setDraftText?.(text);
  }, [composerPrefillPending, consumeComposerPrefill]);

  useEffect(
    () =>
      registerRestartBlocker('chat-composer', () =>
        composerDraft.trim() || composerHasAttachments
          ? {
              id: 'chat-composer',
              message: 'A chat message or attachment is still waiting to be sent.',
            }
          : null
      ),
    [composerDraft, composerHasAttachments]
  );

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

  const handledRepoWorkspaceActionsRef = useRef<Set<string>>(new Set());
  const handledLearningProposalsRef = useRef<Set<string>>(new Set());
  const handledParticipantRequestPromptsRef = useRef<Set<string>>(new Set());

  // Clear stale chat width from localStorage
  useEffect(() => {
    localStorage.removeItem('main-chat-area-width');
  }, []);

  const preferSidebarChips = toolbarChipsPlacement === 'sidebar';
  const useSidebarChips = !isWideViewport || preferSidebarChips;
  const showTopToolbarChips = isWideViewport && !preferSidebarChips;
  const chatResizable = isWideViewport && chatPanelVisible;

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

  const {
    api,
    hubHttp,
    hubSessionToken,
    wsURL,
    wsOutageToastShownRef,
    wsConnectedChannelRef,
    totalAgentsCount,
    loadAgents,
    loadCounts,
    loadChannels,
  } = useChatWindowHubBootstrap({
    serverAddr,
    channel,
    channels,
    layoutSettings,
    loadLayoutSettings,
    setFileExplorerOpen,
    setCodeEditorOpen,
    setPanelOpen,
    setMyAgentsPanelOpen,
  });

  const {
    setActiveCollab,
    setRunbookBuilderDirty,
    activeCollabRef,
    collaborationsByIDRef,
    collaborationsByID,
    mergeCollaborationSnapshot,
    clearActiveCollabIf,
    loadCollaborations,
    activeCollabForChannel,
    trackedCollaborations,
    executingCollaborationForChannel,
    collaborationForChannel,
    panelCollaboration,
    isClosedCollaborationChannel,
    extendableCollaborations,
  } = useChatWindowCollaborations({ api, channel, addToast });

  const { handleSuggestedCommands } = useSuggestedCommands({
    collaborationsByIDRef,
    addToast,
  });

  const {
    activeWorkspaceId,
    explorerWorkspaces,
    resolveScopedRepoPaths,
    handleQuickOpenPath,
    handleImplementationSessionComplete,
    handleCADFilesWritten,
    handleOpenAtLine,
    workspaceGateCollab,
    setWorkspaceGateCollab,
    workspaceGateBusy,
    setWorkspaceGateBusy,
    dismissedWorkspaceGateIdRef,
    workspaceGateToastIdRef,
    channelAwaitingWorkspaceCollab,
    openWorkspaceGate,
    openWorkspaceGateForCollab,
  } = useChatWindowWorkspaceActions({
    channel,
    username,
    collaborationsByID,
    addToast,
    setCodeEditorOpen,
    setFileExplorerOpen,
  });

  const {
    handleWorkspaceGateContinue,
    handleWorkspaceGateDismiss,
    handleSwitchChannel,
    handleNewRunbook,
    handleCreateBlankRunbook,
    handleCreateChannel,
    handleDeleteChannel,
    handleOpenChannelInfo,
  } = useChatChannelActions({
    api,
    channel,
    username,
    agents,
    channels,
    workspaceGateCollab,
    setWorkspaceGateBusy,
    setWorkspaceGateCollab,
    setWorkspaceContextMode,
    dismissedWorkspaceGateIdRef,
    workspaceGateToastIdRef,
    collaborationsByIDRef,
    loadCollaborations,
    loadChannels,
    addToast,
    setActiveCollab,
    setRunbookLibraryOpen,
    setChannelInfoModal,
    updateSettings,
  });

  const { handleCreateDM, handleNewDmCreated } = useChatDmActions({
    api,
    username,
    loadChannels,
    loadAgents,
    handleSwitchChannel,
    updateSettings,
    addToast,
  });

  const { assistantTasks, setAssistantTasks, setAssistantReminders, taskPanelProps } =
    useChatWindowTaskPanel({
      api,
      channel,
      username,
      taskManagementOpen,
      setTaskManagementOpen,
      trackedCollaborations,
      collaborationsByIDRef,
      loadCollaborations,
      handleSwitchChannel,
      setActiveCollab,
      clearActiveCollabIf,
      addToast,
    });

  useEffect(() => {
    mainChatVisibilityRef.current = {
      channelSidebarOpen,
      fileExplorerOpen,
      fileExplorerEmbedded: ideLayout && fileExplorerOpen,
      threadOpen: !!openThreadId,
      collaborationOpen: !!panelCollaboration,
      taskManagementOpen,
    };
    setMainChatReclampKey(
      [
        channelSidebarOpen,
        fileExplorerOpen,
        ideLayout,
        codeEditorOpen,
        openThreadId,
        panelCollaboration?.id,
        taskManagementOpen,
      ].join(':')
    );
  }, [
    channelSidebarOpen,
    fileExplorerOpen,
    ideLayout,
    codeEditorOpen,
    openThreadId,
    panelCollaboration,
    taskManagementOpen,
  ]);

  const {
    surfaceSlackInboundNotification,
    debouncedRefreshAgents,
    surfaceChangeProposal,
    jumpToOldestPendingChange,
    scrollToApproval,
    surfaceToolApproval,
  } = useChatInboundSurfaces({
    api,
    username,
    handleSwitchChannel,
    addToast,
    loadAgents,
    loadCounts,
    activeWorkspaceId,
    explorerWorkspaces,
    fetchPendingChanges,
    fetchPendingGitChanges,
    pendingChangeCount,
    pendingChanges,
    pendingGitChanges,
  });

  const handledHandoffMessagesRef = useRef(new Set<string>());

  const onInboundMessage = useChatInboundMessages({
    api,
    channel,
    handledHandoffMessagesRef,
    handledParticipantRequestPromptsRef,
    handledRepoWorkspaceActionsRef,
    handledLearningProposalsRef,
    collaborationsByIDRef,
    activeCollabRef,
    inputRef,
    loadChannels,
    handleSwitchChannel,
    debouncedRefreshAgents,
    mergeCollaborationSnapshot,
    setActiveCollab,
    addToast,
    surfaceSlackInboundNotification,
    surfaceChangeProposal,
    surfaceToolApproval,
    handleImplementationSessionComplete,
    handleCADFilesWritten,
    handleSuggestedCommands,
    setFileExplorerOpen,
    setLearningProposal,
    setLearningProposalOpen,
  });

  // WebSocket connection
  const { status, connect: reconnectHub } = useWebSocket({
    url: wsURL,
    hasSession: hubSessionToken !== null,
    onMessage: onInboundMessage,
    onConnect: () => {
      devLog('Connected to chat');
      wsOutageToastShownRef.current = false;
      useChatStore.getState().setConnectionStatus('connected');
      const prevChannel = wsConnectedChannelRef.current;
      const activeChannel = useChatStore.getState().channel;
      wsConnectedChannelRef.current = activeChannel;
      // Channel switches rebuild the WS URL and reconnect; handleSwitchChannel already
      // loads history — skip the heavy parallel initial load to avoid races.
      if (prevChannel !== null && prevChannel !== activeChannel) {
        return;
      }
      loadInitialData();
    },
    onDisconnect: () => {
      devLog('Disconnected from chat');
      useChatStore.getState().setConnectionStatus('disconnected');
    },
    onError: (error) => {
      console.error('WebSocket error:', error);
      useChatStore.getState().setConnectionStatus('error');
      if (!wsOutageToastShownRef.current) {
        wsOutageToastShownRef.current = true;
        addToast({
          type: 'error',
          title: 'Connection lost',
          message: 'WebSocket disconnected. Reconnecting…',
        });
      }
    },
  });

  const loadInitialData = useCallback(async () => {
    await loadChatWindowInitialData({
      api,
      loadCollaborations,
      loadAgents,
      loadCounts,
      loadChannels,
      addToast,
      loadCommands: async () => {
        try {
          const defs = await api.fetchCommands();
          setCommandDefs(withClientPaletteCommands(defs));
        } catch (err) {
          console.error('Failed to load command definitions:', err);
          setCommandDefs(withClientPaletteCommands([]));
        }
      },
    });
  }, [api, loadCollaborations, loadAgents, loadCounts, loadChannels, addToast]);

  const {
    dispatchThreadReply,
    dispatchMessage,
    handleChannelInterject,
    appendLocalSlashCommand,
    handleHubAccessConfirm,
  } = useChatOutboundDispatch({
    api,
    channel,
    username,
    workspaceContextMode,
    conversationModeSetting,
    activeChannelMeta,
    composerMode,
    agents,
    activeEditorTab,
    layoutSettings,
    ideEnabled,
    explorerWorkspaces,
    activeWorkspaceId,
    resolveScopedRepoPaths,
    ideLayout,
    hasIdeComposer,
    addToast,
    loadChannels,
    loadAgents,
    handleSwitchChannel,
    loadCollaborations,
    mergeCollaborationSnapshot,
    setActiveCollab,
    executingCollaborationForChannel,
    updateSettings,
    collaborationsByIDRef,
    hubAccessPending,
    setHubAccessPending,
    setHubAccessLoading,
    setHubAccessError,
  });

  const handleBuildPlan = useCallback(
    async (req: { markdown: string; planId: string }) => {
      setComposerMode('agent');
      try {
        localStorage.setItem(COMPOSER_MODE_STORAGE_KEY, 'agent');
      } catch {
        /* ignore */
      }
      await dispatchMessage(
        buildPlanBuildMessage(req.markdown),
        {
          plan_id: req.planId,
          implementation_session: true,
        },
        'agent',
      );
    },
    [dispatchMessage],
  );

  useEffect(() => {
    const onBuild = (ev: Event) => {
      const detail = (ev as CustomEvent<{ markdown?: string; planId?: string }>).detail;
      if (!detail?.markdown) return;
      void handleBuildPlan({ markdown: detail.markdown, planId: detail.planId || 'plan' });
    };
    window.addEventListener(NJ_BUILD_PLAN_EVENT, onBuild);
    return () => window.removeEventListener(NJ_BUILD_PLAN_EVENT, onBuild);
  }, [handleBuildPlan]);

  const handleContinueGeneration = useCallback(
    async (message: Message) => {
      const meta: Record<string, unknown> = {
        [CONTINUATION_OF_METADATA_KEY]: message.id,
        [CONTINUATION_REASON_METADATA_KEY]: 'output_length',
        reply_to: message.id,
      };
      if (message.is_thread_reply && message.thread_id) {
        await dispatchThreadReply(message.thread_id, OUTPUT_LENGTH_CONTINUATION_PROMPT, meta);
        return;
      }
      await dispatchMessage(OUTPUT_LENGTH_CONTINUATION_PROMPT, meta);
    },
    [dispatchMessage, dispatchThreadReply],
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
          supported_bases: ctx.supported_bases,
          prior_adapter_id: ctx.prior_adapter_id,
          active_adapter_version: ctx.active_adapter_version,
          refresh_suggested: ctx.refresh_suggested,
          include_learnings_default: ctx.include_learnings_default,
          eval_min_score: ctx.eval_min_score,
          require_eval_to_assign: ctx.require_eval_to_assign,
          sharpen: Boolean(ctx.suggest_training || ctx.ready),
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

  const handleSendMessage = async (content: string, metadata?: Record<string, unknown>): Promise<boolean> => {
    const trimmed = content.trimStart();
    if (content.trim() === '/nj-open-model-library') {
      setModelLibraryOpen(true);
      appendLocalSlashCommand('/nj-open-model-library');
      return true;
    }
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
      return false;
    }

    // Composer answers pending ask_user cards (keeps focus in the main input).
    const answerText = content.trim();
    if (answerText && !answerText.startsWith('/')) {
      const pendingIds = pendingUserQuestionIds(useChatStore.getState().messages);
      if (pendingIds.length > 0) {
        try {
          await Promise.all(pendingIds.map((id) => api.answerUserQuestion(id, answerText)));
          window.setTimeout(() => inputRef.current?.focus(), 0);
          return true;
        } catch (err) {
          addToast({
            type: 'error',
            title: 'Could not answer agent question',
            message: err instanceof Error ? err.message : 'Failed to submit answer',
          });
          return false;
        }
      }
    }

    const needs = detectHubDataAccessNeeds(content);
    const composerMeta = metadata ?? {};
    if (needs.length > 0 && !hasGrantedHubDataAccess(composerMeta)) {
      setHubAccessError(null);
      setHubAccessPending({ mode: 'main', content, metadata: composerMeta, options: needs });
      return false;
    }

    return dispatchMessage(content, composerMeta);
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

  // Command palette + local /nj-open-* shortcuts (defs load inside openCommandPalette).
  const {
    handleCommandExecute,
    openCommandPalette,
  } = useChatCommandActions({
    api,
    channel,
    username,
    commandDefsLength: commandDefs.length,
    setCommandDefs,
    setCommandPaletteFilter,
    setCommandPaletteOpen,
    setModelLibraryOpen,
    setCodeEditorOpen,
    setFileExplorerOpen,
    setAssistantTasks,
    setAssistantReminders,
    updateLayoutSettings,
    appendLocalSlashCommand,
    handleSendMessage,
    loadCollaborations,
    fetchPendingChanges,
    addToast,
    inputRef,
  });

  const handleFirstWinOpenFiles = useCallback(() => {
    setFileExplorerOpen(true);
  }, []);

  const handleFirstWinPrefill = useCallback((text: string) => {
    useComposerPrefillStore.getState().requestPrefill(text);
  }, []);

  const handleFirstWinOpenModelLibrary = useCallback(() => {
    setModelLibraryOpen(true);
  }, []);

  const handleOpenStartCollaboration = useCallback(() => {
    void (async () => {
      if (commandDefs.length === 0) {
        try {
          const defs = await api.fetchCommands();
          setCommandDefs(withClientPaletteCommands(defs));
        } catch {
          /* keep empty; modal shows fallback */
        }
      }
      setStartCollaborationOpen(true);
    })();
  }, [api, commandDefs.length]);

  useShortcutDispatcher(true);

  const handleOpenSettings = useCallback(
    (tab?: import('../SettingsModal').SettingsTab | 'domain-packs') => {
      if (tab === 'domain-packs') {
        setDomainPacksOpen(true);
        return;
      }
      onOpenSettings?.(tab);
    },
    [onOpenSettings],
  );

  useChatShortcutHandlers({
    onOpenSettings: handleOpenSettings,
    channelSearchRef,
    inputRef,
    ideEnabled,
    ideLayout,
    codeEditorOpen,
    showAgentStop,
    useSidebarChips,
    channelSidebarOpen,
    setChannelSidebarOpen,
    setFileExplorerOpen,
    setCodeEditorOpen,
    setTaskManagementOpen,
    onOpenPendingChanges: jumpToOldestPendingChange,
    setToolbarSidebarOpen,
    setCommandPaletteOpen,
    setModelLibraryOpen,
    setDomainPacksOpen,
    setChatFindOpen,
    setCreateChannelOpen,
    setCreateNewDmOpen,
    chatPanelVisible,
    openCommandPalette,
    handleChannelInterject,
    handleNewRunbook,
    handleSwitchChannel,
    handleCreateDM,
    updateLayoutSettings,
    approveFirstPendingRef,
    rejectFirstPendingRef,
  });

  const closeCommandPalette = useCallback(() => {
    setCommandPaletteOpen(false);
    const input = inputRef.current as (HTMLTextAreaElement & { clearInput?: () => void }) | null;
    input?.clearInput?.();
  }, []);

  const closeModelLibrary = useCallback(() => {
    setModelLibraryOpen(false);
    setModelLibraryInitialTab(undefined);
    setLoraTrainPrefill(null);
  }, []);

  const closeDomainPacks = useCallback(() => {
    setDomainPacksOpen(false);
  }, []);

  useChatShortcutOverlays({
    commandPaletteOpen,
    onCloseCommandPalette: closeCommandPalette,
    ideEnabled,
    createChannelOpen,
    onCloseCreateChannel: () => setCreateChannelOpen(false),
    createNewDmOpen,
    onCloseCreateNewDm: () => setCreateNewDmOpen(false),
    channelInfoModal,
    onCloseChannelInfo: () => setChannelInfoModal(null),
    phoenixModalOpen,
    onClosePhoenix: () => setActivePackModal(null),
    learningProposalOpen,
    onCloseLearningProposal: () => {
      setLearningProposalOpen(false);
      setLearningProposal(null);
    },
    hubAccessPending,
    onCloseHubAccess: () => {
      setHubAccessPending(null);
      setHubAccessError(null);
    },
    chatFindOpen,
    onCloseChatFind: () => setChatFindOpen(false),
    openThreadId,
    onCloseThread: closeThread,
  });

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
      workspaceContextMode,
      onCycleWorkspaceContext: () => {
        const next = cycleWorkspaceContextMode(workspaceContextMode);
        setWorkspaceContextMode(next);
        localStorage.setItem(WORKSPACE_CONTEXT_MODE_KEY, next);
      },
      workspaceContextButtonTitle: `Workspace context: ${workspaceContextModeLabel(workspaceContextMode)} (click to cycle). Next send: ${contextScopePreview.scope}`,
      onOpenPendingChanges: jumpToOldestPendingChange,
      pendingChangeCount,
      onOpenFileExplorer: () => {
        setFileExplorerOpen(true);
        void updateLayoutSettings({ filesPanelVisible: true });
      },
      onOpenCodeEditor: () => {
        setCodeEditorOpen(true);
        void updateLayoutSettings({ editorPanelVisible: true });
      },
      onOpenKnowledgeGraph: () => {
        const ws = explorerWorkspaces.find((w) => w.id === activeWorkspaceId);
        if (ws?.path && activeWorkspaceId) {
          openKnowledgeGraphWorkbench(activeWorkspaceId, ws.path);
          setCodeEditorOpen(true);
          void updateLayoutSettings({ editorPanelVisible: true });
        } else {
          addToast({
            type: 'warning',
            title: 'No workspace',
            message: 'Open a workspace in the Files panel first.',
          });
        }
      },
      customPackToolbarActions,
      onCustomPackToolbarAction: (modal: string) => setActivePackModal(modal),
      taskManagementOpen,
      onToggleTaskManagement: () => setTaskManagementOpen((o) => !o),
      onNewRunbook: () => void handleNewRunbook(),
      onOpenMyAgents: () => setMyAgentsPanelOpen(true),
      totalAgentsCount,
      ideEnabled,
      ideLayoutAvailable,
      onOpenProblems: () => setProblemsOpen(true),
      gitModalOpen,
      onToggleGitModal: () => setGitModalOpen(!gitModalOpen),
      ideLayout,
      onToggleIdeLayout: () => {
        const next: LayoutPreset = ideLayout ? 'team' : 'ide';
        void updateLayoutSettings(panelsForPreset(next));
      },
      ideLayoutButtonTitle: `Layout: ${layoutPresetLabel(ideLayout ? 'ide' : 'team')} (click to switch)`,
      onOpenModelLibrary: () => setModelLibraryOpen(true),
      onOpenDomainPacks: () => setDomainPacksOpen(true),
      enabledPackCount,
      onOpenSettings: handleOpenSettings,
      onLogout: onLogout ? handleLogout : undefined,
      username,
      serverAddr,
      connectionStatus: status,
      onReconnectHub: reconnectHub,
    }),
    [
      openCommandPalette,
      jumpToOldestPendingChange,
      pendingChangeCount,
      updateLayoutSettings,
      workspaceContextMode,
      contextScopePreview.scope,
      taskManagementOpen,
      handleNewRunbook,
      totalAgentsCount,
      ideEnabled,
      ideLayoutAvailable,
      customPackToolbarActions,
      gitModalOpen,
      ideLayout,
      handleOpenSettings,
      onLogout,
      enabledPackCount,
      handleLogout,
      username,
      serverAddr,
      status,
      reconnectHub,
      activeWorkspaceId,
      explorerWorkspaces,
      openKnowledgeGraphWorkbench,
      addToast,
    ]
  );

  return (
    <ErrorBoundary>
      <div className="flex flex-col h-full min-h-0 overflow-hidden bg-slack-bg">
      <CollaborationWorkspaceGate
        collaboration={workspaceGateCollab}
        busy={workspaceGateBusy}
        onContinue={handleWorkspaceGateContinue}
        onNotNow={handleWorkspaceGateDismiss}
      />
      {/* Top Toolbar - always visible, spans full width */}
      <ChatChannelToolbar
        channel={channel}
        channels={channels}
        status={status}
        channelSidebarOpen={channelSidebarOpen}
        onToggleChannelSidebar={toggleChannelSidebar}
        chatPanelVisible={chatPanelVisible}
        onToggleChatPanel={() => void updateLayoutSettings({ chatPanelVisible: !chatPanelVisible })}
        useSidebarChips={useSidebarChips}
        toolbarSidebarOpen={toolbarSidebarOpen}
        onToggleToolbarSidebar={toggleToolbarSidebar}
        showTopToolbarChips={showTopToolbarChips}
        toolbarActionsProps={toolbarActionsProps}
      />

      <PendingApprovalsBar
        channel={channel}
        api={api}
        collaboration={collaborationForChannel}
        onOpenTerminal={() => setPanelOpen(true)}
        onScrollToApproval={scrollToApproval}
        onSwitchChannel={handleSwitchChannel}
        approveFirstPendingRef={approveFirstPendingRef}
        rejectFirstPendingRef={rejectFirstPendingRef}
      />
      <LocationRequestModal api={api} />

      {/* Main Content Area */}
      <div className="flex flex-1 min-h-0 min-w-0 overflow-hidden" data-testid="chat-main-content-row">
        <div ref={mainContentRef} className="flex flex-1 min-h-0 min-w-0 overflow-hidden" data-testid="chat-main-inner-column">
        {/* Channel Sidebar */}
        {channelSidebarOpen && (
          <ChannelSidebar
            channels={visibleChannels}
            agents={agents}
            searchInputRef={channelSearchRef}
            onSwitchChannel={handleSwitchChannel}
            onCreateChannel={() => setCreateChannelOpen(true)}
            onCreateDM={handleCreateDM}
            onOpenNewDM={() => setCreateNewDmOpen(true)}
            onDeleteChannel={handleDeleteChannel}
            onOpenChannelInfo={handleOpenChannelInfo}
            activeRoom={activeRoomSidebarMeta}
          />
        )}

        {/* File Explorer */}
        {fileExplorerOpen && (
          <LazyPanelShell
            panelName="File explorer"
            Component={LazyFileExplorerPanel}
            props={{
              variant: ideLayout ? 'embedded' : 'overlay',
              onClose: () => {
                setFileExplorerOpen(false);
                if (ideLayout) {
                  void updateLayoutSettings({ filesPanelVisible: false });
                }
              },
              onFileOpen: () => setCodeEditorOpen(true),
            }}
          />
        )}

        {/* Workspace slot — editor stays visible when chat is hidden (IDE layout).
            Do not gate on chatPanelVisible: that incorrectly unmounted the editor with chat. */}
        {(ideLayout || chatResizable) && (
          <div
            data-shrinkable-workspace
            className="flex flex-1 min-w-0 min-h-0 flex-col h-full border-r border-slack-border"
          >
            {codeEditorOpen ? (
              <LazyPanelShell
                panelName="Code editor"
                Component={LazyCodeEditorPanel}
                props={{
                  variant: 'embedded',
                  onClose: () => {
                    setCodeEditorOpen(false);
                    void updateLayoutSettings({ editorPanelVisible: false });
                  },
                }}
              />
            ) : ideLayout ? (
              <div className="flex flex-1 min-h-0 items-center justify-center px-6 text-center text-sm text-slack-textMuted">
                Open a file from the explorer to start editing
              </div>
            ) : (
              <div className="flex-1 min-w-0" aria-hidden="true" />
            )}
          </div>
        )}

        {/* Code Editor — overlay when chat is not docked (narrow viewport) */}
        {codeEditorOpen && !ideLayout && !chatResizable && (
          <LazyPanelShell
            panelName="Code editor"
            Component={LazyCodeEditorPanel}
            props={{
              variant: 'overlay',
              onClose: () => {
                setCodeEditorOpen(false);
                void updateLayoutSettings({ editorPanelVisible: false });
              },
            }}
          />
        )}

        {/* Main Chat Area */}
        {chatPanelVisible && (
        <div
          ref={chatPanelRef}
          className={
            chatResizable
              ? 'flex flex-col h-full min-h-0 relative border-l border-slack-border'
              : 'flex flex-col flex-1 min-h-0 min-w-[220px] sm:min-w-[260px] transition-all duration-300 ease-in-out relative overflow-hidden'
          }
          style={chatResizable ? shrinkablePanelStyle(mainChatResize.width, 220) : undefined}
        >
        {chatResizable && (
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

        {currentChannelType === 'room' && (
          <div className="shrink-0 border-b border-slack-border px-3 py-2 text-xs text-gray-400 flex items-center justify-between">
            <div className="min-w-0">
              <span className="text-gray-500">Room</span>
              {activeRoomId ? (
                <>
                  <span className="text-gray-600"> · </span>
                  <span className="font-mono text-gray-300">{activeRoomId.slice(0, 8)}</span>
                </>
              ) : null}
            </div>
            <div className="shrink-0">
              <span className="text-gray-500">Online:</span>{' '}
              <span className="text-gray-200">{connectedRoomMembers.length}</span>
            </div>
          </div>
        )}

        <ChatMessageList
          channel={channel}
          messageSearchQuery={messageSearchQuery}
          chatFindOpen={chatFindOpen}
          onMessageSearchQueryChange={setMessageSearchQuery}
          onCloseFind={() => {
            setChatFindOpen(false);
            setMessageSearchQuery('');
          }}
          isClosedCollaborationChannel={isClosedCollaborationChannel}
          collaborationForChannel={collaborationForChannel}
          channelAwaitingWorkspaceCollab={channelAwaitingWorkspaceCollab}
          onOpenWorkspaceGate={openWorkspaceGate}
          onOpenFiles={handleFirstWinOpenFiles}
          onOpenCommandPalette={openCommandPalette}
          onOpenStartCollaboration={handleOpenStartCollaboration}
          onOpenAgentDM={handleCreateDM}
          onPrefillComposer={handleFirstWinPrefill}
          onOpenModelLibrary={handleFirstWinOpenModelLibrary}
          onOpenDomainPacks={() => setDomainPacksOpen(true)}
          onContinueGeneration={handleContinueGeneration}
        />

        <ChatInputArea
          channel={channel}
          channelHeld={channelHeld}
          hasPendingUserQuestion={hasPendingUserQuestion}
          thinkingAgentsForChannel={thinkingAgentsForChannel}
          showAgentStop={showAgentStop}
          onChannelInterject={() => void handleChannelInterject()}
          composerMode={composerMode}
          composerModeDisabled={status !== 'connected' || isClosedCollaborationChannel}
          onComposerModeChange={(mode) => {
            setComposerMode(mode);
            localStorage.setItem(COMPOSER_MODE_STORAGE_KEY, mode);
            if (ideEnabled) {
              void updateLayoutSettings({
                editorAgentMode: mode,
              });
            }
          }}
          onSend={handleSendMessage}
          inputDisabled={status !== 'connected' || isClosedCollaborationChannel}
          inputPlaceholder={
            isClosedCollaborationChannel
              ? 'Collaboration closed — read-only (slash commands still work)'
              : hasPendingUserQuestion
                ? 'Answer the agent question…'
                : status === 'connected'
                  ? composerModePlaceholder(composerMode)
                  : 'Connecting...'
          }
          agents={agents}
          inputRef={inputRef}
          composerDraft={composerDraft}
          onDraftChange={setComposerDraft}
          onAttachmentStateChange={setComposerHasAttachments}
          showContextIndicator={
            workspaceContextMode === 'auto' || conversationModeSetting === 'auto'
          }
          contextIndicatorLabel={contextIndicatorLabel}
          contextScopeReason={contextScopePreview.reason}
          ideRoutingLabel={ideRoutingLabel}
          api={api}
        />
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
          <LazyPanelShell
            panelName="Runbook builder"
            Component={LazyRunbookBuilderPanel}
            props={{
              collaboration: panelCollaboration,
              hubAgents: agentsToCollaborationAgents(agents),
              onClose: () => {
                setRunbookBuilderDirty(false);
                setActiveCollab(null);
              },
              onDirtyChange: setRunbookBuilderDirty,
              onSaved: (snap) => {
                mergeCollaborationSnapshot(snap);
                setActiveCollab(snap);
                void loadCollaborations(snap.channel || channel);
              },
              onStarted: (snap) => {
                mergeCollaborationSnapshot(snap);
                setActiveCollab(snap);
                void loadCollaborations(snap.channel || channel);
              },
              onWorkspaceGateRequest: openWorkspaceGateForCollab,
            }}
          />
        ) : panelCollaboration ? (
          <LazyPanelShell
            panelName="Collaboration"
            Component={LazyCollaborationPanel}
            props={{
              collaboration: panelCollaboration,
              extendableCollaborations: extendableCollaborations,
              executingCollaboration: executingCollaborationForChannel,
              onClose: () => setActiveCollab(null),
              onConfirmWorkspace: openWorkspaceGate,
              onAfterCollaborationCommand: async () => {
                await loadCollaborations(panelCollaboration.channel || channel);
              },
              onOpenRunbookRun: async (collabId, collabChannel) => {
                if (collabChannel && collabChannel !== channel) {
                  await handleSwitchChannel(collabChannel);
                }
                try {
                  setActiveCollab(await api.getRunbook(collabId));
                } catch {
                  /* optional snapshot */
                }
                await loadCollaborations(collabChannel || channel);
              },
            }}
          />
        ) : null}

        {/* Task Management Panel */}
        {taskManagementOpen && (
          <LazyPanelShell
            panelName="Tasks"
            Component={LazyTaskManagementPanel}
            props={taskPanelProps}
          />
        )}

        {secondaryAnalysisOpen && (
          <LazyPanelShell
            panelName="Secondary analysis"
            Component={LazySecondaryAnalysisPanel}
            props={{ onClose: () => setSecondaryAnalysisOpen(false) }}
          />
        )}

        {/* My Agents Panel - slides in from right */}
        {myAgentsPanelOpen && (
          <ErrorBoundary panelName="My agents">
            <MyAgentsPanel
              onClose={() => setMyAgentsPanelOpen(false)}
              onTrainLoRA={handleTrainLoRAForAgent}
            />
          </ErrorBoundary>
        )}

        </div>

        {useSidebarChips && (
          <ChatToolbarSidebar open={toolbarSidebarOpen} className="self-stretch h-full">
            <ChatToolbarActions layout="vertical" {...toolbarActionsProps} />
          </ChatToolbarSidebar>
        )}
      </div>

      {/* Terminal Panel - slides up from bottom.
          maxHeight keeps the helper footer on-screen when the requested panel
          height would otherwise overflow the window and clip the last rows. */}
      <div
        className="shrink-0 transition-all duration-300 ease-in-out overflow-hidden"
        style={{
          height: isPanelOpen ? `${panelHeight}px` : '0px',
          maxHeight: isPanelOpen ? 'calc(100% - 2.75rem)' : '0px',
        }}
      >
        <ErrorBoundary panelName="Terminal">
          <TerminalPanel
            channel={channel}
            api={api}
            collaboration={collaborationForChannel}
          />
        </ErrorBoundary>
      </div>
      
      <ChatModalHost
        api={api}
        hubHttp={hubHttp}
        channel={channel}
        username={username}
        agents={agents}
        channels={channels}
        trackedCollaborations={trackedCollaborations}
        assistantTasks={assistantTasks}
        pendingChanges={pendingChanges}
        commandDefs={commandDefs}
        ideEnabled={ideEnabled}
        activeWorkspaceId={activeWorkspaceId}
        explorerWorkspaces={explorerWorkspaces}
        gitModalOpen={gitModalOpen}
        setGitModalOpen={setGitModalOpen}
        quickOpenOpen={quickOpenOpen}
        setQuickOpenOpen={setQuickOpenOpen}
        symbolModalOpen={symbolModalOpen}
        setSymbolModalOpen={setSymbolModalOpen}
        problemsOpen={problemsOpen}
        setProblemsOpen={setProblemsOpen}
        fastEditOpen={fastEditOpen}
        setFastEditOpen={setFastEditOpen}
        commandPaletteOpen={commandPaletteOpen}
        commandPaletteFilter={commandPaletteFilter}
        closeCommandPalette={closeCommandPalette}
        handleCommandExecute={handleCommandExecute}
        startCollaborationOpen={startCollaborationOpen}
        setStartCollaborationOpen={setStartCollaborationOpen}
        createChannelOpen={createChannelOpen}
        setCreateChannelOpen={setCreateChannelOpen}
        handleCreateChannel={handleCreateChannel}
        channelInfoModal={channelInfoModal}
        setChannelInfoModal={setChannelInfoModal}
        createNewDmOpen={createNewDmOpen}
        setCreateNewDmOpen={setCreateNewDmOpen}
        handleNewDmCreated={handleNewDmCreated}
        modelLibraryOpen={modelLibraryOpen}
        closeModelLibrary={closeModelLibrary}
        switchAllAgentProviders={switchAllAgentProviders}
        switchAgentProvider={switchAgentProvider}
        modelLibraryInitialTab={modelLibraryInitialTab}
        loraTrainPrefill={loraTrainPrefill}
        domainPacksOpen={domainPacksOpen}
        closeDomainPacks={closeDomainPacks}
        phoenixModalOpen={phoenixModalOpen}
        roomChatModalOpen={roomChatModalOpen}
        modelArenaModalOpen={modelArenaModalOpen}
        aiInterviewModalOpen={aiInterviewModalOpen}
        setActivePackModal={setActivePackModal}
        openArenaWorkbench={openArenaWorkbench}
        setCodeEditorOpen={setCodeEditorOpen}
        updateLayoutSettings={updateLayoutSettings}
        learningProposalOpen={learningProposalOpen}
        learningProposal={learningProposal}
        setLearningProposalOpen={setLearningProposalOpen}
        setLearningProposal={setLearningProposal}
        activeCollabForChannel={activeCollabForChannel ?? null}
        hubAccessPending={hubAccessPending}
        hubAccessLoading={hubAccessLoading}
        hubAccessError={hubAccessError}
        setHubAccessPending={setHubAccessPending}
        setHubAccessError={setHubAccessError}
        handleHubAccessConfirm={handleHubAccessConfirm}
        runbookLibraryOpen={runbookLibraryOpen}
        setRunbookLibraryOpen={setRunbookLibraryOpen}
        handleSwitchChannel={handleSwitchChannel}
        setActiveCollab={setActiveCollab}
        loadCollaborations={loadCollaborations}
        handleCreateBlankRunbook={handleCreateBlankRunbook}
        handleQuickOpenPath={handleQuickOpenPath}
        handleOpenAtLine={handleOpenAtLine}
      />

      {/* Toast Notifications */}
      <ToastContainer />
      </div>
    </ErrorBoundary>
  );
}
