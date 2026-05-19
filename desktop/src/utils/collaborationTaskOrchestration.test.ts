import { describe, expect, it } from 'vitest';
import type { CollaborationTask } from '../types/protocol';
import { taskBlockedByTitles, taskOrchestrationLabel } from './collaborationTaskOrchestration';

function task(partial: Partial<CollaborationTask> & Pick<CollaborationTask, 'id' | 'title' | 'status'>): CollaborationTask {
  return {
    id: partial.id,
    title: partial.title,
    description: '',
    assigned_to: '',
    assigned_name: '',
    status: partial.status,
    dependencies: partial.dependencies,
    prompt_dispatched: partial.prompt_dispatched,
    created_at: '',
    updated_at: '',
  };
}

describe('collaborationTaskOrchestration', () => {
  it('taskBlockedByTitles lists incomplete dependencies', () => {
    const tasks = [
      task({ id: 'a', title: 'Scaffold', status: 'in_progress' }),
      task({ id: 'b', title: 'Integrate', status: 'pending', dependencies: ['a'] }),
    ];
    expect(taskBlockedByTitles(tasks[1], tasks)).toEqual(['Scaffold']);
  });

  it('taskOrchestrationLabel shows waiting when deps incomplete', () => {
    const tasks = [
      task({ id: 'a', title: 'A', status: 'pending' }),
      task({ id: 'b', title: 'B', status: 'pending', dependencies: ['a'] }),
    ];
    expect(taskOrchestrationLabel(tasks[1], tasks, 'executing')).toBe('Waiting on: A');
  });

  it('taskOrchestrationLabel shows Ready when deps done', () => {
    const tasks = [
      task({ id: 'a', title: 'A', status: 'completed' }),
      task({ id: 'b', title: 'B', status: 'pending', dependencies: ['a'] }),
    ];
    expect(taskOrchestrationLabel(tasks[1], tasks, 'executing')).toBe('Ready');
  });

  it('taskOrchestrationLabel shows Dispatched when prompt sent', () => {
    const tasks = [
      task({ id: 'a', title: 'A', status: 'completed' }),
      task({ id: 'b', title: 'B', status: 'pending', dependencies: ['a'], prompt_dispatched: true }),
    ];
    expect(taskOrchestrationLabel(tasks[1], tasks, 'executing')).toBe('Dispatched');
  });
});
