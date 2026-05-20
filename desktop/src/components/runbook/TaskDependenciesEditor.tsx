import { useState, type CSSProperties } from 'react';
import type { CollaborationTask } from '../../types/protocol';

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

export function TaskDependenciesEditor({
  taskIndex,
  tasks,
  disabled,
  onChange,
}: {
  taskIndex: number;
  tasks: CollaborationTask[];
  disabled: boolean;
  onChange: (deps: string[]) => void;
}) {
  const deps = tasks[taskIndex].dependencies ?? [];
  const [pick, setPick] = useState('');

  const candidates = tasks.map((other, j) => ({ other, j })).filter(({ j }) => j !== taskIndex);
  const available = candidates.filter(({ other }) => !deps.includes(other.id));

  const addPicked = () => {
    if (!pick || deps.includes(pick)) return;
    onChange([...deps, pick]);
    setPick('');
  };

  return (
    <div style={{ marginTop: 8 }}>
      <div style={{ ...labelStyle, marginBottom: 6 }}>Depends on</div>
      {deps.length > 0 ? (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 8 }}>
          {deps.map((depId) => {
            const j = tasks.findIndex((t) => t.id === depId);
            const title = j >= 0 ? tasks[j].title?.trim() || '(untitled)' : depId.slice(0, 8);
            return (
              <span
                key={depId}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  padding: '2px 8px',
                  borderRadius: 12,
                  background: 'var(--bg-tertiary, #333)',
                  fontSize: 11,
                }}
              >
                Task {j >= 0 ? j + 1 : '?'}: {title}
                {!disabled ? (
                  <button type="button" onClick={() => onChange(deps.filter((d) => d !== depId))} style={{ border: 'none', background: 'none', color: '#f87171', cursor: 'pointer' }}>
                    ×
                  </button>
                ) : null}
              </span>
            );
          })}
        </div>
      ) : (
        <p style={{ fontSize: 11, color: 'var(--text-secondary, #999)', marginBottom: 8 }}>No dependencies — runs in the first wave.</p>
      )}
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
          <button type="button" onClick={addPicked} disabled={!pick} style={{ padding: '6px 10px', fontSize: 12, borderRadius: 6, border: '1px solid #444', background: 'transparent', color: '#ccc', cursor: 'pointer' }}>
            Add
          </button>
        </div>
      ) : null}
    </div>
  );
}
