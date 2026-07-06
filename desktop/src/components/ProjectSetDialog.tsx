import { useEffect, useMemo, useRef, useState } from 'react';
import type { Workspace } from '../stores/fileExplorerStore';
import type { ProjectSet } from '../stores/projectSetsStore';

export interface ProjectSetDialogResult {
  name: string;
  primaryWorkspaceId: string;
  memberWorkspaceIds: string[];
}

interface ProjectSetDialogProps {
  mode: 'create' | 'edit';
  workspaces: Workspace[];
  activeWorkspaceId: string | null;
  initial?: ProjectSet;
  onSave: (result: ProjectSetDialogResult) => void;
  onDelete?: () => void;
  onCancel: () => void;
}

function initialSelectedIds(
  mode: 'create' | 'edit',
  workspaces: Workspace[],
  activeWorkspaceId: string | null,
  initial?: ProjectSet
): Set<string> {
  if (mode === 'edit' && initial) {
    const ids = new Set<string>([initial.primary_workspace_id, ...initial.member_workspace_ids]);
    return ids;
  }
  const active = activeWorkspaceId ?? workspaces[0]?.id;
  if (!active) return new Set();
  return new Set([active]);
}

export function ProjectSetDialog({
  mode,
  workspaces,
  activeWorkspaceId,
  initial,
  onSave,
  onDelete,
  onCancel,
}: ProjectSetDialogProps) {
  const nameInputRef = useRef<HTMLInputElement>(null);
  const [name, setName] = useState(initial?.name ?? '');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() =>
    initialSelectedIds(mode, workspaces, activeWorkspaceId, initial)
  );
  const [primaryId, setPrimaryId] = useState(
    () => initial?.primary_workspace_id ?? activeWorkspaceId ?? workspaces[0]?.id ?? ''
  );

  useEffect(() => {
    nameInputRef.current?.focus();
    nameInputRef.current?.select();
  }, []);

  const selectedList = useMemo(
    () => workspaces.filter((w) => selectedIds.has(w.id)),
    [workspaces, selectedIds]
  );

  const toggleWorkspace = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
        if (primaryId === id) {
          const fallback = [...next][0] ?? '';
          setPrimaryId(fallback);
        }
      } else {
        next.add(id);
        if (!primaryId || !next.has(primaryId)) {
          setPrimaryId(id);
        }
      }
      return next;
    });
  };

  const trimmedName = name.trim();
  const canSave =
    trimmedName.length > 0 &&
    selectedList.length > 0 &&
    selectedIds.has(primaryId);

  const submit = () => {
    if (!canSave) return;
    const memberWorkspaceIds = selectedList
      .map((w) => w.id)
      .filter((id) => id !== primaryId);
    onSave({
      name: trimmedName,
      primaryWorkspaceId: primaryId,
      memberWorkspaceIds,
    });
  };

  return (
    <>
      <div className="fixed inset-0 z-[260] bg-black/50" onClick={onCancel} aria-hidden />
      <div
        className="fixed z-[261] top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-slack-bg border border-slack-border rounded-lg shadow-xl p-5 min-w-[360px] max-w-[min(480px,calc(100vw-2rem))] max-h-[min(80vh,560px)] flex flex-col"
        role="dialog"
        aria-labelledby="project-set-dialog-title"
      >
        <h3 id="project-set-dialog-title" className="text-sm font-semibold text-slack-text mb-1">
          {mode === 'create' ? 'Create project set' : 'Edit project set'}
        </h3>
        <p className="text-xs text-slack-textMuted mb-4">
          Choose which registered workspaces belong in this set. The primary repo is the default
          focus; other selected repos stay in agent scope.
        </p>

        <label className="block text-xs text-slack-textMuted mb-1">Name</label>
        <input
          ref={nameInputRef}
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && canSave) submit();
            if (e.key === 'Escape') onCancel();
          }}
          placeholder="Neural Junkie Platform"
          className="w-full px-3 py-2 mb-4 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text focus:outline-none focus:border-slack-accent"
        />

        <div className="text-xs text-slack-textMuted mb-2">Workspaces</div>
        <div className="flex-1 min-h-0 overflow-y-auto border border-slack-border rounded mb-4 divide-y divide-slack-border">
          {workspaces.length === 0 ? (
            <p className="px-3 py-4 text-xs text-slack-textMuted">No workspaces registered in Files.</p>
          ) : (
            workspaces.map((workspace) => {
              const checked = selectedIds.has(workspace.id);
              const isPrimary = primaryId === workspace.id;
              return (
                <label
                  key={workspace.id}
                  className={`flex items-start gap-3 px-3 py-2.5 cursor-pointer hover:bg-slack-bgHover ${
                    checked ? 'bg-slack-bgHover/60' : ''
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => toggleWorkspace(workspace.id)}
                    className="mt-0.5 accent-slack-accent"
                  />
                  <span className="flex-1 min-w-0">
                    <span className="block text-sm text-slack-text truncate">{workspace.name}</span>
                    <span className="block text-[11px] text-slack-textMuted truncate font-mono">
                      {workspace.path}
                    </span>
                  </span>
                  <span className="flex items-center gap-1.5 flex-shrink-0 pt-0.5">
                    <input
                      type="radio"
                      name="project-set-primary"
                      checked={isPrimary}
                      disabled={!checked}
                      onChange={() => setPrimaryId(workspace.id)}
                      className="accent-slack-accent"
                      title="Primary workspace"
                    />
                    <span className="text-[10px] uppercase tracking-wide text-slack-textMuted">
                      Primary
                    </span>
                  </span>
                </label>
              );
            })
          )}
        </div>

        {selectedList.length === 1 && (
          <p className="text-[11px] text-slack-textMuted mb-3">
            Add another workspace to link multiple repos in agent scope.
          </p>
        )}

        <div className="flex items-center justify-between gap-2">
          {mode === 'edit' && onDelete ? (
            <button
              type="button"
              onClick={onDelete}
              className="px-3 py-1.5 text-xs rounded text-red-400 hover:text-red-300 hover:bg-red-500/10"
            >
              Delete set
            </button>
          ) : (
            <span />
          )}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onCancel}
              className="px-3 py-1.5 text-xs rounded bg-slack-bgHover text-slack-text hover:bg-slack-border"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={submit}
              disabled={!canSave}
              className="px-3 py-1.5 text-xs rounded bg-slack-accent text-white hover:opacity-90 disabled:opacity-50"
            >
              {mode === 'create' ? 'Create' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
