import { useCallback, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useProjectSetsStore } from '../stores/projectSetsStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useEditorStore } from '../stores/editorStore';
import { useSettingsStore } from '../stores/settingsStore';
import { useWorkspaceIndexLabel } from './WorkspaceIndexStatus';
import { ProjectSetDialog, type ProjectSetDialogResult } from './ProjectSetDialog';
import { useWorkspaceAgentScope } from '../hooks/useWorkspaceAgentScope';

const POPOVER_WIDTH = 300;

interface FilesPanelMenuProps {
  repoPath: string | undefined;
  variant?: 'inline' | 'bar';
}

function indexDotClass(label: string): string {
  if (label === 'Index ready') return 'bg-green-400';
  if (label === 'Indexing…') return 'bg-amber-400 animate-pulse';
  if (label === 'Index pending') return 'bg-slack-textMuted';
  return 'bg-transparent';
}

export function FilesPanelMenu({ repoPath, variant = 'inline' }: FilesPanelMenuProps) {
  const workspaces = useFileExplorerStore((s) => s.workspaces);
  const activeWorkspaceId = useFileExplorerStore((s) => s.activeWorkspaceId);
  const setActiveProjectSet = useProjectSetsStore((s) => s.setActiveProjectSet);
  const createProjectSet = useProjectSetsStore((s) => s.createProjectSet);
  const updateProjectSet = useProjectSetsStore((s) => s.updateProjectSet);
  const deleteProjectSet = useProjectSetsStore((s) => s.deleteProjectSet);
  const { scopeLabel, activeProjectSet, projectSets, activeProjectSetId } = useWorkspaceAgentScope();

  const [open, setOpen] = useState(false);
  const [dialogMode, setDialogMode] = useState<'create' | 'edit' | null>(null);
  const [popoverPos, setPopoverPos] = useState({ top: 0, left: 0 });
  const anchorRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  const indexLabel = useWorkspaceIndexLabel(repoPath);
  const openKnowledgeGraphWorkbench = useEditorStore((s) => s.openKnowledgeGraphWorkbench);
  const updateLayoutSettings = useSettingsStore((s) => s.updateLayoutSettings);

  const menuTitle = [activeProjectSet?.name, scopeLabel, indexLabel].filter(Boolean).join(' · ');

  const openKnowledgeGraph = () => {
    const ws = workspaces.find((w) => w.id === activeWorkspaceId);
    if (!ws?.path || !activeWorkspaceId) return;
    closeMenu();
    openKnowledgeGraphWorkbench(activeWorkspaceId, ws.path);
    void updateLayoutSettings({ editorPanelVisible: true });
  };

  useLayoutEffect(() => {
    if (!open || !anchorRef.current) return;

    const pad = 8;
    const rect = anchorRef.current.getBoundingClientRect();
    const popoverH = popoverRef.current?.offsetHeight ?? 320;

    let left = rect.left;
    let top = rect.bottom + 4;

    if (left + POPOVER_WIDTH > window.innerWidth - pad) {
      left = Math.max(pad, window.innerWidth - POPOVER_WIDTH - pad);
    }
    if (top + popoverH > window.innerHeight - pad) {
      top = Math.max(pad, rect.top - popoverH - 4);
    }
    if (top < pad) top = pad;

    setPopoverPos({ top, left });
  }, [open, indexLabel, scopeLabel, activeProjectSetId, projectSets.length]);

  const closeMenu = useCallback(() => setOpen(false), []);

  const openCreateDialog = () => {
    closeMenu();
    setDialogMode('create');
  };

  const openEditDialog = () => {
    closeMenu();
    setDialogMode('edit');
  };

  const handleSaveProjectSet = async (result: ProjectSetDialogResult) => {
    if (dialogMode === 'edit' && activeProjectSetId) {
      const ps = await updateProjectSet(activeProjectSetId, {
        name: result.name,
        primaryWorkspaceId: result.primaryWorkspaceId,
        memberWorkspaceIds: result.memberWorkspaceIds,
      });
      if (ps) setDialogMode(null);
      return;
    }

    const ps = await createProjectSet({
      name: result.name,
      primaryWorkspaceId: result.primaryWorkspaceId,
      memberWorkspaceIds: result.memberWorkspaceIds,
    });
    if (ps) {
      setActiveProjectSet(ps.id);
      setDialogMode(null);
    }
  };

  const handleDeleteProjectSet = async () => {
    if (!activeProjectSetId) return;
    const ok = await deleteProjectSet(activeProjectSetId);
    if (ok) setDialogMode(null);
  };

  const popover = open
    ? createPortal(
        <>
          <div className="fixed inset-0 z-[250]" aria-hidden onMouseDown={closeMenu} />
          <div
            ref={popoverRef}
            className="fixed z-[251] rounded-lg border border-slack-border bg-slack-bg shadow-xl p-3 space-y-3"
            style={{ top: popoverPos.top, left: popoverPos.left, width: POPOVER_WIDTH }}
            role="dialog"
            aria-label="Files options"
          >
            {(indexLabel || repoPath) && (
              <section>
                <h4 className="text-[10px] uppercase tracking-wide text-slack-textMuted mb-1.5">
                  Code index
                </h4>
                {indexLabel && (
                  <div className="flex items-center gap-2 text-xs text-slack-text">
                    <span
                      className={`w-2 h-2 rounded-full flex-shrink-0 ${indexDotClass(indexLabel)}`}
                      aria-hidden
                    />
                    {indexLabel}
                  </div>
                )}
                <p className="text-[11px] text-slack-textMuted mt-1">
                  Powers @codebase search and specialist consult for the active workspace.
                </p>
                {repoPath && activeWorkspaceId && (
                  <button
                    type="button"
                    onClick={openKnowledgeGraph}
                    className="mt-2 w-full text-left text-xs px-2 py-1.5 rounded bg-slack-bgHover hover:bg-slack-border text-slack-text"
                  >
                    Open knowledge graph
                  </button>
                )}
              </section>
            )}

            <section className="border-t border-slack-border pt-3">
              <h4 className="text-[10px] uppercase tracking-wide text-slack-textMuted mb-1.5">
                Agent scope
              </h4>
              <p className="text-xs text-slack-text">
                {scopeLabel ?? 'Single workspace — no linked repos in scope'}
              </p>
              <p className="text-[11px] text-slack-textMuted mt-1">
                Repos included when you message agents from this panel.
              </p>
            </section>

            <section className="border-t border-slack-border pt-3">
              <div className="flex items-center justify-between mb-2">
                <h4 className="text-[10px] uppercase tracking-wide text-slack-textMuted">
                  Project sets
                </h4>
                <button
                  type="button"
                  onClick={openCreateDialog}
                  className="text-[10px] text-slack-accent hover:underline"
                >
                  New
                </button>
              </div>

              {projectSets.length === 0 ? (
                <p className="text-xs text-slack-textMuted">
                  {workspaces.length < 2
                    ? 'Register another workspace to link repos.'
                    : 'No project sets yet.'}
                </p>
              ) : (
                <div className="space-y-1 max-h-36 overflow-y-auto">
                  <label className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-slack-bgHover cursor-pointer text-xs text-slack-text">
                    <input
                      type="radio"
                      name="files-panel-project-set"
                      checked={!activeProjectSetId}
                      onChange={() => setActiveProjectSet(null)}
                      className="accent-slack-accent"
                    />
                    No project set
                  </label>
                  {projectSets.map((ps) => (
                    <label
                      key={ps.id}
                      className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-slack-bgHover cursor-pointer text-xs text-slack-text"
                    >
                      <input
                        type="radio"
                        name="files-panel-project-set"
                        checked={activeProjectSetId === ps.id}
                        onChange={() => setActiveProjectSet(ps.id)}
                        className="accent-slack-accent"
                      />
                      <span className="truncate">{ps.name}</span>
                    </label>
                  ))}
                </div>
              )}

              {activeProjectSet && (
                <button
                  type="button"
                  onClick={openEditDialog}
                  className="mt-2 text-[11px] text-slack-textMuted hover:text-slack-text"
                >
                  Edit “{activeProjectSet.name}”…
                </button>
              )}

              {projectSets.length === 0 && workspaces.length >= 2 && (
                <button
                  type="button"
                  onClick={openCreateDialog}
                  className="mt-2 text-[11px] text-slack-accent hover:underline"
                >
                  Create project set…
                </button>
              )}
            </section>
          </div>
        </>,
        document.body
      )
    : null;

  const dialog =
    dialogMode && workspaces.length > 0 ? (
      <ProjectSetDialog
        mode={dialogMode}
        workspaces={workspaces}
        activeWorkspaceId={activeWorkspaceId}
        initial={dialogMode === 'edit' ? activeProjectSet : undefined}
        onSave={(result) => void handleSaveProjectSet(result)}
        onDelete={dialogMode === 'edit' ? () => void handleDeleteProjectSet() : undefined}
        onCancel={() => setDialogMode(null)}
      />
    ) : null;

  const isBar = variant === 'bar';

  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={
          isBar
            ? 'flex items-center justify-center gap-1.5 px-3 py-1 rounded text-[10px] uppercase tracking-wide text-slack-textMuted hover:text-slack-text hover:bg-slack-bg transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-slack-accent flex-shrink-0'
            : 'flex items-center gap-1 px-1 py-0.5 rounded text-slack-textMuted hover:text-slack-text hover:bg-slack-bg transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-slack-accent'
        }
        title={menuTitle ? `Files options — ${menuTitle}` : 'Files options'}
        aria-label="Files options"
        aria-expanded={open}
        aria-haspopup="dialog"
      >
        {indexLabel && (
          <span
            className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${indexDotClass(indexLabel)}`}
            aria-hidden
          />
        )}
        {isBar && <span>Options</span>}
        <svg
          className={`w-3 h-3 flex-shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {popover}
      {dialog}
    </>
  );
}
