import type { Collaboration } from '../types/protocol';

/** Panel uses the latest hub snapshot when the store map is ahead of activeCollab state. */
export function resolvePanelCollaboration(
  active: Collaboration | null,
  byId: Record<string, Collaboration>
): Collaboration | null {
  if (!active?.id) {
    return null;
  }
  return byId[active.id] ?? active;
}

export function isNonTerminalCollaborationPhase(phase?: Collaboration['phase']): boolean {
  return phase !== 'completed' && phase !== 'cancelled';
}
