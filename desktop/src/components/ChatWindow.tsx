import { useState, useRef, useCallback, useEffect, useMemo, startTransition } from 'react';
import { shallow } from 'zustand/shallow';
import { useChatStore } from '../stores/chatStore';
import { useTerminalStore, createNewTab } from '../stores/terminalStore';
import { useSettingsStore } from '../stores/settingsStore';
import { usePacksStore } from '../stores/packsStore';
import { GitModal } from './GitPanel';
import { QuickOpenModal } from './QuickOpenModal';
import { SymbolModal } from './SymbolModal';
import { ProblemsPanel } from './ProblemsPanel';
import { FastEditModal } from './FastEditModal';
import { useEditorStore } from '../stores/editorStore';
import { getLanguageFromPath } from '../utils/editorLanguage';
import { isEditableCsvPath } from '../utils/csvTable';
import { useToastStore } from '../stores/toastStore';
import { useApprovalStore } from '../stores/approvalStore';
import { formatToolApprovalSummary } from '../utils/approvalDisplay';
import { PendingApprovalsBar } from './PendingApprovalsBar';
import { LocationRequestModal } from './LocationRequestModal';
import { useLocationShareStore } from '../stores/locationShareStore';
import { useComposerPrefillStore } from '../stores/composerPrefillStore';
import { ChatAPI } from '../api/chatAPI';
import { clearCredentials } from '../utils/secureStorage';
import {
  buildHumanOutboundMetadata,
  cycleWorkspaceContextMode,
  loadWorkspaceContextMode,
  loadScopedWorkspaceContext,
  workspaceContextModeLabel,
  WORKSPACE_CONTEXT_MODE_KEY,
} from '../utils/outboundChatMetadata';
import { attachAmbientStateMetadata } from '../utils/ambientState';
import { applyContextRequestToMetadata } from '../utils/contextRequestAttach';
import {
  clearPendingSendThinking,
  markPendingSendThinking,
  NJ_PENDING_SEND_AGENT_ID,
} from '../utils/pendingSendThinking';
import {
  formatContextIndicator,
  loadConversationModeSetting,
  resolveConversationMode,
} from '../utils/conversationMode';
import { channelNameToKind, resolveContextScope } from '../utils/inferContextScope';
import type { ConversationModeSetting, WorkspaceContextMode } from '../constants/promptMetadata';
import { METADATA_CHANNEL_HOLD } from '../types/protocol';
import { GRANTED_HUB_DATA_ACCESS_KEY, IMPLEMENTATION_FILES_CHANGED_KEY, IMPLEMENTATION_SESSION_COMPLETE_KEY, IMPLEMENTATION_SESSION_OUTCOME_KEY, CAD_FILES_WRITTEN_KEY } from '../constants/promptMetadata';
import {
  detectHubDataAccessNeeds,
  hasGrantedHubDataAccess,
  type HubDataAccessOption,
} from '../utils/hubDataAccess';
import { HubDataAccessModal } from './HubDataAccessModal';
import { shouldSendChannelJoinMessage } from '../utils/joinMessage';
import { devLog } from '../utils/devLog';
import {
  fileChangeProposalPaths,
  refreshFileExplorerForPaths,
} from '../utils/refreshFileExplorer';
import { useWebSocket } from '../hooks/useWebSocket';
import { useSidebarAutoUnhide } from '../hooks/useSidebarAutoUnhide';
import { agentSidebarHideKey, dmChannelNamesForAgent, predictedDmChannelName } from '../utils/dmChannelDisplay';
import {
  patchRevealForChannel,
  patchRevealSidebarItems,
} from '../utils/sidebarVisibility';
import { chatScrollerElRef } from './MessageList';
import { ChatMessageList } from './chat/ChatMessageList';
import { ChatInputArea } from './chat/ChatInputArea';
import {
  shouldNotifySlackInbound,
  slackChannelLabel,
  slackInboundPreview,
  slackInboundSenderLabel,
} from '../utils/slackNotification';
import { isSlackHubChannelName } from '../utils/slackChannelDisplay';
import { ThreadPanel } from './ThreadPanel';
import { MyAgentsPanel } from './MyAgentsPanel';
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
  isAwaitingWorkspaceConfirmation,
  isNonTerminalCollaborationPhase,
  resolvePanelCollaboration,
} from '../utils/collaborationPanelState';
import {
  collaboratorsAddedSince,
  decideCollabPanelOpen,
  isTerminalCollaborationPhase as isTerminalCollabPhaseHelper,
  parseCollabParticipantAddRequest,
  shouldToastCollaboratorAdds,
} from '../utils/chatInboundCollab';
import { RunbookBuilderPanel } from './RunbookBuilderPanel';
import { RunbookLibraryModal } from './runbook/RunbookLibraryModal';
import { CollaborationWorkspaceGate } from './CollaborationWorkspaceGate';
import { TaskManagementPanel } from './TaskManagementPanel';
import { SecondaryAnalysisPanel } from './SecondaryAnalysisPanel';
import { useSecondaryAnalysisStore } from '../stores/secondaryAnalysisStore';
import { useCollaborationsStore, collaborationsByIDSnapshot } from '../stores/collaborationsStore';
import { useIdeOverlayStore } from '../stores/ideOverlayStore';
import { ModelLibraryModal } from './ModelLibraryModal';
import { DomainPacksModal } from './DomainPacksModal';
import { PhoenixBrowserModal } from './PhoenixBrowserModal';
import { RoomChatModal } from './RoomChatModal';
import { ModelArenaModal } from './ModelArenaModal';
import { AIInterviewPrepModal } from './AIInterviewPrepModal';
import { LearningProposalModal } from './LearningProposalModal';
import type { LearningProposalAction } from '../api/chatAPI';
import type { LoraTrainPrefill } from './LoraTrainingPanel';
import { LeftSidebarIcon, RightSidebarIcon, ChatPanelIcon } from './Icons';
import { ChatToolbarActions } from './ChatToolbarActions';
import { ChatToolbarSidebar } from './ChatToolbarSidebar';
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
import {
  CONTINUATION_OF_METADATA_KEY,
  CONTINUATION_REASON_METADATA_KEY,
  OUTPUT_LENGTH_CONTINUATION_PROMPT,
} from '../types/protocol';
import { isCollaborationMessage, getCollaborationId, getChangeProposalCard, showThreadReplyInMainTimeline, isToolStepStreamDelta, isReasoningStreamDelta, THINKING_ACTIVITY_DETAIL_KEY, THINKING_ACTIVITY_REASONING, THINKING_ACTIVITY_USING_TOOL, THINKING_ACTIVITY_WRITING } from '../types/protocol';
import { findThreadParentMessage } from '../utils/slackThread';
import { isSlackMirrorChannelName, showSlackHubChannelIdInHeader, slackChannelDisplayName } from '../utils/slackChannelDisplay';
import { confirmStartCollaborationWhileExecuting } from '../utils/collaborationConfirm';
import { ensureCollaborationExecutionWorkspace } from '../utils/collaborationExecutionWorkspace';
import { syncCollabTurnThinking } from '../utils/collabThinking';
import {
  appendTurnTelemetryFromAgentStatus,
  appendTurnTelemetryFromToolStep,
} from '../utils/turnTelemetry';
import { mirrorAgentCommandInTerminal } from '../utils/mirrorAgentCommandInTerminal';
import { resolveTerminalCwd } from '../utils/terminalCwd';
import { useSuggestedCommands } from '../hooks/useSuggestedCommands';
import {
  ensureRepoAgentWorkspace,
  isRepoAgentWorkspaceAction,
  parseCreateRepoAgentCommand,
} from '../utils/repoAgentWorkspace';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { useGitChangeStore } from '../stores/gitChangeStore';
import { getHubBaseURL } from '../config/hubUrl';
import { isIdeLayout, layoutPresetLabel, panelsForPreset } from '../utils/layoutPresets';
import { shrinkablePanelStyle } from '../utils/panelLayout';
import {
  mainChatMaxWidth,
  measureMainChatMaxWidth,
  type MainChatPanelVisibility,
} from '../utils/mainChatMaxWidth';
import { useHorizontalPanelResize } from '../hooks/useHorizontalPanelResize';
import { useChatShortcutHandlers } from '../hooks/useChatShortcutHandlers';
import { useChatShortcutOverlays } from '../hooks/useChatShortcutOverlays';
import { useShortcutDispatcher } from '../shortcuts/useShortcutDispatcher';
import { formatChord } from '../shortcuts/format';
import { MAX_COLLAB_AGENTS } from '../utils/collaborationLimits';
import type { LayoutPreset } from '../stores/settingsStore';
import {
  loadComposerMode,
  composerModePlaceholder,
  COMPOSER_MODE_STORAGE_KEY,
  type ComposerMode,
} from '../constants/composerMode';
import { prepareOutboundPayload } from '../utils/prepareOutboundPayload';
import { NJ_BUILD_PLAN_EVENT, buildPlanBuildMessage } from '../utils/planCard';
import { resolveWorkspaceScope, scopedRepoPaths } from '../utils/workspaceScope';
import { useProjectSetsStore } from '../stores/projectSetsStore';
import { ideRoutingChipLabel } from '../utils/ideComposer';
import { resolveEditorAgentTrust } from '../utils/editorAgentTrust';
import { registerRestartBlocker } from '../utils/restartSafety';
import {
  messageForPendingChangeId,
  oldestPendingChangeNavTarget,
  oldestPendingProposalMessage,
  pendingProposalCount,
} from '../utils/pendingChangeNavigation';
import { useRoomStore } from '../stores/roomStore';
import {
  agentsToCollaborationAgents,
  showRunbookBuilderForCollab,
  withClientPaletteCommands,
} from './chat/chatWindowHelpers';
import {
  pendingUserQuestionIds,
  pendingUserQuestionMessages,
} from '../utils/pendingUserQuestions';
import { handoffNavigationTarget } from '../utils/capabilityPolicy';

const EMPTY_THINKING_AGENTS: ThinkingAgent[] = [];

interface ChatWindowProps {
  onOpenSettings?: (tab?: import('./SettingsModal').SettingsTab) => void;
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

  const roomMode = useRoomStore((s) => s.mode);
  const activeRoomId = useRoomStore((s) => s.room?.id ?? null);
  const activeRoomJoinCode = useRoomStore((s) => s.joinCode);
  const activeRoomName = useRoomStore((s) => s.room?.name ?? null);
  const roomPresenceMembers = useRoomStore((s) => s.presenceMembers);
  const refreshRoomPresence = useRoomStore((s) => s.refreshPresence);

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

  useEffect(() => {
    if (roomMode !== 'guest' || !activeRoomId) return;
    void refreshRoomPresence();
    const id = window.setInterval(() => {
      void refreshRoomPresence();
    }, 3000);
    return () => window.clearInterval(id);
  }, [roomMode, activeRoomId, refreshRoomPresence]);

  const visibleChannels = useMemo(() => {
    if (roomMode === 'guest' && activeRoomId) {
      return channels.filter((c) => c.type === 'room' && c.room_id === activeRoomId);
    }
    return channels;
  }, [channels, roomMode, activeRoomId]);

  const activeRoomSidebarMeta = useMemo(() => {
    if (!activeRoomId) return null;
    return {
      id: activeRoomId,
      joinCode: activeRoomJoinCode || undefined,
      name: activeRoomName || undefined,
    };
  }, [activeRoomId, activeRoomJoinCode, activeRoomName]);

  const currentChannelType = useMemo(() => {
    return channels.find((c) => c.name === channel)?.type ?? null;
  }, [channels, channel]);

  const connectedRoomMembers = useMemo(() => {
    if (currentChannelType !== 'room') return [];
    return (roomPresenceMembers ?? []).filter((m: any) => m?.connected === true);
  }, [currentChannelType, roomPresenceMembers]);

  // State for tracking counts
  const [totalAgentsCount, setTotalAgentsCount] = useState(0);

  // State for file explorer and code editor panels
  const [fileExplorerOpen, setFileExplorerOpen] = useState(false);
  const [codeEditorOpen, setCodeEditorOpen] = useState(false);
  const {
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
  } = useIdeOverlayStore(
    (s) => ({
      quickOpenOpen: s.quickOpenOpen,
      symbolModalOpen: s.symbolModalOpen,
      fastEditOpen: s.fastEditOpen,
      problemsOpen: s.problemsOpen,
      gitModalOpen: s.gitModalOpen,
      setQuickOpenOpen: s.setQuickOpenOpen,
      setSymbolModalOpen: s.setSymbolModalOpen,
      setFastEditOpen: s.setFastEditOpen,
      setProblemsOpen: s.setProblemsOpen,
      setGitModalOpen: s.setGitModalOpen,
    }),
    shallow
  );
  const [activePackModal, setActivePackModal] = useState<string | null>(null);
  const phoenixModalOpen = activePackModal === 'phoenix-import';
  const roomChatModalOpen = activePackModal === 'room-chat';
  const modelArenaModalOpen = activePackModal === 'model-arena';
  const aiInterviewModalOpen = activePackModal === 'ai-interview-prep';
  const layoutProfile = usePacksStore((s) => s.layoutProfile);
  const hasIdeV2 = usePacksStore((s) => s.hasCapability('ide-v2'));
  const idePackActive = usePacksStore((s) => s.idePackActive());
  const hasIdeComposer = usePacksStore((s) => s.hasCapability('ide-v3-composer'));
  const ideLayout = layoutProfile === 'ide' && isIdeLayout(layoutSettings);
  const ideEnabled = hasIdeV2 || idePackActive;
  const ideLayoutAvailable = ideEnabled;
  const enabledPackCount = usePacksStore((s) => s.packs.filter((p) => p.enabled).length);
  const customPackToolbarActions = usePacksStore((s) => s.getToolbarActions());
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
  const { activeWorkspaceId, workspaces: explorerWorkspaces } = useFileExplorerStore(
    (s) => ({ activeWorkspaceId: s.activeWorkspaceId, workspaces: s.workspaces }),
    shallow
  );
  const openFileInEditor = useEditorStore((s) => s.openFile);
  const openArenaWorkbench = useEditorStore((s) => s.openArenaWorkbench);
  const openKnowledgeGraphWorkbench = useEditorStore((s) => s.openKnowledgeGraphWorkbench);
  const revealLineInEditor = useEditorStore((s) => s.revealLine);
  const activeEditorTab = useEditorStore((s) => {
    const id = s.activeTabId;
    return id ? (s.tabs.find((t) => t.id === id) ?? null) : null;
  });
  const editorTabs = useEditorStore((s) => s.tabs);
  const activeTabId = useEditorStore((s) => s.activeTabId);
  const activeProjectSetId = useProjectSetsStore((s) => s.activeProjectSetId);
  const getProjectSetMemberIds = useProjectSetsStore((s) => s.getMemberIds);

  const resolveScopedRepoPaths = useCallback((): string[] => {
    const scope = resolveWorkspaceScope({
      workspaces: explorerWorkspaces,
      activeWorkspaceId,
      editorTabs,
      activeTabId,
      projectSetMemberIds: activeProjectSetId
        ? getProjectSetMemberIds(activeProjectSetId)
        : undefined,
    });
    return scopedRepoPaths(scope);
  }, [
    explorerWorkspaces,
    activeWorkspaceId,
    editorTabs,
    activeTabId,
    activeProjectSetId,
    getProjectSetMemberIds,
  ]);
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

  const handleQuickOpenPath = useCallback(
    async (path: string) => {
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      try {
        const api = new ChatAPI(getHubBaseURL());
        const content = await api.fetchFileContent(ws.id, path);
        if (isEditableCsvPath(path)) {
          openFileInEditor(ws.id, path, content, 'plaintext', { viewMode: 'csv-table' });
        } else {
          openFileInEditor(ws.id, path, content, getLanguageFromPath(path));
        }
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
      const outcomeRaw = metadata?.[IMPLEMENTATION_SESSION_OUTCOME_KEY];
      if (outcomeRaw && typeof outcomeRaw === 'object') {
        const outcome = outcomeRaw as Record<string, unknown>;
        if (outcome.verify_failed === true || outcome.circuit_breaker_triggered === true) {
          const reason =
            (typeof outcome.failure_type === 'string' && outcome.failure_type) ||
            (outcome.circuit_breaker_triggered ? 'circuit breaker' : 'verify failed');
          addToast({
            type: 'warning',
            title: 'Implementation session issue',
            message: `Session finished with ${reason}. Expand the outcome card in chat for details.`,
          });
        } else if (outcome.outcome === 'wrong_route' && typeof outcome.suggested_agent === 'string') {
          addToast({
            type: 'info',
            title: 'Wrong specialist for this task',
            message: `Try ${outcome.suggested_agent} in Agent mode for boot-fix work.`,
          });
        }
      }
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
            if (isEditableCsvPath(relPath)) {
              openFileInEditor(ws.id, relPath, content, 'plaintext', { viewMode: 'csv-table' });
            } else {
              openFileInEditor(ws.id, relPath, content, getLanguageFromPath(relPath));
            }
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
      addToast,
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
        if (isEditableCsvPath(path)) {
          openFileInEditor(ws.id, path, content, 'plaintext', { viewMode: 'csv-table' });
        } else {
          openFileInEditor(ws.id, path, content, getLanguageFromPath(path));
        }
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
  const [commandDefs, setCommandDefs] = useState<CommandDefinition[]>([]);
  const [modelLibraryOpen, setModelLibraryOpen] = useState(false);
  const [modelLibraryInitialTab, setModelLibraryInitialTab] = useState<'installed' | 'ollama' | 'huggingface' | 'train' | undefined>();
  const [domainPacksOpen, setDomainPacksOpen] = useState(false);
  const [loraTrainPrefill, setLoraTrainPrefill] = useState<LoraTrainPrefill | null>(null);
  const [learningProposal, setLearningProposal] = useState<LearningProposalAction | null>(null);
  const [learningProposalOpen, setLearningProposalOpen] = useState(false);

  // State for active collaboration panel
  const [activeCollab, setActiveCollab] = useState<Collaboration | null>(null);
  const [runbookBuilderDirty, setRunbookBuilderDirty] = useState(false);
  const [runbookLibraryOpen, setRunbookLibraryOpen] = useState(false);
  const activeCollabRef = useRef<Collaboration | null>(null);
  const collaborationsByIDRef = useRef<Record<string, Collaboration>>({});
  const { handleSuggestedCommands } = useSuggestedCommands({
    collaborationsByIDRef,
    addToast,
  });
  const [taskManagementOpen, setTaskManagementOpen] = useState(false);
  const secondaryAnalysisOpen = useSecondaryAnalysisStore((s) => s.panelOpen);
  const setSecondaryAnalysisOpen = useSecondaryAnalysisStore((s) => s.setPanelOpen);
  const collaborationsByID = useCollaborationsStore((s) => s.byID);
  const setCollaborationsByID = useCollaborationsStore((s) => s.setByID);
  const mergeCollabSnapshot = useCollaborationsStore((s) => s.mergeSnapshot);
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

  const isTerminalCollaborationPhase = isTerminalCollabPhaseHelper;

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

  const [workspaceGateCollab, setWorkspaceGateCollab] = useState<Collaboration | null>(null);
  const [workspaceGateBusy, setWorkspaceGateBusy] = useState(false);
  const dismissedWorkspaceGateIdRef = useRef<string | null>(null);
  const workspaceGateToastIdRef = useRef<string | null>(null);
  const handledRepoWorkspaceActionsRef = useRef<Set<string>>(new Set());
  const handledLearningProposalsRef = useRef<Set<string>>(new Set());
  const handledChangeProposalNoticesRef = useRef<Set<string>>(new Set());
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
    if (next && workspaceGateToastIdRef.current !== next.id) {
      workspaceGateToastIdRef.current = next.id;
      addToast({
        type: 'info',
        title: 'Confirm execution workspace',
        message: 'Agents are paused until you click Continue in the workspace dialog.',
      });
    }
    if (!next) {
      workspaceGateToastIdRef.current = null;
    }
  }, [collaborationsByID, channel, addToast]);

  const channelAwaitingWorkspaceCollab = useMemo(() => {
    for (const c of Object.values(collaborationsByID)) {
      if (c.channel !== channel) continue;
      if (isAwaitingWorkspaceConfirmation(c)) return c;
    }
    return null;
  }, [collaborationsByID, channel]);

  const openWorkspaceGate = useCallback(() => {
    if (!channelAwaitingWorkspaceCollab) return;
    dismissedWorkspaceGateIdRef.current = null;
    setWorkspaceGateCollab(channelAwaitingWorkspaceCollab);
  }, [channelAwaitingWorkspaceCollab]);

  const openWorkspaceGateForCollab = useCallback((collab: Collaboration) => {
    if (!isAwaitingWorkspaceConfirmation(collab)) return;
    dismissedWorkspaceGateIdRef.current = null;
    setWorkspaceGateCollab(collab);
  }, []);

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

  const api = useMemo(() => new ChatAPI(serverAddr), [serverAddr]);
  const hubHttp = useMemo(
    () => (serverAddr.startsWith('http') ? serverAddr : `http://${serverAddr}`),
    [serverAddr]
  );
  const channelNamesKey = useMemo(
    () =>
      channels
        .map((c) => c.name)
        .slice()
        .sort()
        .join('\0'),
    [channels],
  );
  const wsURL = useMemo(() => {
    const slackExtra = channelNamesKey
      .split('\0')
      .filter((name) => name && isSlackHubChannelName(name) && name !== channel);
    return api.getWebSocketURL(channel, slackExtra);
  }, [api, channel, channelNamesKey]);
  
  // Debounce timeout ref for agent list refresh
  const agentRefreshTimeoutRef = useRef<number | null>(null);
  /** Prevents rapid DM sidebar clicks from spamming POST /api/channels/create (HTTP 429). */
  const dmCreateInFlightRef = useRef<Map<string, Promise<void>>>(new Map());
  /** Serialize all DM opens so parallel agent clicks cannot burn the mutate budget. */
  const dmOpenChainRef = useRef<Promise<void>>(Promise.resolve());
  
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
      const channelList = await new ChatAPI(getHubBaseURL()).fetchChannels();
      useChatStore.getState().setChannels(channelList);
      return channelList;
    } catch (error) {
      console.error('Failed to load channels:', error);
      return null;
    }
  }, []);

  // Pick up new Slack inbox peer channels for background WS watch.
  useEffect(() => {
    void loadChannels();
    const onSlackInboxUpdated = () => void loadChannels();
    const onRoomChannelsUpdated = () => void loadChannels();
    window.addEventListener('nj-slack-inbox-updated', onSlackInboxUpdated);
    window.addEventListener('nj-room-channels-updated', onRoomChannelsUpdated);
    const id = window.setInterval(() => void loadChannels(), 5_000);
    return () => {
      window.clearInterval(id);
      window.removeEventListener('nj-slack-inbox-updated', onSlackInboxUpdated);
      window.removeEventListener('nj-room-channels-updated', onRoomChannelsUpdated);
    };
  }, [loadChannels]);

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
    const prevSnap = collaborationsByIDSnapshot()[snapshot.id];
    const isTerminal = isTerminalCollaborationPhase(snapshot.phase);
    const wasTerminal = prevSnap && isTerminalCollaborationPhase(prevSnap.phase);

    mergeCollabSnapshot(snapshot, pruneTerminalCollaborations);

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
      if (snapshot.phase === 'reviewing') {
        useChatStore.getState().clearThinkingAgents(ch);
      } else {
        syncCollabTurnThinking(snapshot, ch);
      }
    }
  }, [addToast, mergeCollabSnapshot]);

  const clearActiveCollabIf = useCallback((collaborationID: string) => {
    setActiveCollab(current => (current?.id === collaborationID ? null : current));
  }, []);

  const loadCollaborations = useCallback(async (targetChannel: string) => {
    try {
      const includeTerminal = targetChannel.startsWith('collab-');
      const snapshots = await api.fetchCollaborations(undefined, includeTerminal);
      setCollaborationsByID((prev) => {
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
        // Keep channel snapshots when hub list is briefly stale (e.g. right after /runbook redirect).
        for (const existing of Object.values(prev)) {
          if (!existing?.id || next[existing.id]) continue;
          if (existing.channel !== targetChannel) continue;
          if (!isNonTerminalCollaborationPhase(existing.phase)) continue;
          next[existing.id] = existing;
        }
        if (includeTerminal) {
          return pruneTerminalCollaborations(next, targetChannel);
        }
        return next;
      });
      setActiveCollab(current => {
        if (!current || current.channel !== targetChannel) return current;
        const refreshed = snapshots.find(snapshot => snapshot.id === current.id);
        if (refreshed) return refreshed;
        const cached = collaborationsByIDRef.current[current.id];
        if (cached && cached.channel === targetChannel) return cached;
        return null;
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
      workspaceGateToastIdRef.current = null;
    } catch (e) {
      console.error('[workspace gate]', e);
      addToast({
        type: 'error',
        title: 'Workspace confirmation failed',
        message: e instanceof Error ? e.message : 'Could not confirm workspace',
      });
    } finally {
      setWorkspaceGateBusy(false);
    }
  }, [workspaceGateCollab, api, channel, loadCollaborations, addToast]);

  const handleWorkspaceGateDismiss = useCallback(() => {
    if (workspaceGateCollab) {
      dismissedWorkspaceGateIdRef.current = workspaceGateCollab.id;
    }
    setWorkspaceGateCollab(null);
    addToast({
      type: 'info',
      title: 'Workspace confirmation pending',
      message: 'Use the banner in the collaboration panel or chat strip when you are ready.',
    });
  }, [workspaceGateCollab, addToast]);

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
      void import('../stores/activityLogStore').then(({ logActivity }) => {
        logActivity({
          kind: 'channel',
          title: 'Switched channel',
          channel: channelName,
          detail: prevChannel ? `from #${prevChannel}` : undefined,
        });
      });
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

  const navigateToMessage = useCallback(
    async (channelName: string, messageId: string) => {
      useChatStore.getState().setPendingScrollToMessageId(messageId);
      await handleSwitchChannel(channelName);
    },
    [handleSwitchChannel]
  );

  const surfaceSlackInboundNotification = useCallback(
    (message: Message) => {
      if (!message.channel || !shouldNotifySlackInbound(message)) return;
      const label = slackChannelLabel(useChatStore.getState().channels, message.channel);
      const sender = slackInboundSenderLabel(message);
      addToast({
        type: 'info',
        variant: 'slack',
        title: sender,
        message: slackInboundPreview(message),
        duration: 8000,
        action: {
          label: `Open ${label}`,
          onClick: () => void navigateToMessage(message.channel!, message.id),
        },
      });
    },
    [addToast, navigateToMessage]
  );

  const handleNewRunbook = useCallback(async () => {
    setRunbookLibraryOpen(true);
  }, []);

  const handleCreateBlankRunbook = useCallback(async () => {
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
      const ch = useChatStore.getState().channels.find((c) => c.name === name);
      const label = ch?.type === 'collaboration' ? 'collaboration' : 'channel';
      if (!window.confirm(`Delete ${label} #${name}? This cannot be undone.`)) return;
      try {
        await api.deleteChannel(name);
        const wasActive = useChatStore.getState().channel === name;
        await loadChannels();
        if (wasActive) {
          await handleSwitchChannel('general');
        }
        setChannelInfoModal((cur) => (cur?.name === name ? null : cur));
        const { logActivity } = await import('../stores/activityLogStore');
        logActivity({
          kind: 'channel',
          title: `Deleted ${label}`,
          detail: name,
          channel: name,
        });
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
    const pending = dmCreateInFlightRef.current.get(agentId);
    if (pending) {
      await pending;
      return;
    }

    const run = (async () => {
      const openOne = async () => {
        try {
          const st = useChatStore.getState();
          const agent = st.agents.find((a) => a.id === agentId);
          if (agent) {
            const predicted = predictedDmChannelName(username, agent.name);
            const existingByName = st.channels.find((c) => c.name === predicted);
            if (existingByName) {
              await handleSwitchChannel(existingByName.name);
              return;
            }
            const existingName = dmChannelNamesForAgent(st.channels, agent)[0];
            if (existingName) {
              await handleSwitchChannel(existingName);
              return;
            }
            const byMembership = st.channels.find(
              (c) =>
                c.type === 'dm' &&
                (c.agents?.some((a) => a.id === agentId) || c.members?.includes(agentId)),
            );
            if (byMembership) {
              await handleSwitchChannel(byMembership.name);
              return;
            }
          }

          const ch = await api.openDM(agentId, username);
          const prevChannels = useChatStore.getState().channels;
          if (!prevChannels.some((c) => c.name === ch.name)) {
            useChatStore.getState().setChannels([...prevChannels, ch]);
          }
          const { settings, isLoaded } = useSettingsStore.getState();
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
          void loadChannels();
          await handleSwitchChannel(ch.name);
        } catch (error) {
          console.error('Failed to create DM channel:', error);
          const msg = error instanceof Error ? error.message : 'Failed to create DM channel.';
          addToast({
            type: 'error',
            title: 'Could not open direct message',
            message: /too many requests/i.test(msg)
              ? 'Too many channel requests — wait a few seconds and try again.'
              : msg,
          });
        }
      };

      const chained = dmOpenChainRef.current.then(openOne, openOne);
      dmOpenChainRef.current = chained.then(
        () => undefined,
        () => undefined,
      );
      await chained;
    })();

    dmCreateInFlightRef.current.set(agentId, run);
    try {
      await run;
    } finally {
      dmCreateInFlightRef.current.delete(agentId);
    }
  }, [api, username, loadChannels, handleSwitchChannel, updateSettings, addToast]);

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

  const explorerRefreshTimeoutRef = useRef<number | null>(null);
  const debouncedRefreshExplorer = useCallback(
    (message: Message) => {
      if (explorerRefreshTimeoutRef.current) {
        clearTimeout(explorerRefreshTimeoutRef.current);
      }
      explorerRefreshTimeoutRef.current = window.setTimeout(() => {
        refreshExplorerForFileChange(message);
      }, 200);
    },
    [refreshExplorerForFileChange]
  );

  const surfaceChangeProposal = useCallback(
    (message: Message, isActiveChannel: boolean) => {
      const proposal = getChangeProposalCard(message);
      if (!proposal) return;
      if (proposal.kind === 'file_change') {
        void fetchPendingChanges(username || 'default').catch((error) =>
          console.error('Failed to refresh pending file changes:', error),
        );
        if (proposal.status === 'approved') {
          debouncedRefreshExplorer(message);
        }
      } else {
        void fetchPendingGitChanges(username || 'default').catch((error) =>
          console.error('Failed to refresh pending Git changes:', error),
        );
      }
      if (
        proposal.status === 'pending' &&
        !isActiveChannel &&
        !handledChangeProposalNoticesRef.current.has(message.id)
      ) {
        handledChangeProposalNoticesRef.current.add(message.id);
        addToast({
          type: 'info',
          title: 'Change needs review',
          message: `${message.from.name} proposed a ${proposal.kind === 'file_change' ? 'file change' : 'Git operation'} in #${message.channel}.`,
        });
      }
    },
    [
      addToast,
      debouncedRefreshExplorer,
      fetchPendingChanges,
      fetchPendingGitChanges,
      username,
    ],
  );

  const jumpToOldestPendingChange = useCallback(async () => {
    if (pendingChangeCount === 0) {
      addToast({
        type: 'info',
        title: 'No pending changes',
        message: 'All proposed file and Git changes have been resolved.',
      });
      return;
    }

    const pendingIds = new Set([
      ...pendingChanges.map((change) => change.id),
      ...pendingGitChanges.map((change) => change.id),
    ]);
    const navTarget = oldestPendingChangeNavTarget([
      ...pendingChanges,
      ...pendingGitChanges,
    ]);

    const focusProposal = (messages: Message[]) => {
      const byId = navTarget
        ? messageForPendingChangeId(messages, navTarget.id)
        : null;
      const target =
        byId ?? oldestPendingProposalMessage(messages, pendingIds);
      if (!target) return false;
      const store = useChatStore.getState();
      store.setPendingScrollToMessageId(target.id);
      store.setHighlightMessageId(target.id);
      return true;
    };

    if (focusProposal(useChatStore.getState().messages)) {
      return;
    }

    if (!navTarget?.channel) {
      addToast({
        type: 'info',
        title: 'Pending change unavailable',
        message: 'A change is pending but its chat channel is unknown.',
      });
      return;
    }

    await handleSwitchChannel(navTarget.channel);

    let messages = useChatStore.getState().messages;
    if (!focusProposal(messages)) {
      try {
        messages = await api.fetchMessages(navTarget.channel, 200);
        useChatStore.getState().setMessages(messages);
      } catch {
        // Fall through to toast below.
      }
    }

    if (focusProposal(messages)) {
      return;
    }

    addToast({
      type: 'info',
      title: `Opened #${navTarget.channel}`,
      message: 'Switched to the chat with the pending change. Scroll to find the approval card.',
    });
  }, [
    addToast,
    api,
    handleSwitchChannel,
    pendingChangeCount,
    pendingChanges,
    pendingGitChanges,
  ]);

  const scrollToApproval = useCallback((approvalId: string) => {
    const el = document.querySelector(`[data-approval-id="${approvalId}"]`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      return;
    }
    chatScrollerElRef.current?.scrollTo({
      top: chatScrollerElRef.current.scrollHeight,
      behavior: 'smooth',
    });
  }, []);

  const surfaceToolApproval = useCallback(
    (message: Message, isActiveChannel: boolean) => {
      const approvalId = message.metadata?.approval_id as string | undefined;
      const status = message.metadata?.status as string | undefined;
      if (!approvalId) return;

      const toolName = (message.metadata?.tool_name as string) || 'tool';
      const toolInput = (message.metadata?.tool_input as Record<string, unknown>) || {};
      const msgChannel = message.channel || useChatStore.getState().channel;

      if (status === 'pending') {
        useApprovalStore.getState().upsertPendingTool({
          id: approvalId,
          agentId: message.from.id,
          agentName: message.from.name,
          toolName,
          toolInput,
          channel: msgChannel,
          messageId: message.id,
          createdAt: message.timestamp,
        });
        const summary = formatToolApprovalSummary({
          id: approvalId,
          agentId: message.from.id,
          agentName: message.from.name,
          toolName,
          toolInput,
          channel: msgChannel,
          createdAt: message.timestamp,
        });
        addToast({
          type: 'warning',
          title: `${message.from.name} needs your approval`,
          message: isActiveChannel
            ? summary
            : `Waiting in #${msgChannel} — ${summary}`,
          duration: 0,
          action: isActiveChannel
            ? {
                label: 'Review now',
                onClick: () => scrollToApproval(approvalId),
              }
            : {
                label: `Open #${msgChannel}`,
                onClick: () => useChatStore.getState().setChannel(msgChannel),
              },
        });
      } else {
        useApprovalStore.getState().removePendingTool(approvalId);
      }
    },
    [addToast, scrollToApproval],
  );

  const handledHandoffMessagesRef = useRef(new Set<string>());

  // WebSocket connection
  const { status, connect: reconnectHub } = useWebSocket({
    url: wsURL,
    onMessage: async (message: Message) => {
      try {
        const st = useChatStore.getState();
        const activeChannel = st.channel;

      if (
        message.type === 'system_info' &&
        !handledHandoffMessagesRef.current.has(message.id)
      ) {
        const target = handoffNavigationTarget(message.metadata);
        if (target) {
          handledHandoffMessagesRef.current.add(message.id);
          const channels = await loadChannels();
          if (
            channels?.some((candidate) => candidate.name === target) &&
            useChatStore.getState().channel !== target
          ) {
            await handleSwitchChannel(target);
            return;
          }
        }
      }

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
          const pendingAsks = pendingUserQuestionMessages(st.messages).length > 0;
          if (thinkingStatus === 'started') {
            // Ignore "responding" while the channel is waiting on ask_user.
            if (!(pendingAsks && msgChannel === st.channel)) {
            const activity =
              typeof message.metadata.thinking_activity === 'string'
                ? message.metadata.thinking_activity
                : undefined;
            const activityDetail =
              typeof message.metadata[THINKING_ACTIVITY_DETAIL_KEY] === 'string'
                ? (message.metadata[THINKING_ACTIVITY_DETAIL_KEY] as string)
                : typeof message.metadata.thinking_activity_detail === 'string'
                  ? message.metadata.thinking_activity_detail
                  : undefined;
            st.addThinkingAgent(
              msgChannel,
              message.from.id,
              message.from.name,
              message.from.type,
              activity,
              activityDetail
            );
            if (message.from.id !== NJ_PENDING_SEND_AGENT_ID) {
              clearPendingSendThinking(msgChannel);
            }
            if (
              msgChannel !== activeChannel &&
              msgChannel.startsWith('collab-')
            ) {
              st.addThinkingAgent(
                activeChannel,
                message.from.id,
                message.from.name,
                message.from.type,
                activity,
                activityDetail
              );
              if (message.from.id !== NJ_PENDING_SEND_AGENT_ID) {
                clearPendingSendThinking(activeChannel);
              }
            }
            appendTurnTelemetryFromAgentStatus(msgChannel, message);
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
        const streamMeta = message.metadata ?? {};
        const agentChannel = message.channel || activeChannel;
        if (isToolStepStreamDelta(streamMeta)) {
          st.updateThinkingAgentActivity(agentChannel, message.from.id, {
            activity: THINKING_ACTIVITY_USING_TOOL,
            activityDetail:
              typeof streamMeta.tool_preview === 'string' ? streamMeta.tool_preview : undefined,
            toolStep: {
              kind: String(streamMeta.tool_step ?? ''),
              name: String(streamMeta.tool_name ?? ''),
              iteration:
                typeof streamMeta.tool_iteration === 'number' ? streamMeta.tool_iteration : undefined,
              preview:
                typeof streamMeta.tool_preview === 'string' ? streamMeta.tool_preview : undefined,
            },
          });
          appendTurnTelemetryFromToolStep(agentChannel, message);
        } else if (isReasoningStreamDelta(streamMeta)) {
          st.updateThinkingAgentActivity(agentChannel, message.from.id, {
            activity: THINKING_ACTIVITY_REASONING,
          });
        } else if ((message.content ?? '').length > 0) {
          st.updateThinkingAgentActivity(agentChannel, message.from.id, {
            activity: THINKING_ACTIVITY_WRITING,
          });
        }
        if (streamOnMainTimeline) {
          st.appendStreamDelta(message);
        }
        return;
      }
      if (message.type === 'stream_end') {
        if (streamOnMainTimeline) {
          st.finalizeStream(message.id, message.metadata as Record<string, unknown> | undefined);
        }
        st.removeThinkingAgent(message.channel || activeChannel, message.from.id);
        return;
      }

      const collabData = message.metadata?.collaboration_data as Collaboration | undefined;
      if (collabData?.id) {
        startTransition(() => {
          const collabChannel = collabData.channel || message.channel;
          const isActiveChannelCollab = !collabChannel || collabChannel === activeChannel;
          const previousSnapshot = collaborationsByIDRef.current[collabData.id];
          if (
            shouldToastCollaboratorAdds({
              previous: previousSnapshot,
              snapshot: collabData,
              isActiveChannelCollab,
            })
          ) {
            const addedAgents = collaboratorsAddedSince(previousSnapshot, collabData);
            const names = addedAgents.map((a) => `@${a.agent_name}`).join(', ');
            addToast({
              type: 'info',
              title: 'Collaborator added',
              message: `${names} joined "${collabData.title}".`,
            });
          }
          mergeCollaborationSnapshot(collabData);
          const decision = decideCollabPanelOpen({
            snapshot: collabData,
            activeChannel,
            currentlyOpen: activeCollabRef.current,
            message,
          });
          if (decision.action === 'open' || decision.action === 'update_open') {
            setActiveCollab(decision.snapshot);
          }
        });
      }

      const participantReq = parseCollabParticipantAddRequest(message);
      if (participantReq) {
        const { collabID, agentID, agentName, requestedBy } = participantReq;
        const key = `${collabID}:${agentID}:${message.id}`;
        if (!handledParticipantRequestPromptsRef.current.has(key)) {
          handledParticipantRequestPromptsRef.current.add(key);
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
            if (shouldNotifySlackInbound(message)) {
              surfaceSlackInboundNotification(message);
            }
          }
        }
        return;
      } else if (message.channel && message.channel !== activeChannel) {
        // Message belongs to a different channel -- cache it and mark unread
        st.addMessageToCache(message.channel, message);
        st.markChannelUnread(message.channel);
        if (shouldNotifySlackInbound(message)) {
          surfaceSlackInboundNotification(message);
        }
        if (isCollaborationMessage(message) || getCollaborationId(message)) {
          addToast({
            type: 'info',
            title: 'Collaboration update',
            message: `Activity in #${message.channel} — switch there to see messages.`,
          });
        }
        if (getChangeProposalCard(message)) {
          surfaceChangeProposal(message, false);
        }
        if (message.type === 'tool_approval') {
          surfaceToolApproval(message, false);
          st.addMessageToCache(message.channel, message);
        }
        if (message.type === 'user_question') {
          st.addMessageToCache(message.channel, message);
          if (message.metadata?.status === 'pending') {
            st.clearThinkingAgents(message.channel);
            st.stopAllStreamsForChannel(message.channel);
          }
        }
        if (message.type === 'command_output') {
          mirrorAgentCommandInTerminal(message);
        }
      } else {
        if (message.type === 'tool_approval') {
          surfaceToolApproval(message, true);
          st.upsertToolApprovalMessage(message);
        } else if (message.type === 'user_question') {
          st.upsertUserQuestionMessage(message);
          if (message.metadata?.status === 'pending') {
            const qChannel = message.channel || activeChannel;
            st.clearThinkingAgents(qChannel);
            st.stopAllStreamsForChannel(qChannel);
            window.setTimeout(() => inputRef.current?.focus(), 0);
          }
        } else {
        // Message belongs to the active channel (never wrap addMessage in startTransition —
        // high-frequency agent_status updates can starve transitions and leave the chat empty).
        st.addMessage(message);
        }

        if (message.metadata?.[IMPLEMENTATION_SESSION_COMPLETE_KEY] === true) {
          void handleImplementationSessionComplete(
            message.metadata as Record<string, unknown> | undefined
          );
        }

        if (message.metadata?.[CAD_FILES_WRITTEN_KEY]) {
          void handleCADFilesWritten(message.metadata as Record<string, unknown> | undefined);
        }

        handleSuggestedCommands(message, activeChannel);

        if (message.type === 'command_output') {
          mirrorAgentCommandInTerminal(message);
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

        if (getChangeProposalCard(message)) {
          surfaceChangeProposal(message, true);
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
        clearPendingSendThinking(ch);
        if (ch !== activeChannel) {
          st.removeThinkingAgent(activeChannel, message.from.id);
          clearPendingSendThinking(activeChannel);
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

    void useApprovalStore.getState().syncPendingFromHub(api);
    void useLocationShareStore.getState().syncPending(api);

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
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      const payload = await prepareOutboundPayload({
        content,
        composerMode,
        agents,
        activeTab: activeEditorTab,
        editorAgentTrust: resolveEditorAgentTrust(layoutSettings, composerMode),
        composerMetadata: metadata,
        api: ideEnabled ? api : undefined,
        repoPath: ideEnabled ? ws?.path : undefined,
        repoPaths: ideEnabled ? resolveScopedRepoPaths() : undefined,
        ideEnabled,
        channel,
        channelMeta: activeChannelMeta,
      });
      const baseMetadata = buildHumanOutboundMetadata({
        contextMode: workspaceContextMode,
        conversationMode: conversationModeSetting,
        message: payload.content,
        channel,
        channelType: activeChannelMeta?.type,
        composerMetadata: payload.metadata,
        ideCoding: ideLayout && hasIdeComposer,
        recentChannelMessages: useChatStore.getState().messages,
      });
      const mergedMetadata =
        workspaceContextMode === 'off'
          ? baseMetadata
          : await attachAmbientStateMetadata(
              baseMetadata,
              payload.content,
              ideLayout && hasIdeComposer,
            );
      await api.sendThreadReply(
        threadId,
        channel,
        payload.content,
        { name: username, type: 'human' },
        mergedMetadata
      );
    },
    [
      api,
      channel,
      username,
      workspaceContextMode,
      conversationModeSetting,
      activeChannelMeta?.type,
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
    ]
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

  const appendLocalSlashCommand = useCallback(
    (commandText: string) => {
      const now = new Date().toISOString();
      useChatStore.getState().addMessage({
        id: `local-cmd-${Date.now()}`,
        type: 'question',
        channel,
        from: {
          id: username || 'user',
          name: username || 'You',
          type: 'human',
          expertise: [],
          status: 'active',
          model: '',
          is_paused: false,
        },
        content: commandText,
        timestamp: now,
        metadata: { slash_command: true, client_only: true },
      });
    },
    [channel, username],
  );

  const dispatchMessage = useCallback(
    async (
      content: string,
      metadata?: Record<string, unknown>,
      modeOverride?: ComposerMode,
    ): Promise<boolean> => {
      useChatStore.getState().setChannelHold(channel, false);

      let sendContent = content;
      let composerMeta = metadata ?? {};
      const effectiveComposerMode = modeOverride ?? composerMode;
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      const payload = await prepareOutboundPayload({
        content,
        composerMode: effectiveComposerMode,
        agents,
        activeTab: activeEditorTab,
        editorAgentTrust: resolveEditorAgentTrust(layoutSettings, effectiveComposerMode),
        composerMetadata: composerMeta,
        api: ideEnabled ? api : undefined,
        repoPath: ideEnabled ? ws?.path : undefined,
        repoPaths: ideEnabled ? resolveScopedRepoPaths() : undefined,
        ideEnabled,
        channel,
        channelMeta: activeChannelMeta,
      });
      sendContent = payload.content;
      composerMeta = payload.metadata;

      const baseMetadata = buildHumanOutboundMetadata({
        contextMode: workspaceContextMode,
        conversationMode: conversationModeSetting,
        message: sendContent,
        channel,
        channelType: activeChannelMeta?.type,
        composerMetadata: composerMeta,
        ideCoding: ideLayout && hasIdeComposer,
        recentChannelMessages: useChatStore.getState().messages,
      });
      const mergedMetadata =
        workspaceContextMode === 'off'
          ? baseMetadata
          : await attachAmbientStateMetadata(
              baseMetadata,
              sendContent,
              ideLayout && hasIdeComposer,
            );

      useChatStore.getState().setIsTyping(true);
      markPendingSendThinking(channel);
      try {
        const trimmed = sendContent.trimStart();
        const slashCommand = trimmed.startsWith('/');
        if (trimmed.startsWith('/collaborate')) {
          if (!confirmStartCollaborationWhileExecuting(executingCollaborationForChannel)) {
            clearPendingSendThinking(channel);
            return false;
          }
        }
        if (slashCommand) {
          appendLocalSlashCommand(sendContent.trim());
        }

        let sendResult;
        const from = { name: username, type: 'human' };
        // Slash commands stay on /api/send; semantic turns use prepare/fetch/dispatch.
        if (slashCommand || workspaceContextMode === 'off') {
          sendResult = await api.sendMessage(channel, sendContent, from, 'question', mergedMetadata);
        } else {
          try {
            const prepareMeta = { ...mergedMetadata };
            // Prepare envelope: identity + tree only; bodies come after context_request.
            if (prepareMeta.workspace_context && typeof prepareMeta.workspace_context === 'object') {
              const ws = { ...(prepareMeta.workspace_context as Record<string, unknown>) };
              ws.open_files = [];
              prepareMeta.workspace_context = ws;
              prepareMeta.context_scope = 'hint';
              prepareMeta.context_scope_reason = 'prepare envelope — structural availability';
            }
            const prepared = await api.prepareTurn(
              channel,
              sendContent,
              from,
              'question',
              prepareMeta,
            );
            const { primary } = loadScopedWorkspaceContext();
            const activePath = useEditorStore.getState().tabs.find(
              (t) => t.id === useEditorStore.getState().activeTabId,
            )?.path;
            const dispatchMeta = await applyContextRequestToMetadata({
              api,
              metadata: mergedMetadata ?? {},
              message: sendContent,
              contextRequest: prepared.context_request ?? {},
              prepareToken: prepared.prepare_token,
              fullWorkspace: primary,
              activeTabPath: activePath,
            });
            const req = prepared.context_request ?? {};
            const withAmbient =
              req.include_git_status || req.include_diagnostics
                ? await attachAmbientStateMetadata(
                    dispatchMeta,
                    sendContent,
                    ideLayout && hasIdeComposer,
                    {
                      force: true,
                      includeGit: Boolean(req.include_git_status),
                      includeDiagnostics: Boolean(req.include_diagnostics),
                    },
                  )
                : dispatchMeta;
            sendResult = await api.dispatchTurn(
              channel,
              sendContent,
              from,
              'question',
              withAmbient,
            );
          } catch (prepareErr) {
            console.warn('[dispatchMessage] prepare/dispatch fallback to /api/send', prepareErr);
            sendResult = await api.sendMessage(
              channel,
              sendContent,
              from,
              'question',
              mergedMetadata,
            );
          }
        }
        let timelineChannel = channel;
        if (sendResult.collaboration_channel) {
          clearPendingSendThinking(channel);
          markPendingSendThinking(sendResult.collaboration_channel);
          await loadChannels();
          await handleSwitchChannel(sendResult.collaboration_channel);
          timelineChannel = sendResult.collaboration_channel;
          await loadCollaborations(timelineChannel);
          let collab =
            (sendResult.collaboration_id
              ? collaborationsByIDRef.current[sendResult.collaboration_id]
              : undefined) ??
            Object.values(collaborationsByIDRef.current).find(
              (c) => c.channel === sendResult.collaboration_channel
            );
          if (!collab && sendResult.collaboration_id) {
            try {
              collab = await api.getRunbook(sendResult.collaboration_id);
            } catch (e) {
              console.error('[dispatchMessage] failed to load runbook after redirect:', e);
            }
          }
          await loadCollaborations(timelineChannel);
          if (collab) {
            mergeCollaborationSnapshot(collab);
            setActiveCollab(collab);
            syncCollabTurnThinking(collab, sendResult.collaboration_channel);
          }
        }
        if (sendResult.dm_channel) {
          clearPendingSendThinking(channel);
          const dmName = sendResult.dm_channel;
          markPendingSendThinking(dmName);
          await loadAgents();
          await loadChannels();
          const channelList = useChatStore.getState().channels;
          const { settings, isLoaded } = useSettingsStore.getState();
          if (isLoaded) {
            const patch = patchRevealForChannel(
              settings,
              dmName,
              channelList,
              useChatStore.getState().agents
            );
            if (patch) {
              void updateSettings(patch);
            }
          }
          await handleSwitchChannel(dmName);
          timelineChannel = dmName;
          addToast({
            type: 'success',
            title: 'Expert ready',
            message: 'Opened the new expert direct message.',
          });
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
        // Keep the pending row briefly after HTTP returns — classify may finish before
        // the agent emits thinking_status. Clear when real thinking arrives (below) or
        // after this safety timeout.
        // Keep the pending row briefly after HTTP returns — classify may finish before
        // the agent emits thinking_status. Clear when real thinking arrives or timeout.
        window.setTimeout(() => {
          clearPendingSendThinking(timelineChannel);
        }, 20_000);
        return true;
      } catch (error) {
        console.error('Failed to send message:', error);
        clearPendingSendThinking(channel);
        addToast({
          type: 'error',
          title: 'Message not sent',
          message: error instanceof Error ? error.message : 'Failed to send message.',
        });
        return false;
      } finally {
        useChatStore.getState().setIsTyping(false);
      }
    },
    [
      api,
      channel,
      username,
      appendLocalSlashCommand,
      workspaceContextMode,
      conversationModeSetting,
      activeChannelMeta?.type,
      loadChannels,
      loadAgents,
      handleSwitchChannel,
      loadCollaborations,
      mergeCollaborationSnapshot,
      executingCollaborationForChannel,
      addToast,
      updateSettings,
      ideLayout,
      ideEnabled,
      agents,
      activeEditorTab,
      composerMode,
      hasIdeComposer,
      explorerWorkspaces,
      activeWorkspaceId,
      resolveScopedRepoPaths,
    ]
  );

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
    const trimmed = commandString.trim();
    if (trimmed === '/nj-open-model-library') {
      appendLocalSlashCommand(trimmed);
      setModelLibraryOpen(true);
      (inputRef.current as (HTMLTextAreaElement & { clearInput?: () => void }) | null)?.clearInput?.();
      return;
    }
    if (trimmed === '/nj-open-knowledge-graph') {
      appendLocalSlashCommand(trimmed);
      const fe = useFileExplorerStore.getState();
      const ws = fe.workspaces.find((w) => w.id === fe.activeWorkspaceId);
      if (ws?.path && fe.activeWorkspaceId) {
        useEditorStore.getState().openKnowledgeGraphWorkbench(fe.activeWorkspaceId, ws.path);
        setCodeEditorOpen(true);
        void updateLayoutSettings({ editorPanelVisible: true });
      } else {
        addToast({
          type: 'warning',
          title: 'No workspace',
          message: 'Open a workspace in the Files panel first.',
        });
      }
      (inputRef.current as (HTMLTextAreaElement & { clearInput?: () => void }) | null)?.clearInput?.();
      return;
    }
    if (trimmed === '/nj-open-neural-canvas') {
      appendLocalSlashCommand(trimmed);
      const fe = useFileExplorerStore.getState();
      useEditorStore.getState().openArtifact(
        fe.activeWorkspaceId ?? '',
        '__library__',
        'Neural Canvas',
      );
      setCodeEditorOpen(true);
      void updateLayoutSettings({ editorPanelVisible: true });
      (inputRef.current as (HTMLTextAreaElement & { clearInput?: () => void }) | null)?.clearInput?.();
      return;
    }
    const repoAgentCmd = parseCreateRepoAgentCommand(trimmed);
    const sent = await handleSendMessage(commandString, metadata);
    if (sent !== false) {
      (inputRef.current as (HTMLTextAreaElement & { clearInput?: () => void }) | null)?.clearInput?.();
    }
    if (repoAgentCmd && sent !== false) {
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

  // Open command palette from toolbar button or Cmd/Ctrl+Shift+P
  const openCommandPalette = useCallback((filter = '') => {
    setCommandPaletteFilter(filter);
    setCommandPaletteOpen(true);
    void ensureCommandDefs(true);
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
  }, [api, channel, ensureCommandDefs, fetchPendingChanges, loadCollaborations, username]);

  const handleFirstWinOpenFiles = useCallback(() => {
    setFileExplorerOpen(true);
  }, []);

  const handleFirstWinPrefill = useCallback((text: string) => {
    useComposerPrefillStore.getState().requestPrefill(text);
  }, []);

  const handleFirstWinOpenModelLibrary = useCallback(() => {
    setModelLibraryOpen(true);
  }, []);

  useShortcutDispatcher(true);

  const handleOpenSettings = useCallback(
    (tab?: import('./SettingsModal').SettingsTab | 'domain-packs') => {
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
      if (
        runbookBuilderDirty &&
        activeCollab.source === 'runbook' &&
        (activeCollab.phase === 'draft' || activeCollab.phase === 'reviewing')
      ) {
        return;
      }
      const latest = collaborationsByIDRef.current[activeCollab.id];
      if (!latest || !isNonTerminalCollaborationPhase(latest.phase)) {
        return;
      }
      void loadCollaborations(targetChannel);
    };
    tick();
    const id = window.setInterval(tick, 10_000);
    return () => window.clearInterval(id);
  }, [activeCollab?.id, activeCollab?.channel, activeCollab?.phase, activeCollab?.source, channel, loadCollaborations, runbookBuilderDirty]);

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
                {ch && showSlackHubChannelIdInHeader(ch.name) && (
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
        
        <div className="flex items-center gap-1.5 shrink min-w-0 max-w-[min(100%,72rem)] justify-end" aria-label="Sidebar toggles">
          <div className="flex shrink-0 items-center gap-1.5">
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
            <button
              type="button"
              onClick={() => void updateLayoutSettings({ chatPanelVisible: !chatPanelVisible })}
              className={`w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 ${
                chatPanelVisible
                  ? 'bg-slack-accent text-white'
                  : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text hover:bg-slack-border'
              }`}
              title={`${chatPanelVisible ? 'Hide main chat' : 'Show main chat'} (${formatChord('mod+shift+c')})`}
              aria-label={chatPanelVisible ? 'Hide main chat panel' : 'Show main chat panel'}
              aria-pressed={chatPanelVisible}
            >
              <ChatPanelIcon className="w-3.5 h-3.5" />
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
          </div>
          {showTopToolbarChips && (
            <>
              <div className="w-px h-5 bg-slack-border shrink-0" />
              <div className="flex min-w-0 overflow-x-auto overflow-y-visible">
                <ChatToolbarActions layout="horizontal" {...toolbarActionsProps} />
              </div>
            </>
          )}
        </div>
      </div>

      <PendingApprovalsBar
        channel={channel}
        api={api}
        collaboration={collaborationForChannel}
        onOpenTerminal={() => setPanelOpen(true)}
        onScrollToApproval={scrollToApproval}
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

        {/* Workspace slot — editor stays visible when chat is hidden (IDE layout).
            Do not gate on chatPanelVisible: that incorrectly unmounted the editor with chat. */}
        {(ideLayout || chatResizable) && (
          <div
            data-shrinkable-workspace
            className="flex flex-1 min-w-0 min-h-0 flex-col h-full border-r border-slack-border"
          >
            {codeEditorOpen ? (
              <CodeEditorPanel
                variant="embedded"
                onClose={() => {
                  setCodeEditorOpen(false);
                  void updateLayoutSettings({ editorPanelVisible: false });
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
          <CodeEditorPanel
            variant="overlay"
            onClose={() => {
              setCodeEditorOpen(false);
              void updateLayoutSettings({ editorPanelVisible: false });
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
          onOpenAgentDM={handleCreateDM}
          onPrefillComposer={handleFirstWinPrefill}
          onOpenModelLibrary={handleFirstWinOpenModelLibrary}
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
          <RunbookBuilderPanel
            collaboration={panelCollaboration}
            hubAgents={agentsToCollaborationAgents(agents)}
            onClose={() => {
              setRunbookBuilderDirty(false);
              setActiveCollab(null);
            }}
            onDirtyChange={setRunbookBuilderDirty}
            onSaved={(snap) => {
              mergeCollaborationSnapshot(snap);
              setActiveCollab(snap);
              void loadCollaborations(snap.channel || channel);
            }}
            onStarted={(snap) => {
              mergeCollaborationSnapshot(snap);
              setActiveCollab(snap);
              void loadCollaborations(snap.channel || channel);
            }}
            onWorkspaceGateRequest={openWorkspaceGateForCollab}
          />
        ) : panelCollaboration ? (
          <CollaborationPanel
            collaboration={panelCollaboration}
            extendableCollaborations={extendableCollaborations}
            executingCollaboration={executingCollaborationForChannel}
            onClose={() => setActiveCollab(null)}
            onConfirmWorkspace={openWorkspaceGate}
            onAfterCollaborationCommand={async () => {
              await loadCollaborations(panelCollaboration.channel || channel);
            }}
            onOpenRunbookRun={async (collabId, collabChannel) => {
              if (collabChannel && collabChannel !== channel) {
                await handleSwitchChannel(collabChannel);
              }
              try {
                setActiveCollab(await api.getRunbook(collabId));
              } catch {
                /* optional snapshot */
              }
              await loadCollaborations(collabChannel || channel);
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
        <TerminalPanel
          channel={channel}
          api={api}
          collaboration={collaborationForChannel}
        />
      </div>
      
      <GitModal isOpen={gitModalOpen && ideEnabled} onClose={() => setGitModalOpen(false)} />

      <QuickOpenModal
        isOpen={quickOpenOpen && ideEnabled}
        workspaceId={
          explorerWorkspaces.find((w) => w.id === activeWorkspaceId)?.id ??
          explorerWorkspaces[0]?.id
        }
        onClose={() => setQuickOpenOpen(false)}
        onOpenPath={handleQuickOpenPath}
      />

      <SymbolModal
        isOpen={symbolModalOpen && ideEnabled}
        workspaceId={
          explorerWorkspaces.find((w) => w.id === activeWorkspaceId)?.id ??
          explorerWorkspaces[0]?.id
        }
        onClose={() => setSymbolModalOpen(false)}
        onOpenSymbol={handleOpenAtLine}
      />

      <ProblemsPanel
        isOpen={problemsOpen && ideEnabled}
        onClose={() => setProblemsOpen(false)}
        onOpenAt={handleOpenAtLine}
      />

      <FastEditModal
        isOpen={fastEditOpen && ideEnabled}
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
        activeChannel={channel}
        collaborations={trackedCollaborations}
        assistantTasks={assistantTasks}
        pendingChanges={pendingChanges}
        api={api}
        isOpen={commandPaletteOpen}
        initialFilter={commandPaletteFilter}
        onClose={closeCommandPalette}
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
          api={api}
          onClose={() => setChannelInfoModal(null)}
          onClearHistory={async (name) => {
            await api.clearChannelHistory(name);
            const msgs = await api.fetchMessages(name, 50);
            const st = useChatStore.getState();
            st.replaceChannelMessagesCache(name, msgs);
            if (name === st.channel) {
              st.setMessages(msgs);
              st.cleanupStaleThinking(name, msgs);
            }
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
        onClose={closeModelLibrary}
        serverAddr={hubHttp}
        switchAllAgentProviders={switchAllAgentProviders}
        switchAgentProvider={switchAgentProvider}
        runtimeAgents={agents.map((a) => ({ id: a.id, name: a.name, type: a.type }))}
        defaultChannel={channel}
        initialTab={modelLibraryInitialTab}
        loraTrainPrefill={loraTrainPrefill}
      />

      <DomainPacksModal
        isOpen={domainPacksOpen}
        onClose={closeDomainPacks}
        serverAddr={hubHttp}
      />

      <PhoenixBrowserModal isOpen={phoenixModalOpen} onClose={() => setActivePackModal(null)} />
      <RoomChatModal isOpen={roomChatModalOpen} onClose={() => setActivePackModal(null)} />
      <ModelArenaModal
        isOpen={modelArenaModalOpen}
        onClose={() => setActivePackModal(null)}
        onOpenInEditor={(workspaceId) => {
          openArenaWorkbench(workspaceId, 'arena/model-arena.nj-arena.json');
          setCodeEditorOpen(true);
          void updateLayoutSettings({ editorPanelVisible: true });
        }}
      />
      <AIInterviewPrepModal
        isOpen={aiInterviewModalOpen}
        onClose={() => setActivePackModal(null)}
      />

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
            if (stats.suggest_training || stats.ready_for_lora) {
              addToast({
                type: 'info',
                title: 'Sharpen expert',
                message: `10+ turns with ${learningProposal?.agent_name ?? 'this expert'} — open agent info to sharpen.`,
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

      <RunbookLibraryModal
        isOpen={runbookLibraryOpen}
        api={api}
        hubAgents={agentsToCollaborationAgents(agents)}
        channel={channel}
        username={username || 'User'}
        onClose={() => setRunbookLibraryOpen(false)}
        onInstantiated={(collabId, collabChannel) => {
          void (async () => {
            if (collabChannel && collabChannel !== channel) {
              await handleSwitchChannel(collabChannel);
            }
            try {
              setActiveCollab(await api.getRunbook(collabId));
            } catch {
              /* snapshot optional */
            }
            void loadCollaborations(channel);
          })();
        }}
        onNewBlank={() => void handleCreateBlankRunbook()}
      />

      {/* Toast Notifications */}
      <ToastContainer />
      </div>
    </ErrorBoundary>
  );
}
