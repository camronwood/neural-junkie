import type { Message } from '../types/protocol';
import { getChangeProposalCard } from '../types/protocol';

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
