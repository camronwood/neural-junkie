import { useCallback, useEffect, useState } from 'react';
import type { Collaboration, CollaborationAgent, CollaborationTask } from '../../types/protocol';
import { MAX_RUNBOOK_TASKS, createEmptyTask } from '../../utils/runbookTaskUtils';
import { removeTask, validateDAG } from '../../utils/runbookDAG';
import { RunbookGraphCanvas, useRunbookGraphLayoutActions } from './RunbookGraphCanvas';
import { RunbookGraphToolbar } from './RunbookGraphToolbar';
import { RunbookTaskInspector } from './RunbookTaskInspector';

export interface RunbookGraphModalProps {
  isOpen: boolean;
  collaboration: Collaboration;
  agents: CollaborationAgent[];
  tasks: CollaborationTask[];
  editable: boolean;
  busy?: boolean;
  onClose: () => void;
  onTasksChange: (tasks: CollaborationTask[]) => void;
  /** Return false to keep the modal open (e.g. save failed). */
  onSave?: () => Promise<boolean | void>;
}

export function RunbookGraphModal({
  isOpen,
  collaboration,
  agents,
  tasks,
  editable,
  busy = false,
  onClose,
  onTasksChange,
  onSave,
}: RunbookGraphModalProps) {
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [layoutVersion, setLayoutVersion] = useState(0);
  const { runAutoLayout } = useRunbookGraphLayoutActions(
    collaboration.id,
    tasks,
    setLayoutVersion
  );

  useEffect(() => {
    if (!isOpen) {
      setSelectedTaskId(null);
      setValidationError(null);
    }
  }, [isOpen]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, onClose]);

  const selectedTask = tasks.find((t) => t.id === selectedTaskId);
  const selectedIndex = selectedTask ? tasks.findIndex((t) => t.id === selectedTaskId) : -1;

  const handleAddTask = useCallback(() => {
    if (tasks.length >= MAX_RUNBOOK_TASKS) return;
    const t = createEmptyTask();
    onTasksChange([...tasks, t]);
    setSelectedTaskId(t.id);
    setLayoutVersion((v) => v + 1);
  }, [tasks, onTasksChange]);

  const handleSave = useCallback(async () => {
    const v = validateDAG(tasks);
    if (!v.ok) {
      setValidationError(v.error);
      return;
    }
    if (onSave) {
      const ok = await onSave();
      if (ok === false) return;
    }
    onClose();
  }, [tasks, onSave, onClose]);

  const updateSelectedTask = useCallback(
    (patch: Partial<CollaborationTask>) => {
      if (!selectedTaskId) return;
      onTasksChange(
        tasks.map((t) => (t.id === selectedTaskId ? { ...t, ...patch } : t))
      );
    },
    [selectedTaskId, tasks, onTasksChange]
  );

  const deleteSelectedTask = useCallback(() => {
    if (!selectedTaskId) return;
    onTasksChange(removeTask(tasks, selectedTaskId));
    setSelectedTaskId(null);
    setLayoutVersion((v) => v + 1);
  }, [selectedTaskId, tasks, onTasksChange]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="relative flex flex-col w-[95vw] h-[95vh] rounded-lg overflow-hidden border border-slack-border bg-[#1a1d21]"
        onClick={(e) => e.stopPropagation()}
      >
        <RunbookGraphToolbar
          editable={editable}
          taskCount={tasks.length}
          maxTasks={MAX_RUNBOOK_TASKS}
          validationError={validationError}
          busy={busy}
          onAutoLayout={runAutoLayout}
          onAddTask={handleAddTask}
          onSave={() => void handleSave()}
          onClose={onClose}
        />
        <div style={{ display: 'flex', flex: 1, minHeight: 0 }}>
          <RunbookGraphCanvas
            collaborationId={collaboration.id}
            tasks={tasks}
            phase={collaboration.phase}
            editable={editable}
            onTasksChange={onTasksChange}
            selectedTaskId={selectedTaskId}
            onSelectTask={setSelectedTaskId}
            onValidationError={setValidationError}
            layoutVersion={layoutVersion}
          />
          {selectedTask && selectedIndex >= 0 ? (
            <RunbookTaskInspector
              task={selectedTask}
              taskIndex={selectedIndex}
              agents={agents}
              editable={editable}
              onUpdate={updateSelectedTask}
              onDelete={deleteSelectedTask}
            />
          ) : null}
        </div>
      </div>
    </div>
  );
}
