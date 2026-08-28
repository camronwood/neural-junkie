import { useCallback, useEffect, useRef } from 'react';
import { createChatDmActions, type ChatDmActionsDeps } from './createChatDmActions';
import type { Channel } from '../types/protocol';

/** Stable DM open/create callbacks; deps are read from a ref each call. */
export function useChatDmActions(deps: Omit<ChatDmActionsDeps, 'dmCreateInFlightRef' | 'dmOpenChainRef'>) {
  const depsRef = useRef(deps);
  const dmCreateInFlightRef = useRef<Map<string, Promise<void>>>(new Map());
  const dmOpenChainRef = useRef<Promise<void>>(Promise.resolve());

  useEffect(() => {
    depsRef.current = deps;
  });

  const handleCreateDM = useCallback(
    (agentId: string) =>
      createChatDmActions({
        ...depsRef.current,
        dmCreateInFlightRef,
        dmOpenChainRef,
      }).handleCreateDM(agentId),
    []
  );

  const handleNewDmCreated = useCallback(
    (ch: Channel) =>
      createChatDmActions({
        ...depsRef.current,
        dmCreateInFlightRef,
        dmOpenChainRef,
      }).handleNewDmCreated(ch),
    []
  );

  return {
    handleCreateDM,
    handleNewDmCreated,
  };
}
