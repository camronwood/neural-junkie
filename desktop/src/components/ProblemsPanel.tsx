import { shallow } from 'zustand/shallow';
import { useDiagnosticsStore } from '../stores/diagnosticsStore';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';

interface ProblemsPanelProps {
  isOpen: boolean;
  onClose: () => void;
  onOpenAt: (path: string, line: number) => void | Promise<void>;
}

export function ProblemsPanel({ isOpen, onClose, onOpenAt }: ProblemsPanelProps) {
  const { activeWorkspaceId } = useFileExplorerStore(
    (s) => ({ activeWorkspaceId: s.activeWorkspaceId }),
    shallow
  );
  const items = useDiagnosticsStore((s) => s.allForWorkspace());

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[95] flex items-end justify-center pb-8 bg-black/40"
      role="dialog"
      aria-label="Problems"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="w-full max-w-2xl max-h-[50vh] rounded-lg border border-slack-border bg-slack-bg shadow-xl flex flex-col overflow-hidden">
        <div className="flex items-center justify-between px-4 py-2 border-b border-slack-border">
          <span className="text-sm font-medium text-slack-text">Problems</span>
          <button
            type="button"
            className="text-slack-textMuted hover:text-slack-text text-sm"
            onClick={onClose}
          >
            Close
          </button>
        </div>
        <ul className="overflow-y-auto text-sm flex-1">
          {items.length === 0 && (
            <li className="px-4 py-3 text-slack-textMuted">No problems reported.</li>
          )}
          {items.map((d, i) => (
            <li key={`${d.path}:${d.line}:${i}`}>
              <button
                type="button"
                className="w-full text-left px-4 py-2 hover:bg-slack-bgHover text-slack-text"
                onClick={() => {
                  if (activeWorkspaceId) {
                    useEditorStore.getState().revealLine(activeWorkspaceId, d.path, d.line);
                  }
                  void onOpenAt(d.path, d.line);
                }}
              >
                <span
                  className={
                    d.severity === 'error'
                      ? 'text-red-400'
                      : d.severity === 'warning'
                        ? 'text-yellow-400'
                        : 'text-slack-textMuted'
                  }
                >
                  {d.severity}
                </span>
                <span className="ml-2 text-slack-text">{d.message}</span>
                <span className="block text-xs text-slack-textMuted mt-0.5">
                  {d.path}:{d.line}
                </span>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
