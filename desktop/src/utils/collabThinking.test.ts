import { describe, expect, it } from 'vitest';
import { getCurrentTurnAgent } from './collabThinking';
import type { Collaboration } from '../types/protocol';

describe('getCurrentTurnAgent', () => {
  it('uses discussion participants and current_turn_index', () => {
    const collab: Collaboration = {
      id: 'c1',
      title: 't',
      description: 'd',
      phase: 'planning',
      channel: 'collab-c1',
      created_by: 'u',
      created_at: '',
      updated_at: '',
      agents: [
        { agent_id: 'a1', agent_name: 'A', role: 'r', agent_type: 'assistant', expertise: [] },
        { agent_id: 'a2', agent_name: 'B', role: 'r', agent_type: 'cli', expertise: [] },
      ],
      discussion: {
        id: 'd1',
        collaboration_id: 'c1',
        topic: 't',
        participants: ['a1', 'a2'],
        max_rounds: 3,
        current_round: 1,
        turn_budget: 1,
        total_message_count: 0,
        max_total_messages: 20,
        status: 'active',
        current_turn_index: 1,
        consensus: {},
      },
    };
    expect(getCurrentTurnAgent(collab)?.agent_name).toBe('B');
  });
});
