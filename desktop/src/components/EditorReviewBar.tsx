import { useFileChangeStore } from '../stores/fileChangeStore';

interface EditorReviewBarProps {
  /** Absolute workspace root path; when set, only changes under this root are shown. */
  workspaceRoot?: string;
}

function requestIdForChange(change: { id: string; metadata?: Record<string, unknown> }): string | null {
  const raw = change.metadata?.request_id;
  return typeof raw === 'string' && raw.trim() !== '' ? raw : null;
}

export function EditorReviewBar({ workspaceRoot }: EditorReviewBarProps) {
  const { pendingChanges, approveChange, rejectChange, approveRequest, rejectRequest, busyById } =
    useFileChangeStore();
  const relevant = pendingChanges.filter((c) => {
    if (!workspaceRoot) return true;
    const fp = c.file_path || c.new_path || c.old_path || '';
    return fp.startsWith(workspaceRoot);
  });
  const busy = relevant.some((change) => {
    const requestId = requestIdForChange(change);
    return busyById[change.id] || (requestId ? busyById[requestId] : false);
  });

  if (relevant.length === 0) return null;

  const acceptAll = async () => {
    const seenRequests = new Set<string>();
    for (const c of [...relevant]) {
      const requestId = requestIdForChange(c);
      if (requestId) {
        if (seenRequests.has(requestId)) continue;
        seenRequests.add(requestId);
        await approveRequest(requestId);
        continue;
      }
      await approveChange(c.id);
    }
    await useFileChangeStore.getState().fetchPendingChanges();
  };

  const rejectAll = async () => {
    const seenRequests = new Set<string>();
    for (const c of [...relevant]) {
      const requestId = requestIdForChange(c);
      if (requestId) {
        if (seenRequests.has(requestId)) continue;
        seenRequests.add(requestId);
        await rejectRequest(requestId, 'Rejected from editor review bar');
        continue;
      }
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
