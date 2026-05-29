import type { Collaboration, CollaborationTask } from '../types/protocol';

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

/** Planning started but no agent has posted yet (common right after /collaborate). */
export function isPlanningAwaitingFirstTurn(collaboration: Collaboration | null): boolean {
  if (!collaboration || collaboration.phase !== 'planning') {
    return false;
  }
  return (collaboration.discussion?.total_message_count ?? 0) === 0;
}

/** Heuristic: task text asks for a concrete file deliverable. */
export function taskNeedsFileDeliverable(task: CollaborationTask): boolean {
  const combined = `${task.title} ${task.description ?? ''}`.toLowerCase();
  if (!/\b(write|create|draft|produce|emit)\b/.test(combined)) {
    return false;
  }
  return (
    /\.(md|markdown|yaml|yml|json|txt|go|rs|ts|tsx|py)\b/.test(combined) ||
    combined.includes('collabs/')
  );
}

/** Approved plan; workspace auto-ack path is dispatching tasks without the Continue gate. */
export function isApprovedAwaitingDispatch(collaboration: Collaboration | null): boolean {
  if (!collaboration || collaboration.phase !== 'approved') {
    return false;
  }
  return !!collaboration.workspace_acknowledged && !collaboration.tasks_dispatched;
}
