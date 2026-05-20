import type { CollaborationTask } from '../types/protocol';

export const MAX_RUNBOOK_TASKS = 25;

export function newTaskId(): string {
  return typeof crypto !== 'undefined' && crypto.randomUUID
    ? crypto.randomUUID()
    : `task-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

export function createEmptyTask(): CollaborationTask {
  const now = new Date().toISOString();
  return {
    id: newTaskId(),
    title: '',
    description: '',
    assigned_to: '',
    assigned_name: '',
    status: 'pending',
    dependencies: [],
    created_at: now,
    updated_at: now,
  };
}
