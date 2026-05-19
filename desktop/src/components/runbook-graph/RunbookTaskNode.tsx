import { Handle, Position, type Node, type NodeProps } from '@xyflow/react';
import { taskOrchestrationLabel } from '../../utils/collaborationTaskOrchestration';
import type { RunbookTaskNodeData } from '../../utils/runbookDAG';

const statusColors: Record<string, string> = {
  pending: '#64748b',
  in_progress: '#8b5cf6',
  completed: '#10b981',
  blocked: '#ef4444',
};

type TaskNode = Node<RunbookTaskNodeData, 'runbookTask'>;

export function RunbookTaskNode({ data, selected }: NodeProps<TaskNode>) {
  const { task, index, phase, editable, allTasks } = data;
  const orch = allTasks ? taskOrchestrationLabel(task, allTasks, phase) : null;
  const borderColor = statusColors[task.status] ?? statusColors.pending;
  const title = task.title?.trim() || `Task ${index + 1}`;
  const assignee = task.assigned_name ? `@${task.assigned_name}` : 'Unassigned';

  return (
    <div
      className="runbook-task-node"
      style={{
        minWidth: 200,
        maxWidth: 240,
        padding: '10px 12px',
        borderRadius: 8,
        border: `2px solid ${selected ? '#3b82f6' : borderColor}`,
        backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
        color: 'var(--text-primary, #eee)',
        fontSize: 12,
        boxShadow: selected ? '0 0 0 2px rgba(59, 130, 246, 0.35)' : undefined,
      }}
    >
      {editable ? (
        <>
          <Handle type="target" position={Position.Left} className="!w-2 !h-2 !bg-slate-400" />
          <Handle type="source" position={Position.Right} className="!w-2 !h-2 !bg-slate-400" />
        </>
      ) : null}
      <div style={{ fontSize: 10, color: '#94a3b8', marginBottom: 4 }}>Task {index + 1}</div>
      <div style={{ fontWeight: 600, marginBottom: 4, lineHeight: 1.3 }}>{title}</div>
      <div style={{ fontSize: 11, color: '#a3a3a3' }}>{assignee}</div>
      <div style={{ marginTop: 6, display: 'flex', gap: 6, flexWrap: 'wrap', alignItems: 'center' }}>
        <span
          style={{
            fontSize: 10,
            textTransform: 'uppercase',
            padding: '2px 6px',
            borderRadius: 4,
            backgroundColor: `${borderColor}33`,
            color: borderColor,
          }}
        >
          {task.status}
        </span>
        {orch ? <span style={{ fontSize: 10, color: '#94a3b8' }}>{orch}</span> : null}
      </div>
    </div>
  );
}
