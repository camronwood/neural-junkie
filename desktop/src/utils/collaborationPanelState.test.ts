import { describe, expect, it } from 'vitest';
import type { Collaboration } from '../types/protocol';
import { resolvePanelCollaboration } from './collaborationPanelState';

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
