import { describe, expect, it } from 'vitest';
import {
  canSubmitCollaborationForReview,
  collaborationPrimaryActionLabel,
  collaborationPrimaryActionTitle,
  isPlanningDiscussionComplete,
} from './collaborationActionLabels';
import type { DiscussionSession } from '../types/protocol';

function discussion(status: DiscussionSession['status']): DiscussionSession {
  return {
    id: 'd1',
    collaboration_id: 'c1',
    topic: 't',
    participants: [],
    max_rounds: 3,
    current_round: 3,
    turn_budget: 1,
    total_message_count: 20,
    max_total_messages: 20,
    status,
    current_turn_index: 0,
    consensus: {},
  };
}

describe('collaborationPrimaryActionLabel', () => {
  it('returns Approve & start for reviewing', () => {
    expect(collaborationPrimaryActionLabel('reviewing')).toBe('Approve & start');
  });

  it('returns Re-dispatch tasks for executing', () => {
    expect(collaborationPrimaryActionLabel('executing')).toBe('Re-dispatch tasks');
  });

  it('returns null for planning', () => {
    expect(collaborationPrimaryActionLabel('planning')).toBeNull();
  });

  it('annotates when another collab is executing', () => {
    expect(
      collaborationPrimaryActionLabel('reviewing', { anotherCollabExecuting: true })
    ).toBe('Approve & start (stop other run)');
  });
});

describe('collaborationPrimaryActionTitle', () => {
  it('describes approve action', () => {
    expect(collaborationPrimaryActionTitle('reviewing')).toContain('Approve');
  });
});

describe('planning discussion completion', () => {
  it('treats active discussion as incomplete', () => {
    expect(isPlanningDiscussionComplete(discussion('active'))).toBe(false);
    expect(canSubmitCollaborationForReview('planning', discussion('active'))).toBe(false);
  });

  it('allows submit when discussion finished', () => {
    expect(isPlanningDiscussionComplete(discussion('converged'))).toBe(true);
    expect(canSubmitCollaborationForReview('planning', discussion('budget_exhausted'))).toBe(true);
  });
});
