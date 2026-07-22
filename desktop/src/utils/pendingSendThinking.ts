import { useChatStore } from '../stores/chatStore';

/** Placeholder typing-indicator row while send/classify is in flight. */
export const NJ_PENDING_SEND_AGENT_ID = 'nj-pending-send';

export function clearPendingSendThinking(channel: string): void {
  useChatStore.getState().removeThinkingAgent(channel, NJ_PENDING_SEND_AGENT_ID);
}

export function markPendingSendThinking(channel: string): void {
  useChatStore
    .getState()
    .addThinkingAgent(channel, NJ_PENDING_SEND_AGENT_ID, 'Neural Junkie', 'assistant', 'routing');
}
