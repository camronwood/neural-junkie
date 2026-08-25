import { startTransition, type MutableRefObject, type RefObject } from 'react';
import { ChatAPI } from '../api/chatAPI';
import type { LearningProposalAction } from '../api/chatAPI';
import { useChatStore } from '../stores/chatStore';
import { createNewTab, useTerminalStore } from '../stores/terminalStore';
import {
  CAD_FILES_WRITTEN_KEY,
  IMPLEMENTATION_SESSION_COMPLETE_KEY,
} from '../constants/promptMetadata';
import {
  METADATA_CHANNEL_HOLD,
  THINKING_ACTIVITY_DETAIL_KEY,
  THINKING_ACTIVITY_REASONING,
  THINKING_ACTIVITY_USING_TOOL,
  THINKING_ACTIVITY_WRITING,
  getChangeProposalCard,
  getCollaborationId,
  isCollaborationMessage,
  isReasoningStreamDelta,
  isToolStepStreamDelta,
  showThreadReplyInMainTimeline,
  type Collaboration,
  type Message,
  type ThinkingStatusMetadata,
} from '../types/protocol';
import {
  clearPendingSendThinking,
  NJ_PENDING_SEND_AGENT_ID,
} from '../utils/pendingSendThinking';
import { pendingUserQuestionMessages } from '../utils/pendingUserQuestions';
import { handoffNavigationTarget } from '../utils/capabilityPolicy';
import {
  appendTurnTelemetryFromAgentStatus,
  appendTurnTelemetryFromToolStep,
} from '../utils/turnTelemetry';
import {
  collaboratorsAddedSince,
  decideCollabPanelOpen,
  parseCollabParticipantAddRequest,
  shouldToastCollaboratorAdds,
} from '../utils/chatInboundCollab';
import { shouldNotifySlackInbound } from '../utils/slackNotification';
import { mirrorAgentCommandInTerminal } from '../utils/mirrorAgentCommandInTerminal';
import { resolveTerminalCwd } from '../utils/terminalCwd';
import {
  ensureRepoAgentWorkspace,
  isRepoAgentWorkspaceAction,
} from '../utils/repoAgentWorkspace';
import type { Toast } from '../stores/toastStore';

export type ChatInboundMessageDeps = {
  api: ChatAPI;
  channel: string;
  handledHandoffMessagesRef: MutableRefObject<Set<string>>;
  handledParticipantRequestPromptsRef: MutableRefObject<Set<string>>;
  handledRepoWorkspaceActionsRef: MutableRefObject<Set<string>>;
  handledLearningProposalsRef: MutableRefObject<Set<string>>;
  collaborationsByIDRef: MutableRefObject<Record<string, Collaboration>>;
  activeCollabRef: MutableRefObject<Collaboration | null>;
  inputRef: RefObject<HTMLTextAreaElement | null>;
  loadChannels: () => Promise<unknown>;
  handleSwitchChannel: (target: string) => Promise<void>;
  debouncedRefreshAgents: () => void;
  mergeCollaborationSnapshot: (snapshot: Collaboration) => void;
  setActiveCollab: React.Dispatch<React.SetStateAction<Collaboration | null>>;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
  surfaceSlackInboundNotification: (message: Message) => void;
  surfaceChangeProposal: (message: Message, isActiveChannel: boolean) => void;
  surfaceToolApproval: (message: Message, isActiveChannel: boolean) => void;
  handleImplementationSessionComplete: (
    metadata: Record<string, unknown> | undefined
  ) => void | Promise<void>;
  handleCADFilesWritten: (metadata: Record<string, unknown> | undefined) => void | Promise<void>;
  handleSuggestedCommands: (message: Message, activeChannel: string) => void;
  setFileExplorerOpen: (open: boolean) => void;
  setLearningProposal: (proposal: LearningProposalAction) => void;
  setLearningProposalOpen: (open: boolean) => void;
};

/** WebSocket inbound message handler extracted from ChatWindow. */
export function createChatInboundMessageHandler(deps: ChatInboundMessageDeps) {
  return async (message: Message) => {
    try {
      const st = useChatStore.getState();
      const activeChannel = st.channel;

      if (
        message.type === 'system_info' &&
        !deps.handledHandoffMessagesRef.current.has(message.id)
      ) {
        const target = handoffNavigationTarget(message.metadata);
        if (target) {
          deps.handledHandoffMessagesRef.current.add(message.id);
          const channels = await deps.loadChannels();
          if (
            Array.isArray(channels) &&
            channels.some((candidate: { name: string }) => candidate.name === target) &&
            useChatStore.getState().channel !== target
          ) {
            await deps.handleSwitchChannel(target);
            return;
          }
        }
      }

      if (message.type === 'agent_status') {
        const msgChannel = message.channel || activeChannel;
        if (message.metadata?.history_resync === true) {
          const ch = message.channel || deps.channel;
          try {
            const msgs = await deps.api.fetchMessages(ch, 50);
            const store = useChatStore.getState();
            store.replaceChannelMessagesCache(ch, msgs);
            if (ch === store.channel) {
              store.setMessages(msgs);
              store.cleanupStaleThinking(ch, msgs);
            }
          } catch (e) {
            console.error('[ChatWindow] history_resync refetch failed:', e);
          }
          return;
        }

        if (message.metadata?.thinking_status) {
          const thinkingStatus = message.metadata.thinking_status as ThinkingStatusMetadata['thinking_status'];
          const pendingAsks = pendingUserQuestionMessages(st.messages).length > 0;
          if (thinkingStatus === 'started') {
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
              if (msgChannel !== activeChannel && msgChannel.startsWith('collab-')) {
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

        if (
          message.metadata?.indexing_status !== undefined ||
          message.metadata?.index_progress !== undefined ||
          message.metadata?.status !== undefined ||
          message.from.is_paused !== undefined
        ) {
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

        return;
      }

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
          const previousSnapshot = deps.collaborationsByIDRef.current[collabData.id];
          if (
            shouldToastCollaboratorAdds({
              previous: previousSnapshot,
              snapshot: collabData,
              isActiveChannelCollab,
            })
          ) {
            const addedAgents = collaboratorsAddedSince(previousSnapshot, collabData);
            const names = addedAgents.map((a) => `@${a.agent_name}`).join(', ');
            deps.addToast({
              type: 'info',
              title: 'Collaborator added',
              message: `${names} joined "${collabData.title}".`,
            });
          }
          deps.mergeCollaborationSnapshot(collabData);
          const decision = decideCollabPanelOpen({
            snapshot: collabData,
            activeChannel,
            currentlyOpen: deps.activeCollabRef.current,
            message,
          });
          if (decision.action === 'open' || decision.action === 'update_open') {
            deps.setActiveCollab(decision.snapshot);
          }
        });
      }

      const participantReq = parseCollabParticipantAddRequest(message);
      if (participantReq) {
        const { collabID, agentID, agentName, requestedBy } = participantReq;
        const key = `${collabID}:${agentID}:${message.id}`;
        if (!deps.handledParticipantRequestPromptsRef.current.has(key)) {
          deps.handledParticipantRequestPromptsRef.current.add(key);
          void (async () => {
            const approved = window.confirm(
              `${requestedBy} wants to add ${agentName} to this collaboration. Allow?`
            );
            try {
              const updated = approved
                ? await deps.api.approveCollabParticipantRequest(collabID, agentID)
                : await deps.api.denyCollabParticipantRequest(collabID, agentID);
              deps.mergeCollaborationSnapshot(updated);
              if (deps.activeCollabRef.current?.id === updated.id) {
                deps.setActiveCollab(updated);
              }
              deps.addToast({
                type: approved ? 'success' : 'info',
                title: approved ? 'Agent added' : 'Agent add denied',
                message: approved
                  ? `@${agentName} joined "${updated.title}".`
                  : `@${agentName} was not added to "${updated.title}".`,
              });
            } catch (error) {
              deps.addToast({
                type: 'error',
                title: 'Participant request failed',
                message:
                  error instanceof Error
                    ? error.message
                    : 'Could not update collaboration participants.',
              });
            }
          })();
        }
      }

      if (message.is_thread_reply && message.thread_id) {
        const threadChannel = message.channel || activeChannel;
        void deps.api
          .fetchThreadMetadata(message.thread_id)
          .then((metadata) =>
            useChatStore.getState().updateThreadMetadata(message.thread_id!, metadata)
          )
          .catch((error) => console.error('Failed to fetch thread metadata:', error));
        if (showThreadReplyInMainTimeline(threadChannel)) {
          if (threadChannel === activeChannel) {
            st.addMessage(message);
          } else if (message.channel) {
            st.addMessageToCache(message.channel, message);
            st.markChannelUnread(message.channel);
            if (shouldNotifySlackInbound(message)) {
              deps.surfaceSlackInboundNotification(message);
            }
          }
        }
        return;
      } else if (message.channel && message.channel !== activeChannel) {
        st.addMessageToCache(message.channel, message);
        st.markChannelUnread(message.channel);
        if (shouldNotifySlackInbound(message)) {
          deps.surfaceSlackInboundNotification(message);
        }
        if (isCollaborationMessage(message) || getCollaborationId(message)) {
          deps.addToast({
            type: 'info',
            title: 'Collaboration update',
            message: `Activity in #${message.channel} — switch there to see messages.`,
          });
        }
        if (getChangeProposalCard(message)) {
          deps.surfaceChangeProposal(message, false);
        }
        if (message.type === 'tool_approval') {
          deps.surfaceToolApproval(message, false);
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
          deps.surfaceToolApproval(message, true);
          st.upsertToolApprovalMessage(message);
        } else if (message.type === 'user_question') {
          st.upsertUserQuestionMessage(message);
          if (message.metadata?.status === 'pending') {
            const qChannel = message.channel || activeChannel;
            st.clearThinkingAgents(qChannel);
            st.stopAllStreamsForChannel(qChannel);
            window.setTimeout(() => deps.inputRef.current?.focus(), 0);
          }
        } else {
          st.addMessage(message);
        }

        if (message.metadata?.[IMPLEMENTATION_SESSION_COMPLETE_KEY] === true) {
          void deps.handleImplementationSessionComplete(
            message.metadata as Record<string, unknown> | undefined
          );
        }

        if (message.metadata?.[CAD_FILES_WRITTEN_KEY]) {
          void deps.handleCADFilesWritten(message.metadata as Record<string, unknown> | undefined);
        }

        deps.handleSuggestedCommands(message, activeChannel);

        if (message.type === 'command_output') {
          mirrorAgentCommandInTerminal(message);
        }

        if (message.metadata?.event === 'agent-open-terminal') {
          const agentName = (message.metadata.agent_name as string) || 'Agent';
          const msgCh = message.channel || activeChannel;
          const collabCtx = Object.values(deps.collaborationsByIDRef.current).find(
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
          !deps.handledRepoWorkspaceActionsRef.current.has(message.id)
        ) {
          deps.handledRepoWorkspaceActionsRef.current.add(message.id);
          void ensureRepoAgentWorkspace(clientAction.path, {
            preferredName: clientAction.name,
          }).then((workspaceId) => {
            if (workspaceId) {
              deps.setFileExplorerOpen(true);
            }
          });
        }

        if (
          clientAction &&
          typeof clientAction === 'object' &&
          clientAction.type === 'learning_proposal' &&
          !deps.handledLearningProposalsRef.current.has(message.id)
        ) {
          deps.handledLearningProposalsRef.current.add(message.id);
          deps.setLearningProposal(clientAction as LearningProposalAction);
          deps.setLearningProposalOpen(true);
        }

        if (getChangeProposalCard(message)) {
          deps.surfaceChangeProposal(message, true);
        }
      }

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

      if (message.type === 'agent_join' || message.type === 'agent_leave') {
        deps.debouncedRefreshAgents();
        void deps.loadChannels();
      }
    } catch (err) {
      console.error('[ChatWindow] WebSocket message handler error:', err);
    }
  };
}
