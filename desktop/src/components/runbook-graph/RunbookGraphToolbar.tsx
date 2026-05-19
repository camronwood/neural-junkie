import type { CSSProperties } from 'react';

interface RunbookGraphToolbarProps {
  editable: boolean;
  taskCount: number;
  maxTasks: number;
  validationError: string | null;
  busy: boolean;
  onAutoLayout: () => void;
  onAddTask: () => void;
  onSave: () => void;
  onClose: () => void;
}

export function RunbookGraphToolbar({
  editable,
  taskCount,
  maxTasks,
  validationError,
  busy,
  onAutoLayout,
  onAddTask,
  onSave,
  onClose,
}: RunbookGraphToolbarProps) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '10px 16px',
        borderBottom: '1px solid var(--border-color, #333)',
        backgroundColor: 'var(--bg-secondary, #1e1e1e)',
        flexWrap: 'wrap',
      }}
    >
      <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary, #eee)', marginRight: 'auto' }}>
        Task graph
      </span>
      {validationError ? (
        <span style={{ fontSize: 12, color: '#f87171', flex: '1 1 100%' }}>{validationError}</span>
      ) : null}
      <button
        type="button"
        onClick={onAutoLayout}
        disabled={busy || taskCount === 0}
        style={btnSecondary}
      >
        Auto-layout
      </button>
      {editable ? (
        <button
          type="button"
          onClick={onAddTask}
          disabled={busy || taskCount >= maxTasks}
          style={btnSecondary}
          title={taskCount >= maxTasks ? `Maximum ${maxTasks} tasks` : undefined}
        >
          + Add task
        </button>
      ) : null}
      {editable ? (
        <button type="button" onClick={onSave} disabled={busy || !!validationError} style={btnPrimary}>
          Save &amp; close
        </button>
      ) : null}
      <button type="button" onClick={onClose} disabled={busy} style={btnSecondary}>
        {editable ? 'Cancel' : 'Close'}
      </button>
    </div>
  );
}

const btnSecondary: CSSProperties = {
  padding: '6px 12px',
  fontSize: 12,
  borderRadius: 6,
  border: '1px solid var(--border-color, #444)',
  background: 'var(--bg-tertiary, #2a2a2a)',
  color: 'var(--text-primary, #eee)',
  cursor: 'pointer',
};

const btnPrimary: React.CSSProperties = {
  ...btnSecondary,
  border: '1px solid #3b82f6',
  background: '#2563eb',
  color: '#fff',
};
