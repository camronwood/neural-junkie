import { useCallback, useEffect, useRef } from 'react';
import type { Message } from '../types/protocol';
import {
  createChatInboundMessageHandler,
  type ChatInboundMessageDeps,
} from './createChatInboundMessageHandler';

/** Stable WebSocket inbound handler; deps are read from a ref each message. */
export function useChatInboundMessages(deps: ChatInboundMessageDeps) {
  const depsRef = useRef(deps);
  useEffect(() => {
    depsRef.current = deps;
  });
  return useCallback(async (message: Message) => {
    await createChatInboundMessageHandler(depsRef.current)(message);
  }, []);
}
