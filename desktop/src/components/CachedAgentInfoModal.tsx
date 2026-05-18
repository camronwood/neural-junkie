import type { CachedAgentInfo } from '../types/protocol';
import { getAgentColor } from '../types/protocol';

interface CachedAgentInfoModalProps {
  agent: CachedAgentInfo;
  isOpen: boolean;
  onClose: () => void;
  onLoad: (agent: CachedAgentInfo) => void;
  onDelete: (agent: CachedAgentInfo) => void;
  loading?: boolean;
  deleting?: boolean;
}

export function CachedAgentInfoModal({
  agent,
  isOpen,
  onClose,
  onLoad,
  onDelete,
  loading = false,
  deleting = false,
}: CachedAgentInfoModalProps) {
  if (!isOpen) return null;

  const agentColor = getAgentColor(agent.type as Parameters<typeof getAgentColor>[0]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6 border-b border-slack-border">
          <div className="flex items-center gap-3">
            <div
              className="w-10 h-10 rounded flex items-center justify-center text-white text-lg"
              style={{ backgroundColor: agentColor }}
            >
              {agent.type === 'repo' ? '📦' : agent.type === 'cli' ? '⌨️' : '📚'}
            </div>
            <div>
              <h2 className="text-lg font-semibold text-slack-text">{agent.name}</h2>
              <p className="text-sm text-slack-textMuted capitalize">{agent.type} · not loaded</p>
            </div>
          </div>
        </div>

        <div className="p-6 space-y-3 text-sm">
          {agent.path && (
            <div>
              <div className="text-slack-textMuted text-xs mb-1">Path</div>
              <div className="text-slack-text font-mono text-xs break-all">{agent.path}</div>
            </div>
          )}
          <div className="flex gap-4 text-slack-textMuted text-xs">
            {agent.last_used && <span>Last used: {new Date(agent.last_used).toLocaleString()}</span>}
            {agent.cache_size > 0 && <span>Cache: {(agent.cache_size / 1024).toFixed(1)} KB</span>}
          </div>
          <p className="text-xs text-slack-textMuted">
            This agent is saved on disk but not running. Load it to chat, or delete to remove its cache entry.
          </p>
        </div>

        <div className="p-6 border-t border-slack-border bg-slack-bgHover flex justify-between gap-2">
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => onLoad(agent)}
              disabled={loading || deleting}
              className="px-4 py-2 text-sm bg-slack-accent hover:bg-slack-accentHover text-white rounded transition-colors disabled:opacity-50"
            >
              {loading ? 'Loading…' : 'Load agent'}
            </button>
            <button
              type="button"
              onClick={() => onDelete(agent)}
              disabled={loading || deleting}
              className="px-4 py-2 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors border border-red-500/30 disabled:opacity-50"
            >
              {deleting ? 'Deleting…' : 'Delete permanently'}
            </button>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm text-slack-textMuted hover:text-slack-text rounded transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
