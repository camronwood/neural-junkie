import { useCallback, useEffect, useRef } from 'react';
import {
  createChatInboundSurfaces,
  type ChatInboundSurfacesDeps,
} from './createChatInboundSurfaces';
import type { Message } from '../types/protocol';

type HookDeps = Omit<
  ChatInboundSurfacesDeps,
  | 'agentRefreshTimeoutRef'
  | 'explorerRefreshTimeoutRef'
  | 'handledChangeProposalNoticesRef'
>;

/** Stable inbound surface callbacks; deps are read from a ref each call. */
export function useChatInboundSurfaces(deps: HookDeps) {
  const depsRef = useRef(deps);
  const agentRefreshTimeoutRef = useRef<number | null>(null);
  const explorerRefreshTimeoutRef = useRef<number | null>(null);
  const handledChangeProposalNoticesRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    depsRef.current = deps;
  });

  const withRefs = () =>
    createChatInboundSurfaces({
      ...depsRef.current,
      agentRefreshTimeoutRef,
      explorerRefreshTimeoutRef,
      handledChangeProposalNoticesRef,
    });

  const navigateToMessage = useCallback(
    (channelName: string, messageId: string) =>
      withRefs().navigateToMessage(channelName, messageId),
    []
  );

  const surfaceSlackInboundNotification = useCallback(
    (message: Message) => withRefs().surfaceSlackInboundNotification(message),
    []
  );

  const debouncedRefreshAgents = useCallback(
    () => withRefs().debouncedRefreshAgents(),
    []
  );

  const refreshExplorerForFileChange = useCallback(
    (message: Message) => withRefs().refreshExplorerForFileChange(message),
    []
  );

  const debouncedRefreshExplorer = useCallback(
    (message: Message) => withRefs().debouncedRefreshExplorer(message),
    []
  );

  const surfaceChangeProposal = useCallback(
    (message: Message, isActiveChannel: boolean) =>
      withRefs().surfaceChangeProposal(message, isActiveChannel),
    []
  );

  const jumpToOldestPendingChange = useCallback(
    () => withRefs().jumpToOldestPendingChange(),
    []
  );

  const scrollToApproval = useCallback(
    (approvalId: string) => withRefs().scrollToApproval(approvalId),
    []
  );

  const surfaceToolApproval = useCallback(
    (message: Message, isActiveChannel: boolean) =>
      withRefs().surfaceToolApproval(message, isActiveChannel),
    []
  );

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
