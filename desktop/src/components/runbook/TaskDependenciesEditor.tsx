import { useState, type CSSProperties } from 'react';
import type { CollaborationTask, DependencyEdge, DependencyGroup, EdgeCondition } from '../../types/protocol';

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
};

function edgeForTask(task: CollaborationTask, depId: string): DependencyEdge | undefined {
  return (task.dependency_edges ?? []).find((e) => e.from_task_id === depId);
}

function defaultCondition(): EdgeCondition {
  return { mode: 'always' };
}

export function TaskDependenciesEditor({
  taskIndex,
  tasks,
  disabled,
  onChange,
}: {
  taskIndex: number;
  tasks: CollaborationTask[];
  disabled: boolean;
  onChange: (patch: Pick<CollaborationTask, 'dependencies' | 'dependency_edges' | 'dependency_groups'>) => void;
}) {
  const task = tasks[taskIndex];
  const deps = task.dependencies ?? [];
  const [pick, setPick] = useState('');

  const candidates = tasks.map((other, j) => ({ other, j })).filter(({ j }) => j !== taskIndex);
  const available = candidates.filter(({ other }) => !deps.includes(other.id));

  const emit = (
    nextDeps: string[],
    nextEdges: DependencyEdge[] | undefined,
    nextGroups: DependencyGroup[] | undefined
  ) => {
    onChange({
      dependencies: nextDeps,
      dependency_edges: nextEdges,
      dependency_groups: nextGroups,
    });
  };

  const addPicked = () => {
    if (!pick || deps.includes(pick)) return;
    const nextDeps = [...deps, pick];
    const edges = [...(task.dependency_edges ?? [])];
    if (!edges.some((e) => e.from_task_id === pick)) {
      edges.push({ from_task_id: pick, condition: defaultCondition() });
    }
    emit(nextDeps, edges, task.dependency_groups);
    setPick('');
  };

  const removeDep = (depId: string) => {
    emit(
      deps.filter((d) => d !== depId),
      (task.dependency_edges ?? []).filter((e) => e.from_task_id !== depId),
      task.dependency_groups
    );
  };

  const updateEdgeCondition = (depId: string, condition: EdgeCondition) => {
    const edges = [...(task.dependency_edges ?? [])];
    const idx = edges.findIndex((e) => e.from_task_id === depId);
    if (idx >= 0) {
      edges[idx] = { ...edges[idx], condition };
    } else {
      edges.push({ from_task_id: depId, condition });
    }
    emit(deps, edges, task.dependency_groups);
  };

  const joinMode = task.dependency_groups?.[0]?.mode ?? 'all';
  const setJoinMode = (mode: 'all' | 'any') => {
    if (deps.length < 2) {
      emit(deps, task.dependency_edges, undefined);
      return;
    }
    emit(deps, task.dependency_edges, [{ mode, task_ids: [...deps] }]);
  };

  return (
    <div style={{ marginTop: 8 }}>
      <div style={{ ...labelStyle, marginBottom: 6 }}>Depends on</div>
      {deps.length > 0 ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 8 }}>
          {deps.map((depId) => {
            const j = tasks.findIndex((t) => t.id === depId);
            const title = j >= 0 ? tasks[j].title?.trim() || '(untitled)' : depId.slice(0, 8);
            const edge = edgeForTask(task, depId);
            const cond = edge?.condition ?? defaultCondition();
            return (
              <div
                key={depId}
                style={{
                  padding: 8,
                  borderRadius: 8,
                  border: '1px solid var(--border-color, #333)',
                  background: 'var(--bg-tertiary, #252525)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: disabled ? 0 : 6 }}>
                  <span style={{ fontSize: 11, flex: 1 }}>
                    Task {j >= 0 ? j + 1 : '?'}: {title}
                  </span>
                  {!disabled ? (
                    <button
                      type="button"
                      onClick={() => removeDep(depId)}
                      style={{ border: 'none', background: 'none', color: '#f87171', cursor: 'pointer' }}
                    >
                      ×
                    </button>
                  ) : null}
                </div>
                {!disabled ? (
                  <EdgeConditionFields
                    condition={cond}
                    onChange={(c) => updateEdgeCondition(depId, c)}
                  />
                ) : cond.mode !== 'always' ? (
                  <span style={{ fontSize: 10, color: '#94a3b8' }}>{formatConditionLabel(cond)}</span>
                ) : null}
              </div>
            );
          })}
        </div>
      ) : (
        <p style={{ fontSize: 11, color: 'var(--text-secondary, #999)', marginBottom: 8 }}>
          No dependencies — runs in the first wave.
        </p>
      )}
      {deps.length > 1 && !disabled ? (
        <label style={{ ...labelStyle, marginBottom: 8 }}>
          Join mode (fan-in)
          <select
            value={joinMode}
            onChange={(e) => setJoinMode(e.target.value as 'all' | 'any')}
            style={inputStyle}
          >
            <option value="all">Wait for all upstream tasks</option>
            <option value="any">Wait for any upstream task</option>
          </select>
        </label>
      ) : null}
      {!disabled && available.length > 0 ? (
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <select value={pick} onChange={(e) => setPick(e.target.value)} style={{ ...inputStyle, flex: 1, marginTop: 0 }}>
            <option value="">Select upstream task…</option>
            {available.map(({ other, j }) => (
              <option key={other.id} value={other.id}>
                Task {j + 1}: {other.title?.trim() || '(untitled)'}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={addPicked}
            disabled={!pick}
            style={{
              padding: '6px 10px',
              fontSize: 12,
              borderRadius: 6,
              border: '1px solid #444',
              background: 'transparent',
              color: '#ccc',
              cursor: 'pointer',
            }}
          >
            Add
          </button>
        </div>
      ) : null}
    </div>
  );
}

function EdgeConditionFields({
  condition,
  onChange,
}: {
  condition: EdgeCondition;
  onChange: (c: EdgeCondition) => void;
}) {
  const mode = condition.mode || 'always';
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <select
        value={mode}
        onChange={(e) => {
          const next = e.target.value as EdgeCondition['mode'];
          if (next === 'always') onChange({ mode: 'always' });
          else if (next === 'on_status') onChange({ mode: 'on_status', status: 'completed' });
          else onChange({ mode: 'on_output', contains: '' });
        }}
        style={{ ...inputStyle, marginTop: 0 }}
      >
        <option value="always">Always (when upstream completes)</option>
        <option value="on_status">On upstream status</option>
        <option value="on_output">On upstream output</option>
      </select>
      {mode === 'on_status' ? (
        <select
          value={condition.status ?? 'completed'}
          onChange={(e) => onChange({ ...condition, mode: 'on_status', status: e.target.value })}
          style={inputStyle}
        >
          <option value="completed">completed</option>
          <option value="blocked">blocked</option>
          <option value="in_progress">in_progress</option>
        </select>
      ) : null}
      {mode === 'on_output' ? (
        <>
          <input
            type="text"
            placeholder='Output contains…'
            value={condition.contains ?? ''}
            onChange={(e) => onChange({ ...condition, mode: 'on_output', contains: e.target.value })}
            style={inputStyle}
          />
          <input
            type="text"
            placeholder="Or regex (optional)"
            value={condition.regex ?? ''}
            onChange={(e) => onChange({ ...condition, mode: 'on_output', regex: e.target.value })}
            style={inputStyle}
          />
        </>
      ) : null}
    </div>
  );
}

function formatConditionLabel(cond: EdgeCondition): string {
  if (cond.mode === 'on_output' && cond.contains) return `if output contains "${cond.contains}"`;
  if (cond.mode === 'on_status' && cond.status) return `if status ${cond.status}`;
  return cond.mode;
}
