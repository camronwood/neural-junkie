import { describe, expect, it } from 'vitest';
import type { ChangeProposalCard, Message } from '../types/protocol';
import {
  oldestPendingProposalMessage,
  pendingProposalCount,
} from './pendingChangeNavigation';

function message(id: string, proposal: ChangeProposalCard): Message {
  return {
    id,
    type: proposal.kind === 'file_change' ? 'file_change' : 'chat',
    channel: 'active',
    from: { id: 'agent', name: 'Agent', type: 'cli' } as Message['from'],
    content: '',
    timestamp: new Date().toISOString(),
    metadata: { change_proposal: proposal },
  };
}

describe('pending change navigation', () => {
  it('counts unique file and Git proposals', () => {
    expect(pendingProposalCount(['a', 'b'], ['b', 'c'])).toBe(3);
  });

  it('returns the oldest unresolved card in the active timeline', () => {
    const approved = message('resolved', {
      version: 1,
      kind: 'file_change',
      id: 'a',
      status: 'approved',
      operation: 'edit',
    });
    const oldestPending = message('oldest', {
      version: 1,
      kind: 'git_change',
      id: 'b',
      status: 'pending',
      operation: 'commit',
    });
    const newerPending = message('newer', {
      version: 1,
      kind: 'file_change',
      id: 'c',
      status: 'pending',
      operation: 'create',
    });

    expect(
      oldestPendingProposalMessage(
        [approved, oldestPending, newerPending],
        new Set(['a', 'b', 'c']),
      )?.id,
    ).toBe('oldest');
  });

  it('returns null when pending proposals belong to another channel cache', () => {
    expect(oldestPendingProposalMessage([], new Set(['other-chat-change']))).toBeNull();
  });
});
