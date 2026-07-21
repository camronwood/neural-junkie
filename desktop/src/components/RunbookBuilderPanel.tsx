import { useCallback, useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { shallow } from 'zustand/shallow';
import { ChatAPI } from '../api/chatAPI';
import { useChatStore } from '../stores/chatStore';
import type { Collaboration, CollaborationAgent, CollaborationTask } from '../types/protocol';
import { ensureCollaborationExecutionWorkspace } from '../utils/collaborationExecutionWorkspace';
import { shouldAutoAckWorkspaceOnApprove, isAwaitingWorkspaceConfirmation } from '../utils/collaborationPanelState';
import { RunbookImportModal } from './RunbookImportModal';
import { RunbookGraphModal } from './runbook-graph';
import { RunbookActionConfigEditor } from './runbook/RunbookActionConfigEditor';
import { TaskDependenciesEditor } from './runbook/TaskDependenciesEditor';
import { RunbookSaveDialog } from './runbook/RunbookSaveDialog';
import { RunbookStartModal } from './runbook/RunbookStartModal';
import { defaultActionSpec } from '../utils/runbookActionUtils';
import type { GraphLayout, RunInputSpec } from '../types/protocol';
import { MAX_RUNBOOK_TASKS, createEmptyTask } from '../utils/runbookTaskUtils';
import { registerRestartBlocker } from '../utils/restartSafety';

interface RunbookBuilderPanelProps {
  collaboration: Collaboration;
  hubAgents: CollaborationAgent[];
  onClose: () => void;
  onSaved: (collab: Collaboration) => void;
  onStarted?: (collab: Collaboration) => void;
  onWorkspaceGateRequest?: (collab: Collaboration) => void;
  onDirtyChange?: (dirty: boolean) => void;
}

export function RunbookBuilderPanel({
  collaboration,
  hubAgents,
  onClose,
  onSaved,
  onStarted,
  onWorkspaceGateRequest,
  onDirtyChange,
}: RunbookBuilderPanelProps) {
  const { serverAddr } = useChatStore((s) => ({ serverAddr: s.serverAddr }), shallow);
  const [api] = useState(() => new ChatAPI(serverAddr));
  const [description, setDescription] = useState(collaboration.description);
  const [agentPool, setAgentPool] = useState<string[]>(collaboration.agents.map((a) => a.agent_id));
  const [tasks, setTasks] = useState<CollaborationTask[]>(
    collaboration.tasks?.length ? collaboration.tasks : [createEmptyTask()]
  );
  const [executionPolicy, setExecutionPolicy] = useState(
    collaboration.execution_policy ?? { blocked_upstream_policy: 'block' as const, strict_task_status: true }
  );
  const [importOpen, setImportOpen] = useState(false);
  const [graphOpen, setGraphOpen] = useState(false);
  const [saveOpen, setSaveOpen] = useState(false);
  const [startOpen, setStartOpen] = useState(false);
  const [startInputs, setStartInputs] = useState<RunInputSpec[]>([]);
  const [graphLayout, setGraphLayout] = useState<GraphLayout>(collaboration.graph_layout ?? {});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [dirty, setDirty] = useState(false);

  const markDirty = useCallback(() => {
    setDirty(true);
  }, []);

  useEffect(() => {
    onDirtyChange?.(dirty);
  }, [dirty, onDirtyChange]);

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
        execution_policy: executionPolicy,
        graph_layout: graphLayout,
      });
      onSaved(snap);
      setDirty(false);
      return snap;
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return null;
    } finally {
      setBusy(false);
    }
  }, [api, collaboration.id, description, agentPool, tasks, executionPolicy, graphLayout, onSaved]);

  useEffect(
    () =>
      registerRestartBlocker(`runbook-builder:${collaboration.id}`, () =>
        dirty
          ? {
              id: `runbook-builder:${collaboration.id}`,
              message: 'Unsaved runbook changes must be saved before restarting.',
              save: async () => Boolean(await persist()),
            }
          : null
      ),
    [collaboration.id, dirty, persist]
  );

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
      markDirty();
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
      markDirty();
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

  const openStartModal = async () => {
    let inputs: RunInputSpec[] = [];
    if (collaboration.definition_id) {
      try {
        const def = await api.getRunbookDefinition(collaboration.definition_id, collaboration.definition_version);
        inputs = def.inputs ?? [];
      } catch {
        inputs = [];
      }
    }
    setStartInputs(inputs);
    setStartOpen(true);
  };

  const runStart = async (inputValues: Record<string, string>) => {
    setStartOpen(false);
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
      let started = await api.startRunbook(collaboration.id, inputValues);
      onSaved(started);
      if (!started.workspace_acknowledged) {
        await ensureCollaborationExecutionWorkspace(started);
        if (shouldAutoAckWorkspaceOnApprove(started)) {
          await api.acknowledgeCollaborationWorkspace(started.id);
          started = await api.getRunbook(started.id);
          onSaved(started);
        } else if (isAwaitingWorkspaceConfirmation(started)) {
          onWorkspaceGateRequest?.(started);
        }
      }
      onStarted?.(started);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const handleStart = async () => {
    await openStartModal();
  };

  const collabIdRef = useRef(collaboration.id);

  useEffect(() => {
    if (collaboration.id !== collabIdRef.current) {
      collabIdRef.current = collaboration.id;
      setDescription(collaboration.description);
      setAgentPool(collaboration.agents.map((a) => a.agent_id));
      setTasks(collaboration.tasks?.length ? collaboration.tasks : [createEmptyTask()]);
      setExecutionPolicy(
        collaboration.execution_policy ?? { blocked_upstream_policy: 'block' as const, strict_task_status: true }
      );
      setDirty(false);
      return;
    }
    if (dirty) return;
    setDescription(collaboration.description);
    setAgentPool(collaboration.agents.map((a) => a.agent_id));
    if (collaboration.tasks?.length) {
      setTasks(collaboration.tasks);
    }
    if (collaboration.execution_policy) {
      setExecutionPolicy(collaboration.execution_policy);
    }
    if (collaboration.graph_layout) {
      setGraphLayout(collaboration.graph_layout);
    }
  }, [
    collaboration.id,
    collaboration.description,
    collaboration.agents,
    collaboration.tasks,
    collaboration.execution_policy,
    collaboration.graph_layout,
    dirty,
  ]);

  return (
    <div style={panelStyle}>
      <div style={panelHeaderStyle}>
        <h3 style={panelTitleStyle}>
          Runbook builder
          {dirty ? <span style={unsavedBadgeStyle}>Unsaved</span> : null}
        </h3>
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
            onChange={(e) => {
              setDescription(e.target.value);
              markDirty();
            }}
            disabled={!editable || busy}
            rows={2}
            style={inputStyle}
          />
        </label>

        <SectionTitle>Execution policy</SectionTitle>
        <label style={labelStyle}>
          Blocked upstream
          <select
            value={executionPolicy.blocked_upstream_policy ?? 'block'}
            onChange={(e) => {
              setExecutionPolicy((p) => ({
                ...p,
                blocked_upstream_policy: e.target.value as 'block' | 'skip_branch' | 'fail_run',
              }));
              markDirty();
            }}
            disabled={!editable || busy}
            style={inputStyle}
          >
            <option value="block">Block downstream</option>
            <option value="skip_branch">Skip branch</option>
            <option value="fail_run">Fail run</option>
          </select>
        </label>
        <label style={labelStyle}>
          Max parallel tasks (0 = unlimited)
          <input
            type="number"
            min={0}
            value={executionPolicy.max_concurrent_tasks ?? 0}
            onChange={(e) => {
              setExecutionPolicy((p) => ({
                ...p,
                max_concurrent_tasks: parseInt(e.target.value, 10) || 0,
              }));
              markDirty();
            }}
            disabled={!editable || busy}
            style={inputStyle}
          />
        </label>

        <SectionTitle>Agent pool</SectionTitle>
        <AgentPoolPicker
          hubAgents={hubAgents}
          selected={agentPool}
          onChange={(ids) => {
            setAgentPool(ids);
            markDirty();
          }}
          disabled={!editable || busy}
        />

        <SectionTitle>Tasks ({tasks.length}/{MAX_RUNBOOK_TASKS})</SectionTitle>
        {tasks.map((task, i) => (
          <div key={task.id} style={taskCardStyle}>
            <div style={{ ...taskTitleStyle }}>Task {i + 1}</div>
            <select
              value={task.kind ?? 'agent'}
              onChange={(e) => {
                const kind = e.target.value as 'agent' | 'action';
                setTasks((prev) => {
                  const next = [...prev];
                  next[i] = {
                    ...next[i],
                    kind,
                    action:
                      kind === 'action'
                        ? next[i].action ?? defaultActionSpec('http_get')
                        : undefined,
                  };
                  return next;
                });
                markDirty();
              }}
              disabled={!editable || busy}
              style={{ ...inputStyle, marginBottom: 6 }}
            >
              <option value="agent">Agent task</option>
              <option value="action">Action (HTTP / webhook / …)</option>
            </select>
            {task.kind === 'action' && task.action ? (
              <RunbookActionConfigEditor
                action={task.action}
                api={api}
                disabled={!editable || busy}
                inputStyle={inputStyle}
                labelStyle={labelStyle}
                onChange={(action) => {
                  setTasks((prev) => {
                    const next = [...prev];
                    next[i] = { ...next[i], action };
                    return next;
                  });
                  markDirty();
                }}
              />
            ) : null}
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
                markDirty();
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
                markDirty();
              }}
              disabled={!editable || busy}
              rows={2}
              style={{ ...inputStyle, marginTop: 6 }}
            />
            {(task.kind ?? 'agent') !== 'action' ? (
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
                  markDirty();
                }}
                onAuto={() => void handleAutoAssign(i)}
              />
            ) : null}
            {tasks.length > 1 ? (
              <TaskDependenciesEditor
                taskIndex={i}
                tasks={tasks}
                disabled={!editable || busy}
                onChange={(patch) => {
                  setTasks((prev) => {
                    const next = [...prev];
                    next[i] = { ...next[i], ...patch };
                    return next;
                  });
                  markDirty();
                }}
              />
            ) : null}
            {editable && tasks.length > 1 ? (
              <button
                type="button"
                onClick={() => {
                  setTasks((prev) => prev.filter((_, j) => j !== i));
                  markDirty();
                }}
                disabled={busy}
                style={dangerBtn}
              >
                Remove task
              </button>
            ) : null}
          </div>
        ))}

        {editable && tasks.length < MAX_RUNBOOK_TASKS ? (
          <button
            type="button"
            onClick={() => {
              setTasks((prev) => [...prev, createEmptyTask()]);
              markDirty();
            }}
            disabled={busy}
            style={secondaryBtn}
          >
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
          {editable ? (
            <button type="button" onClick={() => setSaveOpen(true)} disabled={busy} style={secondaryBtn}>
              Save to library
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

      <RunbookSaveDialog
        isOpen={saveOpen}
        api={api}
        collaboration={collaboration}
        tasks={tasks}
        executionPolicy={executionPolicy}
        graphLayout={graphLayout}
        onClose={() => setSaveOpen(false)}
        onSaved={() => setSaveOpen(false)}
      />

      <RunbookStartModal
        isOpen={startOpen}
        inputs={startInputs}
        busy={busy}
        onClose={() => setStartOpen(false)}
        onStart={(values) => void runStart(values)}
      />

      <RunbookImportModal
        isOpen={importOpen}
        busy={busy}
        onClose={() => setImportOpen(false)}
        onImport={handleImport}
      />

      <RunbookGraphModal
        isOpen={graphOpen}
        collaboration={{ ...collaboration, graph_layout: graphLayout }}
        agents={allAgents}
        tasks={tasks}
        api={api}
        editable={editable}
        busy={busy}
        onClose={() => setGraphOpen(false)}
        onTasksChange={(next) => {
          setTasks(next);
          markDirty();
        }}
        onLayoutChange={(layout) => {
          setGraphLayout(layout);
          markDirty();
        }}
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
  width: 'min(380px, 100%)',
  minWidth: 'min(260px, 100%)',
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
  display: 'flex',
  alignItems: 'center',
  gap: 8,
};

const unsavedBadgeStyle: CSSProperties = {
  fontSize: 10,
  fontWeight: 500,
  textTransform: 'uppercase',
  letterSpacing: 0.4,
  padding: '2px 6px',
  borderRadius: 4,
  backgroundColor: '#78350f',
  color: '#fcd34d',
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
