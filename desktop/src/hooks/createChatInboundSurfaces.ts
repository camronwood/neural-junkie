import type { MutableRefObject } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { useApprovalStore } from '../stores/approvalStore';
import { useChatStore } from '../stores/chatStore';
import { useEditorStore } from '../stores/editorStore';
import type { Toast } from '../stores/toastStore';
import { formatToolApprovalSummary } from '../utils/approvalDisplay';
import {
  messageForPendingChangeId,
  oldestPendingChangeNavTarget,
  oldestPendingProposalMessage,
} from '../utils/pendingChangeNavigation';
import {
  fileChangeProposalPaths,
  refreshFileExplorerForPaths,
} from '../utils/refreshFileExplorer';
import {
  shouldNotifySlackInbound,
  slackChannelLabel,
  slackInboundPreview,
  slackInboundSenderLabel,
} from '../utils/slackNotification';
import { chatScrollerElRef } from '../components/MessageList';
import { getChangeProposalCard, type Message } from '../types/protocol';

export type ChatInboundSurfacesDeps = {
  api: ChatAPI;
  username: string;
  handleSwitchChannel: (channelName: string) => Promise<void>;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
  loadAgents: () => void | Promise<unknown>;
  loadCounts: () => void | Promise<unknown>;
  activeWorkspaceId: string | null;
  explorerWorkspaces: ReadonlyArray<{ id: string }>;
  fetchPendingChanges: (userId: string) => Promise<unknown>;
  fetchPendingGitChanges: (userId: string) => Promise<unknown>;
  pendingChangeCount: number;
  pendingChanges: ReadonlyArray<{ id: string; channel?: string; requested_at?: string }>;
  pendingGitChanges: ReadonlyArray<{ id: string; channel?: string; requested_at?: string }>;
  agentRefreshTimeoutRef: MutableRefObject<number | null>;
  explorerRefreshTimeoutRef: MutableRefObject<number | null>;
  handledChangeProposalNoticesRef: MutableRefObject<Set<string>>;
};

/** Inbound notification / navigation surfaces extracted from ChatWindow. */
export function createChatInboundSurfaces(deps: ChatInboundSurfacesDeps) {
  const navigateToMessage = async (channelName: string, messageId: string) => {
    useChatStore.getState().setPendingScrollToMessageId(messageId);
    await deps.handleSwitchChannel(channelName);
  };

  const surfaceSlackInboundNotification = (message: Message) => {
    if (!message.channel || !shouldNotifySlackInbound(message)) return;
    const label = slackChannelLabel(useChatStore.getState().channels, message.channel);
    const sender = slackInboundSenderLabel(message);
    deps.addToast({
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
  };

  // Debounced agent refresh (prevents excessive API calls).
  // Channel list is only refreshed on agent_join/agent_leave, not on every status tick.
  const debouncedRefreshAgents = () => {
    if (deps.agentRefreshTimeoutRef.current) {
      clearTimeout(deps.agentRefreshTimeoutRef.current);
    }
    deps.agentRefreshTimeoutRef.current = window.setTimeout(() => {
      void deps.loadAgents();
      void deps.loadCounts();
    }, 300);
  };

  const refreshExplorerForFileChange = (message: Message) => {
    const paths = fileChangeProposalPaths(message);
    if (paths.length === 0) return;
    const ws =
      deps.explorerWorkspaces.find((w) => w.id === deps.activeWorkspaceId) ??
      deps.explorerWorkspaces[0];
    if (!ws) return;
    void refreshFileExplorerForPaths(ws.id, paths);
    for (const relPath of paths) {
      void useEditorStore.getState().refreshTabFromDisk(ws.id, relPath);
    }
  };

  const debouncedRefreshExplorer = (message: Message) => {
    if (deps.explorerRefreshTimeoutRef.current) {
      clearTimeout(deps.explorerRefreshTimeoutRef.current);
    }
    deps.explorerRefreshTimeoutRef.current = window.setTimeout(() => {
      refreshExplorerForFileChange(message);
    }, 200);
  };

  const surfaceChangeProposal = (message: Message, isActiveChannel: boolean) => {
    const proposal = getChangeProposalCard(message);
    if (!proposal) return;
    if (proposal.kind === 'file_change') {
      void deps.fetchPendingChanges(deps.username || 'default').catch((error) =>
        console.error('Failed to refresh pending file changes:', error),
      );
      if (proposal.status === 'approved') {
        debouncedRefreshExplorer(message);
      }
    } else {
      void deps.fetchPendingGitChanges(deps.username || 'default').catch((error) =>
        console.error('Failed to refresh pending Git changes:', error),
      );
    }
    if (
      proposal.status === 'pending' &&
      !isActiveChannel &&
      !deps.handledChangeProposalNoticesRef.current.has(message.id)
    ) {
      deps.handledChangeProposalNoticesRef.current.add(message.id);
      deps.addToast({
        type: 'info',
        title: 'Change needs review',
        message: `${message.from.name} proposed a ${proposal.kind === 'file_change' ? 'file change' : 'Git operation'} in #${message.channel}.`,
      });
    }
  };

  const jumpToOldestPendingChange = async () => {
    if (deps.pendingChangeCount === 0) {
      deps.addToast({
        type: 'info',
        title: 'No pending changes',
        message: 'All proposed file and Git changes have been resolved.',
      });
      return;
    }

    const pendingIds = new Set([
      ...deps.pendingChanges.map((change) => change.id),
      ...deps.pendingGitChanges.map((change) => change.id),
    ]);
    const navTarget = oldestPendingChangeNavTarget([
      ...deps.pendingChanges,
      ...deps.pendingGitChanges,
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
      deps.addToast({
        type: 'info',
        title: 'Pending change unavailable',
        message: 'A change is pending but its chat channel is unknown.',
      });
      return;
    }

    await deps.handleSwitchChannel(navTarget.channel);

    let messages = useChatStore.getState().messages;
    if (!focusProposal(messages)) {
      try {
        messages = await deps.api.fetchMessages(navTarget.channel, 200);
        useChatStore.getState().setMessages(messages);
      } catch {
        // Fall through to toast below.
      }
    }

    if (focusProposal(messages)) {
      return;
    }

    deps.addToast({
      type: 'info',
      title: `Opened #${navTarget.channel}`,
      message: 'Switched to the chat with the pending change. Scroll to find the approval card.',
    });
  };

  const scrollToApproval = (approvalId: string) => {
    const el = document.querySelector(`[data-approval-id="${approvalId}"]`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      return;
    }
    chatScrollerElRef.current?.scrollTo({
      top: chatScrollerElRef.current.scrollHeight,
      behavior: 'smooth',
    });
  };

  const surfaceToolApproval = (message: Message, isActiveChannel: boolean) => {
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
      deps.addToast({
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
  };

  return {
    navigateToMessage,
    surfaceSlackInboundNotification,
    debouncedRefreshAgents,
    refreshExplorerForFileChange,
    debouncedRefreshExplorer,
    surfaceChangeProposal,
    jumpToOldestPendingChange,
    scrollToApproval,
    surfaceToolApproval,
  };
}
