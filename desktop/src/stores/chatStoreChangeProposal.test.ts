import { beforeEach, describe, expect, it } from 'vitest';
import type { Message } from '../types/protocol';
import { getChangeProposalCard } from '../types/protocol';
import { useChatStore } from './chatStore';

function proposalMessage(status: 'pending' | 'approved'): Message {
  return {
    id: 'proposal-message',
    type: 'chat',
    channel: 'general',
    from: { id: 'agent', name: 'Agent', type: 'cli' } as Message['from'],
    content: '',
    timestamp: new Date().toISOString(),
    metadata: {
      change_proposal: {
        version: 1,
        kind: 'git_change',
        id: 'git-1',
        status,
        operation: 'commit',
      },
    },
  };
}

beforeEach(() => {
  useChatStore.setState({
    channel: 'general',
    messages: [],
    channelMessages: new Map(),
  });
});

describe('chat proposal lifecycle', () => {
  it('keeps metadata-only proposal cards in the timeline', () => {
    useChatStore.getState().addMessage(proposalMessage('pending'));
    expect(useChatStore.getState().messages).toHaveLength(1);
  });

  it('replaces a proposal message by ID when its durable status changes', () => {
    useChatStore.getState().addMessage(proposalMessage('pending'));
    useChatStore.getState().addMessage(proposalMessage('approved'));
    const messages = useChatStore.getState().messages;
    expect(messages).toHaveLength(1);
    expect(getChangeProposalCard(messages[0])?.status).toBe('approved');
  });

  it('updates cached channels instead of discarding lifecycle updates', () => {
    useChatStore.getState().addMessageToCache('other', proposalMessage('pending'));
    useChatStore.getState().addMessageToCache('other', proposalMessage('approved'));
    const messages = useChatStore.getState().channelMessages.get('other') ?? [];
    expect(messages).toHaveLength(1);
    expect(getChangeProposalCard(messages[0])?.status).toBe('approved');
  });
});
