import { useCallback, useEffect, useState, type CSSProperties, type ReactNode } from 'react';
import { shallow } from 'zustand/shallow';
import { ChatAPI } from '../api/chatAPI';
import { useChatStore } from '../stores/chatStore';
import type { Collaboration, CollaborationAgent, CollaborationTask } from '../types/protocol';
import { ensureCollaborationExecutionWorkspace } from '../utils/collaborationExecutionWorkspace';
import { RunbookImportModal } from './RunbookImportModal';
import { RunbookGraphModal } from './runbook-graph';
import { createEmptyTask } from '../utils/runbookTaskUtils';

interface RunbookBuilderPanelProps {
  collaboration: Collaboration;
  hubAgents: CollaborationAgent[];
  onClose: () => void;
  onSaved: (collab: Collaboration) => void;
  onStarted?: (collab: Collaboration) => void;
}

export function RunbookBuilderPanel({
  collaboration,
  hubAgents,
  onClose,
  onSaved,
  onStarted,
}: RunbookBuilderPanelProps) {
  const { serverAddr } = useChatStore((s) => ({ serverAddr: s.serverAddr }), shallow);
  const [api] = useState(() => new ChatAPI(serverAddr));
  const [description, setDescription] = useState(collaboration.description);
  const [agentPool, setAgentPool] = useState<string[]>(collaboration.agents.map((a) => a.agent_id));
  const [tasks, setTasks] = useState<CollaborationTask[]>(
    collaboration.tasks?.length ? collaboration.tasks : [createEmptyTask()]
  );
  const [importOpen, setImportOpen] = useState(false);
  const [graphOpen, setGraphOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const editable = collaboration.phase === 'draft' || collaboration.phase === 'reviewing';
  const poolAgents = hubAgents.filter((a) => agentPool.includes(a.agent_id));
  const allAgents = poolAgents.length > 0 ? poolAgents : collaboration.agents;

  const persist = useCallback(async () => {
    setBusy(true);
    setError('');
    try {
      const normalized = tasks.map((t, i) => ({
        ...t,
        title: t.title.trim() || `Task ${i + 1}`,
        description: t.description.trim() || t.title.trim() || `Task ${i + 1}`,
      }));
      const snap = await api.updateRunbook(collaboration.id, {
        description: description.trim(),
        agent_ids: agentPool,
        tasks: normalized,
      });
      onSaved(snap);
      return snap;
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return null;
    } finally {
      setBusy(false);
    }
  }, [api, collaboration.id, description, agentPool, tasks, onSaved]);

  const handleAutoAssign = async (index: number) => {
    const t = tasks[index];
    setBusy(true);
    setError('');
    try {
      const s = await api.suggestRunbookAssignee(collaboration.id, t.title, t.description);
      if (!s) {
        setError('No confident match — pick an agent manually.');
        return;
      }
      setTasks((prev) => {
        const next = [...prev];
        next[index] = { ...next[index], assigned_to: s.agent_id, assigned_name: s.agent_name };
        return next;
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const handleImport = async (markdown: string) => {
    setBusy(true);
    setError('');
    try {
      const parsed = await api.parseRunbookPlan(collaboration.id, markdown);
      setTasks(parsed);
      setImportOpen(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      throw e;
    } finally {
      setBusy(false);
    }
  };

  const handleSubmit = async () => {
    if (!(await persist())) return;
    setBusy(true);
    try {
      onSaved(await api.submitRunbook(collaboration.id));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const handleStart = async () => {
    let snap = collaboration;
    if (editable) {
      const saved = await persist();
      if (!saved) return;
      snap = saved;
    }
    if (snap.phase === 'draft') {
      try {
        snap = await api.submitRunbook(collaboration.id);
        onSaved(snap);
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
        return;
      }
    }
    setBusy(true);
    setError('');
    try {
      const started = await api.startRunbook(collaboration.id);
      onSaved(started);
      if (!started.workspace_acknowledged) {
        await ensureCollaborationExecutionWorkspace(started);
      }
      onStarted?.(started);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    setDescription(collaboration.description);
    setAgentPool(collaboration.agents.map((a) => a.agent_id));
    if (collaboration.tasks?.length) {
      setTasks(collaboration.tasks);
    }
  }, [collaboration.id, collaboration.description, collaboration.agents, collaboration.tasks]);

  return (
    <div style={panelStyle}>
      <div style={panelHeaderStyle}>
        <h3 style={panelTitleStyle}>Runbook builder</h3>
        <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
          <button
            type="button"
            onClick={() => setGraphOpen(true)}
            disabled={busy}
            style={secondaryBtn}
            title="Open task dependency graph"
          >
            Graph
          </button>
          <button type="button" onClick={onClose} style={closeBtn}>
            ✕
          </button>
        </div>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
        {error ? <div style={{ color: '#ef4444', fontSize: 12, marginBottom: 8 }}>{error}</div> : null}

        <label style={labelStyle}>
          Goal
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={!editable || busy}
            rows={2}
            style={inputStyle}
          />
        </label>

        <SectionTitle>Agent pool</SectionTitle>
        <AgentPoolPicker hubAgents={hubAgents} selected={agentPool} onChange={setAgentPool} disabled={!editable || busy} />

        <SectionTitle>Tasks ({tasks.length}/10)</SectionTitle>
        {tasks.map((task, i) => (
          <div key={task.id} style={taskCardStyle}>
            <div style={{ ...taskTitleStyle }}>Task {i + 1}</div>
            <input
              placeholder="Title"
              value={task.title}
              onChange={(e) => {
                const v = e.target.value;
                setTasks((prev) => {
                  const next = [...prev];
                  next[i] = { ...next[i], title: v };
                  return next;
                });
              }}
              disabled={!editable || busy}
              style={inputStyle}
            />
            <textarea
              placeholder="Description"
              value={task.description}
              onChange={(e) => {
                const v = e.target.value;
                setTasks((prev) => {
                  const next = [...prev];
                  next[i] = { ...next[i], description: v };
                  return next;
                });
              }}
              disabled={!editable || busy}
              rows={2}
              style={{ ...inputStyle, marginTop: 6 }}
            />
            <AssignRow
              agents={allAgents}
              task={task}
              disabled={!editable || busy}
              onAssign={(agentId, agentName) => {
                setTasks((prev) => {
                  const next = [...prev];
                  next[i] = { ...next[i], assigned_to: agentId, assigned_name: agentName };
                  return next;
                });
              }}
              onAuto={() => void handleAutoAssign(i)}
            />
            {tasks.length > 1 ? (
              <TaskDependenciesEditor
                taskIndex={i}
                tasks={tasks}
                disabled={!editable || busy}
                onChange={(deps) => {
                  setTasks((prev) => {
                    const next = [...prev];
                    next[i] = { ...next[i], dependencies: deps };
                    return next;
                  });
                }}
              />
            ) : null}
            {editable && tasks.length > 1 ? (
              <button type="button" onClick={() => setTasks((prev) => prev.filter((_, j) => j !== i))} disabled={busy} style={dangerBtn}>
                Remove task
              </button>
            ) : null}
          </div>
        ))}

        {editable && tasks.length < 10 ? (
          <button type="button" onClick={() => setTasks((prev) => [...prev, createEmptyTask()])} disabled={busy} style={secondaryBtn}>
            + Add task
          </button>
        ) : null}

        {editable ? (
          <div style={{ marginTop: 12 }}>
            <button
              type="button"
              onClick={() => setImportOpen(true)}
              disabled={busy}
              style={{ ...secondaryBtn, width: '100%' }}
            >
              Import from markdown…
            </button>
          </div>
        ) : null}

        <div style={actionBarStyle}>
          {editable ? (
            <button type="button" onClick={() => void persist()} disabled={busy} style={secondaryBtn}>
              Save draft
            </button>
          ) : null}
          {collaboration.phase === 'draft' ? (
            <button type="button" onClick={() => void handleSubmit()} disabled={busy} style={secondaryBtn}>
              Submit for review
            </button>
          ) : null}
          {collaboration.phase === 'draft' || collaboration.phase === 'reviewing' ? (
            <button type="button" onClick={() => void handleStart()} disabled={busy} style={primaryBtn}>
              Start execution
            </button>
          ) : null}
        </div>
      </div>

      <RunbookImportModal
        isOpen={importOpen}
        busy={busy}
        onClose={() => setImportOpen(false)}
        onImport={handleImport}
      />

      <RunbookGraphModal
        isOpen={graphOpen}
        collaboration={collaboration}
        agents={allAgents}
        tasks={tasks}
        editable={editable}
        busy={busy}
        onClose={() => setGraphOpen(false)}
        onTasksChange={setTasks}
        onSave={async () => {
          const snap = await persist();
          return !!snap;
        }}
      />
    </div>
  );
}


function SectionTitle({ children }: { children: ReactNode }) {
  return <div style={sectionTitleStyle}>{children}</div>;
}

function AgentPoolPicker({
  hubAgents,
  selected,
  onChange,
  disabled,
}: {
  hubAgents: CollaborationAgent[];
  selected: string[];
  onChange: (ids: string[]) => void;
  disabled: boolean;
}) {
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 12 }}>
      {hubAgents.map((a) => (
        <label key={a.agent_id} style={agentChipLabelStyle(disabled)}>
          <input
            type="checkbox"
            checked={selected.includes(a.agent_id)}
            onChange={() => {
              if (selected.includes(a.agent_id)) {
                onChange(selected.filter((x) => x !== a.agent_id));
              } else {
                onChange([...selected, a.agent_id]);
              }
            }}
            disabled={disabled}
          />{' '}
          @{a.agent_name}
        </label>
      ))}
    </div>
  );
}

function TaskDependenciesEditor({
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

  const candidates = tasks
    .map((other, j) => ({ other, j }))
    .filter(({ j }) => j !== taskIndex);
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
              <span key={depId} style={depChipStyle}>
                Task {j >= 0 ? j + 1 : '?'}: {title}
                {!disabled ? (
                  <button
                    type="button"
                    aria-label="Remove dependency"
                    style={depChipRemoveBtn}
                    onClick={() => onChange(deps.filter((d) => d !== depId))}
                  >
                    ×
                  </button>
                ) : null}
              </span>
            );
          })}
        </div>
      ) : (
        <p style={{ fontSize: 11, color: 'var(--text-secondary, #999)', marginBottom: 8 }}>
          No dependencies — runs in the first wave.
        </p>
      )}
      {!disabled && available.length > 0 ? (
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <select
            value={pick}
            onChange={(e) => setPick(e.target.value)}
            style={{ ...inputStyle, flex: 1, marginTop: 0 }}
          >
            <option value="">Select upstream task…</option>
            {available.map(({ other, j }) => (
              <option key={other.id} value={other.id}>
                Task {j + 1}: {other.title?.trim() || '(untitled)'}
              </option>
            ))}
          </select>
          <button type="button" onClick={addPicked} disabled={!pick} style={secondaryBtn}>
            Add
          </button>
        </div>
      ) : null}
      {!disabled && available.length === 0 && candidates.length > 0 ? (
        <p style={{ fontSize: 11, color: 'var(--text-secondary, #999)' }}>All other tasks are already linked.</p>
      ) : null}
    </div>
  );
}

function AssignRow({
  agents,
  task,
  disabled,
  onAssign,
  onAuto,
}: {
  agents: CollaborationAgent[];
  task: CollaborationTask;
  disabled: boolean;
  onAssign: (id: string, name: string) => void;
  onAuto: () => void;
}) {
  return (
    <div style={{ display: 'flex', gap: 8, marginTop: 8, alignItems: 'center' }}>
      <select
        value={task.assigned_to}
        onChange={(e) => {
          const ag = agents.find((a) => a.agent_id === e.target.value);
          onAssign(e.target.value, ag?.agent_name ?? '');
        }}
        disabled={disabled}
        style={{ ...inputStyle, flex: 1 }}
      >
        <option value="">Assign agent…</option>
        {agents.map((a) => (
          <option key={a.agent_id} value={a.agent_id}>@{a.agent_name}</option>
        ))}
      </select>
      <button type="button" onClick={onAuto} disabled={disabled} style={secondaryBtn}>Auto</button>
    </div>
  );
}

const panelStyle: CSSProperties = {
  position: 'absolute',
  top: 0,
  right: 0,
  width: 380,
  height: '100%',
  backgroundColor: 'var(--bg-secondary, #1e1e1e)',
  color: 'var(--text-primary, #e8e8e8)',
  borderLeft: '1px solid var(--border-color, #333)',
  zIndex: 21,
  display: 'flex',
  flexDirection: 'column',
  overflow: 'hidden',
};

const panelTitleStyle: CSSProperties = {
  margin: 0,
  fontSize: 14,
  fontWeight: 600,
  color: 'var(--text-primary, #e8e8e8)',
};

const taskTitleStyle: CSSProperties = {
  fontWeight: 600,
  marginBottom: 6,
  color: 'var(--text-primary, #e8e8e8)',
};

function agentChipLabelStyle(disabled: boolean): CSSProperties {
  return {
    fontSize: 12,
    color: 'var(--text-primary, #e8e8e8)',
    cursor: disabled ? 'default' : 'pointer',
  };
}

const panelHeaderStyle: CSSProperties = {
  padding: '12px 16px',
  borderBottom: '1px solid var(--border-color, #333)',
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
};

const sectionTitleStyle: CSSProperties = {
  fontSize: 11,
  textTransform: 'uppercase',
  letterSpacing: 0.5,
  color: 'var(--text-secondary, #a3a3a3)',
  marginBottom: 6,
};

const actionBarStyle: CSSProperties = {
  marginTop: 16,
  paddingTop: 12,
  borderTop: '1px solid var(--border-color, #333)',
  display: 'flex',
  flexWrap: 'wrap',
  gap: 8,
};

const labelStyle: CSSProperties = {
  display: 'block',
  fontSize: 12,
  color: 'var(--text-secondary, #b4b4b4)',
  marginBottom: 12,
};

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

const taskCardStyle: CSSProperties = {
  padding: 10,
  marginBottom: 10,
  borderRadius: 8,
  backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
  border: '1px solid var(--border-color, #333)',
};

const primaryBtn: CSSProperties = {
  border: 'none',
  borderRadius: 6,
  backgroundColor: '#8b5cf6',
  color: '#fff',
  fontSize: 12,
  padding: '6px 12px',
  cursor: 'pointer',
};

const secondaryBtn: CSSProperties = {
  border: '1px solid var(--border-color, #444)',
  borderRadius: 6,
  backgroundColor: 'transparent',
  color: 'var(--text-primary, #eee)',
  fontSize: 12,
  padding: '6px 12px',
  cursor: 'pointer',
};

const dangerBtn: CSSProperties = { ...secondaryBtn, color: '#ef4444', marginTop: 8 };

const closeBtn: CSSProperties = {
  border: 'none',
  background: 'transparent',
  color: 'var(--text-secondary, #999)',
  cursor: 'pointer',
  fontSize: 16,
};

const depChipStyle: CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 6,
  padding: '4px 8px',
  borderRadius: 6,
  fontSize: 11,
  backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
  border: '1px solid var(--border-color, #444)',
  color: 'var(--text-primary, #e8e8e8)',
};

const depChipRemoveBtn: CSSProperties = {
  border: 'none',
  background: 'transparent',
  color: 'var(--text-secondary, #aaa)',
  cursor: 'pointer',
  fontSize: 14,
  lineHeight: 1,
  padding: 0,
};
