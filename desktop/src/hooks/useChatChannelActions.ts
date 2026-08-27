import { useCallback, useEffect, useRef } from 'react';
import type { Channel } from '../types/protocol';
import {
  createChatChannelActions,
  type ChatChannelActionsDeps,
} from './createChatChannelActions';

/** Stable channel / workspace-gate / runbook callbacks; deps are read from a ref each call. */
export function useChatChannelActions(deps: ChatChannelActionsDeps) {
  const depsRef = useRef(deps);
  useEffect(() => {
    depsRef.current = deps;
  });

  const handleWorkspaceGateContinue = useCallback(
    () => createChatChannelActions(depsRef.current).handleWorkspaceGateContinue(),
    [],
  );

  const handleWorkspaceGateDismiss = useCallback(
    () => createChatChannelActions(depsRef.current).handleWorkspaceGateDismiss(),
    [],
  );

  const handleSwitchChannel = useCallback(
    (channelName: string) =>
      createChatChannelActions(depsRef.current).handleSwitchChannel(channelName),
    [],
  );

  const handleNewRunbook = useCallback(
    () => createChatChannelActions(depsRef.current).handleNewRunbook(),
    [],
  );

  const handleCreateBlankRunbook = useCallback(
    () => createChatChannelActions(depsRef.current).handleCreateBlankRunbook(),
    [],
  );

  const handleCreateChannel = useCallback(
    (name: string, description: string, agentIds: string[]) =>
      createChatChannelActions(depsRef.current).handleCreateChannel(name, description, agentIds),
    [],
  );

  const handleDeleteChannel = useCallback(
    (name: string) => createChatChannelActions(depsRef.current).handleDeleteChannel(name),
    [],
  );

  const handleOpenChannelInfo = useCallback(
    (ch: Channel) => createChatChannelActions(depsRef.current).handleOpenChannelInfo(ch),
    [],
  );

  return {
    handleWorkspaceGateContinue,
    handleWorkspaceGateDismiss,
    handleSwitchChannel,
    handleNewRunbook,
    handleCreateBlankRunbook,
    handleCreateChannel,
    handleDeleteChannel,
    handleOpenChannelInfo,
  };
}
