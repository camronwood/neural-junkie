import { describe, expect, it } from 'vitest';
import type { CollaborationTask } from '../types/protocol';
import {
  applyEdgeConnect,
  applyEdgeRemove,
  autoLayoutDagre,
  removeTask,
  tasksToFlow,
  validateDAG,
} from './runbookDAG';

function task(
  partial: Partial<CollaborationTask> & Pick<CollaborationTask, 'id' | 'title'>
): CollaborationTask {
  return {
    id: partial.id,
    title: partial.title,
    description: '',
    assigned_to: '',
    assigned_name: '',
    status: partial.status ?? 'pending',
    dependencies: partial.dependencies,
    created_at: '',
    updated_at: '',
  };
}

describe('validateDAG', () => {
  it('accepts acyclic graph', () => {
    const tasks = [
      task({ id: 'a', title: 'A' }),
      task({ id: 'b', title: 'B', dependencies: ['a'] }),
    ];
    expect(validateDAG(tasks)).toEqual({ ok: true });
  });

  it('rejects cycle', () => {
    const tasks = [
      task({ id: 'a', title: 'A', dependencies: ['b'] }),
      task({ id: 'b', title: 'B', dependencies: ['a'] }),
    ];
    const r = validateDAG(tasks);
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error).toContain('cycle');
  });

  it('rejects unknown dependency', () => {
    const tasks = [task({ id: 'a', title: 'A', dependencies: ['missing'] })];
    const r = validateDAG(tasks);
    expect(r.ok).toBe(false);
  });
});

describe('tasksToFlow', () => {
  it('builds edges from dependencies', () => {
    const tasks = [
      task({ id: 'a', title: 'A' }),
      task({ id: 'b', title: 'B', dependencies: ['a'] }),
    ];
    const { nodes, edges } = tasksToFlow(tasks, {});
    expect(nodes).toHaveLength(2);
    expect(edges).toHaveLength(1);
    expect(edges[0].source).toBe('a');
    expect(edges[0].target).toBe('b');
  });
});

describe('applyEdgeConnect', () => {
  it('adds dependency on target', () => {
    const tasks = [task({ id: 'a', title: 'A' }), task({ id: 'b', title: 'B' })];
    const { tasks: next, error } = applyEdgeConnect(tasks, 'a', 'b');
    expect(error).toBeUndefined();
    expect(next[1].dependencies).toEqual(['a']);
  });

  it('rejects cycle', () => {
    const tasks = [
      task({ id: 'a', title: 'A', dependencies: ['b'] }),
      task({ id: 'b', title: 'B' }),
    ];
    const { error } = applyEdgeConnect(tasks, 'a', 'b');
    expect(error).toBeDefined();
  });
});

describe('applyEdgeRemove', () => {
  it('removes one dep', () => {
    const tasks = [task({ id: 'b', title: 'B', dependencies: ['a'] })];
    const next = applyEdgeRemove(tasks, 'b', 'a');
    expect(next[0].dependencies).toEqual([]);
  });
});

describe('removeTask', () => {
  it('strips deps pointing at removed id', () => {
    const tasks = [
      task({ id: 'a', title: 'A' }),
      task({ id: 'b', title: 'B', dependencies: ['a'] }),
    ];
    const next = removeTask(tasks, 'a');
    expect(next).toHaveLength(1);
    expect(next[0].dependencies).toEqual([]);
  });
});

describe('autoLayoutDagre', () => {
  it('assigns positions for all tasks', () => {
    const tasks = [
      task({ id: 'a', title: 'A' }),
      task({ id: 'b', title: 'B', dependencies: ['a'] }),
    ];
    const layout = autoLayoutDagre(tasks, {});
    expect(layout.a).toBeDefined();
    expect(layout.b).toBeDefined();
  });
});
