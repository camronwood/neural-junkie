import { useState } from 'react';
import type { FileChange } from '../types/protocol';
import { useFileChangeStore } from '../stores/fileChangeStore';

interface FileChangePreviewProps {
  change: FileChange;
  onClose: () => void;
  onApprove: (changeId: string) => void;
  onReject: (changeId: string, reason: string) => void;
}

export function FileChangePreview({ change, onClose, onApprove, onReject }: FileChangePreviewProps) {
  const [rejectReason, setRejectReason] = useState('');
  const [showRejectForm, setShowRejectForm] = useState(false);
  const { previewData, loading } = useFileChangeStore();

  const getOperationIcon = (operation: string) => {
    switch (operation) {
      case 'create': return '📄';
      case 'edit': return '✏️';
      case 'delete': return '🗑️';
      case 'move': return '📁';
      default: return '📄';
    }
  };


  const formatTimeRemaining = (expiresAt: string) => {
    const now = new Date();
    const expires = new Date(expiresAt);
    const diff = expires.getTime() - now.getTime();
    
    if (diff <= 0) return 'Expired';
    
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) {
      return `${hours}h ${minutes % 60}m remaining`;
    }
    return `${minutes}m remaining`;
  };

  const renderContent = () => {
    if (loading) {
      return (
        <div className="flex items-center justify-center p-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <span className="ml-2 text-gray-600">Loading preview...</span>
        </div>
      );
    }

    switch (change.operation) {
      case 'create':
        return (
          <div className="space-y-4">
            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
              <h4 className="font-semibold text-green-800 mb-2">New File Content</h4>
              <pre className="bg-white border rounded p-3 text-sm overflow-x-auto">
                <code>{change.new_content}</code>
              </pre>
            </div>
          </div>
        );

      case 'edit':
        return (
          <div className="space-y-4">
            {previewData?.diff ? (
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <h4 className="font-semibold text-blue-800 mb-2">Changes (Diff)</h4>
                <pre className="bg-white border rounded p-3 text-sm overflow-x-auto whitespace-pre-wrap">
                  <code>{previewData.diff}</code>
                </pre>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                  <h4 className="font-semibold text-red-800 mb-2">Old Content</h4>
                  <pre className="bg-white border rounded p-3 text-sm overflow-x-auto">
                    <code>{change.old_content}</code>
                  </pre>
                </div>
                <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                  <h4 className="font-semibold text-green-800 mb-2">New Content</h4>
                  <pre className="bg-white border rounded p-3 text-sm overflow-x-auto">
                    <code>{change.new_content}</code>
                  </pre>
                </div>
              </div>
            )}
          </div>
        );

      case 'delete':
        return (
          <div className="space-y-4">
            <div className="bg-red-100 border-2 border-red-300 rounded-lg p-4">
              <div className="flex items-center mb-2">
                <span className="text-red-600 text-xl mr-2">⚠️</span>
                <h4 className="font-bold text-red-800">DANGER: File Deletion</h4>
              </div>
              <p className="text-red-700 mb-3">
                This will permanently delete the file. A backup will be created before deletion.
              </p>
              <div className="bg-white border border-red-200 rounded p-3">
                <h5 className="font-semibold text-red-800 mb-2">File to be deleted:</h5>
                <pre className="text-sm overflow-x-auto">
                  <code>{change.file_path}</code>
                </pre>
              </div>
            </div>
          </div>
        );

      case 'move':
        return (
          <div className="space-y-4">
            <div className="bg-purple-50 border border-purple-200 rounded-lg p-4">
              <h4 className="font-semibold text-purple-800 mb-2">File Move</h4>
              <div className="space-y-2">
                <div>
                  <span className="text-sm font-medium text-gray-600">From:</span>
                  <pre className="bg-white border rounded p-2 text-sm mt-1">
                    <code>{change.old_path}</code>
                  </pre>
                </div>
                <div>
                  <span className="text-sm font-medium text-gray-600">To:</span>
                  <pre className="bg-white border rounded p-2 text-sm mt-1">
                    <code>{change.new_path}</code>
                  </pre>
                </div>
              </div>
            </div>
          </div>
        );

      default:
        return (
          <div className="text-center text-gray-500 p-8">
            Unknown operation type: {change.operation}
          </div>
        );
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center space-x-3">
            <span className="text-2xl">{getOperationIcon(change.operation)}</span>
            <div>
              <h3 className="text-lg font-semibold">
                {change.operation.charAt(0).toUpperCase() + change.operation.slice(1)} File
              </h3>
              <p className="text-sm text-gray-600">{change.file_path}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded text-gray-500 hover:text-gray-700 hover:bg-gray-100 text-xl leading-none"
            aria-label="Close preview"
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {/* Change Info */}
          <div className="mb-6 bg-gray-50 rounded-lg p-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="font-medium text-gray-600">Agent:</span>
                <span className="ml-2">{change.agent.name}</span>
              </div>
              <div>
                <span className="font-medium text-gray-600">Channel:</span>
                <span className="ml-2">{change.channel}</span>
              </div>
              <div>
                <span className="font-medium text-gray-600">Requested:</span>
                <span className="ml-2">{new Date(change.requested_at).toLocaleString()}</span>
              </div>
              <div>
                <span className="font-medium text-gray-600">Expires:</span>
                <span className={`ml-2 ${formatTimeRemaining(change.expires_at).includes('Expired') ? 'text-red-600' : 'text-gray-900'}`}>
                  {formatTimeRemaining(change.expires_at)}
                </span>
              </div>
            </div>
          </div>

          {/* Change Content */}
          {renderContent()}
        </div>

        {/* Actions */}
        <div className="border-t p-4">
          {showRejectForm ? (
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Rejection Reason
                </label>
                <input
                  type="text"
                  value={rejectReason}
                  onChange={(e) => setRejectReason(e.target.value)}
                  placeholder="Enter reason for rejection..."
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex space-x-2">
                <button
                  onClick={() => {
                    onReject(change.id, rejectReason);
                    setShowRejectForm(false);
                    setRejectReason('');
                  }}
                  className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
                >
                  Reject
                </button>
                <button
                  onClick={() => setShowRejectForm(false)}
                  className="px-4 py-2 bg-gray-300 text-gray-700 rounded-md hover:bg-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-500"
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <div className="flex justify-between">
              <div className="flex space-x-2">
                {change.operation === 'delete' ? (
                  <button
                    onClick={() => onApprove(change.id)}
                    className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
                  >
                    🗑️ Approve Delete
                  </button>
                ) : (
                  <button
                    onClick={() => onApprove(change.id)}
                    className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500"
                  >
                    ✅ Approve
                  </button>
                )}
                <button
                  onClick={() => setShowRejectForm(true)}
                  className="px-4 py-2 bg-gray-300 text-gray-700 rounded-md hover:bg-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-500"
                >
                  ❌ Reject
                </button>
              </div>
              <button
                onClick={onClose}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-500"
              >
                Close
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
