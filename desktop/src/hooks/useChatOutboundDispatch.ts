import { useCallback, useEffect, useRef } from 'react';
import type { ComposerMode } from '../constants/composerMode';
import {
  createChatOutboundDispatch,
  type ChatOutboundDispatchDeps,
} from './createChatOutboundDispatch';

/** Stable outbound send/interject callbacks; deps are read from a ref each call. */
export function useChatOutboundDispatch(deps: ChatOutboundDispatchDeps) {
  const depsRef = useRef(deps);
  useEffect(() => {
    depsRef.current = deps;
  });

  const dispatchThreadReply = useCallback(
    (threadId: string, content: string, metadata?: Record<string, unknown>) =>
      createChatOutboundDispatch(depsRef.current).dispatchThreadReply(threadId, content, metadata),
    [],
  );

  const dispatchMessage = useCallback(
    (content: string, metadata?: Record<string, unknown>, modeOverride?: ComposerMode) =>
      createChatOutboundDispatch(depsRef.current).dispatchMessage(content, metadata, modeOverride),
    [],
  );

  const handleChannelInterject = useCallback(
    () => createChatOutboundDispatch(depsRef.current).handleChannelInterject(),
    [],
  );

  const appendLocalSlashCommand = useCallback(
    (commandText: string) =>
      createChatOutboundDispatch(depsRef.current).appendLocalSlashCommand(commandText),
    [],
  );

  return {
    dispatchThreadReply,
    dispatchMessage,
    handleChannelInterject,
    appendLocalSlashCommand,
  };
}
