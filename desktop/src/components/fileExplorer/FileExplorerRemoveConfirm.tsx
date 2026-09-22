export interface FileExplorerRemoveConfirmProps {
  pendingRemove: { id: string; name: string };
  onCancel: () => void;
  onConfirm: () => void;
}

export function FileExplorerRemoveConfirm({
  pendingRemove,
  onCancel,
  onConfirm,
}: FileExplorerRemoveConfirmProps) {
  return (
    <>
      <div className="fixed inset-0 z-50 bg-black/50" onClick={onCancel} />
      <div className="fixed z-50 top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-slack-bg border border-slack-border rounded-lg shadow-xl p-5 min-w-[300px]">
        <h3 className="text-sm font-semibold text-slack-text mb-2">Remove Workspace</h3>
        <p className="text-xs text-slack-textMuted mb-4">
          Remove <span className="font-semibold text-slack-text">"{pendingRemove.name}"</span> from
          the file explorer? No files will be deleted.
        </p>
        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 text-xs rounded bg-slack-bgHover text-slack-text hover:bg-slack-border transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="px-3 py-1.5 text-xs rounded bg-red-600 text-white hover:bg-red-700 transition-colors"
          >
            Remove
          </button>
        </div>
      </div>
    </>
  );
}
