import { describe, expect, it } from 'vitest';
import {
  mergeCollaborationSnapshotMap,
  shouldReplaceCollaborationSnapshot,
} from './collaborationSnapshots';
import type { Collaboration } from '../types/protocol';

function snap(partial: Partial<Collaboration> & { id: string }): Collaboration {
  return {
    id: partial.id,
    title: partial.title ?? 't',
    phase: partial.phase ?? 'planning',
    channel: partial.channel ?? 'collab-1',
    updated_at: partial.updated_at ?? '2026-07-14T12:00:00Z',
    workspace_acknowledged: partial.workspace_acknowledged ?? false,
    agents: [],
    tasks: [],
  } as Collaboration;
}

describe('collaborationSnapshots', () => {
  it('rejects older snapshots', () => {
    const existing = snap({ id: 'a', updated_at: '2026-07-14T13:00:00Z' });
    const older = snap({ id: 'a', updated_at: '2026-07-14T12:00:00Z', phase: 'executing' });
    expect(shouldReplaceCollaborationSnapshot(existing, older)).toBe(false);
  });

  it('merges newer snapshot', () => {
    const prev = { a: snap({ id: 'a', phase: 'planning' }) };
    const next = mergeCollaborationSnapshotMap(
      prev,
      snap({ id: 'a', phase: 'reviewing', updated_at: '2026-07-14T14:00:00Z' })
    );
    expect(next.a.phase).toBe('reviewing');
  });
});
