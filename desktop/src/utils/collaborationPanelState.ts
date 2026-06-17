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

/** Agent display names that have not spoken this round while discussion is underway. */
export function planningStalledParticipantNames(collaboration: Collaboration | null): string[] {
  if (!collaboration || collaboration.phase !== 'planning') {
    return [];
  }
  const d = collaboration.discussion;
  if (!d || (d.total_message_count ?? 0) === 0) {
    return [];
  }
  const turns = d.turns_this_round ?? {};
  const participantIds = d.participants ?? [];
  if (participantIds.length === 0) {
    return [];
  }
  const stalled: string[] = [];
  for (const agentId of participantIds) {
    if ((turns[agentId] ?? 0) < 1) {
      const name =
        collaboration.agents?.find((a) => a.agent_id === agentId)?.agent_name?.trim() ||
        agentId.slice(0, 8);
      if (name && !stalled.includes(name)) {
        stalled.push(name);
      }
    }
  }
  return stalled;
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

/** Sandbox + bound project repo: hub may acknowledge workspace immediately after approve/start. */
export function shouldAutoAckWorkspaceOnApprove(collaboration: Collaboration | null): boolean {
  if (!collaboration) {
    return false;
  }
  if (collaboration.execution_mode === 'worktree') {
    return false;
  }
  return !!collaboration.source_repo_path?.trim();
}

/** Executing phase blocked until user confirms workspace (worktree or sandbox with path). */
export function isAwaitingWorkspaceConfirmation(collaboration: Collaboration | null): boolean {
  if (!collaboration || collaboration.phase !== 'executing') {
    return false;
  }
  if (collaboration.workspace_acknowledged) {
    return false;
  }
  const isWorktree = collaboration.execution_mode === 'worktree';
  return isWorktree || !!collaboration.working_directory?.trim();
}

/** Approved plan; workspace auto-ack path is dispatching tasks without the Continue gate. */
export function isApprovedAwaitingDispatch(collaboration: Collaboration | null): boolean {
  if (!collaboration || collaboration.phase !== 'approved') {
    return false;
  }
  return !!collaboration.workspace_acknowledged && !collaboration.tasks_dispatched;
}
