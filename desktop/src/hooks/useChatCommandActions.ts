import { useCallback, useEffect, useRef } from 'react';
import {
  createChatCommandActions,
  type ChatCommandActionsDeps,
} from './createChatCommandActions';

/** Stable command-palette callbacks; deps are read from a ref each call. */
export function useChatCommandActions(deps: ChatCommandActionsDeps) {
  const depsRef = useRef(deps);
  useEffect(() => {
    depsRef.current = deps;
  });

  const ensureCommandDefs = useCallback(
    (forceRefresh: boolean = false) =>
      createChatCommandActions(depsRef.current).ensureCommandDefs(forceRefresh),
    []
  );

  const handleCommandExecute = useCallback(
    (commandString: string, metadata?: Record<string, unknown>) =>
      createChatCommandActions(depsRef.current).handleCommandExecute(commandString, metadata),
    []
  );

  const openCommandPalette = useCallback(
    (filter = '') => createChatCommandActions(depsRef.current).openCommandPalette(filter),
    []
  );

  return {
    ensureCommandDefs,
    handleCommandExecute,
    openCommandPalette,
  };
}
