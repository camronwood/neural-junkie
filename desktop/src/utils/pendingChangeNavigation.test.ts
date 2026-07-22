import { describe, expect, it } from 'vitest';
import type { ChangeProposalCard, Message } from '../types/protocol';
import {
  messageForPendingChangeId,
  oldestPendingChangeNavTarget,
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

  it('picks the oldest pending change that has a channel', () => {
    expect(
      oldestPendingChangeNavTarget([
        { id: 'newer', channel: 'general', requested_at: '2026-07-22T10:00:00Z' },
        { id: 'oldest', channel: 'dm-camron-frontendengineer', requested_at: '2026-07-22T08:00:00Z' },
        { id: 'no-channel', requested_at: '2026-07-22T07:00:00Z' },
      ]),
    ).toEqual({
      id: 'oldest',
      channel: 'dm-camron-frontendengineer',
      requestedAt: '2026-07-22T08:00:00Z',
    });
  });

  it('finds a proposal message by change id', () => {
    const target = message('msg-1', {
      version: 1,
      kind: 'file_change',
      id: '77eecda8',
      status: 'pending',
      operation: 'edit',
    });
    expect(messageForPendingChangeId([target], '77eecda8')?.id).toBe('msg-1');
    expect(messageForPendingChangeId([target], 'missing')).toBeNull();
  });
});
