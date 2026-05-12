import type { FileChange } from '../types/protocol';

interface FileChangeCardProps {
  change: FileChange;
  onPreview: (change: FileChange) => void;
  onApprove: (changeId: string) => void;
  onReject: (changeId: string, reason: string) => void;
}

export function FileChangeCard({ change, onPreview, onApprove, onReject }: FileChangeCardProps) {
  const getOperationIcon = (operation: string) => {
    switch (operation) {
      case 'create':
        return '📄';
      case 'edit':
        return '✏️';
      case 'delete':
        return '🗑️';
      case 'move':
        return '📁';
      default:
        return '📄';
    }
  };

  const getOperationStyles = (operation: string) => {
    switch (operation) {
      case 'create':
        return 'border-emerald-500/35 bg-emerald-950/15';
      case 'edit':
        return 'border-sky-500/35 bg-sky-950/15';
      case 'delete':
        return 'border-red-500/35 bg-red-950/15';
      case 'move':
        return 'border-violet-500/35 bg-violet-950/15';
      default:
        return 'border-slack-border bg-slack-bgHover/40';
    }
  };

  const formatTimeRemaining = (expiresAt: string) => {
    const now = new Date();
    const expires = new Date(expiresAt);
    const diff = expires.getTime() - now.getTime();

    if (diff <= 0) return { text: 'Expired', color: 'text-red-400' };

    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) {
      return { text: `${hours}h ${minutes % 60}m`, color: 'text-slack-textMuted' };
    }
    return { text: `${minutes}m`, color: 'text-slack-textMuted' };
  };

  const getDisplayPath = () => {
    if (change.operation === 'move' && change.old_path && change.new_path) {
      return `${change.old_path} → ${change.new_path}`;
    }
    return change.file_path;
  };

  const timeInfo = formatTimeRemaining(change.expires_at);
  const isExpired = timeInfo.text === 'Expired';

  return (
    <div
      className={`border rounded-lg p-4 mb-3 text-slack-text ${getOperationStyles(change.operation)} ${
        isExpired ? 'opacity-60' : ''
      }`}
    >
      <div className="flex items-start justify-between mb-3 gap-2">
        <div className="flex items-center space-x-2 min-w-0">
          <span className="text-lg shrink-0">{getOperationIcon(change.operation)}</span>
          <div className="min-w-0">
            <h4 className="font-semibold text-sm text-slack-text">
              {change.operation.charAt(0).toUpperCase() + change.operation.slice(1)} File
            </h4>
            <p className="text-xs text-slack-textMuted truncate">{change.agent.name}</p>
          </div>
        </div>
        <div className="text-right shrink-0">
          <div className={`text-xs ${timeInfo.color}`}>{timeInfo.text}</div>
          <div className="text-xs text-slack-textMuted">
            {new Date(change.requested_at).toLocaleTimeString()}
          </div>
        </div>
      </div>

      <div className="mb-3">
        <div className="text-xs font-medium text-slack-textMuted mb-1">File</div>
        <div className="text-sm font-mono bg-[#0f1115] border border-slack-border rounded px-2 py-1 break-all text-slack-text">
          {getDisplayPath()}
        </div>
      </div>

      <div className="mb-3">
        <div className="text-xs text-slack-textMuted">
          Channel: <span className="font-medium text-slack-text">{change.channel}</span>
        </div>
      </div>

      <div className="flex items-center justify-between gap-2 flex-wrap">
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => onPreview(change)}
            className="px-3 py-1 text-xs rounded border border-slack-border bg-slack-bgHover text-slack-text hover:bg-slack-border focus:outline-none focus:ring-2 focus:ring-slack-accent disabled:opacity-50"
            disabled={isExpired}
          >
            Preview
          </button>

          {change.operation === 'delete' ? (
            <button
              type="button"
              onClick={() => onApprove(change.id)}
              className="px-3 py-1 text-xs rounded bg-red-600 text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:opacity-50"
              disabled={isExpired}
            >
              Approve Delete
            </button>
          ) : (
            <button
              type="button"
              onClick={() => onApprove(change.id)}
              className="px-3 py-1 text-xs rounded bg-emerald-600 text-white hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-emerald-500 disabled:opacity-50"
              disabled={isExpired}
            >
              Approve
            </button>
          )}

          <button
            type="button"
            onClick={() => onReject(change.id, 'No reason provided')}
            className="px-3 py-1 text-xs rounded border border-slack-border bg-slack-bg text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover focus:outline-none focus:ring-2 focus:ring-slack-accent disabled:opacity-50"
            disabled={isExpired}
          >
            Reject
          </button>
        </div>

        <div className="flex items-center gap-1">
          {isExpired && <span className="text-xs text-red-400 font-medium">EXPIRED</span>}
          {change.operation === 'delete' && (
            <span className="text-xs text-red-400 font-medium">DELETE</span>
          )}
        </div>
      </div>
    </div>
  );
}
