import type { Collaboration } from '../types/protocol';

interface CollaborationWorkspaceGateProps {
  collaboration: Collaboration | null;
  busy: boolean;
  onContinue: () => void | Promise<void>;
  onNotNow: () => void;
}

/**
 * Blocks the collaboration channel until the user confirms adding the execution
 * sandbox as a workspace and telling the hub to deliver task prompts to agents.
 */
export function CollaborationWorkspaceGate({
  collaboration,
  busy,
  onContinue,
  onNotNow,
}: CollaborationWorkspaceGateProps) {
  if (!collaboration?.working_directory) {
    return null;
  }

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/70 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="collab-workspace-gate-title"
    >
      <div className="max-w-lg w-full rounded-lg border border-gray-600 bg-gray-900 p-6 shadow-xl">
        <h2 id="collab-workspace-gate-title" className="text-lg font-semibold text-white mb-2">
          Collaboration workspace
        </h2>
        <p className="text-gray-300 text-sm mb-4">
          Execution uses a sandbox folder on disk. Add it as a workspace here, then confirm so agents receive
          their task prompts and file changes resolve correctly.
        </p>
        <p className="text-xs font-mono text-gray-400 break-all mb-6">{collaboration.working_directory}</p>
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
            disabled={busy}
          >
            {busy ? 'Working…' : 'Continue'}
          </button>
        </div>
      </div>
    </div>
  );
}
