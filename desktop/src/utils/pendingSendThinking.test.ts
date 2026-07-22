import { describe, expect, it, beforeEach } from 'vitest';
import { useChatStore } from '../stores/chatStore';
import {
  clearPendingSendThinking,
  markPendingSendThinking,
  NJ_PENDING_SEND_AGENT_ID,
} from './pendingSendThinking';
import { formatThinkingActivityLabel } from './thinkingActivityLabel';

describe('pendingSendThinking', () => {
  beforeEach(() => {
    useChatStore.setState({ channelThinkingAgents: new Map() });
  });

  it('marks and clears the pending send row', () => {
    markPendingSendThinking('dm-user-fe');
    const row = useChatStore.getState().channelThinkingAgents.get('dm-user-fe')?.get(NJ_PENDING_SEND_AGENT_ID);
    expect(row?.activity).toBe('routing');
    expect(formatThinkingActivityLabel(row?.activity)).toMatch(/delivering/i);
    clearPendingSendThinking('dm-user-fe');
    expect(useChatStore.getState().channelThinkingAgents.get('dm-user-fe')?.has(NJ_PENDING_SEND_AGENT_ID)).toBeFalsy();
  });
});
