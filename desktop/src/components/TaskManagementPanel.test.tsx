import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { TaskManagementPanel } from './TaskManagementPanel';
import type { Collaboration } from '../types/protocol';

function makeCollaboration(overrides: Partial<Collaboration> = {}): Collaboration {
  return {
    id: 'collab-12345678',
    title: 'Ship collaboration UI fixes',
    description: 'Validate task board behavior for collaboration workflows',
    phase: 'reviewing',
    agents: [],
    tasks: [],
    channel: 'general',
    created_by: 'tester',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

afterEach(() => {
  cleanup();
});

describe('TaskManagementPanel collaboration regressions', () => {
  it('shows collaboration rows even when there are zero tasks and supports approve/revise/cancel actions', async () => {
    const onCollaborationCommand = vi.fn(async () => {});
    const collab = makeCollaboration();

    render(
      <TaskManagementPanel
        collaborations={[collab]}
        assistantTasks={[]}
        assistantReminders={[]}
        onClose={() => {}}
        onOpenCollaboration={() => {}}
        onAssistantTaskDone={() => {}}
        onAssistantReminderDismiss={() => {}}
        onCollaborationCommand={onCollaborationCommand}
      />
    );

    expect(screen.getByText('Collaborations • 1 tracked')).toBeInTheDocument();
    expect(screen.getByText('Ship collaboration UI fixes')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Resume plan'));
    await waitFor(() =>
      expect(onCollaborationCommand).toHaveBeenCalledWith('approve', collab.id)
    );

    fireEvent.change(screen.getByPlaceholderText(/Revision feedback/i), {
      target: { value: 'Please split the plan into two phases' },
    });
    fireEvent.click(screen.getByText('Revise'));
    await waitFor(() =>
      expect(onCollaborationCommand).toHaveBeenCalledWith(
        'revise',
        collab.id,
        'Please split the plan into two phases'
      )
    );

    fireEvent.click(screen.getByText('Cancel'));
    await waitFor(() =>
      expect(onCollaborationCommand).toHaveBeenCalledWith('cancel', collab.id)
    );
  });

  it('shows @unassigned fallback when a task has no assignee name', () => {
    const collab = makeCollaboration({
      phase: 'executing',
      tasks: [
        {
          id: 'task-1',
          title: 'Implement hydration',
          description: 'Hydrate collaboration snapshots from API',
          assigned_to: '',
          assigned_name: '',
          status: 'pending',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    });

    render(
      <TaskManagementPanel
        collaborations={[collab]}
        assistantTasks={[]}
        assistantReminders={[]}
        onClose={() => {}}
        onOpenCollaboration={() => {}}
        onAssistantTaskDone={() => {}}
        onAssistantReminderDismiss={() => {}}
        onCollaborationCommand={async () => {}}
      />
    );

    expect(screen.getByText('@unassigned')).toBeInTheDocument();
  });

  it('mark done calls onCollaborationCommand with complete when user confirms', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
    const onCollaborationCommand = vi.fn(async () => {});
    const collab = makeCollaboration({
      phase: 'executing',
      tasks: [
        {
          id: 'task-1',
          title: 'Open work',
          description: 'd',
          assigned_to: 'a1',
          assigned_name: 'Dev',
          status: 'pending',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    });

    render(
      <TaskManagementPanel
        collaborations={[collab]}
        assistantTasks={[]}
        assistantReminders={[]}
        onClose={() => {}}
        onOpenCollaboration={() => {}}
        onAssistantTaskDone={() => {}}
        onAssistantReminderDismiss={() => {}}
        onCollaborationCommand={onCollaborationCommand}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Mark done \(1 open\)/i }));

    await waitFor(() =>
      expect(onCollaborationCommand).toHaveBeenCalledWith('complete', collab.id)
    );
    confirmSpy.mockRestore();
  });

  it('mark done without open tasks calls complete without confirm', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm');
    const onCollaborationCommand = vi.fn(async () => {});
    const collab = makeCollaboration({
      phase: 'executing',
      tasks: [
        {
          id: 'task-1',
          title: 'Done work',
          description: 'd',
          assigned_to: 'a1',
          assigned_name: 'Dev',
          status: 'completed',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    });

    render(
      <TaskManagementPanel
        collaborations={[collab]}
        assistantTasks={[]}
        assistantReminders={[]}
        onClose={() => {}}
        onOpenCollaboration={() => {}}
        onAssistantTaskDone={() => {}}
        onAssistantReminderDismiss={() => {}}
        onCollaborationCommand={onCollaborationCommand}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /^Mark done$/ }));

    await waitFor(() =>
      expect(onCollaborationCommand).toHaveBeenCalledWith('complete', collab.id)
    );
    expect(confirmSpy).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });
});
