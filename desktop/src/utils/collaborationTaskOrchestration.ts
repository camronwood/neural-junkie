import type { CollaborationTask } from '../types/protocol';

export function taskBlockedByTitles(task: CollaborationTask, tasks: CollaborationTask[]): string[] {
  if (!task.dependencies?.length) {
    return [];
  }
  const byId = new Map(tasks.map((t) => [t.id, t]));
  const titles: string[] = [];
  for (const depId of task.dependencies) {
    const dep = byId.get(depId);
    if (dep && dep.status !== 'completed') {
      titles.push(dep.title || depId.slice(0, 8));
    }
  }
  return titles;
}

export function taskOrchestrationLabel(
  task: CollaborationTask,
  tasks: CollaborationTask[],
  phase: string
): string | null {
  if (phase !== 'executing' && phase !== 'reviewing' && phase !== 'draft') {
    return null;
  }
  if (task.status === 'completed') {
    return null;
  }
  const blocked = taskBlockedByTitles(task, tasks);
  if (blocked.length > 0) {
    return `Waiting on: ${blocked.join(', ')}`;
  }
  if (task.prompt_dispatched) {
    return 'Dispatched';
  }
  if (task.status === 'pending') {
    return 'Ready';
  }
  return null;
}
