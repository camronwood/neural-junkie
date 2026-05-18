import type { Collaboration } from '../types/protocol';
import { useFileExplorerStore } from '../stores/fileExplorerStore';

interface CollaborationWorkspaceGateProps {
  collaboration: Collaboration | null;
  busy: boolean;
  onContinue: () => void | Promise<void>;
  onNotNow: () => void;
}

/**
 * Blocks the collaboration channel until the user confirms execution workspace
 * setup and tells the hub to deliver task prompts to agents.
 */
export function CollaborationWorkspaceGate({
  collaboration,
  busy,
  onContinue,
  onNotNow,
}: CollaborationWorkspaceGateProps) {
  if (!collaboration) {
    return null;
  }

  const isWorktree = collaboration.execution_mode === 'worktree';
  if (!isWorktree && !collaboration.working_directory) {
    return null;
  }

  const activeWorkspace = useFileExplorerStore((s) => s.getActiveWorkspace());
  const sourceRepo =
    collaboration.source_repo_path?.trim() ||
    (isWorktree && !collaboration.source_repo_path?.trim() ? activeWorkspace?.path : '') ||
    '';
  const branchPreview =
    collaboration.worktree_branch ||
    (isWorktree ? `nj/collab-${collaboration.id.slice(0, 8)}` : '');

  const canContinueWorktree =
    !isWorktree ||
    !!collaboration.source_repo_path?.trim() ||
    (activeWorkspace?.is_git_repo && !!activeWorkspace.path?.trim());

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/70 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="collab-workspace-gate-title"
    >
      <div className="max-w-lg w-full rounded-lg border border-gray-600 bg-gray-900 p-6 shadow-xl">
        <h2 id="collab-workspace-gate-title" className="text-lg font-semibold text-white mb-2">
          {isWorktree ? 'Collaboration git worktree' : 'Collaboration workspace'}
        </h2>
        {isWorktree ? (
          <>
            <p className="text-gray-300 text-sm mb-4">
              Execution uses a git worktree on a dedicated branch. Your main checkout stays untouched;
              merge the branch when you are done.
            </p>
            {sourceRepo ? (
              <p className="text-xs text-gray-400 mb-2">
                <span className="text-gray-500">Source repo:</span>{' '}
                <span className="font-mono break-all">{sourceRepo}</span>
              </p>
            ) : (
              <p className="text-xs text-amber-200/90 mb-2">
                Source repo will be taken from your active workspace in the file explorer.
                {!canContinueWorktree && ' Select a git repository workspace to continue.'}
              </p>
            )}
            {branchPreview ? (
              <p className="text-xs text-gray-400 mb-2">
                <span className="text-gray-500">Branch:</span>{' '}
                <span className="font-mono">{branchPreview}</span>
              </p>
            ) : null}
            {collaboration.working_directory ? (
              <p className="text-xs font-mono text-gray-400 break-all mb-6">
                Worktree: {collaboration.working_directory}
              </p>
            ) : (
              <p className="text-xs text-gray-500 mb-6">
                Worktree path is created under your Collaboration output folder when you continue.
              </p>
            )}
          </>
        ) : (
          <>
            <p className="text-gray-300 text-sm mb-4">
              Execution uses a sandbox folder on disk. Add it as a workspace here, then confirm so agents
              receive their task prompts and file changes resolve correctly.
            </p>
            <p className="text-xs font-mono text-gray-400 break-all mb-6">{collaboration.working_directory}</p>
          </>
        )}
        <div className="flex justify-end gap-2">
          <button
            type="button"
            className="px-4 py-2 text-sm rounded bg-gray-700 text-gray-200 hover:bg-gray-600 disabled:opacity-50"
            onClick={onNotNow}
            disabled={busy}
          >
            Not now
          </button>
          <button
            type="button"
            className="px-4 py-2 text-sm rounded bg-emerald-700 text-white hover:bg-emerald-600 disabled:opacity-50"
            onClick={() => void onContinue()}
            disabled={busy || (isWorktree && !canContinueWorktree)}
          >
            {busy ? 'Working…' : 'Continue'}
          </button>
        </div>
      </div>
    </div>
  );
}
