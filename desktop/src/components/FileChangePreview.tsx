import { useState } from 'react';
import type { FileChange } from '../types/protocol';
import { useFileChangeStore } from '../stores/fileChangeStore';

interface FileChangePreviewProps {
  change: FileChange;
  onClose: () => void;
  onApprove: (changeId: string) => void;
  onReject: (changeId: string, reason: string) => void;
}

const codeBox = 'bg-[#0f1115] border border-slack-border rounded p-3 text-sm overflow-x-auto text-slack-text';

export function FileChangePreview({ change, onClose, onApprove, onReject }: FileChangePreviewProps) {
  const [rejectReason, setRejectReason] = useState('');
  const [showRejectForm, setShowRejectForm] = useState(false);
  const { previewData, loading } = useFileChangeStore();

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
          <div className="animate-spin rounded-full h-8 w-8 border-2 border-slack-border border-t-slack-accent" />
          <span className="ml-2 text-sm text-slack-textMuted">Loading preview...</span>
        </div>
      );
    }

    switch (change.operation) {
      case 'create':
        return (
          <div className="space-y-4">
            <div className="rounded-lg border border-emerald-500/30 bg-emerald-950/20 p-4">
              <h4 className="font-semibold text-emerald-300 mb-2">New File Content</h4>
              <pre className={codeBox}>
                <code>{change.new_content}</code>
              </pre>
            </div>
          </div>
        );

      case 'edit':
        return (
          <div className="space-y-4">
            {previewData?.diff ? (
              <div className="rounded-lg border border-sky-500/30 bg-sky-950/20 p-4">
                <h4 className="font-semibold text-sky-300 mb-2">Changes (Diff)</h4>
                <pre className={`${codeBox} whitespace-pre-wrap`}>
                  <code>{previewData.diff}</code>
                </pre>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="rounded-lg border border-red-500/30 bg-red-950/20 p-4">
                  <h4 className="font-semibold text-red-300 mb-2">Old Content</h4>
                  <pre className={codeBox}>
                    <code>{change.old_content}</code>
                  </pre>
                </div>
                <div className="rounded-lg border border-emerald-500/30 bg-emerald-950/20 p-4">
                  <h4 className="font-semibold text-emerald-300 mb-2">New Content</h4>
                  <pre className={codeBox}>
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
            <div className="rounded-lg border border-red-500/40 bg-red-950/25 p-4">
              <div className="flex items-center mb-2">
                <span className="text-red-400 text-xl mr-2">⚠️</span>
                <h4 className="font-bold text-red-200">File Deletion</h4>
              </div>
              <p className="text-red-200/90 mb-3 text-sm">
                This will permanently delete the file. A backup will be created before deletion.
              </p>
              <div className="rounded border border-red-500/30 bg-[#0f1115] p-3">
                <h5 className="font-semibold text-red-300 mb-2 text-sm">File to be deleted:</h5>
                <pre className="text-sm overflow-x-auto text-slack-text">
                  <code>{change.file_path}</code>
                </pre>
              </div>
            </div>
          </div>
        );

      case 'move':
        return (
          <div className="space-y-4">
            <div className="rounded-lg border border-violet-500/30 bg-violet-950/20 p-4">
              <h4 className="font-semibold text-violet-300 mb-2">File Move</h4>
              <div className="space-y-2">
                <div>
                  <span className="text-xs font-medium text-slack-textMuted">From:</span>
                  <pre className={`${codeBox} mt-1`}>
                    <code>{change.old_path}</code>
                  </pre>
                </div>
                <div>
                  <span className="text-xs font-medium text-slack-textMuted">To:</span>
                  <pre className={`${codeBox} mt-1`}>
                    <code>{change.new_path}</code>
                  </pre>
                </div>
              </div>
            </div>
          </div>
        );

      default:
        return (
          <div className="text-center text-slack-textMuted p-8 text-sm">
            Unknown operation type: {change.operation}
          </div>
        );
    }
  };

  const expiresLabel = formatTimeRemaining(change.expires_at);
  const expired = expiresLabel === 'Expired';

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50">
      <div className="bg-slack-bg border border-slack-border rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] flex flex-col text-slack-text">
        <div className="flex items-center justify-between p-4 border-b border-slack-border bg-slack-bgHover/50">
          <div className="flex items-center space-x-3 min-w-0">
            <span className="text-2xl shrink-0">{getOperationIcon(change.operation)}</span>
            <div className="min-w-0">
              <h3 className="text-lg font-semibold truncate">
                {change.operation.charAt(0).toUpperCase() + change.operation.slice(1)} File
              </h3>
              <p className="text-sm text-slack-textMuted truncate">{change.file_path}</p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover text-xl leading-none shrink-0"
            aria-label="Close preview"
          >
            ×
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          <div className="mb-6 rounded-lg border border-slack-border bg-slack-bgHover/40 p-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="font-medium text-slack-textMuted">Agent:</span>
                <span className="ml-2 text-slack-text">{change.agent.name}</span>
              </div>
              <div>
                <span className="font-medium text-slack-textMuted">Channel:</span>
                <span className="ml-2 text-slack-text">{change.channel}</span>
              </div>
              <div>
                <span className="font-medium text-slack-textMuted">Requested:</span>
                <span className="ml-2 text-slack-text">{new Date(change.requested_at).toLocaleString()}</span>
              </div>
              <div>
                <span className="font-medium text-slack-textMuted">Expires:</span>
                <span className={`ml-2 ${expired ? 'text-red-400' : 'text-slack-text'}`}>{expiresLabel}</span>
              </div>
            </div>
          </div>

          {renderContent()}
        </div>

        <div className="border-t border-slack-border p-4 bg-slack-bgHover/30">
          {showRejectForm ? (
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-slack-textMuted mb-1">Rejection Reason</label>
                <input
                  type="text"
                  value={rejectReason}
                  onChange={(e) => setRejectReason(e.target.value)}
                  placeholder="Enter reason for rejection..."
                  className="w-full px-3 py-2 rounded-md border border-slack-border bg-slack-bg text-slack-text placeholder:text-slack-textMuted focus:outline-none focus:ring-2 focus:ring-slack-accent"
                />
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
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
                  type="button"
                  onClick={() => setShowRejectForm(false)}
                  className="px-4 py-2 rounded-md border border-slack-border bg-slack-bgHover text-slack-text hover:bg-slack-border focus:outline-none focus:ring-2 focus:ring-slack-accent"
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <div className="flex justify-between gap-2 flex-wrap">
              <div className="flex gap-2 flex-wrap">
                {change.operation === 'delete' ? (
                  <button
                    type="button"
                    onClick={() => onApprove(change.id)}
                    className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
                  >
                    Approve Delete
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => onApprove(change.id)}
                    className="px-4 py-2 bg-emerald-600 text-white rounded-md hover:bg-emerald-700 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  >
                    Approve
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setShowRejectForm(true)}
                  className="px-4 py-2 rounded-md border border-slack-border bg-slack-bgHover text-slack-text hover:bg-slack-border focus:outline-none focus:ring-2 focus:ring-slack-accent"
                >
                  Reject
                </button>
              </div>
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 rounded-md border border-slack-border bg-slack-bg text-slack-text hover:bg-slack-bgHover focus:outline-none focus:ring-2 focus:ring-slack-accent"
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
