import type { CSSProperties } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { CollaborationAgent, CollaborationTask, TaskKind } from '../../types/protocol';
import { defaultActionSpec } from '../../utils/runbookActionUtils';
import { RunbookActionConfigEditor } from '../runbook/RunbookActionConfigEditor';
import { TaskDependenciesEditor } from '../runbook/TaskDependenciesEditor';

const inputStyle: CSSProperties = {
  width: '100%',
  boxSizing: 'border-box',
  marginTop: 4,
  padding: '6px 8px',
  borderRadius: 6,
  border: '1px solid var(--border-color, #444)',
  backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
  color: 'var(--text-primary, #eee)',
  fontSize: 12,
  fontFamily: 'inherit',
};

const labelStyle: CSSProperties = {
  display: 'block',
  fontSize: 11,
  color: '#a3a3a3',
  marginBottom: 10,
};

interface RunbookTaskInspectorProps {
  task: CollaborationTask;
  taskIndex: number;
  tasks: CollaborationTask[];
  agents: CollaborationAgent[];
  api: ChatAPI;
  editable: boolean;
  onUpdate: (patch: Partial<CollaborationTask>) => void;
  onUpdateDependencies: (deps: string[]) => void;
  onDelete: () => void;
}

export function RunbookTaskInspector({
  task,
  taskIndex,
  tasks,
  agents,
  api,
  editable,
  onUpdate,
  onUpdateDependencies,
  onDelete,
}: RunbookTaskInspectorProps) {
  const kind: TaskKind = task.kind ?? 'agent';

  return (
    <div
      style={{
        width: 280,
        flexShrink: 0,
        borderLeft: '1px solid var(--border-color, #333)',
        padding: 16,
        overflow: 'auto',
        backgroundColor: 'var(--bg-secondary, #1e1e1e)',
      }}
    >
      <h4 style={{ margin: '0 0 12px', fontSize: 13, color: 'var(--text-primary, #eee)' }}>
        Task {taskIndex + 1}
      </h4>
      <label style={labelStyle}>
        Task type
        <select
          value={kind}
          onChange={(e) => {
            const nextKind = e.target.value as TaskKind;
            onUpdate({
              kind: nextKind,
              action:
                nextKind === 'action'
                  ? task.action ?? defaultActionSpec('http_get')
                  : undefined,
            });
          }}
          disabled={!editable}
          style={inputStyle}
        >
          <option value="agent">Agent task</option>
          <option value="action">Action task</option>
        </select>
      </label>
      <label style={labelStyle}>
        Title
        <input
          type="text"
          value={task.title}
          onChange={(e) => onUpdate({ title: e.target.value })}
          disabled={!editable}
          style={inputStyle}
        />
      </label>
      <label style={labelStyle}>
        Description
        <textarea
          value={task.description}
          onChange={(e) => onUpdate({ description: e.target.value })}
          disabled={!editable}
          rows={3}
          style={{ ...inputStyle, resize: 'vertical' }}
        />
      </label>
      {kind === 'action' && task.action ? (
        <RunbookActionConfigEditor
          action={task.action}
          api={api}
          disabled={!editable}
          inputStyle={inputStyle}
          labelStyle={{ ...labelStyle, marginBottom: 8 }}
          onChange={(action) => onUpdate({ action })}
        />
      ) : null}
      {kind !== 'action' ? (
        <label style={labelStyle}>
          Assignee
          <select
            value={task.assigned_to}
            onChange={(e) => {
              const a = agents.find((x) => x.agent_id === e.target.value);
              onUpdate({
                assigned_to: e.target.value,
                assigned_name: a?.agent_name ?? '',
              });
            }}
            disabled={!editable}
            style={inputStyle}
          >
            <option value="">Unassigned</option>
            {agents.map((a) => (
              <option key={a.agent_id} value={a.agent_id}>
                @{a.agent_name}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      {tasks.length > 1 ? (
        <TaskDependenciesEditor
          taskIndex={taskIndex}
          tasks={tasks}
          disabled={!editable}
          onChange={onUpdateDependencies}
        />
      ) : null}
      {editable ? (
        <button
          type="button"
          onClick={onDelete}
          style={{
            marginTop: 8,
            padding: '6px 10px',
            fontSize: 12,
            borderRadius: 6,
            border: '1px solid #7f1d1d',
            background: 'transparent',
            color: '#f87171',
            cursor: 'pointer',
          }}
        >
          Remove task
        </button>
      ) : null}
    </div>
  );
}
