import type { Message } from '../types/protocol';
import { getChangeProposalCard } from '../types/protocol';

export type PendingChangeNavTarget = {
  id: string;
  channel: string;
  requestedAt?: string;
};

export function pendingProposalCount(fileIds: string[], gitIds: string[]): number {
  return new Set([...fileIds, ...gitIds]).size;
}

export function oldestPendingProposalMessage(
  messages: Message[],
  pendingIds: ReadonlySet<string>,
): Message | null {
  for (const message of messages) {
    const proposal = getChangeProposalCard(message);
    if (proposal?.status === 'pending' && pendingIds.has(proposal.id)) {
      return message;
    }
  }
  return null;
}

/** Oldest pending file/Git change that has a channel we can open. */
export function oldestPendingChangeNavTarget(
  changes: ReadonlyArray<{ id: string; channel?: string; requested_at?: string }>,
): PendingChangeNavTarget | null {
  const withChannel = changes.filter(
    (c): c is { id: string; channel: string; requested_at?: string } =>
      typeof c.id === 'string' && typeof c.channel === 'string' && c.channel.trim() !== '',
  );
  if (withChannel.length === 0) return null;

  withChannel.sort((a, b) => {
    const at = a.requested_at ? Date.parse(a.requested_at) : Number.POSITIVE_INFINITY;
    const bt = b.requested_at ? Date.parse(b.requested_at) : Number.POSITIVE_INFINITY;
    if (Number.isNaN(at) && Number.isNaN(bt)) return 0;
    if (Number.isNaN(at)) return 1;
    if (Number.isNaN(bt)) return -1;
    return at - bt;
  });

  const oldest = withChannel[0];
  return {
    id: oldest.id,
    channel: oldest.channel,
    requestedAt: oldest.requested_at,
  };
}

export function messageForPendingChangeId(
  messages: Message[],
  changeId: string,
): Message | null {
  for (const message of messages) {
    const proposal = getChangeProposalCard(message);
    if (proposal?.id === changeId) {
      return message;
    }
  }
  return null;
}
