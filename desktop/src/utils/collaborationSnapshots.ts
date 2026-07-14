/** Pure helpers for collaboration snapshot maps (extracted from ChatWindow). */

import type { Collaboration } from '../types/protocol';

function isTerminalPhase(phase?: Collaboration['phase']): boolean {
  return phase === 'completed' || phase === 'cancelled';
}

export function shouldReplaceCollaborationSnapshot(
  existing: Collaboration | undefined,
  snapshot: Collaboration
): boolean {
  if (!existing) return true;
  if (
    existing.updated_at === snapshot.updated_at &&
    existing.phase === snapshot.phase &&
    existing.workspace_acknowledged === snapshot.workspace_acknowledged
  ) {
    return false;
  }
  const nextTime = Date.parse(snapshot.updated_at || '');
  const existingTime = Date.parse(existing.updated_at || '');
  if (!Number.isNaN(nextTime) && !Number.isNaN(existingTime) && nextTime < existingTime) {
    return false;
  }
  return true;
}

export function mergeCollaborationSnapshotMap(
  prev: Record<string, Collaboration>,
  snapshot: Collaboration,
  pruneTerminalForChannel?: (
    next: Record<string, Collaboration>,
    channel: string
  ) => Record<string, Collaboration>
): Record<string, Collaboration> {
  if (!snapshot?.id) return prev;
  if (!shouldReplaceCollaborationSnapshot(prev[snapshot.id], snapshot)) {
    return prev;
  }
  let next = { ...prev, [snapshot.id]: snapshot };
  if (isTerminalPhase(snapshot.phase) && snapshot.channel && pruneTerminalForChannel) {
    next = pruneTerminalForChannel(next, snapshot.channel);
  }
  return next;
}
