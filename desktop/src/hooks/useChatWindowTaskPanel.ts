import { useCallback, useEffect, useState } from 'react';
import type { ChatAPI } from '../api/chatAPI';
import type { Toast } from '../stores/toastStore';
import type { AssistantReminder, AssistantTask, Collaboration } from '../types/protocol';

export interface UseChatWindowTaskPanelArgs {
  api: ChatAPI;
  channel: string;
  username: string;
  taskManagementOpen: boolean;
  setTaskManagementOpen: (open: boolean) => void;
  trackedCollaborations: Collaboration[];
  collaborationsByIDRef: { current: Record<string, Collaboration> };
  loadCollaborations: (targetChannel: string) => Promise<void>;
  handleSwitchChannel: (channel: string) => Promise<unknown> | void;
  setActiveCollab: (collab: Collaboration | null) => void;
  clearActiveCollabIf: (collaborationID: string) => void;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
}

/**
 * Assistant tasks/reminders state, polling, and task-management panel props for ChatWindow.
 * Extraction loop 6 from ChatWindowRoot.
 */
export function useChatWindowTaskPanel({
  api,
  channel,
  username,
  taskManagementOpen,
  setTaskManagementOpen,
  trackedCollaborations,
  collaborationsByIDRef,
  loadCollaborations,
  handleSwitchChannel,
  setActiveCollab,
  clearActiveCollabIf,
  addToast,
}: UseChatWindowTaskPanelArgs) {
  const [assistantTasks, setAssistantTasks] = useState<AssistantTask[]>([]);
  const [assistantReminders, setAssistantReminders] = useState<AssistantReminder[]>([]);

  const loadAssistantState = useCallback(async () => {
    try {
      const state = await api.fetchAssistantState(channel);
      setAssistantTasks(state.tasks || []);
      setAssistantReminders(state.reminders || []);
    } catch (error) {
      console.error('Failed to load assistant state:', error);
    }
  }, [api, channel]);

  useEffect(() => {
    if (!taskManagementOpen) return;
    void loadCollaborations(channel);
    loadAssistantState();
    const id = window.setInterval(loadAssistantState, 30000);
    return () => window.clearInterval(id);
  }, [taskManagementOpen, loadAssistantState, loadCollaborations, channel]);

  const taskPanelProps = {
    collaborations: trackedCollaborations,
    assistantTasks,
    assistantReminders,
    onClose: () => setTaskManagementOpen(false),
    onOpenCollaboration: async (collab: Collaboration) => {
      if (collab.channel && collab.channel !== channel) {
        try {
          await handleSwitchChannel(collab.channel);
        } catch (e) {
          console.error('[TaskPanel] failed to switch to collaboration channel:', e);
        }
      }
      setActiveCollab(collab);
      setTaskManagementOpen(false);
    },
    onAssistantTaskDone: async (taskID: string) => {
      const previousTasks = assistantTasks;
      const targetTask = previousTasks.find(task => task.id === taskID);
      setAssistantTasks(prev =>
        prev.map(task => (task.id === taskID ? { ...task, status: 'done' } : task))
      );
      try {
        await api.markAssistantTaskDone(taskID);
        addToast({
          type: 'success',
          title: 'Task marked done',
          message: targetTask ? `"${targetTask.title}" moved to done.` : 'Assistant task moved to done.',
        });
        void loadAssistantState();
      } catch (error) {
        console.error('Failed to mark assistant task done:', error);
        setAssistantTasks(previousTasks);
        addToast({
          type: 'error',
          title: 'Task update failed',
          message: error instanceof Error ? error.message : 'Failed to mark assistant task done.',
        });
      }
    },
    onAssistantReminderDismiss: async (reminderID: string) => {
      const previousReminders = assistantReminders;
      const targetReminder = previousReminders.find(reminder => reminder.id === reminderID);
      setAssistantReminders(prev => prev.filter(reminder => reminder.id !== reminderID));
      try {
        await api.dismissAssistantReminder(reminderID);
        addToast({
          type: 'success',
          title: 'Reminder dismissed',
          message: targetReminder ? `"${targetReminder.content}" dismissed.` : 'Assistant reminder dismissed.',
        });
        void loadAssistantState();
      } catch (error) {
        console.error('Failed to dismiss assistant reminder:', error);
        setAssistantReminders(previousReminders);
        addToast({
          type: 'error',
          title: 'Reminder dismiss failed',
          message: error instanceof Error ? error.message : 'Failed to dismiss assistant reminder.',
        });
      }
    },
    onCollaborationCommand: async (
      command: string,
      collaborationID: string,
      feedbackText?: string,
      taskIndex?: number,
    ) => {
      const from = { name: username || 'User', type: 'human' };
      const shortID = collaborationID.slice(0, 8);
      const collab =
        collaborationsByIDRef.current[collaborationID] ??
        trackedCollaborations.find(c => c.id === collaborationID);
      const targetChannel = collab?.channel?.trim() || channel;
      let content = '';
      if (command === 'approve') {
        content = `/resume-plan ${shortID}`;
      } else if (command === 'revise') {
        const trimmed = (feedbackText || '').trim();
        if (!trimmed) {
          throw new Error('Revision feedback is required.');
        }
        content = `/revise-plan ${shortID} ${trimmed}`;
      } else if (command === 'complete') {
        const open = collab?.tasks?.filter(t => t.status !== 'completed') ?? [];
        content = `/complete-collab ${shortID}${open.length > 0 ? ' --force' : ''}`;
      } else if (command === 'task-done') {
        if (taskIndex == null || taskIndex < 0) {
          throw new Error('Task index is required.');
        }
        content = `/collab-task-done ${shortID} ${taskIndex + 1}`;
      } else {
        content = `/cancel-plan ${shortID}`;
      }
      try {
        if (targetChannel !== channel) {
          await handleSwitchChannel(targetChannel);
        }
        await api.sendMessage(targetChannel, content, from);
        await loadCollaborations(targetChannel);
        if (command === 'cancel') {
          clearActiveCollabIf(collaborationID);
          setTaskManagementOpen(false);
        }
      } catch (e) {
        addToast({
          type: 'error',
          title: 'Collaboration command failed',
          message: e instanceof Error ? e.message : 'Request failed.',
        });
        throw e;
      }
    },
  };

  return {
    assistantTasks,
    setAssistantTasks,
    setAssistantReminders,
    taskPanelProps,
  };
}
