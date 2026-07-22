import { useFileChangeStore } from '../stores/fileChangeStore';

interface EditorReviewBarProps {
  /** Absolute workspace root path; when set, only changes under this root are shown. */
  workspaceRoot?: string;
}

export function EditorReviewBar({ workspaceRoot }: EditorReviewBarProps) {
  const { pendingChanges, approveChange, rejectChange, busyById } = useFileChangeStore();
  const relevant = pendingChanges.filter((c) => {
    if (!workspaceRoot) return true;
    const fp = c.file_path || c.new_path || c.old_path || '';
    return fp.startsWith(workspaceRoot);
  });
  const busy = relevant.some((change) => busyById[change.id]);

  if (relevant.length === 0) return null;

  const acceptAll = async () => {
    for (const c of [...relevant]) {
      await approveChange(c.id);
    }
    await useFileChangeStore.getState().fetchPendingChanges();
  };

  const rejectAll = async () => {
    for (const c of [...relevant]) {
      await rejectChange(c.id, 'Rejected from editor review bar');
    }
    await useFileChangeStore.getState().fetchPendingChanges();
  };

  return (
    <div
      className="flex items-center justify-between gap-2 px-3 py-2 bg-slack-accent/20 border-b border-slack-accent text-sm"
      role="region"
      aria-label="Review agent changes"
    >
      <span className="text-slack-text">
        {relevant.length} pending change{relevant.length === 1 ? '' : 's'}
      </span>
      <div className="flex gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void rejectAll()}
          className="px-2 py-1 text-xs rounded border border-slack-border hover:bg-slack-bgHover"
        >
          Reject all
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => void acceptAll()}
          className="px-2 py-1 text-xs rounded bg-green-600 hover:bg-green-700 text-white"
        >
          Accept all
        </button>
      </div>
    </div>
  );
}
