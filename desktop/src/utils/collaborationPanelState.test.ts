import { describe, expect, it } from 'vitest';
import type { Collaboration } from '../types/protocol';
import {
  isApprovedAwaitingDispatch,
  isPlanningAwaitingFirstTurn,
  planningStalledParticipantNames,
  resolvePanelCollaboration,
  taskNeedsFileDeliverable,
} from './collaborationPanelState';

function collab(overrides: Partial<Collaboration> = {}): Collaboration {
  return {
    id: 'cid-1',
    title: 'Test',
    description: 'd',
    phase: 'planning',
    agents: [],
    channel: 'collab-cid-1',
    created_by: 'u',
    created_at: '',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('resolvePanelCollaboration', () => {
  it('prefers fresher snapshot from collaborationsByID', () => {
    const active = collab({ phase: 'planning', updated_at: '2026-01-01T00:00:00Z' });
    const byId = {
      'cid-1': collab({ phase: 'reviewing', updated_at: '2026-01-02T00:00:00Z' }),
    };
    const resolved = resolvePanelCollaboration(active, byId);
    expect(resolved?.phase).toBe('reviewing');
  });

  it('returns null when active is null', () => {
    expect(resolvePanelCollaboration(null, {})).toBeNull();
  });
});

describe('isPlanningAwaitingFirstTurn', () => {
  it('is true when planning with zero messages', () => {
    expect(
      isPlanningAwaitingFirstTurn(
        collab({
          phase: 'planning',
          discussion: {
            id: 'd',
            collaboration_id: 'cid-1',
            current_round: 1,
            max_rounds: 2,
            turn_budget: 1,
            total_message_count: 0,
            max_total_messages: 12,
            status: 'active',
            started_at: '',
          },
        })
      )
    ).toBe(true);
  });
});

describe('taskNeedsFileDeliverable', () => {
  it('detects write md tasks', () => {
    expect(
      taskNeedsFileDeliverable({
        id: 't1',
        title: 'Write collabs/x/findings.md',
        status: 'in_progress',
      })
    ).toBe(true);
  });
});

describe('planningStalledParticipantNames', () => {
  it('lists participants with zero turns when discussion has started', () => {
    const names = planningStalledParticipantNames(
      collab({
        phase: 'planning',
        agents: [
          {
            agent_id: 'a1',
            agent_name: 'BackendEngineer',
            agent_type: 'backend',
            expertise: [],
            role: 'backend',
          },
          {
            agent_id: 'a2',
            agent_name: 'SoftwareArchitect',
            agent_type: 'architecture',
            expertise: [],
            role: 'architect',
          },
        ],
        discussion: {
          id: 'd',
          collaboration_id: 'cid-1',
          participants: ['a1', 'a2'],
          current_round: 1,
          max_rounds: 2,
          turn_budget: 1,
          total_message_count: 2,
          max_total_messages: 12,
          status: 'active',
          turns_this_round: { a1: 1, a2: 0 },
        },
      })
    );
    expect(names).toEqual(['SoftwareArchitect']);
  });
});

describe('isApprovedAwaitingDispatch', () => {
  it('is true when workspace acked but tasks not dispatched', () => {
    expect(
      isApprovedAwaitingDispatch(
        collab({
          phase: 'approved',
          workspace_acknowledged: true,
          tasks_dispatched: false,
        })
      )
    ).toBe(true);
  });
});
