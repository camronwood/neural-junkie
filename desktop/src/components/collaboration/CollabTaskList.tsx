import type { Collaboration, CollaborationTask } from '../../types/protocol';
import {
  FILE_PROPOSAL_QUEUE_LABEL,
  taskNeedsFileDeliverable,
} from '../../utils/collaborationPanelState';
import { taskOrchestrationLabel } from '../../utils/collaborationTaskOrchestration';
import { taskActionBtnStyle } from './collabStyles';

function taskIcon(status: string): string {
  switch (status) {
    case 'in_progress': return '🔄';
    case 'completed': return '✅';
    case 'blocked': return '🚫';
    default: return '⬜';
  }
}

export interface CollabTaskListProps {
  collaboration: Collaboration;
  isTerminal: boolean;
  isSubmitting: boolean;
  reassignTaskId: string | null;
  reassignAgentId: string;
  onReassignTaskIdChange: (updater: (prev: string | null) => string | null) => void;
  onReassignAgentIdChange: (agentId: string) => void;
  onViewGraph: () => void;
  onTaskApprove: (task: CollaborationTask) => void | Promise<void>;
  onTaskDone: (task: CollaborationTask) => void | Promise<void>;
  onTaskSkip: (task: CollaborationTask) => void | Promise<void>;
  onTaskRedispatch: (task: CollaborationTask) => void | Promise<void>;
  onTaskReassign: (task: CollaborationTask) => void | Promise<void>;
}

/** Execution progress bar + task list (with reassign UI) + empty-tasks message. */
export function CollabTaskList({
  collaboration: c,
  isTerminal,
  isSubmitting,
  reassignTaskId,
  reassignAgentId,
  onReassignTaskIdChange,
  onReassignAgentIdChange,
  onViewGraph,
  onTaskApprove,
  onTaskDone,
  onTaskSkip,
  onTaskRedispatch,
  onTaskReassign,
}: CollabTaskListProps) {
  const completedTasks = c.tasks?.filter((t) => t.status === 'completed').length ?? 0;
  const totalTasks = c.tasks?.length ?? 0;
  const progress = totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0;

  return (
    <>
      {/* Progress bar (during execution) */}
      {c.phase === 'executing' && totalTasks > 0 && (
        <div style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
            <span style={{ fontSize: 12, color: 'var(--text-secondary, #888)' }}>Progress</span>
            <span style={{ fontSize: 12, color: 'var(--text-secondary, #888)' }}>{completedTasks}/{totalTasks} tasks ({progress}%)</span>
          </div>
          <div style={{
            height: 6, borderRadius: 3,
            backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
            overflow: 'hidden',
          }}>
            <div style={{
              height: '100%', borderRadius: 3,
              width: `${progress}%`,
              backgroundColor: '#8b5cf6',
              transition: 'width 0.3s ease',
            }} />
          </div>
        </div>
      )}

      {/* Tasks */}
      {c.tasks && c.tasks.length > 0 ? (
        <div style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
            <h4 style={{ margin: 0, fontSize: 12, textTransform: 'uppercase', color: 'var(--text-secondary, #888)', letterSpacing: 0.5 }}>
              Tasks
            </h4>
            <button
              type="button"
              onClick={() => onViewGraph()}
              style={{
                padding: '4px 8px',
                fontSize: 11,
                borderRadius: 4,
                border: '1px solid var(--border-color, #444)',
                background: 'transparent',
                color: 'var(--text-primary, #ccc)',
                cursor: 'pointer',
              }}
            >
              View graph
            </button>
          </div>
          {c.approve_warnings && c.approve_warnings.length > 0 ? (
            <div
              style={{
                marginBottom: 10,
                padding: '8px 10px',
                borderRadius: 6,
                border: '1px solid #b45309',
                backgroundColor: 'rgba(180, 83, 9, 0.12)',
                fontSize: 12,
                color: '#fcd34d',
              }}
            >
              <div style={{ fontWeight: 600, marginBottom: 4 }}>Plan validation</div>
              <ul style={{ margin: 0, paddingLeft: 18 }}>
                {c.approve_warnings.map((w) => (
                  <li key={w}>{w}</li>
                ))}
              </ul>
            </div>
          ) : null}
          {c.tasks.map((task: CollaborationTask, i: number) => {
            const reassignCandidates = (c.agents || []).filter(
              (a) => a.agent_id && a.agent_id !== task.assigned_to
            );
            const showReassign =
              !isTerminal &&
              c.phase === 'executing' &&
              task.status !== 'completed' &&
              reassignCandidates.length > 0;
            return (
            <div key={task.id} style={{
              padding: '8px 10px', marginBottom: 6,
              borderRadius: 6,
              backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
              fontSize: 13,
            }}>
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
                <span>{taskIcon(task.status)}</span>
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 500, color: 'var(--text-primary, #eee)' }}>
                    Task {i + 1}: {task.title}
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary, #999)', marginTop: 2 }}>
                    Assigned to @{task.assigned_name || 'unassigned'}
                  </div>
                  {task.options?.expected_model || task.options?.expected_provider_id ? (
                    <div style={{ fontSize: 11, color: '#a5b4fc', marginTop: 4 }}>
                      Model: {task.options.expected_model || 'agent default'}
                      {task.options.expected_provider_id
                        ? ` · provider ${task.options.expected_provider_id}`
                        : ''}
                      {task.options.routing_reason ? ` · ${task.options.routing_reason}` : ''}
                    </div>
                  ) : null}
                  {c.tasks && taskOrchestrationLabel(task, c.tasks, c.phase) ? (
                    <div style={{ fontSize: 11, color: '#94a3b8', marginTop: 4 }}>
                      {taskOrchestrationLabel(task, c.tasks, c.phase)}
                    </div>
                  ) : null}
                  {c.phase === 'executing' &&
                  task.status === 'in_progress' &&
                  taskNeedsFileDeliverable(task) ? (
                    <div style={{ fontSize: 11, color: '#fbbf24', marginTop: 4, lineHeight: 1.4 }}>
                      File deliverable: assignee must emit a <strong>[FILE_CHANGE]</strong> proposal — approve in{' '}
                      <strong>{FILE_PROPOSAL_QUEUE_LABEL}</strong> (toolbar). Chat-only replies do not write to disk.
                    </div>
                  ) : null}
                  {c.phase === 'executing' && task.awaiting_approval ? (
                    <div
                      data-testid={`collaboration-task-awaiting-approval-${task.id}`}
                      style={{ fontSize: 11, color: '#fbbf24', marginTop: 4, lineHeight: 1.4 }}
                    >
                      Requires your approval before this action runs (webhook / SMS / email / wait_human).
                    </div>
                  ) : null}
                  {!isTerminal && task.status !== 'completed' && c.phase === 'executing' && (
                    <div style={{ marginTop: 6, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                      {task.awaiting_approval ? (
                        <button
                          type="button"
                          data-testid={`collaboration-task-approve-${task.id}`}
                          onClick={() => void onTaskApprove(task)}
                          disabled={isSubmitting}
                          style={taskActionBtnStyle('#f59e0b')}
                        >
                          Approve
                        </button>
                      ) : null}
                      <button
                        type="button"
                        onClick={() => void onTaskDone(task)}
                        disabled={isSubmitting}
                        style={taskActionBtnStyle('#10b981')}
                      >
                        Complete
                      </button>
                      <button
                        type="button"
                        onClick={() => void onTaskSkip(task)}
                        disabled={isSubmitting}
                        style={taskActionBtnStyle('#94a3b8')}
                      >
                        Skip
                      </button>
                      <button
                        type="button"
                        onClick={() => void onTaskRedispatch(task)}
                        disabled={isSubmitting}
                        style={taskActionBtnStyle('#3b82f6')}
                      >
                        Redispatch
                      </button>
                      {showReassign && (
                        <button
                          type="button"
                          data-testid={`collaboration-task-reassign-${task.id}`}
                          onClick={() => {
                            onReassignTaskIdChange((prev) => (prev === task.id ? null : task.id));
                            onReassignAgentIdChange(reassignCandidates[0]?.agent_id || '');
                          }}
                          disabled={isSubmitting}
                          style={taskActionBtnStyle('#f59e0b')}
                        >
                          Reassign
                        </button>
                      )}
                    </div>
                  )}
                  {reassignTaskId === task.id && showReassign && (
                    <div
                      data-testid={`collaboration-task-reassign-form-${task.id}`}
                      style={{ marginTop: 8, display: 'flex', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}
                    >
                      <select
                        aria-label={`Reassign ${task.title}`}
                        value={reassignAgentId}
                        onChange={(e) => onReassignAgentIdChange(e.target.value)}
                        style={{
                          flex: '1 1 140px',
                          padding: '4px 8px',
                          borderRadius: 4,
                          border: '1px solid var(--border-color, #444)',
                          backgroundColor: 'var(--bg-secondary, #1e1e1e)',
                          color: 'var(--text-primary, #eee)',
                          fontSize: 12,
                        }}
                      >
                        {reassignCandidates.map((a) => (
                          <option key={a.agent_id} value={a.agent_id}>
                            @{a.agent_name}
                          </option>
                        ))}
                      </select>
                      <button
                        type="button"
                        onClick={() => void onTaskReassign(task)}
                        disabled={isSubmitting || !reassignAgentId}
                        style={taskActionBtnStyle('#10b981')}
                      >
                        Confirm
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>
            );
          })}
        </div>
      ) : (
        (c.phase === 'reviewing' || c.phase === 'approved' || c.phase === 'executing') && (
          <p
            data-testid="collaboration-tasks-empty"
            style={{ margin: '0 0 16px', fontSize: 12, color: 'var(--text-secondary, #999)', lineHeight: 1.45 }}
          >
            No tasks yet. Approve a plan with task assignments, or revise so agents add concrete work items.
          </p>
        )
      )}
    </>
  );
}
