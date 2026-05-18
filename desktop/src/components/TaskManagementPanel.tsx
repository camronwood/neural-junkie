import { useMemo, useState } from 'react';
import type {
  AssistantReminder,
  AssistantTask,
  Collaboration,
  CollaborationTask,
  CollaborationTaskStatus,
} from '../types/protocol';
import { confirmReplaceCollaborationExecution } from '../utils/collaborationConfirm';

type TaskViewMode = 'by_agent' | 'by_collaboration';
type StatusFilter = 'all' | CollaborationTaskStatus;

interface TaskWithContext {
  task: CollaborationTask;
  collaboration: Collaboration;
}

interface TaskManagementPanelProps {
  collaborations: Collaboration[];
  assistantTasks: AssistantTask[];
  assistantReminders: AssistantReminder[];
  onClose: () => void;
  onOpenCollaboration: (collaboration: Collaboration) => void;
  onAssistantTaskDone: (taskID: string) => void;
  onAssistantReminderDismiss: (reminderID: string) => void;
  onCollaborationCommand: (
    command: 'approve' | 'revise' | 'cancel',
    collaborationID: string,
    feedback?: string
  ) => Promise<void>;
}

function statusIcon(status: CollaborationTaskStatus): string {
  switch (status) {
    case 'in_progress':
      return '🔄';
    case 'completed':
      return '✅';
    case 'blocked':
      return '🚫';
    default:
      return '⬜';
  }
}

const statusOrder: Record<CollaborationTaskStatus, number> = {
  in_progress: 0,
  blocked: 1,
  pending: 2,
  completed: 3,
};

function toTimestamp(value?: string): number {
  if (!value) return 0;
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? 0 : parsed;
}

function formatUpdatedAt(value?: string): string {
  if (!value) return 'unknown';
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) return 'unknown';
  return new Date(timestamp).toLocaleTimeString();
}

export function TaskManagementPanel({
  collaborations,
  assistantTasks,
  assistantReminders,
  onClose,
  onOpenCollaboration,
  onAssistantTaskDone,
  onAssistantReminderDismiss,
  onCollaborationCommand,
}: TaskManagementPanelProps) {
  const [viewMode, setViewMode] = useState<TaskViewMode>('by_agent');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [query, setQuery] = useState('');
  const [assigneeFilter, setAssigneeFilter] = useState<string>('');
  const [revisionFeedback, setRevisionFeedback] = useState<Record<string, string>>({});
  const [submittingByCollab, setSubmittingByCollab] = useState<Record<string, boolean>>({});
  const statusFilters: StatusFilter[] = ['all', 'pending', 'in_progress', 'blocked', 'completed'];

  const tasks = useMemo(() => {
    const rows: TaskWithContext[] = [];
    for (const collab of collaborations) {
      for (const task of collab.tasks ?? []) {
        rows.push({ task, collaboration: collab });
      }
    }

    const queryLower = query.trim().toLowerCase();
    return rows
      .filter(({ task }) => statusFilter === 'all' || task.status === statusFilter)
      .filter(({ task }) => (assigneeFilter ? task.assigned_name === assigneeFilter : true))
      .filter(({ task, collaboration }) => {
        if (!queryLower) return true;
        return (
          task.title.toLowerCase().includes(queryLower) ||
          task.description.toLowerCase().includes(queryLower) ||
          task.assigned_name.toLowerCase().includes(queryLower) ||
          collaboration.title.toLowerCase().includes(queryLower)
        );
      })
      .sort((a, b) => {
        const statusCmp = statusOrder[a.task.status] - statusOrder[b.task.status];
        if (statusCmp !== 0) return statusCmp;
        return toTimestamp(b.task.updated_at) - toTimestamp(a.task.updated_at);
      });
  }, [assigneeFilter, collaborations, query, statusFilter]);

  const totals = useMemo(() => {
    const all = tasks.length;
    const inProgress = tasks.filter(t => t.task.status === 'in_progress').length;
    const completed = tasks.filter(t => t.task.status === 'completed').length;
    const blocked = tasks.filter(t => t.task.status === 'blocked').length;
    return { all, inProgress, completed, blocked };
  }, [tasks]);

  const groupedByAgent = useMemo(() => {
    const out = new Map<string, TaskWithContext[]>();
    for (const row of tasks) {
      const key = row.task.assigned_name || 'Unassigned';
      if (!out.has(key)) out.set(key, []);
      out.get(key)!.push(row);
    }
    return out;
  }, [tasks]);

  const groupedByCollab = useMemo(() => {
    const out = new Map<string, TaskWithContext[]>();
    for (const row of tasks) {
      const key = row.collaboration.id;
      if (!out.has(key)) out.set(key, []);
      out.get(key)!.push(row);
    }
    return out;
  }, [tasks]);

  const visibleCollaborations = useMemo(() => {
    const queryLower = query.trim().toLowerCase();
    return collaborations
      .filter(collab => {
        if (!queryLower) return true;
        return (
          collab.title.toLowerCase().includes(queryLower) ||
          collab.description.toLowerCase().includes(queryLower) ||
          collab.phase.toLowerCase().includes(queryLower)
        );
      })
      .sort((a, b) => toTimestamp(b.updated_at) - toTimestamp(a.updated_at));
  }, [collaborations, query]);

  const assistantPendingTasks = useMemo(
    () => assistantTasks.filter(task => task.status !== 'done'),
    [assistantTasks]
  );
  const assistantDoneTasks = useMemo(
    () => assistantTasks.filter(task => task.status === 'done'),
    [assistantTasks]
  );
  const activeReminders = useMemo(
    () => assistantReminders.filter(reminder => reminder.active),
    [assistantReminders]
  );
  const hasAssistantData = assistantTasks.length > 0 || activeReminders.length > 0;

  const executingCollaboration = useMemo(
    () => collaborations.find(c => c.phase === 'executing') ?? null,
    [collaborations]
  );

  const runCollabCommand = async (command: 'approve' | 'revise' | 'cancel', collab: Collaboration) => {
    const collabID = collab.id;
    if (!collabID) return;
    if (
      command === 'approve' &&
      !confirmReplaceCollaborationExecution(executingCollaboration, collab)
    ) {
      return;
    }
    setSubmittingByCollab(prev => ({ ...prev, [collabID]: true }));
    try {
      if (command === 'revise') {
        await onCollaborationCommand('revise', collabID, revisionFeedback[collabID] || '');
        setRevisionFeedback(prev => ({ ...prev, [collabID]: '' }));
      } else {
        await onCollaborationCommand(command, collabID);
      }
    } finally {
      setSubmittingByCollab(prev => ({ ...prev, [collabID]: false }));
    }
  };

  return (
    <div
      style={{
        width: 430,
        borderLeft: '1px solid var(--border-color, #333)',
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: 'var(--bg-secondary, #1e1e1e)',
        height: '100%',
      }}
    >
      <div
        style={{
          padding: '12px 16px',
          borderBottom: '1px solid var(--border-color, #333)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 8,
        }}
      >
        <div>
          <div style={{ fontWeight: 600, fontSize: 14 }}>Task Management</div>
          <div style={{ fontSize: 12, color: 'var(--text-secondary, #999)' }}>
            {totals.all} task(s) • {visibleCollaborations.length} active collaboration(s) • {totals.inProgress} in progress • {totals.completed} completed • {totals.blocked} blocked
          </div>
        </div>
        <button
          onClick={onClose}
          style={{
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--text-secondary, #888)',
            fontSize: 18,
            padding: '6px',
            borderRadius: 6,
            lineHeight: 1,
          }}
          aria-label="Close task management panel"
        >
          ×
        </button>
      </div>

      <div style={{ padding: 12, borderBottom: '1px solid var(--border-color, #333)', display: 'grid', gap: 8 }}>
        <div style={{ display: 'flex', gap: 6 }}>
          <button
            onClick={() => setViewMode('by_agent')}
            style={{
              padding: '4px 10px',
              borderRadius: 999,
              border: '1px solid var(--border-color, #444)',
              backgroundColor: viewMode === 'by_agent' ? '#3b82f6' : 'transparent',
              color: viewMode === 'by_agent' ? '#fff' : 'var(--text-secondary, #aaa)',
              fontSize: 12,
              cursor: 'pointer',
            }}
          >
            By Agent
          </button>
          <button
            onClick={() => setViewMode('by_collaboration')}
            style={{
              padding: '4px 10px',
              borderRadius: 999,
              border: '1px solid var(--border-color, #444)',
              backgroundColor: viewMode === 'by_collaboration' ? '#3b82f6' : 'transparent',
              color: viewMode === 'by_collaboration' ? '#fff' : 'var(--text-secondary, #aaa)',
              fontSize: 12,
              cursor: 'pointer',
            }}
          >
            By Collaboration
          </button>
        </div>

        <input
          type="text"
          value={query}
          onChange={e => setQuery(e.target.value)}
          placeholder="Filter tasks, agents, collaborations..."
          style={{
            width: '100%',
            padding: '6px 10px',
            borderRadius: 6,
            border: '1px solid var(--border-color, #444)',
            backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
            color: 'var(--text-primary, #eee)',
            fontSize: 12,
          }}
        />

        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
          {statusFilters.map(filter => (
            <button
              key={filter}
              onClick={() => setStatusFilter(filter)}
              style={{
                padding: '3px 8px',
                borderRadius: 999,
                border: '1px solid var(--border-color, #444)',
                backgroundColor: statusFilter === filter ? '#8b5cf6' : 'transparent',
                color: statusFilter === filter ? '#fff' : 'var(--text-secondary, #aaa)',
                fontSize: 11,
                cursor: 'pointer',
              }}
            >
              {filter}
            </button>
          ))}
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
        {tasks.length === 0 && visibleCollaborations.length === 0 && !hasAssistantData ? (
          <div
            style={{
              border: '1px dashed var(--border-color, #444)',
              borderRadius: 8,
              padding: 12,
              color: 'var(--text-secondary, #999)',
              fontSize: 13,
            }}
          >
            No collaboration or assistant tasks match the current filters. Start one with `/collaborate`, `/task-add`, or `remind me in 10m ...`.
          </div>
        ) : (
          <>
            <div style={{ marginBottom: 14 }}>
              <div style={{ fontSize: 12, color: 'var(--text-secondary, #999)', marginBottom: 6 }}>
                Collaborations • {visibleCollaborations.length} tracked
              </div>
              {visibleCollaborations.length === 0 ? (
                <div style={{ marginBottom: 12, color: 'var(--text-secondary, #999)', fontSize: 12 }}>
                  No collaborations match current filters.
                </div>
              ) : (
                visibleCollaborations.map(collab => (
                  <CollaborationRow
                    key={collab.id}
                    collaboration={collab}
                    executingCollaboration={executingCollaboration}
                    onOpenCollaboration={onOpenCollaboration}
                    feedback={revisionFeedback[collab.id] || ''}
                    setFeedback={(value) =>
                      setRevisionFeedback(prev => ({ ...prev, [collab.id]: value }))
                    }
                    isSubmitting={!!submittingByCollab[collab.id]}
                    onRunCommand={(command) => runCollabCommand(command, collab)}
                  />
                ))
              )}
            </div>

            {tasks.length > 0 ? (
              viewMode === 'by_agent' ? (
                Array.from(groupedByAgent.entries()).map(([agentName, rows]) => (
                  <div key={agentName} style={{ marginBottom: 12 }}>
                    <div style={{ fontSize: 12, color: 'var(--text-secondary, #999)', marginBottom: 6 }}>
                      @{agentName} • {rows.length} task(s)
                    </div>
                    {rows.map(({ task, collaboration }) => (
                      <TaskRow
                        key={task.id}
                        task={task}
                        collaboration={collaboration}
                        onOpenCollaboration={onOpenCollaboration}
                        onFilterAssignee={setAssigneeFilter}
                      />
                    ))}
                  </div>
                ))
              ) : (
                Array.from(groupedByCollab.entries()).map(([collabID, rows]) => {
                  const collab = rows[0].collaboration;
                  return (
                    <div key={collabID} style={{ marginBottom: 12 }}>
                      <div
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                          marginBottom: 6,
                          gap: 8,
                        }}
                      >
                        <div style={{ fontSize: 12, color: 'var(--text-secondary, #999)' }}>
                          {collab.title} • {rows.length} task(s)
                        </div>
                        <button
                          onClick={() => onOpenCollaboration(collab)}
                          style={{
                            border: '1px solid var(--border-color, #444)',
                            borderRadius: 6,
                            backgroundColor: 'transparent',
                            color: 'var(--text-secondary, #999)',
                            fontSize: 11,
                            padding: '3px 8px',
                            cursor: 'pointer',
                          }}
                        >
                          Open
                        </button>
                      </div>
                      {rows.map(({ task, collaboration }) => (
                        <TaskRow
                          key={task.id}
                          task={task}
                          collaboration={collaboration}
                          onOpenCollaboration={onOpenCollaboration}
                          onFilterAssignee={setAssigneeFilter}
                        />
                      ))}
                    </div>
                  );
                })
              )
            ) : (
              <div style={{ marginBottom: 12, color: 'var(--text-secondary, #999)', fontSize: 12 }}>
                No collaboration tasks in current filters.
              </div>
            )}

            <div style={{ marginTop: 16, borderTop: '1px solid var(--border-color, #333)', paddingTop: 10 }}>
              <div style={{ fontSize: 12, color: 'var(--text-secondary, #999)', marginBottom: 6 }}>
                Assistant • {assistantPendingTasks.length} pending task(s) • {activeReminders.length} active reminder(s)
              </div>
              {assistantPendingTasks.map(task => (
                <div
                  key={task.id}
                  style={{
                    padding: '8px 10px',
                    marginBottom: 6,
                    borderRadius: 6,
                    backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
                    fontSize: 12,
                  }}
                >
                  <div style={{ color: 'var(--text-primary, #eee)', fontWeight: 500 }}>{task.title}</div>
                  <div style={{ marginTop: 4, color: 'var(--text-secondary, #999)', fontSize: 11 }}>
                    {task.status} • priority {task.priority}
                  </div>
                  <div style={{ marginTop: 6, display: 'flex', gap: 8 }}>
                    <button
                      onClick={() => onAssistantTaskDone(task.id)}
                      style={{
                        border: '1px solid var(--border-color, #444)',
                        borderRadius: 6,
                        backgroundColor: 'transparent',
                        color: 'var(--text-secondary, #999)',
                        fontSize: 11,
                        padding: '2px 8px',
                        cursor: 'pointer',
                      }}
                    >
                      Mark Done
                    </button>
                  </div>
                </div>
              ))}
              {activeReminders.map(reminder => (
                <div
                  key={reminder.id}
                  style={{
                    padding: '8px 10px',
                    marginBottom: 6,
                    borderRadius: 6,
                    backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
                    fontSize: 12,
                  }}
                >
                  <div style={{ color: 'var(--text-primary, #eee)', fontWeight: 500 }}>{reminder.content}</div>
                  <div style={{ marginTop: 4, color: 'var(--text-secondary, #999)', fontSize: 11 }}>
                    triggers {formatUpdatedAt(reminder.trigger_time)}
                  </div>
                  <div style={{ marginTop: 6, display: 'flex', gap: 8 }}>
                    <button
                      onClick={() => onAssistantReminderDismiss(reminder.id)}
                      style={{
                        border: '1px solid var(--border-color, #444)',
                        borderRadius: 6,
                        backgroundColor: 'transparent',
                        color: 'var(--text-secondary, #999)',
                        fontSize: 11,
                        padding: '2px 8px',
                        cursor: 'pointer',
                      }}
                    >
                      Dismiss
                    </button>
                  </div>
                </div>
              ))}
              {assistantPendingTasks.length === 0 && activeReminders.length === 0 && assistantDoneTasks.length === 0 && (
                <div style={{ color: 'var(--text-secondary, #999)', fontSize: 12 }}>
                  No assistant tasks/reminders in this channel yet.
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function CollaborationRow({
  collaboration,
  executingCollaboration,
  onOpenCollaboration,
  feedback,
  setFeedback,
  isSubmitting,
  onRunCommand,
}: {
  collaboration: Collaboration;
  executingCollaboration: Collaboration | null;
  onOpenCollaboration: (collaboration: Collaboration) => void;
  feedback: string;
  setFeedback: (value: string) => void;
  isSubmitting: boolean;
  onRunCommand: (command: 'approve' | 'revise' | 'cancel') => void;
}) {
  const isTerminal = collaboration.phase === 'completed' || collaboration.phase === 'cancelled';
  const resumeLabel =
    collaboration.phase === 'executing'
      ? 'Resume plan'
      : executingCollaboration &&
          executingCollaboration.phase === 'executing' &&
          executingCollaboration.id !== collaboration.id
        ? 'Resume plan (stop other)'
        : 'Resume plan';
  return (
    <div
      style={{
        padding: '8px 10px',
        marginBottom: 6,
        borderRadius: 6,
        backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
        fontSize: 12,
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8 }}>
        <div style={{ color: 'var(--text-primary, #eee)', fontWeight: 500 }}>
          {collaboration.title}
        </div>
        <div style={{ color: 'var(--text-secondary, #999)', fontSize: 11 }}>
          {collaboration.phase}
        </div>
      </div>
      <div style={{ marginTop: 4, color: 'var(--text-secondary, #999)', fontSize: 11 }}>
        {collaboration.tasks?.length || 0} task(s) • updated {formatUpdatedAt(collaboration.updated_at)}
      </div>
      <div style={{ marginTop: 6, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <button
          onClick={() => onOpenCollaboration(collaboration)}
          style={{
            border: '1px solid var(--border-color, #444)',
            borderRadius: 6,
            backgroundColor: 'transparent',
            color: 'var(--text-secondary, #999)',
            fontSize: 11,
            padding: '2px 8px',
            cursor: 'pointer',
          }}
        >
          Open
        </button>
        {(collaboration.phase === 'approved' ||
          collaboration.phase === 'reviewing' ||
          collaboration.phase === 'executing') && (
          <button
            type="button"
            title={
              collaboration.phase === 'executing'
                ? 'Re-send task prompts for open work'
                : undefined
            }
            disabled={isSubmitting}
            onClick={() => onRunCommand('approve')}
            style={{
              border:
                collaboration.phase === 'executing' ? '1px solid #8b5cf6' : '1px solid #10b981',
              borderRadius: 6,
              backgroundColor: 'transparent',
              color: collaboration.phase === 'executing' ? '#c4b5fd' : '#10b981',
              fontSize: 11,
              padding: '2px 8px',
              cursor: 'pointer',
              opacity: isSubmitting ? 0.6 : 1,
            }}
          >
            {resumeLabel}
          </button>
        )}
        {collaboration.phase === 'reviewing' && (
          <>
            <div style={{ width: '100%', flexBasis: '100%' }}>
              <textarea
                value={feedback}
                onChange={e => setFeedback(e.target.value)}
                placeholder="Revision feedback… (⌘↵ / Ctrl+↵ to send)"
                rows={3}
                style={{
                  width: '100%',
                  boxSizing: 'border-box',
                  padding: '6px 8px',
                  borderRadius: 6,
                  border: '1px solid var(--border-color, #444)',
                  backgroundColor: 'var(--bg-secondary, #1e1e1e)',
                  color: 'var(--text-primary, #eee)',
                  fontSize: 11,
                  lineHeight: 1.4,
                  resize: 'vertical',
                  minHeight: 64,
                  fontFamily: 'inherit',
                }}
                onKeyDown={e => {
                  if (e.key !== 'Enter') return;
                  if (!(e.metaKey || e.ctrlKey)) return;
                  e.preventDefault();
                  if (!isSubmitting && feedback.trim()) onRunCommand('revise');
                }}
              />
            </div>
            <button
              disabled={isSubmitting || !feedback.trim()}
              onClick={() => onRunCommand('revise')}
              style={{
                border: '1px solid #3b82f6',
                borderRadius: 6,
                backgroundColor: 'transparent',
                color: '#3b82f6',
                fontSize: 11,
                padding: '2px 8px',
                cursor: 'pointer',
                opacity: isSubmitting || !feedback.trim() ? 0.6 : 1,
              }}
            >
              Revise
            </button>
          </>
        )}
        {!isTerminal && (
          <button
            disabled={isSubmitting}
            onClick={() => onRunCommand('cancel')}
            style={{
              border: '1px solid #ef4444',
              borderRadius: 6,
              backgroundColor: 'transparent',
              color: '#ef4444',
              fontSize: 11,
              padding: '2px 8px',
              cursor: 'pointer',
              opacity: isSubmitting ? 0.6 : 1,
            }}
          >
            Cancel
          </button>
        )}
      </div>
    </div>
  );
}

function TaskRow({
  task,
  collaboration,
  onOpenCollaboration,
  onFilterAssignee,
}: {
  task: CollaborationTask;
  collaboration: Collaboration;
  onOpenCollaboration: (collaboration: Collaboration) => void;
  onFilterAssignee: (assignee: string) => void;
}) {
  return (
    <div
      style={{
        padding: '8px 10px',
        marginBottom: 6,
        borderRadius: 6,
        backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
        fontSize: 12,
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8 }}>
        <div style={{ color: 'var(--text-primary, #eee)', fontWeight: 500, display: 'flex', gap: 6, alignItems: 'center' }}>
          <span>{statusIcon(task.status)}</span>
          <span title={task.title}>{task.title}</span>
        </div>
        <span style={{ color: 'var(--text-secondary, #999)', fontSize: 11 }}>{task.status}</span>
      </div>
      <div style={{ marginTop: 4, color: 'var(--text-secondary, #999)' }} title={task.description}>
        {task.description}
      </div>
      <div style={{ marginTop: 6, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8 }}>
        <button
          onClick={() => onFilterAssignee(task.assigned_name)}
          style={{
            border: 'none',
            background: 'none',
            color: '#93c5fd',
            cursor: 'pointer',
            padding: 0,
            fontSize: 11,
          }}
          title="Filter by assignee"
        >
          @{task.assigned_name || 'unassigned'}
        </button>
        <div style={{ fontSize: 11, color: 'var(--text-secondary, #999)' }}>
          {collaboration.title} • {formatUpdatedAt(task.updated_at)}
        </div>
      </div>
      <div style={{ marginTop: 6, display: 'flex', gap: 8 }}>
        <button
          onClick={() => onOpenCollaboration(collaboration)}
          style={{
            border: '1px solid var(--border-color, #444)',
            borderRadius: 6,
            backgroundColor: 'transparent',
            color: 'var(--text-secondary, #999)',
            fontSize: 11,
            padding: '2px 8px',
            cursor: 'pointer',
          }}
        >
          Open Collaboration
        </button>
        <button
          onClick={() => navigator.clipboard?.writeText(task.id)}
          style={{
            border: '1px solid var(--border-color, #444)',
            borderRadius: 6,
            backgroundColor: 'transparent',
            color: 'var(--text-secondary, #999)',
            fontSize: 11,
            padding: '2px 8px',
            cursor: 'pointer',
          }}
        >
          Copy Task ID
        </button>
      </div>
    </div>
  );
}
