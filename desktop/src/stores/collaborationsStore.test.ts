import { afterEach, describe, expect, it } from 'vitest';
import { collaborationsByIDSnapshot, useCollaborationsStore } from './collaborationsStore';
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

describe('collaborationsStore', () => {
  afterEach(() => {
    useCollaborationsStore.getState().clear();
  });

  it('mergeSnapshot updates byID and snapshot accessor', () => {
    useCollaborationsStore.getState().mergeSnapshot(
      snap({ id: 'a', phase: 'planning', updated_at: '2026-07-14T12:00:00Z' })
    );
    expect(collaborationsByIDSnapshot().a.phase).toBe('planning');

    useCollaborationsStore.getState().mergeSnapshot(
      snap({ id: 'a', phase: 'executing', updated_at: '2026-07-14T13:00:00Z' })
    );
    expect(useCollaborationsStore.getState().byID.a.phase).toBe('executing');
  });

  it('mergeSnapshot with prune keeps channel-scoped terminals', () => {
    useCollaborationsStore.getState().setByID({
      old: snap({
        id: 'old',
        channel: 'collab-x',
        phase: 'completed',
        updated_at: '2026-07-14T10:00:00Z',
      }),
    });
    useCollaborationsStore.getState().mergeSnapshot(
      snap({
        id: 'new',
        channel: 'collab-x',
        phase: 'completed',
        updated_at: '2026-07-14T14:00:00Z',
      }),
      (next, channelName) => {
        const terminals = Object.values(next)
          .filter(
            (c) =>
              (c.phase === 'completed' || c.phase === 'cancelled') &&
              (c.channel || '') === channelName
          )
          .sort(
            (a, b) => Date.parse(b.updated_at || '') - Date.parse(a.updated_at || '')
          );
        if (terminals.length <= 1) return next;
        const drop = new Set(terminals.slice(1).map((c) => c.id));
        const pruned = { ...next };
        for (const id of drop) delete pruned[id];
        return pruned;
      }
    );
    const byID = useCollaborationsStore.getState().byID;
    expect(byID.new?.phase).toBe('completed');
    expect(byID.old).toBeUndefined();
  });
});
