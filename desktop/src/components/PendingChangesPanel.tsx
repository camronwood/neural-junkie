import { useEffect, useState } from 'react';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { FileChangeCard } from './FileChangeCard';
import { FileChangePreview } from './FileChangePreview';
import type { FileChange } from '../types/protocol';

interface PendingChangesPanelProps {
  onClose: () => void;
}

export function PendingChangesPanel({ onClose }: PendingChangesPanelProps) {
  const {
    pendingChanges,
    loading,
    error,
    fetchPendingChanges,
    approveChange,
    rejectChange,
    selectChange,
    clearError,
    refreshChanges,
  } = useFileChangeStore();

  const [selectedChange, setSelectedChange] = useState<FileChange | null>(null);

  useEffect(() => {
    fetchPendingChanges();
  }, [fetchPendingChanges]);

  useEffect(() => {
    const interval = setInterval(() => {
      refreshChanges();
    }, 30000);

    return () => clearInterval(interval);
  }, [refreshChanges]);

  const handlePreview = (change: FileChange) => {
    setSelectedChange(change);
    selectChange(change.id);
  };

  const handleClosePreview = () => {
    setSelectedChange(null);
    selectChange(null);
  };

  const handleApprove = async (changeId: string) => {
    try {
      await approveChange(changeId);
      if (selectedChange?.id === changeId) {
        handleClosePreview();
      }
    } catch (error) {
      console.error('Failed to approve change:', error);
    }
  };

  const handleReject = async (changeId: string, reason: string) => {
    try {
      await rejectChange(changeId, reason);
      if (selectedChange?.id === changeId) {
        handleClosePreview();
      }
    } catch (error) {
      console.error('Failed to reject change:', error);
    }
  };

  const getOperationCounts = () => {
    const changes = Array.isArray(pendingChanges) ? pendingChanges : [];

    const counts = {
      create: 0,
      edit: 0,
      delete: 0,
      move: 0,
      total: changes.length,
    };

    changes.forEach(change => {
      counts[change.operation as keyof typeof counts]++;
    });

    return counts;
  };

  const counts = getOperationCounts();

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
    >
      <div className="bg-slack-bg border border-slack-border rounded-lg shadow-xl max-w-6xl w-full mx-4 max-h-[90vh] flex flex-col text-slack-text">
        <div className="flex items-center justify-between p-4 border-b border-slack-border bg-slack-bgHover/50">
          <div>
            <h2 className="text-lg font-semibold text-slack-text">Pending File Changes</h2>
            <p className="text-sm text-slack-textMuted mt-0.5">
              {counts.total} pending changes
              {counts.create > 0 && ` • ${counts.create} create`}
              {counts.edit > 0 && ` • ${counts.edit} edit`}
              {counts.delete > 0 && ` • ${counts.delete} delete`}
              {counts.move > 0 && ` • ${counts.move} move`}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={refreshChanges}
              disabled={loading}
              className="px-3 py-1.5 text-xs rounded bg-slack-accent text-white hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-slack-accent disabled:opacity-50"
            >
              {loading ? '⟳' : '🔄'} Refresh
            </button>
            <button
              type="button"
              onClick={onClose}
              className="p-1.5 rounded text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover text-xl leading-none"
              aria-label="Close pending changes panel"
            >
              ×
            </button>
          </div>
        </div>

        {error && (
          <div className="mx-4 mt-4 p-3 rounded-lg border border-red-500/40 bg-red-950/30">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center min-w-0">
                <span className="text-red-400 mr-2 shrink-0">⚠️</span>
                <span className="text-red-200 text-sm break-words">{error}</span>
              </div>
              <button
                type="button"
                onClick={clearError}
                className="text-red-400 hover:text-red-300 shrink-0"
              >
                ×
              </button>
            </div>
          </div>
        )}

        <div className="flex-1 overflow-y-auto p-4">
          {loading && (Array.isArray(pendingChanges) ? pendingChanges.length === 0 : true) ? (
            <div className="flex items-center justify-center p-8">
              <div className="animate-spin rounded-full h-8 w-8 border-2 border-slack-border border-t-slack-accent" />
              <span className="ml-2 text-sm text-slack-textMuted">Loading changes...</span>
            </div>
          ) : (Array.isArray(pendingChanges) ? pendingChanges.length === 0 : true) ? (
            <div className="text-center text-slack-textMuted p-8">
              <div className="text-4xl mb-4">📝</div>
              <h3 className="text-lg font-medium mb-2 text-slack-text">No Pending Changes</h3>
              <p className="text-sm">All file changes have been processed.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {(Array.isArray(pendingChanges) ? pendingChanges : []).map((change) => (
                <FileChangeCard
                  key={change.id}
                  change={change}
                  onPreview={handlePreview}
                  onApprove={handleApprove}
                  onReject={handleReject}
                />
              ))}
            </div>
          )}
        </div>

        <div className="border-t border-slack-border p-4 bg-slack-bgHover/40">
          <div className="flex items-center justify-between text-xs text-slack-textMuted">
            <div>
              Tip: use Preview to review changes before approving.
            </div>
            <div>Auto-refresh: 30s</div>
          </div>
        </div>
      </div>

      {selectedChange && (
        <FileChangePreview
          change={selectedChange}
          onClose={handleClosePreview}
          onApprove={handleApprove}
          onReject={handleReject}
        />
      )}
    </div>
  );
}
