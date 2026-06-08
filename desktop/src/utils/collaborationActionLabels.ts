import type { CollaborationPhase, DiscussionSession, DiscussionStatus } from '../types/protocol';

const planningDiscussionCompleteStatuses: DiscussionStatus[] = [
  'converged',
  'budget_exhausted',
  'timed_out',
];

/** True when agent planning discussion has finished (not still taking turns). */
export function isPlanningDiscussionComplete(
  discussion?: DiscussionSession
): boolean {
  if (!discussion) return false;
  return planningDiscussionCompleteStatuses.includes(discussion.status);
}

/** Whether Submit for review is allowed during the planning phase. */
export function canSubmitCollaborationForReview(
  phase: CollaborationPhase,
  discussion?: DiscussionSession
): boolean {
  return phase === 'planning' && isPlanningDiscussionComplete(discussion);
}

export function collaborationSubmitForReviewTitle(
  discussion?: DiscussionSession
): string | undefined {
  if (!discussion) {
    return 'Planning discussion has not started yet';
  }
  if (discussion.status === 'active') {
    return 'Unlocks when planning finishes (consensus, message limit, or round limit)';
  }
  if (canSubmitCollaborationForReview('planning', discussion)) {
    return 'Submit the plan for your review and session summary';
  }
  return undefined;
}

/** Primary action on the collaboration panel (approve, re-dispatch, etc.). */
export function collaborationPrimaryActionLabel(
  phase: CollaborationPhase,
  options?: { anotherCollabExecuting?: boolean; awaitingWorkspaceConfirmation?: boolean }
): string | null {
  if (phase === 'executing') {
    if (options?.awaitingWorkspaceConfirmation) {
      return 'Confirm workspace';
    }
    return 'Re-dispatch tasks';
  }
  if (phase === 'reviewing' || phase === 'approved') {
    if (options?.anotherCollabExecuting) {
      return 'Approve & start (stop other run)';
    }
    return 'Approve & start';
  }
  return null;
}

export function collaborationPrimaryActionTitle(
  phase: CollaborationPhase,
  options?: { awaitingWorkspaceConfirmation?: boolean }
): string | undefined {
  if (phase === 'executing') {
    if (options?.awaitingWorkspaceConfirmation) {
      return 'Open the workspace confirmation dialog so agents can receive task prompts';
    }
    return 'Re-send task prompts for open work (pending, in progress, or blocked)';
  }
  if (phase === 'reviewing' || phase === 'approved') {
    return 'Approve the plan, create the collaboration workspace, and start task execution';
  }
  return undefined;
}
