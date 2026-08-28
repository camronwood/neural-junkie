import type { MutableRefObject } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { useChatStore } from '../stores/chatStore';
import { useEditorStore } from '../stores/editorStore';
import { useSettingsStore } from '../stores/settingsStore';
import type { Toast } from '../stores/toastStore';
import type {
  AgentInfo,
  Channel,
  Collaboration,
} from '../types/protocol';
import type { ConversationModeSetting, WorkspaceContextMode } from '../constants/promptMetadata';
import type { ComposerMode } from '../constants/composerMode';
import type { LayoutSettings } from '../stores/settingsStore';
import type { EditorTab } from '../stores/editorStore';
import type { Workspace } from '../stores/fileExplorerStore';
import {
  buildHumanOutboundMetadata,
  loadScopedWorkspaceContext,
} from '../utils/outboundChatMetadata';
import { attachAmbientStateMetadata } from '../utils/ambientState';
import { applyContextRequestToMetadata } from '../utils/contextRequestAttach';
import {
  clearPendingSendThinking,
  markPendingSendThinking,
} from '../utils/pendingSendThinking';
import { confirmStartCollaborationWhileExecuting } from '../utils/collaborationConfirm';
import { syncCollabTurnThinking } from '../utils/collabThinking';
import { prepareOutboundPayload } from '../utils/prepareOutboundPayload';
import { resolveEditorAgentTrust } from '../utils/editorAgentTrust';
import { patchRevealForChannel } from '../utils/sidebarVisibility';
import { GRANTED_HUB_DATA_ACCESS_KEY } from '../constants/promptMetadata';
import type { HubDataAccessOption } from '../utils/hubDataAccess';

export type HubAccessPendingState = {
  mode: 'main' | 'thread';
  threadId?: string;
  content: string;
  metadata?: Record<string, unknown>;
  options: HubDataAccessOption[];
};

export type ChatOutboundDispatchDeps = {
  api: ChatAPI;
  channel: string;
  username: string;
  workspaceContextMode: WorkspaceContextMode;
  conversationModeSetting: ConversationModeSetting;
  activeChannelMeta: Channel | undefined;
  composerMode: ComposerMode;
  agents: AgentInfo[];
  activeEditorTab: EditorTab | null;
  layoutSettings: LayoutSettings;
  ideEnabled: boolean;
  explorerWorkspaces: Workspace[];
  activeWorkspaceId: string | null;
  resolveScopedRepoPaths: () => string[];
  ideLayout: boolean;
  hasIdeComposer: boolean;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
  loadChannels: () => Promise<unknown>;
  loadAgents: () => Promise<unknown>;
  handleSwitchChannel: (target: string) => Promise<void>;
  loadCollaborations: (targetChannel: string) => Promise<unknown>;
  mergeCollaborationSnapshot: (snapshot: Collaboration) => void;
  setActiveCollab: React.Dispatch<React.SetStateAction<Collaboration | null>>;
  executingCollaborationForChannel: Collaboration | null;
  updateSettings: (patch: Partial<import('../stores/settingsStore').Settings>) => Promise<void>;
  collaborationsByIDRef: MutableRefObject<Record<string, Collaboration>>;
  hubAccessPending: HubAccessPendingState | null;
  setHubAccessPending: React.Dispatch<React.SetStateAction<HubAccessPendingState | null>>;
  setHubAccessLoading: React.Dispatch<React.SetStateAction<boolean>>;
  setHubAccessError: React.Dispatch<React.SetStateAction<string | null>>;
};

/** Outbound send/interject handlers extracted from ChatWindow. */
export function createChatOutboundDispatch(deps: ChatOutboundDispatchDeps) {
  const appendLocalSlashCommand = (commandText: string) => {
    const now = new Date().toISOString();
    useChatStore.getState().addMessage({
      id: `local-cmd-${Date.now()}`,
      type: 'question',
      channel: deps.channel,
      from: {
        id: deps.username || 'user',
        name: deps.username || 'You',
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
  };

  const dispatchThreadReply = async (
    threadId: string,
    content: string,
    metadata?: Record<string, unknown>,
  ) => {
    const ws =
      deps.explorerWorkspaces.find((w) => w.id === deps.activeWorkspaceId) ??
      deps.explorerWorkspaces[0];
    const payload = await prepareOutboundPayload({
      content,
      composerMode: deps.composerMode,
      agents: deps.agents,
      activeTab: deps.activeEditorTab,
      editorAgentTrust: resolveEditorAgentTrust(deps.layoutSettings, deps.composerMode),
      composerMetadata: metadata,
      api: deps.ideEnabled ? deps.api : undefined,
      repoPath: deps.ideEnabled ? ws?.path : undefined,
      repoPaths: deps.ideEnabled ? deps.resolveScopedRepoPaths() : undefined,
      ideEnabled: deps.ideEnabled,
      channel: deps.channel,
      channelMeta: deps.activeChannelMeta,
    });
    const baseMetadata = buildHumanOutboundMetadata({
      contextMode: deps.workspaceContextMode,
      conversationMode: deps.conversationModeSetting,
      message: payload.content,
      channel: deps.channel,
      channelType: deps.activeChannelMeta?.type,
      composerMetadata: payload.metadata,
      ideCoding: deps.ideLayout && deps.hasIdeComposer,
      recentChannelMessages: useChatStore.getState().messages,
    });
    const mergedMetadata =
      deps.workspaceContextMode === 'off'
        ? baseMetadata
        : await attachAmbientStateMetadata(
            baseMetadata,
            payload.content,
            deps.ideLayout && deps.hasIdeComposer,
          );
    await deps.api.sendThreadReply(
      threadId,
      deps.channel,
      payload.content,
      { name: deps.username, type: 'human' },
      mergedMetadata,
    );
  };

  const handleChannelInterject = async () => {
    try {
      await deps.api.channelInterject(deps.channel, deps.username);
      const st = useChatStore.getState();
      st.setChannelHold(deps.channel, true);
      st.clearThinkingAgents(deps.channel);
      st.stopAllStreamsForChannel(deps.channel);
    } catch (error) {
      console.error('Channel interject failed:', error);
      deps.addToast({
        type: 'error',
        title: 'Stop failed',
        message: error instanceof Error ? error.message : 'Could not stop agents.',
      });
    }
  };

  const dispatchMessage = async (
    content: string,
    metadata?: Record<string, unknown>,
    modeOverride?: ComposerMode,
  ): Promise<boolean> => {
    useChatStore.getState().setChannelHold(deps.channel, false);

    let sendContent = content;
    let composerMeta = metadata ?? {};
    const effectiveComposerMode = modeOverride ?? deps.composerMode;
    const ws =
      deps.explorerWorkspaces.find((w) => w.id === deps.activeWorkspaceId) ??
      deps.explorerWorkspaces[0];
    const payload = await prepareOutboundPayload({
      content,
      composerMode: effectiveComposerMode,
      agents: deps.agents,
      activeTab: deps.activeEditorTab,
      editorAgentTrust: resolveEditorAgentTrust(deps.layoutSettings, effectiveComposerMode),
      composerMetadata: composerMeta,
      api: deps.ideEnabled ? deps.api : undefined,
      repoPath: deps.ideEnabled ? ws?.path : undefined,
      repoPaths: deps.ideEnabled ? deps.resolveScopedRepoPaths() : undefined,
      ideEnabled: deps.ideEnabled,
      channel: deps.channel,
      channelMeta: deps.activeChannelMeta,
    });
    sendContent = payload.content;
    composerMeta = payload.metadata;

    const baseMetadata = buildHumanOutboundMetadata({
      contextMode: deps.workspaceContextMode,
      conversationMode: deps.conversationModeSetting,
      message: sendContent,
      channel: deps.channel,
      channelType: deps.activeChannelMeta?.type,
      composerMetadata: composerMeta,
      ideCoding: deps.ideLayout && deps.hasIdeComposer,
      recentChannelMessages: useChatStore.getState().messages,
    });
    const mergedMetadata =
      deps.workspaceContextMode === 'off'
        ? baseMetadata
        : await attachAmbientStateMetadata(
            baseMetadata,
            sendContent,
            deps.ideLayout && deps.hasIdeComposer,
          );

    useChatStore.getState().setIsTyping(true);
    markPendingSendThinking(deps.channel);
    try {
      const trimmed = sendContent.trimStart();
      const slashCommand = trimmed.startsWith('/');
      if (trimmed.startsWith('/collaborate')) {
        if (!confirmStartCollaborationWhileExecuting(deps.executingCollaborationForChannel)) {
          clearPendingSendThinking(deps.channel);
          return false;
        }
      }
      if (slashCommand) {
        appendLocalSlashCommand(sendContent.trim());
      }

      let sendResult;
      const from = { name: deps.username, type: 'human' };
      if (slashCommand || deps.workspaceContextMode === 'off') {
        sendResult = await deps.api.sendMessage(
          deps.channel,
          sendContent,
          from,
          'question',
          mergedMetadata,
        );
      } else {
        try {
          const prepareMeta = { ...mergedMetadata };
          if (prepareMeta.workspace_context && typeof prepareMeta.workspace_context === 'object') {
            const wsCtx = { ...(prepareMeta.workspace_context as Record<string, unknown>) };
            wsCtx.open_files = [];
            prepareMeta.workspace_context = wsCtx;
            prepareMeta.context_scope = 'hint';
            prepareMeta.context_scope_reason = 'prepare envelope — structural availability';
          }
          const prepared = await deps.api.prepareTurn(
            deps.channel,
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
            api: deps.api,
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
                  deps.ideLayout && deps.hasIdeComposer,
                  {
                    force: true,
                    includeGit: Boolean(req.include_git_status),
                    includeDiagnostics: Boolean(req.include_diagnostics),
                  },
                )
              : dispatchMeta;
          sendResult = await deps.api.dispatchTurn(
            deps.channel,
            sendContent,
            from,
            'question',
            withAmbient,
          );
        } catch (prepareErr) {
          console.warn('[dispatchMessage] prepare/dispatch fallback to /api/send', prepareErr);
          sendResult = await deps.api.sendMessage(
            deps.channel,
            sendContent,
            from,
            'question',
            mergedMetadata,
          );
        }
      }
      let timelineChannel = deps.channel;
      if (sendResult.collaboration_channel) {
        clearPendingSendThinking(deps.channel);
        markPendingSendThinking(sendResult.collaboration_channel);
        await deps.loadChannels();
        await deps.handleSwitchChannel(sendResult.collaboration_channel);
        timelineChannel = sendResult.collaboration_channel;
        await deps.loadCollaborations(timelineChannel);
        let collab =
          (sendResult.collaboration_id
            ? deps.collaborationsByIDRef.current[sendResult.collaboration_id]
            : undefined) ??
          Object.values(deps.collaborationsByIDRef.current).find(
            (c) => c.channel === sendResult.collaboration_channel,
          );
        if (!collab && sendResult.collaboration_id) {
          try {
            collab = await deps.api.getRunbook(sendResult.collaboration_id);
          } catch (e) {
            console.error('[dispatchMessage] failed to load runbook after redirect:', e);
          }
        }
        await deps.loadCollaborations(timelineChannel);
        if (collab) {
          deps.mergeCollaborationSnapshot(collab);
          deps.setActiveCollab(collab);
          syncCollabTurnThinking(collab, sendResult.collaboration_channel);
        }
      }
      if (sendResult.dm_channel) {
        clearPendingSendThinking(deps.channel);
        const dmName = sendResult.dm_channel;
        markPendingSendThinking(dmName);
        await deps.loadAgents();
        await deps.loadChannels();
        const channelList = useChatStore.getState().channels;
        const { settings, isLoaded } = useSettingsStore.getState();
        if (isLoaded) {
          const patch = patchRevealForChannel(
            settings,
            dmName,
            channelList,
            useChatStore.getState().agents,
          );
          if (patch) {
            void deps.updateSettings(patch);
          }
        }
        await deps.handleSwitchChannel(dmName);
        timelineChannel = dmName;
        deps.addToast({
          type: 'success',
          title: 'Expert ready',
          message: 'Opened the new expert direct message.',
        });
      }
      if (sendContent.trimStart().startsWith('/')) {
        try {
          const msgs = await deps.api.fetchMessages(timelineChannel, 50);
          useChatStore.getState().setMessages(msgs);
          await deps.loadCollaborations(timelineChannel);
        } catch (e) {
          console.error('[dispatchMessage] post-command refresh failed:', e);
        }
      }
      window.setTimeout(() => {
        clearPendingSendThinking(timelineChannel);
      }, 20_000);
      return true;
    } catch (error) {
      console.error('Failed to send message:', error);
      clearPendingSendThinking(deps.channel);
      deps.addToast({
        type: 'error',
        title: 'Message not sent',
        message: error instanceof Error ? error.message : 'Failed to send message.',
      });
      return false;
    } finally {
      useChatStore.getState().setIsTyping(false);
    }
  };

  const handleHubAccessConfirm = async (selected: HubDataAccessOption[]) => {
    if (!deps.hubAccessPending) return;
    deps.setHubAccessLoading(true);
    deps.setHubAccessError(null);
    try {
      const result = await deps.api.readHubDataAccess(
        selected.map((s) => ({ kind: s.kind, relative_path: s.relativePath }))
      );
      const merged = {
        ...(deps.hubAccessPending.metadata ?? {}),
        [GRANTED_HUB_DATA_ACCESS_KEY]: result,
      };
      if (deps.hubAccessPending.mode === 'thread' && deps.hubAccessPending.threadId) {
        await dispatchThreadReply(deps.hubAccessPending.threadId, deps.hubAccessPending.content, merged);
      } else {
        await dispatchMessage(deps.hubAccessPending.content, merged);
      }
      deps.setHubAccessPending(null);
    } catch (err) {
      deps.setHubAccessError(err instanceof Error ? err.message : 'Failed to read hub data');
    } finally {
      deps.setHubAccessLoading(false);
    }
  };

  return {
    dispatchThreadReply,
    dispatchMessage,
    handleChannelInterject,
    appendLocalSlashCommand,
    handleHubAccessConfirm,
  };
}
