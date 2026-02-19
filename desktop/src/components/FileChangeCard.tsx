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
      case 'create': return '📄';
      case 'edit': return '✏️';
      case 'delete': return '🗑️';
      case 'move': return '📁';
      default: return '📄';
    }
  };

  const getOperationColor = (operation: string) => {
    switch (operation) {
      case 'create': return 'text-green-600 bg-green-50 border-green-200';
      case 'edit': return 'text-blue-600 bg-blue-50 border-blue-200';
      case 'delete': return 'text-red-600 bg-red-50 border-red-200';
      case 'move': return 'text-purple-600 bg-purple-50 border-purple-200';
      default: return 'text-gray-600 bg-gray-50 border-gray-200';
    }
  };

  const formatTimeRemaining = (expiresAt: string) => {
    const now = new Date();
    const expires = new Date(expiresAt);
    const diff = expires.getTime() - now.getTime();
    
    if (diff <= 0) return { text: 'Expired', color: 'text-red-600' };
    
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) {
      return { text: `${hours}h ${minutes % 60}m`, color: 'text-gray-600' };
    }
    return { text: `${minutes}m`, color: 'text-gray-600' };
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
    <div className={`border rounded-lg p-4 mb-3 ${getOperationColor(change.operation)} ${
      isExpired ? 'opacity-60' : ''
    }`}>
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center space-x-2">
          <span className="text-lg">{getOperationIcon(change.operation)}</span>
          <div>
            <h4 className="font-semibold text-sm">
              {change.operation.charAt(0).toUpperCase() + change.operation.slice(1)} File
            </h4>
            <p className="text-xs text-gray-600">{change.agent.name}</p>
          </div>
        </div>
        <div className="text-right">
          <div className={`text-xs ${timeInfo.color}`}>
            {timeInfo.text}
          </div>
          <div className="text-xs text-gray-500">
            {new Date(change.requested_at).toLocaleTimeString()}
          </div>
        </div>
      </div>

      {/* File Path */}
      <div className="mb-3">
        <div className="text-sm font-medium text-gray-700 mb-1">File:</div>
        <div className="text-sm font-mono bg-white border rounded px-2 py-1 break-all">
          {getDisplayPath()}
        </div>
      </div>

      {/* Channel */}
      <div className="mb-3">
        <div className="text-xs text-gray-500">
          Channel: <span className="font-medium">{change.channel}</span>
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between">
        <div className="flex space-x-2">
          <button
            onClick={() => onPreview(change)}
            className="px-3 py-1 text-xs bg-white border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
            disabled={isExpired}
          >
            👁️ Preview
          </button>
          
          {change.operation === 'delete' ? (
            <button
              onClick={() => onApprove(change.id)}
              className="px-3 py-1 text-xs bg-red-600 text-white rounded hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:opacity-50"
              disabled={isExpired}
            >
              🗑️ Approve Delete
            </button>
          ) : (
            <button
              onClick={() => onApprove(change.id)}
              className="px-3 py-1 text-xs bg-green-600 text-white rounded hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 disabled:opacity-50"
              disabled={isExpired}
            >
              ✅ Approve
            </button>
          )}
          
          <button
            onClick={() => onReject(change.id, 'No reason provided')}
            className="px-3 py-1 text-xs bg-gray-300 text-gray-700 rounded hover:bg-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-500 disabled:opacity-50"
            disabled={isExpired}
          >
            ❌ Reject
          </button>
        </div>

        {/* Status indicator */}
        <div className="flex items-center space-x-1">
          {isExpired && (
            <span className="text-xs text-red-600 font-medium">EXPIRED</span>
          )}
          {change.operation === 'delete' && (
            <span className="text-xs text-red-600 font-medium">⚠️ DELETE</span>
          )}
        </div>
      </div>
    </div>
  );
}
